generate:
	protoc -I . ./first.proto --go_out=./pkg/services/first --go_opt=paths=source_relative --go-grpc_out=./pkg/services/first --go-grpc_opt=paths=source_relative
	protoc -I . ./second.proto --go_out=./pkg/services/second --go_opt=paths=source_relative --go-grpc_out=./pkg/services/second --go-grpc_opt=paths=source_relative
