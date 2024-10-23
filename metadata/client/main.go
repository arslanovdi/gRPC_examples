package main

import (
	"context"
	pb "gRPC_examples/communication_patterns/pkg/ecommerce" // сгенерированный код
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"io"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// подключение к grpc серверу без TLS
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Warn("did not connect", slog.Any("error", err)) // В моем случае ошибки не возникает даже при отключенном сервере, просто висит ConnectionState: Connecting
		os.Exit(1)
	} // Пока не понятно как диагностировать что сервер не поднят, пробный запрос развечто делать типа HealthCheck
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn) // инициализируем интерфейс через который будут вызываться удаленные методы

	// Add Order
	{
		// metadata set
		md := metadata.Pairs( // создание метаданных типа key-[]value
			"timestamp", time.Now().Format(time.RFC822),
			"kn", "vn",
		)
		mdCtx := metadata.NewOutgoingContext(context.Background(), md)                      // создание контекста с метаданными
		ctxA := metadata.AppendToOutgoingContext(mdCtx, "k1", "v1", "k1", "v2", "k2", "v3") // добавление метаданных к контексту
		var header, trailer metadata.MD

		ctx, cancel := context.WithTimeout(ctxA, time.Second*5) // добавляем таймаут
		defer cancel()

		order1 := pb.Order{Id: "101", Items: []string{"iPhone XS", "Mac Book Pro"}, Destination: "San Jose, CA", Price: 2300.00}

		res, err := client.AddOrder(ctx, &order1, grpc.Header(&header), grpc.Trailer(&trailer)) // вызов метода AddOrder c метаданными в контексте, ответные метаданные добавляются в header и trailer

		if err != nil {
			slog.Error("AddOrder() failed:", slog.Any("error", err))
		}
		if res != nil {
			slog.Info("AddOrder() Response", slog.String("message", res.Value))
		}

		// metadata get
		{
			// читаем headers
			if t, ok := header["timestamp"]; ok {
				slog.Info("AddOrder() timestamp from header")
				for _, e := range t {
					slog.Info("metadata from server", "timestamp", e)
				}
			} else {
				slog.Warn("timestamp expected but doesn't exist in header")
				os.Exit(1)
			}

			if l, ok := header["location"]; ok {
				slog.Info("AddOrder() location from header")
				for _, e := range l {
					slog.Info("metadata from server", "location", e)
				}
			} else {
				slog.Warn("location expected but doesn't exist in header")
				os.Exit(1)
			}
		}
	}

	// Search Order : потоковая передача на стороне сервера
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		searchStream, _ := client.SearchOrders(ctx, &wrappers.StringValue{Value: "Google"}) // получаем поток метода SearchOrders

		// получаем заголовок потока
		header, err := searchStream.Header()
		if err != nil {
			slog.Warn("failed to get metadata")
			os.Exit(1)
		}
		{
			// читаем headers
			if t, ok := header["timestamp"]; ok {
				slog.Info("SearchOrder() timestamp from header")
				for _, e := range t {
					slog.Info("metadata from server", "timestamp", e)
				}
			} else {
				slog.Warn("timestamp expected but doesn't exist in header")
				os.Exit(1)
			}

			if l, ok := header["location"]; ok {
				slog.Info("SearchOrder() location from header")
				for _, e := range l {
					slog.Info("metadata from server", "location", e)
				}
			} else {
				slog.Warn("location expected but doesn't exist in header")
				os.Exit(1)
			}
		}

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
}
