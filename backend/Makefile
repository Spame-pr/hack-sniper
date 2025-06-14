.PHONY: build run-bot run-rpc clean test deps test-mysql test-rpc create-pair test-lp-detection

# Build all binaries
build: build-bot build-rpc

# Build bot service
build-bot:
	go build -o bin/bot cmd/bot/main.go

# Build RPC service
build-rpc:
	go build -o bin/rpc cmd/rpc/main.go

# Run bot service
run-bot:
	go run cmd/bot/main.go

# Run RPC service
run-rpc:
	go run cmd/rpc/main.go

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test ./...

# Test MySQL connection
test-mysql:
	go run scripts/test-mysql.go

# Test RPC proxy
test-rpc:
	@echo "🧪 Testing RPC Proxy..."
	@go run scripts/test-rpc.go

# Create Uniswap pair and add liquidity
create-pair:
	@echo "🚀 Creating Uniswap V2 pair and adding liquidity..."
	@go run scripts/create-pair.go

# Test LP_ADD detection
test-lp-detection:
	@echo "🧪 Testing LP_ADD detection..."
	@go run scripts/test-lp-detection.go

# Clean build artifacts
clean:
	rm -rf bin/

# Create bin directory
bin:
	mkdir -p bin

# Docker build
docker-build:
	docker build -t sniper-bot .

# Docker compose up
docker-up:
	docker-compose up -d

# Docker compose down
docker-down:
	docker-compose down

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate contract bindings
generate:
	go generate ./... 