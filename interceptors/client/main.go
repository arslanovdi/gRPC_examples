package main

import (
	"context"
	pb "gRPC_examples/interceptors/pkg/ecommerce" // сгенерированный код
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log/slog"
	"os"
	"reflect"
	"time"

	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

// orderUnaryClientInterceptor перехватывает унарные методы
func orderUnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// TODO логика до вызова унарного метода
	slog.Info("Перехват до вызова унарного метода", slog.Any("метод", method))

	err := invoker(ctx, method, req, reply, cc, opts...) // вызываем удаленный метод

	// TODO логика после получения ответа
	slog.Info("Перехват после получения ответа", slog.Any("метод", method), slog.Any("ответ", reply))

	return err
}

// clientStreamInterceptor перехватывает потоковые методы
func clientStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	// TODO логика до вызова потокового метода
	slog.Info("Перехват до вызова потокового метода", slog.Any("метод", method))

	s, err := streamer(ctx, desc, cc, method, opts...) // вызываем удаленный метод
	if err != nil {
		return nil, err
	}
	return newWrappedStream(s), nil // возвращаем обернутый поток
}

// wrappedStream обертка над grpc.ClientStream
// перехватываем методы RecvMsg и SendMsg
type wrappedStream struct {
	grpc.ClientStream
}

// RecvMsg перехватчик RecvMsg
func (w *wrappedStream) RecvMsg(m interface{}) error {
	// TODO логика до приема сообщения RecvMsg
	slog.Info("Перехват до приема RecvMsg", slog.Any("type", reflect.TypeOf(m)))

	err := w.ClientStream.RecvMsg(m) // получаем сообщение с удаленного сервера

	// TODO логика после приема сообщения RecvMsg
	slog.Info("Перехват после приема RecvMsg", slog.Any("message", m), slog.Any("error", err))

	return err
}

// SendMsg перехватчик SendMsg
func (w *wrappedStream) SendMsg(m interface{}) error {
	// TODO логика до отправки сообщения SendMsg
	slog.Info("Перехват SendMsg", slog.Any("message", m))

	err := w.ClientStream.SendMsg(m) // отправляем сообщение на удаленный сервер

	// TODO логика после отправки сообщения SendMsg
	slog.Info("Перехват после отправки SendMsg", slog.Any("error", err))

	return err
}

// newWrappedStream конструктор обертки над grpc.ClientStream
func newWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStream{s}
}

func main() {
	// подключение к grpc серверу без TLS
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(orderUnaryClientInterceptor), // регистрация перехватчика осуществляется внутри grpc.Dial
		grpc.WithStreamInterceptor(clientStreamInterceptor))
	if err != nil {
		slog.Warn("did not connect", slog.Any("error", err)) // В моем случае ошибки не возникает даже при отключенном сервере, просто висит ConnectionState: Connecting
		os.Exit(1)
	} // Пока не понятно как диагностировать что сервер не поднят, пробный запрос развечто делать типа HealthCheck
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn) // инициализируем интерфейс через который будут вызываться удаленные методы

	// Add Order
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		order1 := pb.Order{Id: "101", Items: []string{"iPhone XS", "Mac Book Pro"}, Destination: "San Jose, CA", Price: 2300.00}
		res, err := client.AddOrder(ctx, &order1) // вызов метода AddOrder
		if err != nil {
			slog.Error("AddOrder() failed:", slog.Any("error", err))
		}
		if res != nil {
			slog.Info("AddOrder() Response", slog.String("message", res.Value))
		}
	}

	// Get Order
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		retrievedOrder, err := client.GetOrder(ctx, &wrappers.StringValue{Value: "106"}) // вызов метода GetOrder
		if err != nil {
			slog.Error("GetOrder() failed.", slog.Any("error", err))
		}
		slog.Info("GetOrder() Response", slog.Any("order", retrievedOrder))

	}

	// Search Order : потоковая передача на стороне сервера
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		searchStream, _ := client.SearchOrders(ctx, &wrappers.StringValue{Value: "Google"}) // получаем поток метода SearchOrders
		for {
			searchOrder, err := searchStream.Recv() // получаем сообщение из потока
			if err == io.EOF {                      // если поток завершился прилетает io.EOF
				slog.Info("SearchOrder() EOF")
				break
			}

			if err == nil {
				slog.Info("SearchOrder() result", slog.Any("order", searchOrder))
			}
		}
	}

	// Update Orders : // потоковый на стороне клиента
	{
		updOrder1 := pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}
		updOrder2 := pb.Order{Id: "103", Items: []string{"Apple Watch S4", "Mac Book Pro", "iPad Pro"}, Destination: "San Jose, CA", Price: 2800.00}
		updOrder3 := pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub", "iPad Mini"}, Destination: "Mountain View, CA", Price: 2200.00}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		updateStream, err := client.UpdateOrders(ctx) // вызов удаленного метода UpdateOrders, получаем поток
		if err != nil {
			slog.Warn("UpdateOrders() error calling method", slog.Any("error", err))
			os.Exit(1)
		}

		// Updating order 1
		if err := updateStream.Send(&updOrder1); err != nil { // отправка заказов в поток для обработки на сервере
			slog.Warn("UpdateOrders() sending", slog.Any("order", updOrder1), slog.Any("error", err))
			os.Exit(1)
		}

		// Updating order 2
		if err := updateStream.Send(&updOrder2); err != nil { // отправка заказов в поток для обработки на сервере
			slog.Warn("UpdateOrders() sending", slog.Any("order", updOrder2), slog.Any("error", err))
			os.Exit(1)
		}

		// Updating order 3
		if err := updateStream.Send(&updOrder3); err != nil { // отправка заказов в поток для обработки на сервере
			slog.Warn("UpdateOrders() sending", slog.Any("order", updOrder3), slog.Any("error", err))
			os.Exit(1)
		}

		updateRes, err := updateStream.CloseAndRecv() // Закрытие потока и получение результата
		if err != nil {
			slog.Warn("UpdateOrders() closing and receiving", slog.Any("error", err))
			os.Exit(1)
		}
		slog.Info("UpdateOrders()", slog.Any("result", updateRes))
	}

	// Process Order : двусторонняя потоковая передача
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		streamProcOrder, err := client.ProcessOrders(ctx)
		if err != nil {
			slog.Warn("ProcessOrders() error calling", slog.Any("error", err))
			os.Exit(1)
		}

		channel := make(chan struct{})
		go readingProcessOrders(streamProcOrder, channel) // запуск горутины для получения ответа от сервера

		if err := streamProcOrder.Send(&wrappers.StringValue{Value: "102"}); err != nil {
			slog.Warn("ProcessOrders() error sending", slog.Int("ID", 102), slog.Any("error", err))
			os.Exit(1)
		}

		if err := streamProcOrder.Send(&wrappers.StringValue{Value: "103"}); err != nil {
			slog.Warn("ProcessOrders() error sending", slog.Int("ID", 103), slog.Any("error", err))
			os.Exit(1)
		}

		if err := streamProcOrder.Send(&wrappers.StringValue{Value: "104"}); err != nil {
			slog.Warn("ProcessOrders() error sending", slog.Int("ID", 104), slog.Any("error", err))
			os.Exit(1)
		}

		if err := streamProcOrder.Send(&wrappers.StringValue{Value: "101"}); err != nil {
			slog.Warn("ProcessOrders() error sending", slog.Int("ID", 101), slog.Any("error", err))
			os.Exit(1)
		}

		if err := streamProcOrder.Send(&wrappers.StringValue{Value: "105"}); err != nil {
			slog.Warn("ProcessOrders() error sending", slog.Int("ID", 105), slog.Any("error", err))
			os.Exit(1)
		}

		if err := streamProcOrder.Send(&wrappers.StringValue{Value: "108"}); err != nil {
			slog.Warn("ProcessOrders() error sending", slog.Int("ID", 108), slog.Any("error", err))
			os.Exit(1)
		}
		if err := streamProcOrder.CloseSend(); err != nil {
			slog.Warn("ProcessOrders()", slog.Any("error", err))
			os.Exit(1)
		}
		channel <- struct{}{} // завершение работы горутины
	}
}

// readingProcessOrders метод для получения ответа от сервера, запускаем в отдельной горутине
func readingProcessOrders(streamProcOrder pb.OrderManagement_ProcessOrdersClient, c chan struct{}) {
	for {
		combinedShipment, errProcOrder := streamProcOrder.Recv()
		if errProcOrder == io.EOF {
			break
		}
		if combinedShipment == nil {
			slog.Warn("Combined shipment", slog.String("is nil", "true"))
			break
		}
		slog.Info("Combined shipment", slog.Any("order list", combinedShipment.OrdersList))
	}
	<-c
}
