# Sniper Bot System

A system that allows token creators to auction off the right to snipe their tokens, where snipers compete with ETH bribes to get included early in the transaction bundle.

## 🎯 The Problem

Token creators get sniped — and usually walk away with nothing, while sniper bots farm all the upside.

## 💡 The Solution

Let's fix that. This system lets token creators auction off the right to snipe their tokens:

- Snipers compete with ETH bribes to get included early
- Bribes go directly to the token creator
- More bots ⇒ more bribes ⇒ more profit for the creator
- Maximize chaos. Monetize attention.

## 🏗️ Architecture

### Components

1. **🔫 Telegram Bot (for Snipers)**
   - `/register`: Create a wallet (PK stored backend-side)
   - View wallet balances (ETH + tokens)
   - `/snipe <token_address> <bribe_in_ETH>`: Submit snipe requests

2. **🤑 Custom RPC (for Token Creators)**
   - Non-LP_ADD txs → pass through to Base sequencer normally
   - If LP_Add detected → trigger bundle build with sniper bribes

### How It Works

1. **Sniper Registration**: Users create wallets through the Telegram bot
2. **Snipe Bidding**: Snipers submit bids for tokens that aren't live yet
3. **LP_ADD Detection**: When a token creator adds liquidity, the system detects it
4. **Bundle Creation**: System constructs a bundle with:
   - TX 0: LP_ADD from token creator
   - TX 1...N: Sniper swaps (sorted by bribe size)
   - Each sniper transaction includes a bribe transfer to the token creator
5. **Bundle Submission**: Submit to Base sequencer with proper gas price ordering

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or later
- MySQL database
- Telegram Bot Token
- Base network RPC access

### Setup

1. **Clone and setup**:
```bash
git clone <repository-url>
cd sniper-bot
./scripts/setup.sh
```

2. **Configure environment**:
```bash
# Edit .env file with your configuration
vim .env
```

Required environment variables:
- `TELEGRAM_BOT_TOKEN`: Your Telegram bot token
- `BASE_RPC_URL`: Base network RPC URL
- `ADMIN_PRIVATE_KEY`: Admin wallet private key
- `DATABASE_URL`: MySQL connection string

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

4. **Run the services**:
```bash
# Terminal 1 - Bot service
make run-bot

# Terminal 2 - RPC service
make run-rpc
```

### Docker Deployment

```bash
# Using Docker Compose
docker-compose up -d
```

## 📱 Usage

### For Snipers

1. **Register**: Send `/register` to the Telegram bot
2. **Check Balance**: Send `/balance` to see your wallet info
3. **Submit Snipe**: Send `/snipe <token_address> <bribe_amount>`

Example:
```
/register
/snipe 0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6 0.1
```

### For Token Creators

1. **Configure Metamask/Rabby**: Set custom RPC to `http://localhost:8545`
2. **Deploy Token**: Deploy your token contract normally
3. **Add Liquidity**: Add liquidity through the DEX
4. **Receive Bribes**: Bribes are automatically sent to your address

## 🔧 Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Format Code

```bash
make fmt
```

### Clean

```bash
make clean
```

## 📊 Technical Details

### Transaction Bundle Structure

```
Bundle = [LP_Add, Sniper1, Sniper2, ..., SniperN]
```

- **LP_ADD**: Original liquidity addition transaction
- **Sniper Transactions**: Ordered by bribe amount (highest first)
- **Gas Price Ordering**: Each tx has `gasPrice = previous - 1 wei`
- **Bribe Transfers**: ETH sent directly to token creator

### Security Features

- **Secure Wallet Generation**: Cryptographically secure private key generation
- **Encrypted Storage**: Private keys stored encrypted in database
- **Input Validation**: All user inputs validated before processing
- **Transaction Ordering**: Gas price manipulation ensures proper ordering
- **Access Control**: User operations isolated by user ID

### Network Support

- **Base Network**: Primary deployment target
- **Uniswap V2 Compatible**: Works with any Uniswap V2-style DEX
- **EVM Compatible**: Can be extended to other EVM chains

## 🛠️ Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | Yes |
| `BASE_RPC_URL` | Base network RPC URL | Yes |
| `BASE_WS_URL` | Base network WebSocket URL | Yes |
| `ADMIN_PRIVATE_KEY` | Admin wallet private key | Yes |
| `DATABASE_URL` | MySQL connection string | Yes |
| `UNISWAP_V2_ROUTER` | DEX router contract address | Yes |
| `UNISWAP_V2_FACTORY` | DEX factory contract address | Yes |

### Database Schema

The system automatically creates the required MySQL tables:

#### `wallets` Table
- `id`: Auto-increment primary key
- `telegram_user_id`: Unique identifier for Telegram users
- `wallet_address`: Ethereum wallet address
- `private_key`: Encrypted private key storage
- `created_at`: Timestamp of wallet creation

#### `snipe_bids` Table
- Stores sniper bid information with InnoDB engine
- Indexes on `token_address` and `status` for performance
- UTF8MB4 charset for full Unicode support

**Important**: Each user can only have ONE wallet. The system enforces this constraint at the database level with a unique index on `telegram_user_id`.

### MySQL Connection String Format

```
username:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local
```

Example:
```
sniper_user:sniper_password@tcp(localhost:3306)/sniper_bot?charset=utf8mb4&parseTime=True&loc=Local
```

## 🔍 Monitoring

### Health Checks

- Bot service: `http://localhost:8080/health`
- RPC service: `http://localhost:8545/health`

### Logs

Logs are written to stdout and can be redirected to files:

```bash
./bin/bot > logs/bot.log 2>&1 &
./bin/rpc > logs/rpc.log 2>&1 &
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run tests: `make test`
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## ⚠️ Disclaimer

This software is for educational and research purposes. Users are responsible for compliance with applicable laws and regulations. The authors are not responsible for any financial losses or legal issues arising from the use of this software.

## 🆘 Support

For support and questions:

1. Check the [Architecture Documentation](ARCHITECTURE.md)
2. Review the [Issues](https://github.com/your-repo/issues) page
3. Join our community discussions

## 🚧 Roadmap

- [ ] Multi-chain support (Ethereum, Arbitrum, Polygon)
- [ ] Advanced bidding mechanisms (time-based auctions)
- [ ] Web dashboard for analytics
- [ ] REST API for programmatic access
- [ ] Enhanced MEV protection mechanisms
- [ ] Mobile app for snipers

## 🔫 Bot Service HTTP API

The bot service now includes an HTTP server with the following endpoints:

#### Endpoints

1. **Health Check**
   ```
   GET /health
   Response: "OK" (200)
   ```

2. **LP_ADD Notification**
   ```
   POST /api/lp-add
   Authorization: Bearer <BOT_API_KEY>
   Content-Type: application/json
   
   Payload:
   {
     "tokenAddress": "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
     "creatorAddress": "0x8e3cf8fe85a40c70a56f128f8e444c7ea864480d",
     "txCallData": "0xf305d719000000000000000000000000742d35cc..."
   }
   
   Response:
   {
     "status": "success",
     "message": "LP_ADD notification received and processed",
     "data": {
       "tokenAddress": "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
       "creatorAddress": "0x8e3cf8fe85a40c70a56f128f8e444c7ea864480d"
     }
   }
   ```

#### Configuration

Add these environment variables:

```bash
# Bot HTTP server configuration
BOT_HTTP_PORT=8080                    # Port for HTTP server (default: 8080)
BOT_API_KEY=your-secure-api-key-here  # API key for authentication

# RPC service configuration  
BOT_API_URL=http://localhost:8080     # Bot service URL (default: http://localhost:8080)
```

#### Testing

```bash
# Test the API
go run scripts/test-bot-api.go
```

---

**Built with ❤️ for the DeFi community** 