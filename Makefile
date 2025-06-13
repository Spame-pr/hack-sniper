.PHONY: build run-bot run-rpc clean test deps test-mysql

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