package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// Test the RPC proxy
	fmt.Println("🧪 Testing RPC Proxy...")

	// Wait a moment for the service to start
	time.Sleep(2 * time.Second)

	// Test eth_estimateGas request
	testRequest := map[string]interface{}{
		"id":      1,
		"jsonrpc": "2.0",
		"method":  "eth_estimateGas",
		"params": []interface{}{
			map[string]interface{}{
				"from":  "0x8E3cF8Fe85A40C70a56f128f8e444c7ea864480D",
				"to":    "0xcB6EDF6038cce43401761f3a5Bf5975356B772Bd",
				"value": "0xde0b6b3a7640000",
			},
		},
	}

	// Convert to JSON
	requestBody, err := json.Marshal(testRequest)
	if err != nil {
		log.Fatalf("❌ Failed to marshal request: %v", err)
	}

	fmt.Printf("📤 Sending request to proxy: %s\n", string(requestBody))

	// Send request to proxy
	proxyURL := "http://localhost:8545"
	resp, err := http.Post(proxyURL, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		log.Fatalf("❌ Failed to send request to proxy: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("❌ Failed to read response: %v", err)
	}

	fmt.Printf("📥 Proxy response (status %d): %s\n", resp.StatusCode, string(responseBody))

	// Test direct call to Base for comparison
	fmt.Println("\n🔗 Testing direct call to Base...")
	baseURL := os.Getenv("BASE_RPC_URL")
	if baseURL == "" {
		baseURL = "https://mainnet.base.org"
	}

	directResp, err := http.Post(baseURL, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		log.Fatalf("❌ Failed to send request to Base: %v", err)
	}
	defer directResp.Body.Close()

	directResponseBody, err := io.ReadAll(directResp.Body)
	if err != nil {
		log.Fatalf("❌ Failed to read Base response: %v", err)
	}

	fmt.Printf("📥 Base response (status %d): %s\n", directResp.StatusCode, string(directResponseBody))

	// Compare responses
	if resp.StatusCode == directResp.StatusCode {
		fmt.Println("✅ Status codes match!")
	} else {
		fmt.Printf("❌ Status codes differ: proxy=%d, base=%d\n", resp.StatusCode, directResp.StatusCode)
	}

	// Parse and compare JSON responses
	var proxyJSON, baseJSON map[string]interface{}
	if err := json.Unmarshal(responseBody, &proxyJSON); err == nil {
		if err := json.Unmarshal(directResponseBody, &baseJSON); err == nil {
			if fmt.Sprintf("%v", proxyJSON) == fmt.Sprintf("%v", baseJSON) {
				fmt.Println("✅ Response bodies match!")
			} else {
				fmt.Println("⚠️  Response bodies differ (this might be normal due to timing)")
			}
		}
	}

	fmt.Println("\n🎉 RPC Proxy test completed!")
}
