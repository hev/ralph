.PHONY: build install clean test run run-otel up down logs

# Build variables
BINARY_NAME=ralph
VERSION=1.0.0
BUILD_DIR=./build
GO_FILES=$(shell find . -type f -name '*.go')

# Build the binary
build:
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ralph

# Install to GOPATH/bin
install:
	go install ./cmd/ralph

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Run tests
test:
	go test -v ./...

# Run ralph with default settings
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Run ralph with OTEL enabled (requires docker stack)
run-otel: build
	$(BUILD_DIR)/$(BINARY_NAME) --otel-enabled --otel-endpoint localhost:4317

# Start the observability stack
up:
	docker-compose up -d
	@echo "Grafana available at http://localhost:3000 (admin/admin)"
	@echo "Prometheus available at http://localhost:9090"

# Stop the observability stack
down:
	docker-compose down

# View docker logs
logs:
	docker-compose logs -f

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/ralph
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/ralph
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/ralph
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/ralph

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the ralph binary"
	@echo "  install    - Install ralph to GOPATH/bin"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  run        - Build and run ralph"
	@echo "  run-otel   - Build and run ralph with OTEL enabled"
	@echo "  up         - Start the observability stack (docker-compose)"
	@echo "  down       - Stop the observability stack"
	@echo "  logs       - View docker-compose logs"
	@echo "  build-all  - Build for multiple platforms"
