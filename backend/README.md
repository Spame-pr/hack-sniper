# Sniper Bot Backend System

A sophisticated MEV solution that allows token creators to monetize token sniping through a competitive bidding system. Built with Go, it features a Telegram bot interface for snipers and a custom RPC proxy for seamless integration with token deployment workflows.

## 🎯 Overview

Instead of losing value to front-running bots, token creators can now:
- **Monetize sniping activity** through direct bribes from competing snipers
- **Maximize auction profits** as more snipers = more competition = higher bribes
- **Maintain fair competition** through transparent bundle ordering mechanisms

## 🏗️ System Architecture

### Core Components

1. **🤖 Telegram Bot Service**
   - User registration and secure wallet generation
   - Balance monitoring and management
   - Snipe bid submission and validation
   - Real-time status updates and notifications

2. **🔗 RPC Proxy Service**
   - Transparent transaction forwarding to Base sequencer
   - LP_ADD transaction detection and interception
   - Bundle construction coordination
   - MEV-aware transaction processing

3. **💰 Bundle Manager**
   - Snipe bid sorting by bribe amount (highest first)
   - Gas price orchestration for proper transaction ordering
   - Bundle construction and submission to Base
   - Transaction success monitoring

4. **🔐 Wallet Manager**
   - Cryptographically secure wallet generation
   - Encrypted private key storage in MySQL
   - Transaction signing and broadcast capabilities
   - User wallet isolation and access control

5. **🗄️ Database Layer**
   - MySQL 8.0 with InnoDB engine for ACID compliance
   - Optimized indexing for high-performance queries
   - UTF8MB4 support for full Unicode compatibility
   - Connection pooling and transaction management

## 🚀 Quick Start

### Prerequisites

- **Go**: Version 1.21 or later
- **MySQL**: Version 8.0 or later
- **Telegram Bot Token**: From @BotFather
- **Base Network Access**: RPC endpoint URL

### Installation

1. **Clone and setup**:
```bash
git clone <repository-url>
cd sniper-bot/backend
./scripts/setup.sh
```

2. **Configure environment**:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. **Start MySQL**:
```bash
docker run --name sniper-mysql \
  -e MYSQL_ROOT_PASSWORD=root_password \
  -e MYSQL_DATABASE=sniper_bot \
  -e MYSQL_USER=sniper_user \
  -e MYSQL_PASSWORD=sniper_password \
  -p 3306:3306 -d mysql:8.0 \
  --default-authentication-plugin=mysql_native_password
```

4. **Initialize database**:
```bash
make db-migrate
```

5. **Run services**:
```bash
# Development mode
make dev

# Or run services separately
make run-bot    # Terminal 1
make run-rpc    # Terminal 2
```

### Docker Deployment

```bash
# Production deployment
docker-compose up -d

# View logs
docker-compose logs -f
```

## 🔧 Configuration

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `TELEGRAM_BOT_TOKEN` | Telegram bot authentication token | `1234567890:ABC...` |
| `BASE_RPC_URL` | Base network RPC endpoint | `https://base.llamarpc.com` |
| `BASE_WS_URL` | Base network WebSocket endpoint | `wss://base.llamarpc.com` |
| `ADMIN_PRIVATE_KEY` | Admin wallet private key (0x prefixed) | `0xabc123...` |
| `DATABASE_URL` | MySQL connection string | `user:pass@tcp(host:port)/db` |
| `UNISWAP_V2_ROUTER` | DEX router contract address | `0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24` |
| `UNISWAP_V2_FACTORY` | DEX factory contract address | `0x8909Dc15e40173Ff4699343b6eB8132c65e18eC6` |

### Optional Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RPC_PORT` | `8545` | RPC proxy service port |
| `BOT_PORT` | `8080` | Bot service health check port |
| `MAX_SNIPE_BIDS` | `100` | Maximum concurrent snipe bids per token |
| `BUNDLE_TIMEOUT` | `30s` | Bundle construction timeout |
| `DB_MAX_CONNECTIONS` | `25` | Maximum database connections |

## 📱 Usage Guide

### For Snipers

1. **Register Wallet**:
```
/register
```
*Creates a secure wallet and stores encrypted private key*

2. **Check Balance**:
```
/balance
```
*Shows ETH balance and wallet address*

3. **Submit Snipe Bid**:
```
/snipe 0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6 0.1
```
*Bids 0.1 ETH to snipe the specified token*

4. **View Active Bids**:
```
/mybids
```
*Shows all active snipe bids*

### For Token Creators

1. **Configure Metamask**: Set custom RPC to `http://localhost:8545` (or your deployed endpoint)
2. **Deploy Token**: Deploy your ERC-20 token contract normally
3. **Add Liquidity**: Add liquidity through your preferred DEX interface
4. **Receive Bribes**: Bribes are automatically sent to your wallet address

## 🔍 Development

### Build Commands

```bash
# Build all services
make build

# Build specific services
make build-bot
make build-rpc

# Cross-platform builds
make build-linux
make build-darwin
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test packages
go test ./pkg/bot/...
go test ./pkg/rpc/...
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Static analysis
make vet

# Security scan
make security-scan
```

### Database Management

```bash
# Test database connection
make test-mysql

# Run migrations
make db-migrate

# Seed test data
make db-seed

# Backup database
make db-backup
```

## 📊 Technical Details

### Bundle Construction Algorithm

```go
// Pseudo-code for bundle construction
func ConstructBundle(lpAddTx Transaction, bids []SnipeBid) Bundle {
    // Sort bids by bribe amount (descending)
    sort.Slice(bids, func(i, j int) bool {
        return bids[i].BribeAmount.Cmp(bids[j].BribeAmount) > 0
    })
    
    bundle := []Transaction{lpAddTx}
    
    // Add sniper transactions with decreasing gas prices
    baseGasPrice := lpAddTx.GasPrice
    for i, bid := range bids {
        sniperTx := ConstructSniperTx(bid)
        sniperTx.GasPrice = baseGasPrice - big.NewInt(int64(i+1))
        bundle = append(bundle, sniperTx)
    }
    
    return bundle
}
```

### Database Schema

#### Wallets Table
```sql
CREATE TABLE wallets (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    telegram_user_id VARCHAR(255) UNIQUE NOT NULL,
    wallet_address VARCHAR(42) NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_wallets_telegram_user_id (telegram_user_id),
    INDEX idx_wallets_address (wallet_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

#### Snipe Bids Table
```sql
CREATE TABLE snipe_bids (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token_address VARCHAR(42) NOT NULL,
    bribe_amount DECIMAL(36,18) NOT NULL,
    wallet_address VARCHAR(42) NOT NULL,
    status ENUM('pending', 'executed', 'failed', 'expired') DEFAULT 'pending',
    transaction_hash VARCHAR(66),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    executed_at TIMESTAMP NULL,
    INDEX idx_snipe_bids_token_address (token_address),
    INDEX idx_snipe_bids_status (status),
    INDEX idx_snipe_bids_user_id (user_id),
    INDEX idx_snipe_bids_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Security Features

- **🔐 Encrypted Storage**: Private keys encrypted with AES-256-GCM
- **🛡️ Input Validation**: Comprehensive validation for all user inputs
- **🔒 Access Control**: User operations isolated by Telegram user ID
- **⚡ Rate Limiting**: Protection against spam and abuse
- **🔍 Audit Logging**: Complete transaction and operation logging
- **🚫 SQL Injection Protection**: Parameterized queries throughout

### Performance Optimizations

- **Connection Pooling**: Efficient MySQL connection management
- **Query Optimization**: Indexed queries for O(log n) performance
- **Batch Processing**: Bulk operations for improved throughput
- **Caching**: In-memory caching for frequently accessed data
- **Async Processing**: Non-blocking I/O for concurrent operations

## 🔧 Monitoring & Observability

### Health Checks

```bash
# Bot service health
curl http://localhost:8080/health

# RPC service health  
curl http://localhost:8545/health

# Database connectivity
make test-mysql
```

### Logging

Structured logging with multiple levels:
```bash
# View logs in development
make logs

# Production log aggregation
docker-compose logs -f bot
docker-compose logs -f rpc
```

### Metrics

Key metrics to monitor:
- **Snipe Success Rate**: Percentage of successful snipes
- **Bundle Construction Time**: Time to build transaction bundles
- **Database Query Performance**: Query execution times
- **Transaction Processing**: Throughput and latency metrics

## 🚨 Troubleshooting

### Common Issues

1. **Database Connection Errors**:
```bash
# Check MySQL status
docker ps | grep mysql

# Test connection
make test-mysql

# Reset database
make db-reset
```

2. **Telegram Bot Not Responding**:
```bash
# Check bot token validity
curl https://api.telegram.org/bot<TOKEN>/getMe

# Restart bot service
make restart-bot
```

3. **RPC Proxy Issues**:
```bash
# Check Base network connectivity
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}' \
  $BASE_RPC_URL
```

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug
make run-bot

# Enable SQL query logging
export DB_LOG_MODE=true
make run-rpc
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run the test suite: `make test`
5. Format code: `make fmt`
6. Run linter: `make lint`
7. Commit changes: `git commit -m 'Add amazing feature'`
8. Push to branch: `git push origin feature/amazing-feature`
9. Open a Pull Request

### Development Guidelines

- Write comprehensive tests for new functionality
- Follow Go best practices and idioms
- Update documentation for API changes
- Use semantic commit messages
- Ensure backward compatibility

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ⚠️ Disclaimer

This software is provided for educational and research purposes only. Users are solely responsible for compliance with applicable laws and regulations. The authors disclaim all liability for any financial losses, legal issues, or other damages arising from the use of this software.

## 🆘 Support

For support and questions:

- **Issues**: Open an issue on GitHub
- **Documentation**: Check the `/docs` directory
- **Community**: Join our Discord community
- **Security**: Report security issues privately to security@example.com

---

*Built with ❤️ for the DeFi community* 