package main

import (
	"context"
	"gRPC_examples/authentication/jwt/myJWT"
	"gRPC_examples/authentication/oauth2.0/pkg/ecommerce"
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

	jwtToken, err := myJWT.NewToken("root", 5*time.Minute) // токен нужно получить от сервера аутентификации, для примера просто создаю его
	if err != nil {
		slog.Warn("failed to create JWT token", slog.Any("error", err))
		os.Exit(1)
	}

	jwtCreds, err := NewJWTAccessFromKey([]byte(jwtToken))
	if err != nil {
		slog.Warn("failed to create jwy credentials", slog.Any("error", err))
		os.Exit(1)
	}

	opts := []grpc.DialOption{
		grpc.WithPerRPCCredentials(jwtCreds), // авторизация с JWT
		grpc.WithTransportCredentials(creds)} // задаем параметры подключения к серверу, с TLS

	// подключение к grpc серверу с TLS и oauth
	conn, err := grpc.NewClient(address, opts...)
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

// jwtCredentials структура, реализующая интерфейс credentials.PerRPCCredentials
type jwtCredentials struct {
	Token string
}

func NewJWTAccessFromKey(key []byte) (credentials.PerRPCCredentials, error) {
	return &jwtCredentials{
		Token: string(key),
	}, nil
}

func (c jwtCredentials) RequireTransportSecurity() bool {
	return true
}

func (c jwtCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": c.Token,
	}, nil
}
