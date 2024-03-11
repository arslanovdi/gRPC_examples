Пример перехватчиков методов (interceptor, middleware)

На стороне сервера:
orderUnaryServerInterceptor() - Унарные методы
orderServerStreamInterceptor() - потоковые методы, создается обертка над grpc.ServerStream с реализацей методов RecvMsg и SendMsg

На стороне клиента:
orderUnaryClientInterceptor() - Унарные методы
clientStreamInterceptor() - потоковые методы, создается обертка над grpc.ClientStream с реализацей методов RecvMsg и SendMsg
