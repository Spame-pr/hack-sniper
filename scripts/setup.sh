#!/bin/bash

set -e

echo "🚀 Setting up Sniper Bot System..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or later."
    exit 1
fi

echo "✅ Go version $GO_VERSION detected"

# Install dependencies
echo "📦 Installing dependencies..."
go mod download
go mod tidy

# Create necessary directories
echo "📁 Creating directories..."
mkdir -p bin
mkdir -p wallets
mkdir -p logs

# Build the applications
echo "🔨 Building applications..."
go build -o bin/bot cmd/bot/main.go
go build -o bin/rpc cmd/rpc/main.go

echo "✅ Build completed successfully"

# Check if .env file exists
if [ ! -f .env ]; then
    echo "⚙️  Creating .env file..."
    cat > .env << EOF
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# Base Network Configuration
BASE_RPC_URL=https://mainnet.base.org
BASE_WS_URL=wss://mainnet.base.org

# Admin Wallet (KEEP THIS SECURE!)
ADMIN_PRIVATE_KEY=your_admin_private_key_here

# Database Configuration (MySQL)
DATABASE_URL=sniper_user:sniper_password@tcp(localhost:3306)/sniper_bot?charset=utf8mb4&parseTime=True&loc=Local

# DEX Configuration (Base network addresses)
UNISWAP_V2_ROUTER=0x4752ba5dbc23f44d87826276bf6fd6b1c372ad24
UNISWAP_V2_FACTORY=0x8909dc15e40173ff4699343b6eb8132c65e18ec6
EOF
    echo "📝 Created .env file. Please edit it with your configuration."
else
    echo "✅ .env file already exists"
fi

# Run tests
echo "🧪 Running tests..."
go test ./test/

echo ""
echo "🎉 Setup completed successfully!"
echo ""
echo "Next steps:"
echo "1. Edit the .env file with your configuration:"
echo "   - Add your Telegram bot token"
echo "   - Add your admin private key (keep it secure!)"
echo "   - Configure your database connection"
echo ""
echo "2. Set up MySQL database:"
echo "   docker run --name sniper-mysql -e MYSQL_ROOT_PASSWORD=root_password -e MYSQL_DATABASE=sniper_bot -e MYSQL_USER=sniper_user -e MYSQL_PASSWORD=sniper_password -p 3306:3306 -d mysql:8.0 --default-authentication-plugin=mysql_native_password"
echo ""
echo "3. Run the services:"
echo "   # Terminal 1 - Bot service"
echo "   make run-bot"
echo ""
echo "   # Terminal 2 - RPC service"
echo "   make run-rpc"
echo ""
echo "4. Or use Docker Compose:"
echo "   docker-compose up -d"
echo ""
echo "📚 For more information, check the README.md file" 