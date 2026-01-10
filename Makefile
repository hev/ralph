.PHONY: build install clean test test-race run run-otel up down logs snapshot release

# Build variables
BINARY_NAME=ralph
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_DIR=./build
GO_FILES=$(shell find . -type f -name '*.go')

# ldflags for version injection
LDFLAGS=-s -w \
	-X github.com/hev/ralph/internal/config.Version=$(VERSION) \
	-X github.com/hev/ralph/internal/config.Commit=$(COMMIT) \
	-X github.com/hev/ralph/internal/config.Date=$(DATE)

# Build the binary
build:
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ralph

# Install to GOPATH/bin
install:
	go install -ldflags="$(LDFLAGS)" ./cmd/ralph

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf dist/
	go clean

# Run tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -v -race ./...

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
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/ralph
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/ralph
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/ralph
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/ralph
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/ralph

# Create a snapshot release (for testing)
snapshot:
	goreleaser release --snapshot --clean

# Create a release (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the ralph binary"
	@echo "  install    - Install ralph to GOPATH/bin"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  test-race  - Run tests with race detector"
	@echo "  run        - Build and run ralph"
	@echo "  run-otel   - Build and run ralph with OTEL enabled"
	@echo "  up         - Start the observability stack (docker-compose)"
	@echo "  down       - Stop the observability stack"
	@echo "  logs       - View docker-compose logs"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  snapshot   - Create a snapshot release (testing)"
	@echo "  release    - Create a release with goreleaser"
