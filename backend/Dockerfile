# Build stage
FROM golang:1.23.2-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the applications with optimized flags
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-s -w" \
    -trimpath \
    -o bin/bot services/bot/main.go

RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-s -w" \
    -trimpath \
    -o bin/rpc services/rpc/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -g 1001 -S appgroup \
    && adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder /app/bin/bot ./bot
COPY --from=builder /app/bin/rpc ./rpc

# Create necessary directories
RUN mkdir -p wallets logs \
    && chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -f "(bot|rpc)" > /dev/null || exit 1

# Expose ports
EXPOSE 8080 8545

# Default command (can be overridden)
CMD ["./bot"] 