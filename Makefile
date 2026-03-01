# NodeByte Backend Makefile
# Usage: make [target]

# Load environment variables from .env file
ifneq (,$(wildcard .env))
	include .env
endif

# Variables
BINARY_NAME=nodebyte-backend
MAIN_PATH=./cmd/api
BUILD_DIR=./bin
GO?=go

ifeq ($(OS),Windows_NT)
	GOFLAGS=-ldflags "-s -w"
	EXE_EXT=.exe
	BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)$(EXE_EXT)
	DB_TOOL_PATH=$(BUILD_DIR)/db$(EXE_EXT)
	MKDIR_BUILD=if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
	RM_BUILD=if exist "$(BUILD_DIR)" rmdir /s /q "$(BUILD_DIR)"
	RM_TMP=if exist "tmp" rmdir /s /q "tmp"
	RM_COVERAGE_OUT=if exist "coverage.out" del /q "coverage.out"
	RM_COVERAGE_HTML=if exist "coverage.html" del /q "coverage.html"
	RUN_GOOS_LINUX_AMD64=set GOOS=linux&& set GOARCH=amd64&& $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	RUN_GOOS_WINDOWS_AMD64=set GOOS=windows&& set GOARCH=amd64&& $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	RUN_GOOS_DARWIN_AMD64=set GOOS=darwin&& set GOARCH=amd64&& $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	RUN_GOOS_DARWIN_ARM64=set GOOS=darwin&& set GOARCH=arm64&& $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
else
	GOFLAGS=-ldflags="-s -w"
	EXE_EXT=
	BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)$(EXE_EXT)
	DB_TOOL_PATH=$(BUILD_DIR)/db$(EXE_EXT)
	MKDIR_BUILD=mkdir -p $(BUILD_DIR)
	RM_BUILD=rm -rf $(BUILD_DIR)
	RM_TMP=rm -rf tmp
	RM_COVERAGE_OUT=rm -f coverage.out
	RM_COVERAGE_HTML=rm -f coverage.html
	RUN_GOOS_LINUX_AMD64=GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	RUN_GOOS_WINDOWS_AMD64=GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	RUN_GOOS_DARWIN_AMD64=GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	RUN_GOOS_DARWIN_ARM64=GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
endif

# Docker
DOCKER_IMAGE=nodebyte/backend
DOCKER_TAG=latest

# Colors for terminal output (disabled on Windows to avoid cmd parsing issues)
GREEN=
YELLOW=
RED=
NC=

.PHONY: all build build-tools run dev clean test lint fmt vet deps tidy docker-build docker-up docker-down docker-logs swagger help db-init db-migrate db-reset db-list

# Default target
all: build

## Build Commands

# Build the application for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@$(MKDIR_BUILD)
	$(GO) build $(GOFLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Build for multiple platforms
build-all:
	@echo "Building for all platforms..."
	@$(MKDIR_BUILD)
	$(RUN_GOOS_LINUX_AMD64)
	$(RUN_GOOS_WINDOWS_AMD64)
	$(RUN_GOOS_DARWIN_AMD64)
	$(RUN_GOOS_DARWIN_ARM64)
	@echo "All builds complete"

# Build database tools
build-tools:
	@echo "Building database tools..."
	@$(MKDIR_BUILD)
	$(GO) build $(GOFLAGS) -o $(DB_TOOL_PATH) ./cmd/db
	@echo "Database tools built: $(DB_TOOL_PATH)"

## Run Commands

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH)

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

# Build database migration tools
build-db-tools: build-tools

# Initialize fresh database with all schemas
db-init: build-tools
	@echo "Initializing database..."
	$(DB_TOOL_PATH) init -database "$(DATABASE_URL)"

# Run interactive migration
db-migrate: build-tools
	@echo "Running database migration..."
	$(DB_TOOL_PATH) migrate -database "$(DATABASE_URL)"

# Migrate specific schema
db-migrate-schema: build-tools
	@$(if $(SCHEMA),,$(error Usage: make db-migrate-schema SCHEMA=schema_name.sql))
	@echo "Migrating $(SCHEMA)..."
	$(DB_TOOL_PATH) migrate -database "$(DATABASE_URL)" -schema "$(SCHEMA)"

# Reset database (DROP and recreate) - CAREFUL!
db-reset: build-tools
	@echo "WARNING: This will DROP and recreate the database!"
	$(DB_TOOL_PATH) reset -database "$(DATABASE_URL)"

# List available schemas
db-list: build-tools
	$(DB_TOOL_PATH) list

# Generate sqlc code (if using sqlc)
sqlc:
	@echo "Generating sqlc code..."
	sqlc generate

## Utility Commands

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@$(RM_BUILD)
	@$(RM_TMP)
	@$(RM_COVERAGE_OUT)
	@$(RM_COVERAGE_HTML)
	@echo "Clean complete"

# Show environment info
info:
	@echo "Environment Info"
	@echo "Go Version: " && $(GO) version
	@echo "GOOS: " && $(GO) env GOOS
	@echo "GOARCH: " && $(GO) env GOARCH
	@echo "Module: " && $(GO) list -m

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
	@echo "  make build-tools    - Build database tools"
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
	@echo "Database:"
	@echo "  make db-init        - Initialize fresh database"
	@echo "  make db-migrate     - Run interactive migration"
	@echo "  make db-migrate-schema SCHEMA=schema_name.sql - Migrate specific schema"
	@echo "  make db-reset       - Reset database (DROP and recreate)"
	@echo "  make db-list        - List available schemas"
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
