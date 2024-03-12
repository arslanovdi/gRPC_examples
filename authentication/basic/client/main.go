package main

import (
	"context"
	"encoding/base64"
	"gRPC_examples/authentication/basic/pkg/ecommerce"
	"google.golang.org/grpc/credentials"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
)

const (
	address  = "localhost:50051"
	hostname = "localhost"
)

var certFile = filepath.Join("authentication", "tls", "serverTLS", "certs", "server.crt")

func main() {
	creds, err := credentials.NewClientTLSFromFile(certFile, hostname) // загружаем и проверяем публичный сертификат
	if err != nil {
		slog.Warn("failed to load credentials", slog.Any("error", err))
		os.Exit(1)
	}

	auth := basicAuth{ // инициализируем структуру, поля логин/пароль
		username: "root",
		password: "P@$$w0rd",
	}

	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(auth),     // аутентификаци при помощи нашей реализации интерфейса credentials.PerRPCCredentials
		grpc.WithTransportCredentials(creds)} // задаем параметры подключения к серверу, с TLS

	// подключение к grpc серверу с TLS и базовой аутентификацией
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		slog.Warn("did not connect", slog.Any("error", err)) // В моем случае ошибки не возникает даже при отключенном сервере, просто висит ConnectionState: Connecting
		os.Exit(1)
	} // Пока не понятно как диагностировать что сервер не поднят, пробный запрос развечто делать типа HealthCheck
	defer conn.Close()

	client := ecommerce_v1.NewOrderManagementClient(conn) // инициализируем интерфейс через который будут вызываться удаленные методы

	// Add Order
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		order1 := ecommerce_v1.Order{Id: "101", Items: []string{"iPhone XS", "Mac Book Pro"}, Destination: "San Jose, CA", Price: 2300.00}
		res, err := client.AddOrder(ctx, &order1) // вызов метода AddOrder
		if err != nil {
			slog.Error("AddOrder() failed:", slog.Any("error", err))
		}
		if res != nil {
			slog.Info("AddOrder() Response", slog.String("message", res.Value))
		}
	}

}

// basicAuth имплементация интерфейса credentials.PerRPCCredentials
type basicAuth struct {
	username string // поля внедряются в удаленные вызовы
	password string
}

// GetRequestMetadata преобразует пару логин/пароль в метаданные запроса (мапу)
func (b basicAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	auth := b.username + ":" + b.password
	enc := base64.StdEncoding.EncodeToString([]byte(auth))
	return map[string]string{
		"authorization": "Basic " + enc,
	}, nil
}

// RequireTransportSecurity обязательно используем TLS
func (b basicAuth) RequireTransportSecurity() bool {
	return true
}
