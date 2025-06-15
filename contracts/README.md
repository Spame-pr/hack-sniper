# Sniper Bot Smart Contracts

Smart contracts for the sniper bot system built with Foundry. The contracts enable snipers to execute token purchases with built-in bribe payments to token creators in a single atomic transaction.

## 📋 Overview

The smart contract system consists of:

- **Sniper.sol**: Main contract that handles atomic snipe execution with bribes
- **Deployment Scripts**: Automated deployment and configuration scripts
- **Test Suite**: Comprehensive tests for all contract functionality

## 🏗️ Architecture

### Sniper Contract

The core `Sniper.sol` contract provides:

- **Atomic Snipe Execution**: Swap ETH for tokens and send bribe in one transaction
- **Bribe Protection**: Ensures bribes are always sent to token creators
- **Slippage Protection**: Minimum output amount validation
- **Emergency Functions**: Owner-only recovery mechanisms
- **Event Logging**: Complete transaction and bribe tracking

### Key Features

- **🎯 Single Transaction**: Everything happens atomically - no partial executions
- **💰 Guaranteed Bribes**: Bribes are sent before the transaction completes
- **🔒 Slippage Control**: Set minimum tokens to receive for protection
- **⚡ Gas Optimized**: Efficient contract design for lower gas costs
- **🛡️ Secure**: Owner controls with emergency withdrawal capabilities

## 🚀 Quick Start

### Prerequisites

- **Foundry**: Latest version installed
- **Base RPC Access**: For deployment and testing
- **Private Key**: For contract deployment

### Installation

```bash
# Clone and navigate to contracts
cd contracts

# Install dependencies
forge install

# Build contracts
forge build
```

### Configuration

Create a `.env` file:
```bash
# Network configuration
BASE_RPC_URL=https://base.llamarpc.com
PRIVATE_KEY=0xabc123...

# Contract addresses (Base mainnet)
UNISWAP_V2_ROUTER=0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24
```

## 🔧 Development

### Build

```bash
# Compile contracts
forge build

# Build with size optimization
forge build --optimize --optimizer-runs 200
```

### Testing

```bash
# Run all tests
forge test

# Run tests with gas reporting
forge test --gas-report

# Run specific test
forge test --match-test testSnipeWithBribe

# Run tests with verbosity
forge test -vvv
```

### Local Testing

```bash
# Start local Anvil node
anvil

# Run tests against local node
forge test --fork-url http://localhost:8545
```

## 📋 Contract Details

### Sniper.sol

#### Constructor Parameters
- `_router`: Address of the Uniswap V2 compatible router contract

#### Main Functions

##### `snipeWithBribe()`
```solidity
function snipeWithBribe(
    address token,           // Token to purchase
    address payable creator, // Token creator (receives bribe)
    uint256 amountOutMin,    // Minimum tokens to receive
    uint256 deadline,        // Transaction deadline
    uint256 bribeAmount      // ETH amount to send as bribe
) external payable
```

**Process:**
1. Validates input parameters and ETH amount
2. Calculates swap amount (msg.value - bribeAmount)
3. Executes token swap through Uniswap V2 router
4. Transfers bribe to token creator
5. Emits SnipeExecuted event

##### `emergencyWithdraw()`
```solidity
function emergencyWithdraw() external onlyOwner
```
- Owner-only function to withdraw stuck ETH

##### `withdrawToken()`
```solidity
function withdrawToken(address token) external onlyOwner
```
- Owner-only function to withdraw stuck ERC-20 tokens

#### Events

```solidity
event SnipeExecuted(
    address indexed sniper,      // Address that called the function
    address indexed token,       // Token that was purchased
    address indexed creator,     // Creator who received the bribe
    uint256 swapAmount,         // ETH used for token swap
    uint256 bribeAmount,        // ETH sent as bribe
    uint256 tokensReceived      // Tokens received from swap
);
```

## 🚀 Deployment

### Deploy to Base Mainnet

```bash
# Deploy Sniper contract
forge script script/Sniper.s.sol:SniperScript \
    --rpc-url $BASE_RPC_URL \
    --private-key $PRIVATE_KEY \
    --broadcast \
    --verify

# Verify on Basescan
forge verify-contract <contract_address> src/Sniper.sol:Sniper \
    --chain base \
    --constructor-args $(cast abi-encode "constructor(address)" $UNISWAP_V2_ROUTER)
```

### Deploy to Base Testnet

```bash
# Deploy to Base Sepolia
forge script script/Sniper.s.sol:SniperScript \
    --rpc-url https://sepolia.base.org \
    --private-key $PRIVATE_KEY \
    --broadcast
```

## 📊 Gas Analysis

### Typical Gas Costs

| Function | Gas Estimate | Notes |
|----------|-------------|-------|
| `snipeWithBribe()` | ~150,000 | Includes swap + transfer |
| `emergencyWithdraw()` | ~21,000 | Simple ETH transfer |
| `withdrawToken()` | ~35,000 | ERC-20 transfer |

### Optimization Features

- **Immutable Variables**: Router address stored as immutable
- **Efficient Validation**: Early returns for invalid inputs
- **Minimal Storage**: Only essential state variables
- **Direct Transfers**: No unnecessary intermediate steps

## 🧪 Testing

### Test Coverage

The test suite covers:

- ✅ **Successful snipes** with various parameters
- ✅ **Bribe validation** and transfer verification
- ✅ **Slippage protection** with minimum output amounts
- ✅ **Deadline validation** for time-sensitive operations
- ✅ **Access control** for owner-only functions
- ✅ **Emergency scenarios** and recovery mechanisms
- ✅ **Event emission** and parameter validation
- ✅ **Edge cases** and error conditions

### Running Specific Tests

```bash
# Test successful snipe execution
forge test --match-test testSnipeWithBribe

# Test access control
forge test --match-test testOnlyOwner

# Test error conditions
forge test --match-test testRevert
```

## 🔍 Security Features

### Input Validation
- **Non-zero addresses**: Validates token and creator addresses
- **Sufficient ETH**: Ensures msg.value > bribeAmount
- **Positive bribes**: Requires bribeAmount > 0

### Access Control
- **Owner-only functions**: Emergency withdrawal and token recovery
- **Immutable router**: Router address cannot be changed after deployment

### Economic Security
- **Atomic execution**: Swap and bribe happen together or fail together
- **Slippage protection**: Minimum output amount prevents sandwich attacks
- **Deadline enforcement**: Prevents transaction from being held indefinitely

## 🔄 Integration

### Backend Integration

The contracts integrate with the backend system:

1. **Bundle Construction**: Backend creates transaction bundles with contract calls
2. **Gas Management**: Backend sets appropriate gas prices for ordering
3. **Event Monitoring**: Backend listens for SnipeExecuted events
4. **Address Management**: Backend tracks sniper wallets and permissions

### Example Integration

```go
// Go code for contract interaction
contract, err := NewSniper(contractAddress, client)
if err != nil {
    return err
}

// Create transaction
tx, err := contract.SnipeWithBribe(
    &bind.TransactOpts{
        From:     sniperAddress,
        Value:    totalETH,
        GasLimit: 200000,
    },
    tokenAddress,
    creatorAddress,
    minTokens,
    deadline,
    bribeAmount,
)
```

## 📋 Contract Addresses

### Base Mainnet
- **Sniper Contract**: `0x...` (Deploy with script)
- **Uniswap V2 Router**: `0x4752ba5DBc23f44D87826276BF6Fd6b1C372aD24`
- **Uniswap V2 Factory**: `0x8909Dc15e40173Ff4699343b6eB8132c65e18eC6`

### Base Sepolia Testnet
- **Sniper Contract**: `0x...` (Deploy with script)
- **Uniswap V2 Router**: `0x...` (Check Base docs)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass: `forge test`
5. Submit a pull request

### Development Guidelines

- **Test Coverage**: Maintain >95% test coverage
- **Gas Optimization**: Profile and optimize gas usage
- **Security**: Follow best practices for smart contract security
- **Documentation**: Update docs for any interface changes

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## ⚠️ Security Disclaimer

These smart contracts handle user funds and should be thoroughly audited before mainnet deployment. The contracts are provided as-is for educational and research purposes. Deploy at your own risk.

## 🆘 Support

- **Issues**: Report bugs via GitHub issues
- **Security**: Report vulnerabilities privately to security@example.com
- **Community**: Join our Discord for development discussions

---

*Powering the future of fair MEV distribution* 🚀
