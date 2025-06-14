package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	fmt.Println("🧪 Testing LP_ADD Detection...")

	// Configuration
	proxyRPC := "http://localhost:8545"
	routerAddress := os.Getenv("UNISWAP_V2_ROUTER")
	if routerAddress == "" {
		routerAddress = "0x4752ba5dbc23f44d87826276bf6fd6b1c372ad24"
	}

	// Create a mock addLiquidityETH transaction
	// This is the function signature for addLiquidityETH:
	// addLiquidityETH(address token, uint256 amountTokenDesired, uint256 amountTokenMin, uint256 amountETHMin, address to, uint256 deadline)
	// Function selector: 0xf305d719

	mockTxData := "0xf305d719" + // addLiquidityETH function selector
		"000000000000000000000000833589fcd6edb6e08f4c7c32d4f71b54bda02913" + // token (USDC)
		"00000000000000000000000000000000000000000000000000000000b2d05e00" + // amountTokenDesired (3000 USDC)
		"0000000000000000000000000000000000000000000000000000000000000000" + // amountTokenMin (0)
		"0000000000000000000000000000000000000000000000000000000000000000" + // amountETHMin (0)
		"0000000000000000000000008e3cf8fe85a40c70a56f128f8e444c7ea864480d" + // to (recipient)
		"0000000000000000000000000000000000000000000000000000000067654321" // deadline

	// Create a mock raw transaction
	mockRawTx := "0x02f8b5018203e8843b9aca00843b9aca0083049f8094" + routerAddress[2:] + "880de0b6b3a764000080b844" + mockTxData[2:] + "c080a01234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdefa01234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	// Create eth_sendRawTransaction request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_sendRawTransaction",
		"params":  []string{mockRawTx},
		"id":      1,
	}

	// Convert to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("❌ Failed to marshal request: %v", err)
	}

	fmt.Printf("📤 Sending mock LP_ADD transaction to proxy...\n")
	fmt.Printf("🎯 Router address: %s\n", routerAddress)
	fmt.Printf("📊 Transaction data: %s\n", mockTxData)

	// Send request to proxy
	resp, err := http.Post(proxyRPC, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		log.Fatalf("❌ Failed to send request to proxy: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Fatalf("❌ Failed to decode response: %v", err)
	}

	fmt.Printf("📥 Proxy response: %+v\n", response)

	if resp.StatusCode == 200 {
		fmt.Println("✅ Request sent successfully!")
		fmt.Println("🔍 Check your RPC proxy logs to see if LP_ADD was detected")
		fmt.Println("💡 Look for log message: 'LP_ADD transaction detected: ...'")
	} else {
		fmt.Printf("⚠️  Request failed with status: %d\n", resp.StatusCode)
	}

	fmt.Println("\n🎉 LP detection test completed!")
	fmt.Println("📝 Note: The transaction will likely fail (which is expected)")
	fmt.Println("🎯 The goal is to test if your RPC proxy detects the LP_ADD pattern")
}
