# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/bot cmd/bot/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/rpc cmd/rpc/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binaries from builder stage
COPY --from=builder /app/bin/bot .
COPY --from=builder /app/bin/rpc .

# Create wallets directory
RUN mkdir -p wallets

# Expose ports
EXPOSE 8080 8545

# Default command (can be overridden)
CMD ["./bot"] 