# Sniper Bot Backend Architecture

## 🎯 Executive Summary

The Sniper Bot Backend is a sophisticated MEV infrastructure system that transforms token sniping from creator exploitation into a profitable auction mechanism. Built with Go, it provides secure wallet management, transparent bidding systems, and atomic transaction bundling to create fair competition among snipers while maximizing revenue for token creators.

## 🏗️ System Overview

### Core Philosophy

Rather than allowing uncompensated front-running, the system creates a **structured MEV marketplace** where:

1. **Token creators** capture value through competitive sniper auctions
2. **Snipers** compete transparently with ETH bribes for early access
3. **Market efficiency** is maximized through fair price discovery
4. **Atomic execution** ensures bribes are paid before tokens are transferred

### High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Snipers     │    │  Backend Stack  │    │ Token Creators  │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│ Telegram Bot    │◄──►│ Bot Service     │    │ MetaMask/Rabby  │
│ Wallet UI       │    │ RPC Proxy       │◄──►│ DEX Interface   │
│ Bid Management  │    │ Bundle Manager  │    │ Token Contracts │
└─────────────────┘    │ Database Layer  │    └─────────────────┘
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │ Smart Contracts │
                       │ Base Network    │
                       └─────────────────┘
```

## 🔧 Component Architecture

### 1. Telegram Bot Service (`pkg/bot`)

The Telegram Bot Service handles all sniper interactions through a secure, user-friendly interface.

#### Core Responsibilities
- **User Registration**: Secure wallet generation and storage
- **Bid Management**: Snipe request processing and validation
- **Balance Monitoring**: Real-time wallet balance tracking
- **Status Updates**: Transaction confirmations and notifications

#### Technical Implementation
```go
type BotService struct {
    bot           *tgbotapi.BotAPI
    db            database.Interface
    walletManager *wallet.Manager
    rpcClient     *rpc.Client
    logger        *logrus.Logger
}
```

#### Command Processing Pipeline
1. **Input Validation**: Sanitize and validate user commands
2. **Authentication**: Verify user permissions and rate limits
3. **Business Logic**: Execute command-specific operations
4. **Database Operations**: Persist state changes atomically
5. **Response Generation**: Format and send user responses

#### Security Features
- **Rate Limiting**: Prevent spam and abuse
- **Input Sanitization**: SQL injection and XSS protection
- **Access Control**: User-based permission system
- **Audit Logging**: Complete command execution history

### 2. RPC Proxy Service (`pkg/rpc`)

The RPC Proxy Service intercepts token creator transactions and coordinates bundle construction.

#### Core Responsibilities
- **Transaction Interception**: Monitor for LP_ADD transactions
- **Bundle Coordination**: Trigger sniper bundle construction
- **MEV Detection**: Identify profitable MEV opportunities
- **Base Network Forwarding**: Transparent proxy for non-MEV transactions

#### Technical Implementation
```go
type RPCService struct {
    server        *http.Server
    baseClient    *ethclient.Client
    bundleManager *bundle.Manager
    detector      *detector.LPAddDetector
    logger        *logrus.Logger
}
```

#### Transaction Processing Flow
1. **Request Interception**: Capture incoming JSON-RPC requests
2. **Transaction Analysis**: Parse and classify transaction types
3. **LP_ADD Detection**: Identify liquidity addition transactions
4. **Bundle Triggering**: Initiate sniper bundle construction
5. **Response Handling**: Return appropriate responses to clients

#### Performance Optimizations
- **Connection Pooling**: Efficient Base network connection management
- **Async Processing**: Non-blocking transaction analysis
- **Caching**: Frequently accessed data caching
- **Request Batching**: Optimize Base network interactions

### 3. Bundle Manager (`pkg/bundle`)

The Bundle Manager orchestrates the creation and submission of MEV transaction bundles.

#### Core Responsibilities
- **Bid Aggregation**: Collect and sort sniper bids by bribe amount
- **Gas Management**: Calculate optimal gas prices for transaction ordering
- **Bundle Construction**: Create ordered transaction arrays
- **Submission Handling**: Submit bundles to Base sequencer

#### Technical Implementation
```go
type BundleManager struct {
    db           database.Interface
    ethClient    *ethclient.Client
    sniperContract *contracts.Sniper
    gasEstimator *gas.Estimator
    logger       *logrus.Logger
}
```

#### Bundle Construction Algorithm
```go
func (bm *BundleManager) ConstructBundle(
    lpAddTx *types.Transaction,
    tokenAddress common.Address,
    creatorAddress common.Address,
) (*Bundle, error) {
    // 1. Fetch active bids for token
    bids, err := bm.db.GetActiveBids(tokenAddress)
    if err != nil {
        return nil, err
    }
    
    // 2. Sort bids by bribe amount (descending)
    sort.Slice(bids, func(i, j int) bool {
        return bids[i].BribeAmount.Cmp(bids[j].BribeAmount) > 0
    })
    
    // 3. Construct bundle with proper gas ordering
    bundle := &Bundle{
        Transactions: []*types.Transaction{lpAddTx},
    }
    
    baseGasPrice := lpAddTx.GasPrice()
    for i, bid := range bids {
        sniperTx, err := bm.constructSniperTx(bid, creatorAddress)
        if err != nil {
            continue
        }
        
        // Ensure proper ordering with decreasing gas prices
        gasPrice := new(big.Int).Sub(baseGasPrice, big.NewInt(int64(i+1)))
        sniperTx = types.NewTx(&types.DynamicFeeTx{
            // ... transaction parameters
            GasFeeCap: gasPrice,
        })
        
        bundle.Transactions = append(bundle.Transactions, sniperTx)
    }
    
    return bundle, nil
}
```

#### Economic Mechanisms
- **First-Price Auction**: Highest bidders win snipe positions
- **Gas Price Ordering**: Ensures proper transaction sequencing
- **Slippage Protection**: Minimum output validation in smart contracts
- **Atomic Execution**: All-or-nothing bundle execution

### 4. Wallet Manager (`pkg/wallet`)

The Wallet Manager provides secure wallet generation and management for snipers.

#### Core Responsibilities
- **Secure Generation**: Cryptographically secure private key creation
- **Encrypted Storage**: AES-256-GCM encryption for database storage
- **Transaction Signing**: Secure transaction signature generation
- **Balance Monitoring**: Real-time balance tracking and updates

#### Technical Implementation
```go
type WalletManager struct {
    db          database.Interface
    ethClient   *ethclient.Client
    encryptor   *crypto.AESEncryptor
    generator   *crypto.KeyGenerator
    logger      *logrus.Logger
}
```

#### Security Architecture
```go
// Wallet generation with entropy
func (wm *WalletManager) GenerateWallet(userID string) (*Wallet, error) {
    privateKey, err := crypto.GenerateKey()
    if err != nil {
        return nil, err
    }
    
    publicKey := privateKey.Public()
    address := crypto.PubkeyToAddress(publicKey.(*ecdsa.PublicKey))
    
    // Encrypt private key before storage
    encryptedKey, err := wm.encryptor.Encrypt(
        crypto.FromECDSA(privateKey),
    )
    if err != nil {
        return nil, err
    }
    
    wallet := &Wallet{
        UserID:           userID,
        Address:          address,
        EncryptedPrivKey: encryptedKey,
        CreatedAt:        time.Now(),
    }
    
    return wallet, wm.db.SaveWallet(wallet)
}
```

#### Encryption Standards
- **Algorithm**: AES-256-GCM with 256-bit keys
- **Key Derivation**: PBKDF2 with SHA-256 and 100,000 iterations
- **Nonce Generation**: Cryptographically secure random nonces
- **Authentication**: Galois/Counter Mode for authenticated encryption

### 5. Database Layer (`pkg/database`)

The Database Layer provides persistent storage with ACID compliance and performance optimization.

#### Core Responsibilities
- **Data Persistence**: Reliable storage for wallets, bids, and transactions
- **Query Optimization**: Efficient data retrieval with proper indexing
- **Transaction Management**: ACID compliance for critical operations
- **Connection Management**: Pool-based connection handling

#### Schema Design

##### Wallets Table
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

##### Snipe Bids Table
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
    expires_at TIMESTAMP NULL,
    
    INDEX idx_snipe_bids_token_address (token_address),
    INDEX idx_snipe_bids_status (status),
    INDEX idx_snipe_bids_user_id (user_id),
    INDEX idx_snipe_bids_created_at (created_at),
    INDEX idx_snipe_bids_bribe_amount (bribe_amount DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

#### Performance Optimizations

##### Connection Pooling Configuration
```go
func NewDatabase(dsn string) (*Database, error) {
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }
    
    // Connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(1 * time.Minute)
    
    return &Database{db: db}, nil
}
```

##### Query Optimization Strategies
- **Composite Indexes**: Multi-column indexes for complex queries
- **Covering Indexes**: Include frequently accessed columns
- **Partitioning**: Table partitioning for large datasets
- **Query Plan Analysis**: Regular EXPLAIN analysis for optimization

## 🔒 Security Architecture

### Authentication and Authorization

#### Multi-Layer Security Model
1. **Telegram Authentication**: User verification through Telegram's OAuth
2. **API Key Authentication**: Service-to-service authentication
3. **Wallet-Level Permissions**: User-specific access control
4. **Rate Limiting**: Request throttling and abuse prevention

#### Access Control Matrix
| User Type | Wallet Creation | Bid Submission | Admin Functions |
|-----------|----------------|----------------|-----------------|
| Regular Sniper | ✅ (Own) | ✅ (Own) | ❌ |
| Admin | ✅ (Any) | ✅ (Any) | ✅ |
| System Service | ❌ | ❌ | ✅ (Limited) |

### Data Protection

#### Encryption at Rest
- **Private Keys**: AES-256-GCM with unique keys per wallet
- **Database**: TDE (Transparent Data Encryption) for MySQL
- **Backups**: Encrypted backup storage with separate key management

#### Encryption in Transit
- **HTTPS/TLS 1.3**: All external communications encrypted
- **Database Connections**: TLS-encrypted MySQL connections
- **Internal Services**: mTLS for service-to-service communication

### Input Validation and Sanitization

#### Validation Pipeline
```go
type InputValidator struct {
    ethereumAddressRegex *regexp.Regexp
    amountValidator      *big.Int
    deadlineValidator    time.Duration
}

func (iv *InputValidator) ValidateSnipeRequest(req *SnipeRequest) error {
    // Address validation
    if !iv.ethereumAddressRegex.MatchString(req.TokenAddress) {
        return ErrInvalidTokenAddress
    }
    
    // Amount validation
    if req.BribeAmount.Cmp(MinBribeAmount) < 0 {
        return ErrInsufficientBribe
    }
    
    // Rate limiting
    if err := iv.checkRateLimit(req.UserID); err != nil {
        return err
    }
    
    return nil
}
```

## 📊 Performance and Scalability

### Performance Metrics

#### Target Performance Benchmarks
- **Bundle Construction Time**: < 100ms for 50 concurrent bids
- **Database Query Response**: < 10ms for 95th percentile
- **Transaction Processing**: > 1000 TPS theoretical capacity
- **Memory Usage**: < 512MB per service instance

### Scalability Architecture

#### Horizontal Scaling Strategy
```yaml
# Kubernetes deployment example
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sniper-bot-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: sniper-bot
  template:
    metadata:
      labels:
        app: sniper-bot
    spec:
      containers:
      - name: bot-service
        image: sniper-bot:latest
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

#### Database Scaling
- **Read Replicas**: Multiple read-only database instances
- **Connection Pooling**: Efficient connection distribution
- **Query Optimization**: Regular performance tuning
- **Caching Layer**: Redis for frequently accessed data

### Monitoring and Observability

#### Key Performance Indicators (KPIs)
```go
type Metrics struct {
    // Business metrics
    SuccessfulSnipes    prometheus.Counter
    TotalBribeVolume   prometheus.Gauge
    AverageBribeAmount prometheus.Histogram
    
    // Technical metrics
    DatabaseQueryTime  prometheus.Histogram
    BundleConstructionTime prometheus.Histogram
    APIResponseTime    prometheus.Histogram
    ErrorRate         prometheus.Counter
}
```

#### Logging Architecture
- **Structured Logging**: JSON format with correlation IDs
- **Log Levels**: DEBUG, INFO, WARN, ERROR with appropriate filtering
- **Centralized Collection**: ELK stack or similar log aggregation
- **Security Events**: Special handling for authentication failures

## 🔄 Integration Points

### External System Integration

#### Base Network Integration
```go
type BaseClient struct {
    rpcClient    *rpc.Client
    ethClient    *ethclient.Client
    wsClient     *websocket.Conn
    blockMonitor *BlockMonitor
}

func (bc *BaseClient) MonitorLPAddTransactions() {
    headers := make(chan *types.Header)
    sub, err := bc.ethClient.SubscribeNewHead(context.Background(), headers)
    if err != nil {
        log.Fatal(err)
    }
    
    for {
        select {
        case header := <-headers:
            block, err := bc.ethClient.BlockByHash(context.Background(), header.Hash())
            if err != nil {
                continue
            }
            
            bc.analyzeBlockForLPAdd(block)
            
        case err := <-sub.Err():
            log.Printf("Subscription error: %v", err)
        }
    }
}
```

#### Smart Contract Integration
- **Event Monitoring**: Listen for SnipeExecuted events
- **Transaction Construction**: Build contract interaction transactions
- **Gas Estimation**: Dynamic gas price calculation
- **Error Handling**: Comprehensive revert reason parsing

### Internal Service Communication

#### Service Discovery
```go
type ServiceRegistry struct {
    services map[string]*ServiceInfo
    mutex    sync.RWMutex
}

type ServiceInfo struct {
    Address    string
    Port       int
    Health     string
    LastSeen   time.Time
}
```

#### Message Queuing
- **Async Processing**: Redis-based job queues for heavy operations
- **Event Sourcing**: Event-driven architecture for state changes
- **Dead Letter Queues**: Failed job recovery mechanisms

## 🚀 Deployment Architecture

### Environment Configuration

#### Development Environment
```bash
# Local development setup
export NODE_ENV=development
export LOG_LEVEL=debug
export DATABASE_URL=mysql://user:pass@localhost:3306/sniper_bot_dev
export TELEGRAM_BOT_TOKEN=dev_bot_token
export BASE_RPC_URL=https://base-sepolia.g.alchemy.com/v2/api-key
```

#### Production Environment
```bash
# Production configuration
export NODE_ENV=production
export LOG_LEVEL=info
export DATABASE_URL=mysql://user:pass@prod-db:3306/sniper_bot
export TELEGRAM_BOT_TOKEN=prod_bot_token
export BASE_RPC_URL=https://base.llamarpc.com
```

### Docker Configuration

#### Multi-Stage Build
```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bot .
CMD ["./bot"]
```

### Infrastructure as Code

#### Terraform Configuration
```hcl
resource "aws_ecs_service" "sniper_bot" {
  name            = "sniper-bot"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.sniper_bot.arn
  desired_count   = 3

  deployment_configuration {
    maximum_percent         = 200
    minimum_healthy_percent = 100
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.sniper_bot.arn
    container_name   = "sniper-bot"
    container_port   = 8080
  }
}
```

## 🔍 Testing Strategy

### Testing Pyramid

#### Unit Tests (70%)
```go
func TestBundleConstruction(t *testing.T) {
    mockDB := &MockDatabase{}
    bundleManager := NewBundleManager(mockDB, mockEthClient)
    
    // Test data
    lpAddTx := createMockLPAddTransaction()
    bids := createMockBids(5) // 5 mock bids
    
    mockDB.On("GetActiveBids", mock.Anything).Return(bids, nil)
    
    bundle, err := bundleManager.ConstructBundle(lpAddTx, tokenAddr, creatorAddr)
    
    assert.NoError(t, err)
    assert.Len(t, bundle.Transactions, 6) // LP_ADD + 5 snipe txs
    
    // Verify gas price ordering
    for i := 1; i < len(bundle.Transactions); i++ {
        assert.True(t, bundle.Transactions[i-1].GasPrice().Cmp(
            bundle.Transactions[i].GasPrice()) > 0)
    }
}
```

#### Integration Tests (20%)
```go
func TestEndToEndSnipeFlow(t *testing.T) {
    // Setup test environment
    testDB := setupTestDatabase()
    testBot := setupTestBotService(testDB)
    testRPC := setupTestRPCService(testDB)
    
    // Register test user
    userID := "test_user_123"
    wallet, err := testBot.RegisterUser(userID)
    assert.NoError(t, err)
    
    // Fund wallet
    fundWallet(wallet.Address, ethAmount)
    
    // Submit snipe bid
    err = testBot.ProcessSnipeBid(userID, tokenAddr, bribeAmount)
    assert.NoError(t, err)
    
    // Simulate LP_ADD
    lpAddTx := createLPAddTransaction(tokenAddr)
    err = testRPC.ProcessTransaction(lpAddTx)
    assert.NoError(t, err)
    
    // Verify bundle creation and execution
    // ... verification logic
}
```

#### End-to-End Tests (10%)
- **Full System Tests**: Complete user journey testing
- **Performance Tests**: Load and stress testing
- **Security Tests**: Penetration testing and vulnerability assessment

### Continuous Integration

#### GitHub Actions Pipeline
```yaml
name: CI/CD Pipeline
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: 1.21
    
    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
```

## 📈 Future Roadmap

### Phase 1: Core Optimization (Q1 2024)
- **Performance Improvements**: Database query optimization
- **Security Enhancements**: Additional security audits
- **Monitoring**: Advanced observability implementation

### Phase 2: Feature Expansion (Q2 2024)
- **Multi-Chain Support**: Ethereum, Arbitrum, Polygon integration
- **Advanced Bidding**: Time-based and Dutch auction mechanisms
- **REST API**: Programmatic access for advanced users

### Phase 3: Enterprise Features (Q3 2024)
- **White-Label Solutions**: Customizable deployments
- **Analytics Dashboard**: Advanced reporting and insights
- **Institutional Support**: High-volume user management

### Phase 4: Ecosystem Integration (Q4 2024)
- **DEX Partnerships**: Direct integration with major DEXes
- **MEV Infrastructure**: Builder and relay integrations
- **Mobile Applications**: iOS and Android app development

---

## 📚 Additional Resources

### Development Resources
- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [Database Design Patterns](https://en.wikipedia.org/wiki/Database_design)
- [MEV Documentation](https://ethereum.org/en/developers/docs/mev/)

### Security Resources
- [OWASP Go Security Guide](https://owasp.org/www-project-go-secure-coding-practices-guide/)
- [Smart Contract Security](https://consensys.github.io/smart-contract-best-practices/)
- [Cryptography Best Practices](https://cryptography.io/en/latest/)

---

*This architecture document is living documentation that evolves with the system. Last updated: 2024* 