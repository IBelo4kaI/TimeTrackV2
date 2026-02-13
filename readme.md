## Roadmap

- [ ] Создать файл initDb.sql

protoc -I proto proto/permission.proto --go_out=./internal/adapter/grpc/ --go-grpc_out=./internal/adapter/grpc/ --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative
