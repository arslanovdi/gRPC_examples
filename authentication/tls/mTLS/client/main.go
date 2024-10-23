package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"gRPC_examples/authentication/tls/mTLS/pkg/ecommerce"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/credentials"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
)

var (
	address  = "localhost:50051"
	hostname = "localhost"
	crtFile  = filepath.Join("authentication", "tls", "mTLS", "certs", "client.crt")
	keyFile  = filepath.Join("authentication", "tls", "mTLS", "certs", "client.key")
	caFile   = filepath.Join("authentication", "tls", "mTLS", "certs", "ca.crt")
)

func main() {
	cert, err := tls.LoadX509KeyPair(crtFile, keyFile) // загружаем клиентский сертификат
	if err != nil {
		slog.Warn("failed to load certificate", slog.Any("error", err))
		os.Exit(1)
	}

	certPool := x509.NewCertPool() // создаем пул сертификатов (пустой)
	ca, err := os.ReadFile(caFile)
	if err != nil {
		slog.Warn("failed to read ca certificate", slog.Any("error", err))
		os.Exit(1)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok { // добавляем в пул сертификаты CA
		slog.Warn("failed to append ca certificate")
		os.Exit(1)
	}

	opts := []grpc.DialOption{ // задаем параметры подключения к серверу, с TLS
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      certPool,
			ServerName:   hostname, // обязательно, должно быть равно полю CommonName, указанному в сертификате
		}))}

	// подключение к grpc серверу с TLS
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

}
