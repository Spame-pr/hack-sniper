# Sniper Bot Backend Makefile
# ============================

.PHONY: help build run clean test deps docker dev-setup dev-up dev-down logs tools generate fmt lint vet tidy security

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_DIR := bin
BOT_BINARY := $(BINARY_DIR)/bot
RPC_BINARY := $(BINARY_DIR)/rpc
DOCKER_IMAGE := sniper-bot
DOCKER_COMPOSE_FILE := docker-compose.yml
GO_VERSION := 1.23.2

# Build flags
LDFLAGS := -ldflags "-s -w"
BUILD_FLAGS := $(LDFLAGS) -trimpath

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

## Help
help: ## Show this help message
	@echo "$(GREEN)Sniper Bot Backend$(RESET)"
	@echo "=================="
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(RESET) %s\n", $$1, $$2}'

## Build
build: deps $(BINARY_DIR) ## Build all services
	@echo "$(GREEN)Building all services...$(RESET)"
	@$(MAKE) build-bot
	@$(MAKE) build-rpc
	@echo "$(GREEN)✓ Build completed$(RESET)"

build-bot: $(BINARY_DIR) ## Build bot service (includes bot + API)
	@echo "$(GREEN)Building bot service (Telegram bot + API)...$(RESET)"
	@CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BOT_BINARY) services/bot/main.go

build-rpc: $(BINARY_DIR) ## Build RPC service
	@echo "$(GREEN)Building RPC service...$(RESET)"
	@CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(RPC_BINARY) services/rpc/main.go

$(BINARY_DIR):
	@mkdir -p $(BINARY_DIR)

## Run
run-bot: ## Run bot service in development mode (includes Telegram bot + API)
	@echo "$(GREEN)Starting bot service (Telegram bot + API)...$(RESET)"
	@go run services/bot/main.go

run-rpc: ## Run RPC service in development mode
	@echo "$(GREEN)Starting RPC service...$(RESET)"
	@go run services/rpc/main.go

## Dependencies
deps: ## Download and tidy dependencies
	@echo "$(GREEN)Downloading dependencies...$(RESET)"
	@go mod download
	@go mod tidy

## Testing
test: ## Run all tests
	@echo "$(GREEN)Running tests...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	@echo "$(GREEN)Generating coverage report...$(RESET)"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

test-mysql: ## Test MySQL connection
	@echo "$(GREEN)Testing MySQL connection...$(RESET)"
	@go run scripts/init-schema.go

## Scripts
create-pair: ## Create Uniswap pair and add liquidity
	@echo "$(GREEN)Creating Uniswap V2 pair and adding liquidity...$(RESET)"
	@go run scripts/create-pair.go

init-schema: ## Initialize database schema
	@echo "$(GREEN)Initializing database schema...$(RESET)"
	@go run scripts/init-schema.go

## Docker
docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(RESET)"
	@docker build -t $(DOCKER_IMAGE):latest .
	@echo "$(GREEN)✓ Docker image built: $(DOCKER_IMAGE):latest$(RESET)"

docker-build-multi: ## Build multi-platform Docker image
	@echo "$(GREEN)Building multi-platform Docker image...$(RESET)"
	@docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_IMAGE):latest .

docker-run-bot: docker-build ## Run bot service in Docker
	@echo "$(GREEN)Running bot service in Docker...$(RESET)"
	@docker run --rm --name sniper-bot $(DOCKER_IMAGE):latest ./bot

docker-run-rpc: docker-build ## Run RPC service in Docker
	@echo "$(GREEN)Running RPC service in Docker...$(RESET)"
	@docker run --rm --name sniper-rpc -p 8545:8545 $(DOCKER_IMAGE):latest ./rpc

## Development Environment
dev-setup: ## Setup development environment
	@echo "$(GREEN)Setting up development environment...$(RESET)"
	@$(MAKE) tools
	@$(MAKE) deps
	@echo "$(GREEN)✓ Development environment ready$(RESET)"

dev-up: ## Start development environment with Docker Compose
	@echo "$(GREEN)Starting development environment...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
	@echo "$(GREEN)✓ Development environment started$(RESET)"
	@$(MAKE) dev-status

dev-down: ## Stop development environment
	@echo "$(GREEN)Stopping development environment...$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down
	@echo "$(GREEN)✓ Development environment stopped$(RESET)"

dev-restart: dev-down dev-up ## Restart development environment

dev-status: ## Show development environment status
	@echo "$(GREEN)Development environment status:$(RESET)"
	@docker-compose -f $(DOCKER_COMPOSE_FILE) ps

logs: ## Show development environment logs
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

logs-mysql: ## Show MySQL logs
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f mysql

## Code Quality
fmt: ## Format Go code
	@echo "$(GREEN)Formatting code...$(RESET)"
	@go fmt ./...
	@gofumpt -w . 2>/dev/null || true

lint: ## Lint Go code
	@echo "$(GREEN)Linting code...$(RESET)"
	@golangci-lint run --timeout=5m

vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(RESET)"
	@go vet ./...

tidy: ## Tidy Go modules
	@echo "$(GREEN)Tidying modules...$(RESET)"
	@go mod tidy

security: ## Run security scan
	@echo "$(GREEN)Running security scan...$(RESET)"
	@gosec ./... 2>/dev/null || echo "$(YELLOW)⚠ gosec not installed. Run 'make tools' to install$(RESET)"

## Tools
tools: ## Install development tools
	@echo "$(GREEN)Installing development tools...$(RESET)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "$(GREEN)✓ Development tools installed$(RESET)"

generate: ## Generate code (contracts, mocks, etc.)
	@echo "$(GREEN)Generating code...$(RESET)"
	@go generate ./...

## Maintenance
clean: ## Clean build artifacts and cache
	@echo "$(GREEN)Cleaning up...$(RESET)"
	@rm -rf $(BINARY_DIR)/
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache -modcache
	@echo "$(GREEN)✓ Cleanup completed$(RESET)"

clean-docker: ## Clean Docker images and containers
	@echo "$(GREEN)Cleaning Docker artifacts...$(RESET)"
	@docker system prune -f
	@docker rmi $(DOCKER_IMAGE):latest 2>/dev/null || true

## Release
release: clean test security lint build ## Full release build with all checks
	@echo "$(GREEN)✓ Release build completed successfully$(RESET)"

## Database
db-connect: ## Connect to MySQL database
	@echo "$(GREEN)Connecting to MySQL database...$(RESET)"
	@docker exec -it mysqlSniper mysql -u root -p

db-backup: ## Backup database
	@echo "$(GREEN)Creating database backup...$(RESET)"
	@docker exec mysqlSniper mysqldump -u root -p --all-databases > backup_$(shell date +%Y%m%d_%H%M%S).sql

## Monitoring
ps: ## Show running processes
	@echo "$(GREEN)Running processes:$(RESET)"
	@ps aux | grep -E "(bot|rpc)" | grep -v grep || echo "No processes found"

check-ports: ## Check if required ports are available
	@echo "$(GREEN)Checking ports:$(RESET)"
	@netstat -tulpn | grep -E ":(3306|8080|8545)" || echo "Ports are available" 