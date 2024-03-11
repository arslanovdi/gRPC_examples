package main

import (
	"context"
	"crypto/tls"
	"gRPC_examples/authentication/tls/serverTLS/pkg/ecommerce"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"net"
)

var (
	port    = ":50051"
	crtFile = filepath.Join("authentication", "tls", "serverTLS", "certs", "server.crt")
	keyFile = filepath.Join("authentication", "tls", "serverTLS", "certs", "server.key")
)

// ecommerceServer структура имплементирует интерфейс
// OrderManagementServer, который содержит методы описанные в ecommerce.proto
type ecommerceServer struct {
	orderMap                                        map[string]ecommerce_v1.Order // сохранять в мапу обьект сообщения плохая идея, тут это для примера
	mu                                              sync.Mutex
	ecommerce_v1.UnimplementedOrderManagementServer // обязательно встраивать структуру
}

// NewEcommerceServer конструктор
func NewEcommerceServer() *ecommerceServer {
	s := &ecommerceServer{
		orderMap: make(map[string]ecommerce_v1.Order),
	}
	s.initSampleData()
	return s
}

// initSampleData заполняет мапу тестовыми данными
func (s *ecommerceServer) initSampleData() {
	s.mu.Lock()
	s.orderMap["102"] = ecommerce_v1.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}
	s.orderMap["103"] = ecommerce_v1.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	s.orderMap["104"] = ecommerce_v1.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	s.orderMap["105"] = ecommerce_v1.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	s.orderMap["106"] = ecommerce_v1.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
	s.orderMap["107"] = ecommerce_v1.Order{Id: "107", Items: []string{"Apple MacBook Pro"}, Destination: "Mountain View, CA", Price: 600.00}
	s.orderMap["108"] = ecommerce_v1.Order{Id: "108", Items: []string{"Apple MacBook Air"}, Destination: "San Jose, CA", Price: 700.00}
	s.mu.Unlock()
}

// AddOrder Simple RPC
// одиночные (унарные) вызовы
func (s *ecommerceServer) AddOrder(ctx context.Context, orderReq *ecommerce_v1.Order) (*wrappers.StringValue, error) {
	slog.Info("AddOrder() order added", "ID", orderReq.Id)

	s.mu.Lock()
	s.orderMap[orderReq.Id] = *orderReq
	s.mu.Unlock()
	//time.Sleep(time.Second * 6) // Simulate processing тест дедлайна контекста
	return &wrappers.StringValue{Value: "Order Added: " + orderReq.Id}, nil
}

func main() {
	cert, err := tls.LoadX509KeyPair(crtFile, keyFile) // загружаем сертификат
	if err != nil {
		slog.Warn("failed to load certificate", slog.Any("error", err))
		os.Exit(1)
	}
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewServerTLSFromCert(&cert)), // в опциях включаем TLS для всех входящих соединений
	}

	lis, err := net.Listen("tcp", port) // листенер к которому привяжем grpc-сервер
	if err != nil {
		slog.Warn("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}
	s := grpc.NewServer(opts...)                                        // создаем grpc-сервер c опциями TLS
	ecommerce_v1.RegisterOrderManagementServer(s, NewEcommerceServer()) // регистрируем имплементацию в grpc-сервер
	if err := s.Serve(lis); err != nil {                                // запускаем grpc-сервер на листенере
		slog.Warn("failed to serve", slog.Any("error", err))
	}
}
