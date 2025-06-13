# Sniper Bot System Architecture

## Overview

The Sniper Bot System is a sophisticated MEV solution that allows token creators to monetize the sniping of their tokens. Instead of losing value to front-running bots, token creators can auction off the right to snipe their tokens and receive bribes directly.

## Core Concept

1. **Token creators** deploy tokens and add liquidity through our custom RPC
2. **Snipers** compete by offering ETH bribes to be included in the snipe bundle
3. **System** constructs ordered transaction bundles ensuring fair competition
4. **Bribes** go directly to token creators, maximizing their profit

## System Components

### 1. Telegram Bot Service
The Telegram bot handles all user interactions for snipers:
- User registration and wallet creation
- Balance checking and management
- Snipe request processing
- Command validation and response formatting

### 2. RPC Proxy Service
The RPC proxy intercepts transactions from token creators:
- Transaction monitoring and filtering
- LP_ADD detection and processing
- Bundle coordination with bot service
- Transparent request forwarding

### 3. Bundle Manager
Handles the creation and submission of transaction bundles:
- Bid sorting by bribe amount
- Gas price management for ordering
- Transaction construction and signing
- Bundle submission to Base sequencer

### 4. Wallet Manager
Manages sniper wallets and private keys:
- Secure wallet generation
- Encrypted private key storage
- Transaction signing capabilities
- Address management and lookup

### 5. DEX Integration
Handles interactions with Uniswap V2-style DEXes:
- Router contract interactions
- Factory contract management
- Pair contract operations
- ABI definitions and bindings

### 6. Database Layer (MySQL)
Persistent storage using MySQL 8.0:
- Snipe bid storage and retrieval
- User data management
- Transaction history tracking
- Performance-optimized indexing

## Database Schema

### MySQL Configuration
- **Engine**: InnoDB for ACID compliance
- **Charset**: UTF8MB4 for full Unicode support
- **Collation**: utf8mb4_unicode_ci for proper sorting
- **Indexes**: Optimized for token_address and status queries

### Tables
```sql
CREATE TABLE snipe_bids (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token_address VARCHAR(255) NOT NULL,
    bribe_amount VARCHAR(255) NOT NULL,
    wallet VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) NOT NULL,
    INDEX idx_snipe_bids_token_address (token_address),
    INDEX idx_snipe_bids_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

## Security Considerations

- **Private Key Security**: Keys are generated using cryptographically secure methods
- **Database Security**: MySQL with proper authentication and connection encryption
- **Transaction Ordering**: Gas price manipulation ensures proper bundle ordering
- **Input Validation**: All inputs validated before processing
- **Access Control**: User operations isolated by user ID

## Performance Features

### Database Optimization
- **Connection Pooling**: Efficient MySQL connection management
- **Indexing Strategy**: Optimized indexes for common queries
- **Query Optimization**: Efficient sorting using DECIMAL casting for bribe amounts
- **Transaction Management**: Proper ACID compliance for data integrity

### Scalability
- **Horizontal Scaling**: Stateless services support load balancing
- **Database Clustering**: MySQL supports read replicas and clustering
- **Caching**: In-memory caching for frequently accessed data
- **Batch Processing**: Efficient handling of multiple requests

## Deployment

### Development Setup
```bash
# Start MySQL
docker run --name sniper-mysql \
  -e MYSQL_ROOT_PASSWORD=root_password \
  -e MYSQL_DATABASE=sniper_bot \
  -e MYSQL_USER=sniper_user \
  -e MYSQL_PASSWORD=sniper_password \
  -p 3306:3306 -d mysql:8.0 \
  --default-authentication-plugin=mysql_native_password

# Test MySQL connection
make test-mysql

# Run services
make run-bot
make run-rpc
```

### Production Deployment
```bash
# Using Docker Compose
docker-compose up -d
```

The system can be deployed using Docker Compose or manually with the provided scripts. MySQL provides excellent performance and reliability for the persistent storage requirements of the sniper bot system. 