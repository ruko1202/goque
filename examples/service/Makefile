GOBIN			?= $(PWD)/bin

.PHONY: help build run dev docker-up docker-down docker-logs clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the service binary
	@echo "Building service..."
	@go build -o ./bin/service ./cmd/service
	@echo "✓ Build complete: bin/service"

run: build ## Build and run the service locally
	@echo "Starting service..."
	@./bin/service

dev: ## Run the service in development mode
	@echo "Starting service in development mode..."
	@go run ./cmd/service

docker-up: ## Start all services with docker-compose
	@echo "Starting services with Docker Compose..."
	@docker-compose up -d
	@echo "✓ Services started"
	@echo ""
	@echo "Dashboard: http://localhost:8080"
	@echo "API: http://localhost:8080/api/tasks"
	@echo "Health: http://localhost:8080/health"

docker-down: ## Stop all services
	@echo "Stopping services..."
	@docker-compose down
	@echo "✓ Services stopped"

docker-logs: ## Show logs from all services
	@docker-compose logs -f

docker-rebuild: ## Rebuild and restart services
	@echo "Rebuilding services..."
	@docker-compose down
	@docker-compose build --no-cache
	@docker-compose up -d
	@echo "✓ Services rebuilt and started"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf ./bin/
	@echo "✓ Clean complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@echo "✓ Dependencies downloaded"

bin-deps: ## Download binary dependencies
	@echo "Downloading binary dependencies..."
	GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0
	@echo "✓ Dependencies binary downloaded"

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

lint: ## Run linter
	@echo "Running linter..."
	@$(GOBIN)/golangci-lint run ./...

fmt: ## Run formater
	@echo "Running formater..."
	@$(GOBIN)/golangci-lint fmt -E gofmt -E goimports ./...
