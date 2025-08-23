build: proto
	GOOS=linux GOARCH=amd64 go build -o bin/server main.go

clean:
	rm ./telemetry.log

generate:
	protoc --proto_path=.. --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/sensor.proto

dependencies:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go mod vendor

lint:
	go fmt ./...
	golangci-lint run ./...

test:
	go fmt ./...
	go test ./...
