.PHONY: help build run test clean dev docker-build docker-run

help: ## Show this help message
	@echo "Isekai API Gateway - Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the gateway binary
	@echo "Building gateway..."
	@go build -o bin/gateway cmd/gateway/main.go
	@echo "Build complete: bin/gateway"

run: ## Run the gateway
	@echo "Starting gateway..."
	@go run cmd/gateway/main.go

dev: ## Run in development mode with hot reload (requires air)
	@air

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

install: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t isekai-gateway:latest .
	@echo "Docker image built"

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env isekai-gateway:latest

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@go run cmd/gateway/main.go migrate up

migrate-down: ## Run database migrations down
	@echo "Rolling back migrations..."
	@go run cmd/gateway/main.go migrate down

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "Dependencies updated"
