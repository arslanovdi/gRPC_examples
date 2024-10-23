package main

import (
	"context"
	"fmt"
	pb "gRPC_examples/communication_patterns/pkg/ecommerce" // сгенерированный код
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log/slog"
	"os"
	"sync"
	"time"

	"net"
	"strings"
)

const (
	port = ":50051"
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
func (s *ecommerceServer) AddOrder(ctx context.Context, orderReq *pb.Order) (*wrappers.StringValue, error) {
	// метаданные
	{
		md, metadataAvailable := metadata.FromIncomingContext(ctx) // Получение метаданных от клиента
		if !metadataAvailable {
			return nil, status.Errorf(codes.DataLoss, "UnaryEcho: failed to get metadata")
		}
		if t, ok := md["timestamp"]; ok { // читаем поле timestamp
			slog.Info("timestamp from metadata")
			for _, e := range t {
				slog.Info("metadata from client", "timestamp", e)
			}
		}

		header := metadata.New(map[string]string{"location": "San Jose", "timestamp": time.Now().Format(time.RFC822)}) // Создание метаданных типа key-value
		grpc.SendHeader(ctx, header)                                                                                   // Отправка метаданных
	}

	slog.Info("AddOrder() order added", "ID", orderReq.Id)
	s.mu.Lock()
	s.orderMap[orderReq.Id] = *orderReq
	s.mu.Unlock()
	//time.Sleep(time.Second * 6) // Simulate processing тест дедлайна контекста
	return &wrappers.StringValue{Value: "Order Added: " + orderReq.Id}, nil
}

// SearchOrders
// streaming метод со стороны сервера
func (s *ecommerceServer) SearchOrders(searchQuery *wrappers.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {
	// метаданные
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.RFC822))
		stream.SetTrailer(trailer) // отправляем метаданные в заключительном блоке потока
	}()
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.RFC822)}) // создание метаданных типа key-value
	stream.SendHeader(header)                                                                                 // отправляем метаданные в виде заголовка в потоке

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

func main() {
	lis, err := net.Listen("tcp", port) // листенер к которому привяжем grpc-сервер
	if err != nil {
		slog.Warn("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}
	s := grpc.NewServer()                                     // создаем grpc-сервер
	pb.RegisterOrderManagementServer(s, NewEcommerceServer()) // регистрируем имплементацию в grpc-сервер
	if err := s.Serve(lis); err != nil {                      // запускаем grpc-сервер на листенере
		slog.Warn("failed to serve", slog.Any("error", err))
	}
}
