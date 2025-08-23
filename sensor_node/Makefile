BINARY_NAME=sensor_node
BUILD_DIR=bin

build-linux: proto
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go

build-windows: proto
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go

build-macos: proto
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go

build-macos-arm: proto
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go

build-all: build-linux build-windows build-macos build-macos-arm

clean:
	rm -f proto/*.pb.go
	rm -rf $(BUILD_DIR)

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