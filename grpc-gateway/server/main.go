package main

import (
	"context"
	pb "gRPC_examples/grpc-gateway/pkg/gen/go" // сгенерированный код
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	httpServerEndpoint = "localhost:8081"
	gRPCServerEndpoint = "localhost:50051"
)

// ecommerceServer структура имплементирует интерфейс
// OrderManagementServer, который содержит методы описанные в ecommerce.proto
type ecommerceServer struct {
	orderMap                              map[string]pb.Order // сохранять в мапу обьект сообщения плохая идея, тут это для примера
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
	slog.Info("AddOrder() order added", "ID", orderReq.Id)

	s.mu.Lock()
	s.orderMap[orderReq.Id] = *orderReq
	s.mu.Unlock()
	//time.Sleep(time.Second * 6) // Simulate processing тест дедлайна контекста
	return &wrappers.StringValue{Value: "Order Added: " + orderReq.Id}, nil
}

// GetOrder Simple RPC
// одиночные (унарные) вызовы
func (s *ecommerceServer) GetOrder(ctx context.Context, orderId *wrappers.StringValue) (*pb.Order, error) {
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

func main() {
	// gRPC сервер
	lis, err := net.Listen("tcp", gRPCServerEndpoint) // листенер к которому привяжем grpc-сервер
	if err != nil {
		slog.Warn("failed to listen", slog.Any("error", err))
		os.Exit(1)
	}
	s := grpc.NewServer()                                     // создаем grpc-сервер
	pb.RegisterOrderManagementServer(s, NewEcommerceServer()) // регистрируем имплементацию в grpc-сервер

	// htpp сервер

	// dial the gRPC server above to make a client connection
	conn, err := grpc.Dial(gRPCServerEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	rmux := runtime.NewServeMux()
	client := pb.NewOrderManagementClient(conn)
	err = pb.RegisterOrderManagementHandlerClient(context.Background(), rmux, client)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	// mount the gRPC HTTP gateway to the root
	mux.Handle("/", rmux)

	// mount a path to expose the generated OpenAPI specification on disk
	mux.HandleFunc("/swagger-ui/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./grpc-gateway/pkg/gen/openapi/ecommerce.swagger.json")
	})

	// mount the Swagger UI that uses the OpenAPI specification path above
	mux.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir("./grpc-gateway/swagger-ui"))))

	gr, errCtx := errgroup.WithContext(context.Background()) // группа для обработки ошибок запуска gRPC и http сервера

	gr.Go(func() error { // запускаем http-сервер
		slog.Info("http server started", slog.String("address", httpServerEndpoint))
		slog.Info("swagger started", slog.String("address", httpServerEndpoint+"/swagger-ui/"))
		err := http.ListenAndServe(httpServerEndpoint, mux)
		return err
	})
	gr.Go(func() error { // запускаем gRPC-сервер
		slog.Info("gRPC server started", slog.String("address", gRPCServerEndpoint))
		err := s.Serve(lis)
		return err
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM) // подписываемся на сигналы завершения приложения

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := gr.Wait() // получаем первую из ошибок возникших в errgroup
		slog.Error("Error in server:", slog.Any("error", err))
	}()

loop:
	for {
		select {
		case <-stop:
			s.GracefulStop()
			slog.Info("Graceful shutdown...")
			break loop
		case <-errCtx.Done(): // контекст завершится при ошибке запуска gRPC или http сервера
			slog.Info("Shutting down server with error")
			s.GracefulStop()
			wg.Wait() // ждем пока завершится горутина отлавливающая и логирующая ошибку errgroup
			break loop
		}
	}

	slog.Info("Server stopped")
}
