# NodeByte Backend Makefile
# Usage: make [target]

# Variables
BINARY_NAME=nodebyte-backend
MAIN_PATH=./cmd/api
BUILD_DIR=./bin
GO=go
GOFLAGS=-ldflags="-s -w"

# Docker
DOCKER_IMAGE=nodebyte/backend
DOCKER_TAG=latest

# Colors for terminal output (disabled on Windows to avoid cmd parsing issues)
GREEN=
YELLOW=
RED=
NC=

.PHONY: all build run dev clean test lint fmt vet deps tidy docker-build docker-up docker-down docker-logs swagger help

# Default target
all: build

## Build Commands

# Build the application for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "Building for all platforms..."
	@if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "All builds complete"

## Run Commands

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)\$(BINARY_NAME)

# Run with hot reload (requires air)
dev:
	@echo "Starting development server with hot reload..."
	$(GO) run $(MAIN_PATH)

# Run directly without building
run-direct:
	@echo "Running directly..."
	$(GO) run $(MAIN_PATH)

## Code Quality

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Run all checks
check: fmt vet test
	@echo "All checks passed"

# Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	$(GO) run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go
	@echo "Swagger docs generated in docs/ directory"

## Dependencies

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Update dependencies
update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

# Vendor dependencies
vendor:
	@echo "Vendoring dependencies..."
	$(GO) mod vendor

## Docker Commands

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Start Docker containers
docker-up:
	@echo "Starting Docker containers..."
	docker-compose up -d

# Start with monitoring
docker-up-monitor:
	@echo "Starting Docker containers with monitoring..."
	docker-compose --profile monitoring up -d

# Stop Docker containers
docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f backend

# View all logs
docker-logs-all:
	docker-compose logs -f

# Restart backend container
docker-restart:
	@echo "Restarting backend container..."
	docker-compose restart backend

# Clean Docker resources
docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v --rmi local

## Database Commands

# Generate sqlc code (if using sqlc)
sqlc:
	@echo "Generating sqlc code..."
	sqlc generate

## Utility Commands

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rmdir /s /q $(BUILD_DIR) 2>nul || true
	rmdir /s /q tmp 2>nul || true
	del coverage.out coverage.html 2>nul || true
	@echo "Clean complete"

# Show environment info
info:
	@echo "Environment Info"
	@echo "Go Version: " && go version
	@echo "GOOS: " && go env GOOS
	@echo "GOARCH: " && go env GOARCH
	@echo "Module: " && go list -m

# Generate API documentation (if using swag)
docs:
	@echo "Generating API documentation..."
	swag init -g cmd/api/main.go -o docs

## Help

# Show help
help:
	@echo ""
	@echo "NodeByte Backend - Available Commands"
	@echo ""
	@echo "Build:"
	@echo "  make build          - Build the application"
	@echo "  make build-all      - Build for all platforms"
	@echo ""
	@echo "Run:"
	@echo "  make run            - Build and run"
	@echo "  make run-direct     - Run without building"
	@echo "  make dev            - Run with hot reload"
	@echo ""
	@echo "Code Quality:"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make bench          - Run benchmarks"
	@echo "  make lint           - Lint code"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make check          - Run all checks"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps           - Download dependencies"
	@echo "  make tidy           - Tidy dependencies"
	@echo "  make update         - Update dependencies"
	@echo "  make vendor         - Vendor dependencies"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-up      - Start containers"
	@echo "  make docker-up-monitor - Start with monitoring"
	@echo "  make docker-down    - Stop containers"
	@echo "  make docker-logs    - View backend logs"
	@echo "  make docker-restart - Restart backend"
	@echo "  make docker-clean   - Clean Docker resources"
	@echo ""
	@echo "Utility:"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make info           - Show environment info"
	@echo "  make docs           - Generate API docs"
	@echo ""
