# gRPC-gateway

Позволяет считывать определения gRPC-сервисов и генерировать реверс прокси-серверы для перевода API на основе REST и JSON в gRPC.
клиенты смогут обращаться к gRPC-сервису как по gRPC так и по HTTP.

## документация gRPC-gateway
https://grpc-ecosystem.github.io/grpc-gateway/
https://github.com/grpc-ecosystem/grpc-gateway

## install
```
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
go install google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

```go get golang.org/x/sync```    // для использования errgroup

Googleapis(https://github.com/googleapis/googleapis). Эти файлу необходимы для grpc-gateway.
Пока не нашел решения лучше, чем просто скопировать файлы в папку проекта, создал отдельную папку для proto файлов. 
google/api/annotations.proto
google/api/http.proto
Для того чтобы IDE знала где искать эти proto файлы может понадобиться прописать путь к папке c проектом в настройках Protocol Buffers - Import Paths вашей IDE, в моем случае Goland.

## документация buf
https://github.com/bufbuild/buf

buf install in windows
```scoop install buf```

В этом примере файлы генерирую при помощи buf. Также можно сгенерить при помощи protoc, в makefile лежит команда.
Сайт buf.build заблокирован из РФ и РБ. В связи с этим удаленные плагины не работают, их необходимо загружать и запускать локально. https://buf.build/docs/generate/tutorial
Есть смысл задуматься в использовании buf, плохой фактор когда в любой момент могут заблокировать ту или иную локацию.

Загрузка зависимостей deps из Buf Schema Registry(BSR) также не работает, поэтому необходимые proto файлы положил локально в папку proto.
Путь к папке с proto файлами указал в файле buf.work.yaml

Реализована генерация сваггер доки в buf.
Сваггер поднят на том же порту что и http ручки.
Для работы сваггера в проект нужно положить папку: https://github.com/swagger-api/swagger-ui/tree/master/dist , у меня она в swagger-ui.