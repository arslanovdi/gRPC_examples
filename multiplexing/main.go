package main

import (
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"os"

	one "gRPC_examples/multiplexing/pkg/services/first"  // путь к сгенерированному пакету с 1-м сервисом
	two "gRPC_examples/multiplexing/pkg/services/second" // путь к сгенерированному пакету со 2-м сервисом
)

const (
	port = ":50051"
)

// NewFirstServer заглушка реализации первого сервиса
func NewFirstServer() one.OrderManagementServer {
	var s one.OrderManagementServer
	return s
}

// NewSecondServer заглушка реализации второго сервиса
func NewSecondServer() two.OrderManagementServer {
	var s two.OrderManagementServer
	return s
}

func main() {
	lis, err := net.Listen("tcp", port) // листенер к которому привяжем grpc-сервер
	if err != nil {
		slog.Warn("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}
	s := grpc.NewServer()                                   // создаем grpc-сервер
	one.RegisterOrderManagementServer(s, NewFirstServer())  // регистрируем имплементацию первого сервиса в grpc-сервер
	two.RegisterOrderManagementServer(s, NewSecondServer()) // регистрируем имплементацию второго сервиса в grpc-сервер

	if err := s.Serve(lis); err != nil { // запускаем grpc-сервер на листенере
		slog.Warn("failed to serve", slog.Any("error", err))
	}
}
