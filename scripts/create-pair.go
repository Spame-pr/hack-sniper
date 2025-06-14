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

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Configuration
	proxyRPC := "http://localhost:8545" // Your RPC proxy
	baseRPC := os.Getenv("BASE_SEPOLIA_RPC_URL")
	if baseRPC == "" {
		baseRPC = "https://sepolia.base.org"
	}

	factoryAddress := os.Getenv("UNISWAP_V2_FACTORY")
	if factoryAddress == "" {
		factoryAddress = "0x7Ae58f10f7849cA6F5fB71b7f45CB416c9204b1e" // Base Sepolia Factory
	}

	routerAddress := os.Getenv("UNISWAP_V2_ROUTER")
	if routerAddress == "" {
		routerAddress = "0x1689E7B1F10000AE47eBfE339a4f69dECd19F602" // Base Sepolia Router
	}

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

	// Token addresses - you can replace these with actual tokens
	// WETH on Base Sepolia
	wethAddress := common.HexToAddress("0x4200000000000000000000000000000000000006") // WETH on Base
	// Example token address (replace with your token)
	tokenAddress := common.HexToAddress("0x885eD8891b6E18F013B38b54a717312B52a10486") // Replace with your token

	fmt.Printf("🪙 Token A (WETH): %s\n", wethAddress.Hex())
	fmt.Printf("🪙 Token B (Custom): %s\n", tokenAddress.Hex())

	// Define amounts for liquidity
	ethAmount := big.NewInt(1000000000000)         // 0.000001 ETH
	tokenAmount := big.NewInt(1000000000000000000) // 1 Token (assuming 18 decimals)

	// Parse Factory ABI
	factoryABIParsed, err := abi.JSON(strings.NewReader(factoryABI))
	if err != nil {
		log.Fatalf("❌ Failed to parse factory ABI: %v", err)
	}

	// Step 1: Check if pair already exists
	fmt.Println("🔍 Step 1: Checking if pair already exists...")

	getPairData, err := factoryABIParsed.Pack("getPair", tokenAddress, wethAddress)
	if err != nil {
		log.Fatalf("❌ Failed to pack getPair data: %v", err)
	}

	factoryAddr := common.HexToAddress(factoryAddress)
	msg := ethereum.CallMsg{
		To:   &factoryAddr,
		Data: getPairData,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Printf("⚠️  Failed to call getPair: %v", err)
	} else {
		pairAddress := common.BytesToAddress(result)
		if pairAddress != (common.Address{}) {
			fmt.Printf("✅ Pair already exists at: %s\n", pairAddress.Hex())
			fmt.Println("⏭️  Skipping pair creation, proceeding to add liquidity...")
		} else {
			// Step 2: Create new pair
			fmt.Println("📝 Step 2: Creating new pair...")

			createPairData, err := factoryABIParsed.Pack("createPair", tokenAddress, wethAddress)
			if err != nil {
				log.Fatalf("❌ Failed to pack createPair data: %v", err)
			}

			// Create pair transaction
			createPairTx := types.NewTransaction(nonce, common.HexToAddress(factoryAddress), big.NewInt(0), 200000, gasPrice, createPairData)

			// Sign create pair transaction
			signedCreatePairTx, err := types.SignTx(createPairTx, types.NewEIP155Signer(networkID), privateKey)
			if err != nil {
				log.Fatalf("❌ Failed to sign create pair transaction: %v", err)
			}

			fmt.Printf("🎯 Create pair transaction hash: %s\n", signedCreatePairTx.Hash().Hex())

			// Send create pair transaction through proxy
			fmt.Println("📡 Sending create pair transaction through RPC proxy...")
			err = proxyClient.SendTransaction(context.Background(), signedCreatePairTx)
			if err != nil {
				log.Printf("❌ Failed to send create pair transaction through proxy: %v", err)
			} else {
				fmt.Printf("✅ Create pair transaction sent through proxy: %s\n", signedCreatePairTx.Hash().Hex())
			}

			// Wait for pair creation confirmation
			fmt.Println("⏳ Waiting for pair creation confirmation...")
			time.Sleep(10 * time.Second)

			// Increment nonce for next transaction
			nonce++
		}
	}

	return

	// Step 3: Approve token spending
	fmt.Println("📝 Step 3: Approving token spending...")

	fmt.Printf("💰 Adding liquidity: %s ETH + %s Token\n",
		new(big.Float).Quo(new(big.Float).SetInt(ethAmount), big.NewFloat(1e18)).Text('f', 4),
		new(big.Float).Quo(new(big.Float).SetInt(tokenAmount), big.NewFloat(1e18)).Text('f', 4))

	// Parse ERC20 ABI
	erc20ABIParsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		log.Fatalf("❌ Failed to parse ERC20 ABI: %v", err)
	}

	// Create approve transaction data
	approveData, err := erc20ABIParsed.Pack("approve", common.HexToAddress(routerAddress), tokenAmount)
	if err != nil {
		log.Fatalf("❌ Failed to pack approve data: %v", err)
	}

	// Create approve transaction
	approveTx := types.NewTransaction(nonce, tokenAddress, big.NewInt(0), 100000, gasPrice, approveData)

	// Sign approve transaction
	signedApproveTx, err := types.SignTx(approveTx, types.NewEIP155Signer(networkID), privateKey)
	if err != nil {
		log.Fatalf("❌ Failed to sign approve transaction: %v", err)
	}

	// Send approve transaction through proxy
	err = proxyClient.SendTransaction(context.Background(), signedApproveTx)
	if err != nil {
		log.Printf("⚠️  Failed to send approve transaction through proxy: %v", err)
		log.Println("📝 This might be normal if you don't have the custom token")
	} else {
		fmt.Printf("✅ Approve transaction sent: %s\n", signedApproveTx.Hash().Hex())
		time.Sleep(5 * time.Second)
	}

	// Step 4: Add liquidity
	fmt.Println("📝 Step 4: Adding liquidity...")

	// Parse Router ABI
	routerABIParsed, err := abi.JSON(strings.NewReader(routerABI))
	if err != nil {
		log.Fatalf("❌ Failed to parse router ABI: %v", err)
	}

	// Calculate deadline (10 minutes from now)
	deadline := big.NewInt(time.Now().Add(10 * time.Minute).Unix())

	// Create addLiquidityETH transaction data
	liquidityData, err := routerABIParsed.Pack("addLiquidityETH",
		tokenAddress,  // token
		tokenAmount,   // amountTokenDesired
		big.NewInt(0), // amountTokenMin (0 for testing)
		big.NewInt(0), // amountETHMin (0 for testing)
		fromAddress,   // to
		deadline,      // deadline
	)
	if err != nil {
		log.Fatalf("❌ Failed to pack liquidity data: %v", err)
	}

	// Create liquidity transaction
	liquidityTx := types.NewTransaction(nonce+1, common.HexToAddress(routerAddress), ethAmount, 300000, gasPrice, liquidityData)

	// Sign liquidity transaction
	signedLiquidityTx, err := types.SignTx(liquidityTx, types.NewEIP155Signer(networkID), privateKey)
	if err != nil {
		log.Fatalf("❌ Failed to sign liquidity transaction: %v", err)
	}

	fmt.Printf("🎯 Liquidity transaction hash: %s\n", signedLiquidityTx.Hash().Hex())
	fmt.Printf("📊 Transaction details:\n")
	fmt.Printf("   - From: %s\n", fromAddress.Hex())
	fmt.Printf("   - To: %s\n", routerAddress)
	fmt.Printf("   - Value: %s ETH\n", new(big.Float).Quo(new(big.Float).SetInt(ethAmount), big.NewFloat(1e18)).Text('f', 4))
	fmt.Printf("   - Gas Limit: %d\n", liquidityTx.Gas())
	fmt.Printf("   - Gas Price: %s Gwei\n", new(big.Float).Quo(new(big.Float).SetInt(gasPrice), big.NewFloat(1e9)).Text('f', 2))

	// Send liquidity transaction through proxy
	fmt.Println("📡 Sending liquidity transaction through RPC proxy...")
	err = proxyClient.SendTransaction(context.Background(), signedLiquidityTx)
	if err != nil {
		log.Printf("❌ Failed to send liquidity transaction through proxy: %v", err)
		log.Println("💡 This might be expected if you don't have sufficient ETH or tokens")
		log.Println("🔍 Check your RPC proxy logs to see if LP_ADD was detected")
	} else {
		fmt.Printf("✅ Liquidity transaction sent through proxy: %s\n", signedLiquidityTx.Hash().Hex())
		fmt.Println("🔍 Check your RPC proxy logs to see if LP_ADD was detected!")
	}

	// Also try sending directly to Base Sepolia for comparison
	fmt.Println("\n📡 Sending same transaction directly to Base Sepolia for comparison...")
	err = client.SendTransaction(context.Background(), signedLiquidityTx)
	if err != nil {
		log.Printf("❌ Failed to send transaction directly to Base Sepolia: %v", err)
	} else {
		fmt.Printf("✅ Transaction sent directly to Base Sepolia: %s\n", signedLiquidityTx.Hash().Hex())
	}

	fmt.Println("\n🎉 Script completed!")
	fmt.Println("📝 Note: Transactions may fail if you don't have sufficient ETH or tokens")
	fmt.Println("🔍 The important part is testing that your RPC proxy receives and processes the transactions")
	fmt.Println("💡 If pair creation succeeded, you should see the new pair address in the transaction logs")
}
