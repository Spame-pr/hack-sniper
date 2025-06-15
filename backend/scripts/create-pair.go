package main

//Base sepolia
//Router: 0x1689E7B1F10000AE47eBfE339a4f69dECd19F602
//Factory: 0x7Ae58f10f7849cA6F5fB71b7f45CB416c9204b1e

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

// TxParams holds common transaction parameters
type TxParams struct {
	PrivateKey     *ecdsa.PrivateKey
	FromAddress    common.Address
	NetworkID      *big.Int
	GasPrice       *big.Int
	MaxFeePerGas   *big.Int
	MaxPriorityFee *big.Int
	Client         *ethclient.Client
	ProxyClient    *ethclient.Client
	Nonce          *uint64
}

// Factory ABI for creating pairs
const factoryABI = `[
	{
		"inputs": [
			{"internalType": "address", "name": "tokenA", "type": "address"},
			{"internalType": "address", "name": "tokenB", "type": "address"}
		],
		"name": "createPair",
		"outputs": [{"internalType": "address", "name": "pair", "type": "address"}],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"internalType": "address", "name": "tokenA", "type": "address"},
			{"internalType": "address", "name": "tokenB", "type": "address"}
		],
		"name": "getPair",
		"outputs": [{"internalType": "address", "name": "pair", "type": "address"}],
		"stateMutability": "view",
		"type": "function"
	}
]`

// Simplified ABI for Uniswap V2 Router
const routerABI = `[
	{
		"inputs": [
			{"internalType": "address", "name": "token", "type": "address"},
			{"internalType": "uint256", "name": "amountTokenDesired", "type": "uint256"},
			{"internalType": "uint256", "name": "amountTokenMin", "type": "uint256"},
			{"internalType": "uint256", "name": "amountETHMin", "type": "uint256"},
			{"internalType": "address", "name": "to", "type": "address"},
			{"internalType": "uint256", "name": "deadline", "type": "uint256"}
		],
		"name": "addLiquidityETH",
		"outputs": [
			{"internalType": "uint256", "name": "amountToken", "type": "uint256"},
			{"internalType": "uint256", "name": "amountETH", "type": "uint256"},
			{"internalType": "uint256", "name": "liquidity", "type": "uint256"}
		],
		"stateMutability": "payable",
		"type": "function"
	}
]`

// ERC20 ABI for token operations
const erc20ABI = `[
	{
		"inputs": [
			{"internalType": "address", "name": "spender", "type": "address"},
			{"internalType": "uint256", "name": "amount", "type": "uint256"}
		],
		"name": "approve",
		"outputs": [{"internalType": "bool", "name": "", "type": "bool"}],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]`

// createPair creates a new Uniswap V2 pair if it doesn't exist
func createPair(params *TxParams, factoryAddress string, tokenA, tokenB common.Address) error {
	fmt.Println("🔍 Step 1: Checking if pair already exists...")

	// Parse Factory ABI
	factoryABIParsed, err := abi.JSON(strings.NewReader(factoryABI))
	if err != nil {
		return fmt.Errorf("failed to parse factory ABI: %v", err)
	}

	// Check if pair already exists
	getPairData, err := factoryABIParsed.Pack("getPair", tokenA, tokenB)
	if err != nil {
		return fmt.Errorf("failed to pack getPair data: %v", err)
	}

	factoryAddr := common.HexToAddress(factoryAddress)
	msg := ethereum.CallMsg{
		To:   &factoryAddr,
		Data: getPairData,
	}

	result, err := params.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Printf("⚠️  Failed to call getPair: %v", err)
	} else {
		pairAddress := common.BytesToAddress(result)
		if pairAddress != (common.Address{}) {
			fmt.Printf("✅ Pair already exists at: %s\n", pairAddress.Hex())
			fmt.Println("⏭️  Skipping pair creation, proceeding to approve...")
			return nil
		}
	}

	// Create new pair
	fmt.Println("📝 Step 2: Creating new pair...")

	createPairData, err := factoryABIParsed.Pack("createPair", tokenA, tokenB)
	if err != nil {
		return fmt.Errorf("failed to pack createPair data: %v", err)
	}

	// Create pair transaction
	createPairTx := types.NewTransaction(*params.Nonce, factoryAddr, big.NewInt(0), 3000000, params.GasPrice, createPairData)

	// Sign create pair transaction
	signedCreatePairTx, err := types.SignTx(createPairTx, types.NewEIP155Signer(params.NetworkID), params.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to sign create pair transaction: %v", err)
	}

	fmt.Printf("🎯 Create pair transaction hash: %s\n", signedCreatePairTx.Hash().Hex())

	// Send create pair transaction through proxy
	fmt.Println("📡 Sending create pair transaction through RPC proxy...")
	err = params.ProxyClient.SendTransaction(context.Background(), signedCreatePairTx)
	if err != nil {
		log.Printf("❌ Failed to send create pair transaction through proxy: %v", err)
		return err
	} else {
		fmt.Printf("✅ Create pair transaction sent through proxy: %s\n", signedCreatePairTx.Hash().Hex())
	}

	// Wait for pair creation confirmation
	fmt.Println("⏳ Waiting for pair creation confirmation...")
	time.Sleep(10 * time.Second)

	// Increment nonce for next transaction
	*params.Nonce++

	return nil
}

// approve approves token spending for the router
func approve(params *TxParams, tokenAddress, routerAddress common.Address, tokenAmount *big.Int) error {
	fmt.Println("📝 Step 3: Approving token spending...")

	// Parse ERC20 ABI
	erc20ABIParsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return fmt.Errorf("failed to parse ERC20 ABI: %v", err)
	}

	// Create approve transaction data
	approveData, err := erc20ABIParsed.Pack("approve", routerAddress, tokenAmount)
	if err != nil {
		return fmt.Errorf("failed to pack approve data: %v", err)
	}

	// Create approve transaction
	approveTx := types.NewTransaction(*params.Nonce, tokenAddress, big.NewInt(0), 100000, params.GasPrice, approveData)

	// Sign approve transaction
	signedApproveTx, err := types.SignTx(approveTx, types.NewEIP155Signer(params.NetworkID), params.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to sign approve transaction: %v", err)
	}

	// Send approve transaction through proxy
	err = params.ProxyClient.SendTransaction(context.Background(), signedApproveTx)
	if err != nil {
		log.Printf("⚠️  Failed to send approve transaction through proxy: %v", err)
		log.Println("📝 This might be normal if you don't have the custom token")
		return err
	} else {
		fmt.Printf("✅ Approve transaction sent: %s\n", signedApproveTx.Hash().Hex())
		time.Sleep(5 * time.Second)
	}

	// Increment nonce for next transaction
	*params.Nonce++

	return nil
}

// addLiquidity adds liquidity to the pair
func addLiquidity(params *TxParams, routerAddress, tokenAddress common.Address, tokenAmount, ethAmount *big.Int) error {
	fmt.Println("📝 Step 4: Adding liquidity...")

	// Parse Router ABI
	routerABIParsed, err := abi.JSON(strings.NewReader(routerABI))
	if err != nil {
		return fmt.Errorf("failed to parse router ABI: %v", err)
	}

	// Calculate deadline (10 minutes from now)
	deadline := big.NewInt(time.Now().Add(10 * time.Minute).Unix())

	// Create addLiquidityETH transaction data
	liquidityData, err := routerABIParsed.Pack("addLiquidityETH",
		tokenAddress,       // token
		tokenAmount,        // amountTokenDesired
		big.NewInt(0),      // amountTokenMin (0 for testing)
		big.NewInt(0),      // amountETHMin (0 for testing)
		params.FromAddress, // to
		deadline,           // deadline
	)
	if err != nil {
		return fmt.Errorf("failed to pack liquidity data: %v", err)
	}

	// Create EIP-1559 liquidity transaction
	liquidityTx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   params.NetworkID,
		Nonce:     *params.Nonce,
		GasTipCap: params.MaxPriorityFee,
		GasFeeCap: params.MaxFeePerGas,
		Gas:       450000,
		To:        &routerAddress,
		Value:     ethAmount,
		Data:      liquidityData,
	})

	// Sign liquidity transaction
	signedLiquidityTx, err := types.SignTx(liquidityTx, types.NewLondonSigner(params.NetworkID), params.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to sign liquidity transaction: %v", err)
	}

	fmt.Printf("🎯 Liquidity transaction hash: %s\n", signedLiquidityTx.Hash().Hex())
	fmt.Printf("📊 Transaction details:\n")
	fmt.Printf("   - From: %s\n", params.FromAddress.Hex())
	fmt.Printf("   - To: %s\n", routerAddress.Hex())
	fmt.Printf("   - Value: %s ETH\n", new(big.Float).Quo(new(big.Float).SetInt(ethAmount), big.NewFloat(1e18)).Text('f', 4))
	fmt.Printf("   - Gas Limit: %d\n", liquidityTx.Gas())
	fmt.Printf("   - Gas Price: %s Gwei\n", new(big.Float).Quo(new(big.Float).SetInt(params.GasPrice), big.NewFloat(1e9)).Text('f', 2))

	// Send liquidity transaction through proxy
	fmt.Println("📡 Sending liquidity transaction through RPC proxy...")
	err = params.ProxyClient.SendTransaction(context.Background(), signedLiquidityTx)
	if err != nil {
		log.Printf("❌ Failed to send liquidity transaction through proxy: %v", err)
		log.Println("💡 This might be expected if you don't have sufficient ETH or tokens")
		log.Println("🔍 Check your RPC proxy logs to see if LP_ADD was detected")
		return err
	} else {
		fmt.Printf("✅ Liquidity transaction sent through proxy: %s\n", signedLiquidityTx.Hash().Hex())
		fmt.Println("🔍 Check your RPC proxy logs to see if LP_ADD was detected!")
	}

	return nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Configuration
	proxyRPC := "http://localhost:8545" // Your RPC proxy
	baseRPC := os.Getenv("BASE_RPC_URL")

	factoryAddress := os.Getenv("UNISWAP_V2_FACTORY")
	routerAddress := os.Getenv("UNISWAP_V2_ROUTER")

	privateKeyHex := os.Getenv("ADMIN_PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatal("❌ ADMIN_PRIVATE_KEY environment variable is required")
	}

	fmt.Println("🚀 Creating New Uniswap V2 Pair and Adding Liquidity...")
	fmt.Printf("📡 Proxy RPC: %s\n", proxyRPC)
	fmt.Printf("🔗 Base Sepolia RPC: %s\n", baseRPC)
	fmt.Printf("🏭 Factory: %s\n", factoryAddress)
	fmt.Printf("🏭 Router: %s\n", routerAddress)

	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("❌ Failed to parse private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("❌ Cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Printf("👛 From address: %s\n", fromAddress.Hex())

	// Connect to Base Sepolia network (for reading data)
	client, err := ethclient.Dial(baseRPC)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Base Sepolia: %v", err)
	}

	// Connect to proxy (for sending transactions)
	proxyClient, err := ethclient.Dial(proxyRPC)
	if err != nil {
		log.Fatalf("❌ Failed to connect to proxy: %v", err)
	}

	// Get network ID
	networkID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatalf("❌ Failed to get network ID: %v", err)
	}
	fmt.Printf("🌐 Network ID: %s\n", networkID.String())

	// Get nonce
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatalf("❌ Failed to get nonce: %v", err)
	}

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf("❌ Failed to get gas price: %v", err)
	}

	// Get EIP-1559 gas parameters
	gasTipCap := big.NewInt(2100000)

	// Calculate max fee per gas (base fee + tip with buffer)
	maxFeePerGas := new(big.Int).Add(gasPrice, gasTipCap)
	maxFeePerGas = new(big.Int).Mul(maxFeePerGas, big.NewInt(12))
	maxFeePerGas = new(big.Int).Div(maxFeePerGas, big.NewInt(10))

	// Token addresses - you can replace these with actual tokens
	// WETH on Base Sepolia
	wethAddress := common.HexToAddress("0x4200000000000000000000000000000000000006") // WETH on Base
	// Example token address (replace with your token)
	tokenAddress := common.HexToAddress("0xc2159eCD56C1D0162D636e120F73939F0cb01aC5") // Replace with your token

	fmt.Printf("🪙 Token A (WETH): %s\n", wethAddress.Hex())
	fmt.Printf("🪙 Token B (Custom): %s\n", tokenAddress.Hex())

	// Define amounts for liquidity
	ethAmount := big.NewInt(1000000000000)         // 0.000001 ETH
	tokenAmount := big.NewInt(1000000000000000000) // 1 Token (assuming 18 decimals)

	fmt.Printf("💰 Adding liquidity: %s ETH + %s Token\n",
		new(big.Float).Quo(new(big.Float).SetInt(ethAmount), big.NewFloat(1e18)).Text('f', 4),
		new(big.Float).Quo(new(big.Float).SetInt(tokenAmount), big.NewFloat(1e18)).Text('f', 4))

	// Create transaction parameters
	params := &TxParams{
		PrivateKey:     privateKey,
		FromAddress:    fromAddress,
		NetworkID:      networkID,
		GasPrice:       gasPrice,
		MaxFeePerGas:   maxFeePerGas,
		MaxPriorityFee: gasTipCap,
		Client:         client,
		ProxyClient:    proxyClient,
		Nonce:          &nonce,
	}

	//// Step 1: Create pair (if needed)
	//if err := createPair(params, factoryAddress, tokenAddress, wethAddress); err != nil {
	//	log.Printf("❌ Create pair failed: %v", err)
	//	// Continue anyway as pair might already exist
	//}
	//
	//// Step 2: Approve token spending
	//if err := approve(params, tokenAddress, common.HexToAddress(routerAddress), tokenAmount); err != nil {
	//	log.Printf("❌ Approve failed: %v", err)
	//	// Continue anyway as approval might not be needed in some cases
	//}

	// Step 3: Add liquidity
	if err := addLiquidity(params, common.HexToAddress(routerAddress), tokenAddress, tokenAmount, ethAmount); err != nil {
		//log.Printf("❌ Add liquidity failed: %v", err)
	}

	fmt.Println("\n🎉 Script completed!")
	fmt.Println("📝 Note: Transactions may fail if you don't have sufficient ETH or tokens")
	fmt.Println("🔍 The important part is testing that your RPC proxy receives and processes the transactions")
	fmt.Println("💡 If pair creation succeeded, you should see the new pair address in the transaction logs")
}
