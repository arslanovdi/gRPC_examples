package main

import (
	"bytes"
	"context"
	"encoding/json"
	pb "gRPC_examples/grpc-gateway/pkg/gen/go" // сгенерированный код
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
)

const (
	httpServerEndpoint = "localhost:8081"
	gRPCServerEndpoint = "localhost:50051"
)

func main() {

	// подключение к grpc серверу без TLS
	conn, err := grpc.Dial(gRPCServerEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Warn("did not connect", slog.Any("error", err)) // В моем случае ошибки не возникает даже при отключенном сервере, просто висит ConnectionState: Connecting
		os.Exit(1)
	} // Пока не понятно как диагностировать что сервер не поднят, пробный запрос развечто делать типа HealthCheck
	defer conn.Close()

	client := pb.NewOrderManagementClient(conn) // инициализируем интерфейс через который будут вызываться удаленные методы

	// Add Order by gRPC
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

	// Get Order by gRPC
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		retrievedOrder, err := client.GetOrder(ctx, &wrappers.StringValue{Value: "106"}) // вызов метода GetOrder
		if err != nil {
			slog.Error("GetOrder() failed.", slog.Any("error", err))
		}
		slog.Info("GetOrder() Response", slog.Any("order", retrievedOrder))

	}

	// Add Order by HTTP request
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		order2, err := json.Marshal(pb.Order{Id: "115", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00})
		if err != nil {
			slog.Error("error marshalling", slog.Any("error", err))
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8081/v1/order", bytes.NewReader(order2))
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := client.Do(request)
		defer resp.Body.Close()
		if err != nil {
			slog.Error("AddOrder() failed:", slog.Any("error", err))
		}
		slog.Info("AddOrder() Response", slog.Any("order", resp))

	}

	// Get Order by HTTP request
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8081/v1/order/106", nil)
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		resp, err := client.Do(request)
		defer resp.Body.Close()
		if err != nil {
			slog.Error("GetOrder() failed:", slog.Any("error", err))
		}

		order := pb.Order{}
		err = json.NewDecoder(resp.Body).Decode(&order)
		if err != nil {
			slog.Error("error unmarshalling", slog.Any("error", err))
		}

		slog.Info("GetOrder() Response", slog.Any("order", order))
	}

}
