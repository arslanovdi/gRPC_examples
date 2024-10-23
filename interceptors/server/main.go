package main

import (
	"context"
	"fmt"
	pb "gRPC_examples/interceptors/pkg/ecommerce" // сгенерированный код
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log/slog"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
)

const (
	port           = ":50051"
	orderBatchSize = 5
)

// ecommerceServer структура имплементирует интерфейс
// OrderManagementServer, который содержит методы описанные в ecommerce.proto
type ecommerceServer struct {
	orderMap                              map[string]pb.Order // TODO сохранять в мапу обьект сообщения плохая идея, тут это для примера
	mu                                    sync.Mutex
	pb.UnimplementedOrderManagementServer // обязательно встраивать структуру
}

// NewEcommerceServer конструктор
func NewEcommerceServer() *ecommerceServer {
	s := &ecommerceServer{
		orderMap: make(map[string]pb.Order),
	}
	s.initSampleData()
	return s
}

// initSampleData заполняет мапу тестовыми данными
func (s *ecommerceServer) initSampleData() {
	s.mu.Lock()
	s.orderMap["102"] = pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}
	s.orderMap["103"] = pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	s.orderMap["104"] = pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	s.orderMap["105"] = pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	s.orderMap["106"] = pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
	s.orderMap["107"] = pb.Order{Id: "107", Items: []string{"Apple MacBook Pro"}, Destination: "Mountain View, CA", Price: 600.00}
	s.orderMap["108"] = pb.Order{Id: "108", Items: []string{"Apple MacBook Air"}, Destination: "San Jose, CA", Price: 700.00}
	s.mu.Unlock()
}

// AddOrder Simple RPC
// одиночные (унарные) вызовы
func (s *ecommerceServer) AddOrder(_ context.Context, orderReq *pb.Order) (*wrappers.StringValue, error) {
	slog.Info("AddOrder() order added", "ID", orderReq.Id)

	s.mu.Lock()
	s.orderMap[orderReq.Id] = *orderReq
	s.mu.Unlock()
	//time.Sleep(time.Second * 6) // Simulate processing тест дедлайна контекста
	return &wrappers.StringValue{Value: "Order Added: " + orderReq.Id}, nil
}

// GetOrder Simple RPC
// одиночные (унарные) вызовы
func (s *ecommerceServer) GetOrder(_ context.Context, orderId *wrappers.StringValue) (*pb.Order, error) {
	s.mu.Lock()
	ord, exists := s.orderMap[orderId.Value]
	s.mu.Unlock()
	if exists {
		slog.Info("GetOrder() success", "ID", ord.Id)
		return &ord, status.New(codes.OK, "").Err()
	}
	slog.Info("GetOrder() failed", "ID", orderId.Value)
	return nil, status.Errorf(codes.NotFound, "Order does not exist. : ", orderId)

}

// SearchOrders
// streaming метод со стороны сервера
func (s *ecommerceServer) SearchOrders(searchQuery *wrappers.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, order := range s.orderMap {
		for _, itemStr := range order.Items {
			if strings.Contains(itemStr, searchQuery.Value) {
				err := stream.Send(&order) // Отправка сообщения клиенту
				if err != nil {
					return fmt.Errorf("error sending message to stream : %v", err)
				}
				slog.Info("SearchOrders() Order found", "ID", key)
				break
			}
		}
	}
	return nil // потоковая передача завершается отправкой nil
}

// UpdateOrders
// streaming метод со стороны клиента
func (s *ecommerceServer) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {

	ordersStr := "Updated Order IDs : "
	for {
		order, err := stream.Recv() // Получаем сообщение от клиента в потоке
		if err == io.EOF {          // Если прилетел io.EOF значит потоковый прием завершен
			return stream.SendAndClose(&wrappers.StringValue{Value: "Orders processed " + ordersStr}) // Отправляем ответ клиенту
		}

		if err != nil {
			return err
		}
		// Обработка сообщения
		{
			s.mu.Lock()
			s.orderMap[order.Id] = *order
			s.mu.Unlock()

			slog.Info("Order updated", "ID", order.Id)
			ordersStr += order.Id + ", "
		}
	}
}

// ProcessOrders
// двусторонний streaming метод
func (s *ecommerceServer) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {

	batchMarker := 1
	var combinedShipmentMap = make(map[string]pb.CombinedShipment)
	for {
		orderId, err := stream.Recv() // Получаем сообщение от клиента в потоке
		if err == io.EOF {            // Если прилетел io.EOF значит потоковый прием завершен, отправляем ответ клиенту
			slog.Info("ProcessOrders() end of receiving")
			for _, shipment := range combinedShipmentMap {
				if err := stream.Send(&shipment); err != nil { // отправка сообщения клиенту в потоке
					return err
				}
			}
			slog.Info("ProcessOrders() end of sending")
			return nil
		}
		if err != nil {
			slog.Error("Error n receiving", slog.Any("error", err))
			return err
		}
		slog.Info("ProcessOrders() Reading Proc order", "ID", orderId)

		s.mu.Lock()
		destination := s.orderMap[orderId.GetValue()].Destination
		s.mu.Unlock()
		shipment, found := combinedShipmentMap[destination]

		if found {
			s.mu.Lock()
			ord := s.orderMap[orderId.GetValue()]
			s.mu.Unlock()
			shipment.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = shipment
		} else {
			s.mu.Lock()
			comShip := pb.CombinedShipment{Id: "cmb - " + (s.orderMap[orderId.GetValue()].Destination), Status: "Processed!"}
			ord := s.orderMap[orderId.GetValue()]
			s.mu.Unlock()
			comShip.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = comShip
			slog.Info("ProcessOrders()", slog.Int("order list count", len(comShip.OrdersList)), slog.String("id", comShip.GetId()))
		}

		if batchMarker == orderBatchSize {
			for _, comb := range combinedShipmentMap {
				slog.Info("ProcessOrders() shipping", slog.String("ID", comb.Id), slog.Int("order list count", len(comb.OrdersList)))
				if err := stream.Send(&comb); err != nil {
					return err
				}
			}
			batchMarker = 0
			combinedShipmentMap = make(map[string]pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}

// orderUnaryServerInterceptor
// Перехватчик унарных методов
func orderUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// TODO логика до вызова унарного метода
	slog.Info("Перехват до вызова унарного метода", slog.Any("метод", info.FullMethod))

	m, err := handler(ctx, req) // вызываем обработчик метода

	// TODO логика до отправки ответа клиенту
	slog.Info("Перехват до отправки ответа", slog.Any("метод", info.FullMethod), slog.Any("ответ", m))

	return m, err
}

// orderServerStreamInterceptor
// Перехватчик потоковых методов
func orderServerStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// TODO логика до вызова потокового метода
	slog.Info("Перехват до вызова потокового метода", slog.Any("метод", info.FullMethod))

	err := handler(srv, newWrappedStream(ss)) // вызываем обработчик метода
	if err != nil {
		slog.Error("RPC failed with error", slog.Any("error", err))
	}

	return err
}

// wrappedStream обертка над grpc.ServerStream
// перехватываем методы RecvMsg и SendMsg
type wrappedStream struct {
	grpc.ServerStream
}

// RecvMsg перехватчик RecvMsg
func (w *wrappedStream) RecvMsg(m interface{}) error {
	// TODO логика до вызова RecvMsg
	slog.Info("Перехват до вызова RecvMsg", slog.Any("type", reflect.TypeOf(m)))

	err := w.ServerStream.RecvMsg(m)

	// TODO логика после вызова RecvMsg
	slog.Info("Перехват после вызова RecvMsg", slog.Any("message", m), slog.Any("error", err))

	return err
}

// SendMsg перехватчик SendMsg
func (w *wrappedStream) SendMsg(m interface{}) error {
	// TODO логика до вызова SendMsg
	slog.Info("Перехват SendMsg", slog.Any("message", m))

	err := w.ServerStream.SendMsg(m) // отправляем сообщение на удаленный сервер

	// TODO логика после вызова SendMsg
	slog.Info("Перехват после вызова SendMsg", slog.Any("error", err))

	return err
}

// newWrappedStream конструктор обертки над grpc.ServerStream
func newWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s}
}

func main() {
	lis, err := net.Listen("tcp", port) // листенер к которому привяжем grpc-сервер
	if err != nil {
		slog.Warn("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}
	s := grpc.NewServer( // создаем grpc-сервер
		grpc.UnaryInterceptor(orderUnaryServerInterceptor),   // регистрируем Unary Interceptor
		grpc.StreamInterceptor(orderServerStreamInterceptor), // регистрируем Stream Interceptor
	)
	pb.RegisterOrderManagementServer(s, NewEcommerceServer()) // регистрируем имплементацию в grpc-сервер
	if err := s.Serve(lis); err != nil {                      // запускаем grpc-сервер на листенере
		slog.Warn("failed to serve", slog.Any("error", err))
	}
}
