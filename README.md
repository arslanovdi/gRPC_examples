<dl>
<dt>Install</dt>
<dd>
go get -u google.golang.org/grpc
</dd>
</dl>

[communication_patterns](https://github.com/arslanovdi/gRPC_examples/tree/master/communication_patterns)
Пример работы с унарными методами, потоковыми методами на стороне сервера, клиента и двунаправленные.

[interceptors](https://github.com/arslanovdi/gRPC_examples/tree/master/interceptors)
Пример перехватчиков методов (interceptor, middleware)

[authentication](https://github.com/arslanovdi/gRPC_examples/tree/master/authentication)
Безопасность в gRPC

[multiplexing](https://github.com/arslanovdi/gRPC_examples/tree/master/multiplexing)
Запуск нескольких gRPC-сервисов на одном сервере (порту).

[metadata](https://github.com/arslanovdi/gRPC_examples/tree/master/metadata)
Пример приема/передачи метаданных.

[grpc-gateway](https://github.com/arslanovdi/gRPC_examples/tree/master/grpc-gateway)
Пример обработки grpc сервером HTTP запросов. Может быть полезно если у grpc сервера есть grpc и HTTP клиенты, или при переходе.
Для генерации используется Buf. Генерируется OpenAPI спецификация, gRPC, HTTP код.

[errors](https://github.com/arslanovdi/gRPC_examples/tree/master/errors)
список ошибок

Deadlines, cancel request.
Все через контексты. context.WithTimeout.
