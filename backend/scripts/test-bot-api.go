package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// LPAddNotificationPayload represents the payload sent to bot service
type LPAddNotificationPayload struct {
	TokenAddress   string `json:"tokenAddress"`
	CreatorAddress string `json:"creatorAddress"`
	TxCallData     string `json:"txCallData"`
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	fmt.Println("🧪 Testing Bot API Communication...")

	// Get configuration
	botAPIURL := os.Getenv("BOT_API_URL")
	if botAPIURL == "" {
		botAPIURL = "http://localhost:8080"
	}

	botAPIKey := os.Getenv("BOT_API_KEY")
	if botAPIKey == "" {
		botAPIKey = "default-api-key-change-me"
	}

	fmt.Printf("📡 Bot API URL: %s\n", botAPIURL)
	fmt.Printf("🔑 API Key: %s\n", botAPIKey)

	// Test data
	testPayload := LPAddNotificationPayload{
		TokenAddress:   "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
		CreatorAddress: "0x8e3cf8fe85a40c70a56f128f8e444c7ea864480d",
		TxCallData:     "0xf305d719000000000000000000000000742d35cc6634c0532925a3b8d4c9db96c4b4d8b6",
	}

	// Test 1: Health check
	fmt.Println("\n🔍 Test 1: Health Check")
	if err := testHealthCheck(botAPIURL); err != nil {
		log.Printf("❌ Health check failed: %v", err)
	} else {
		fmt.Println("✅ Health check passed")
	}

	// Test 2: Unauthorized request
	fmt.Println("\n🔍 Test 2: Unauthorized Request")
	if err := testUnauthorized(botAPIURL, testPayload); err != nil {
		fmt.Printf("✅ Unauthorized test passed (expected error): %v\n", err)
	} else {
		fmt.Println("❌ Unauthorized test failed - should have been rejected")
	}

	// Test 3: Valid request
	fmt.Println("\n🔍 Test 3: Valid LP_ADD Notification")
	if err := testValidRequest(botAPIURL, botAPIKey, testPayload); err != nil {
		log.Printf("❌ Valid request failed: %v", err)
	} else {
		fmt.Println("✅ Valid request passed")
	}

	fmt.Println("\n🎉 Bot API tests completed!")
}

func testHealthCheck(baseURL string) error {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(body) != "OK" {
		return fmt.Errorf("expected 'OK', got '%s'", string(body))
	}

	return nil
}

func testUnauthorized(baseURL string, payload LPAddNotificationPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/lp-add", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	// Intentionally not setting Authorization header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	return fmt.Errorf("unauthorized") // This is expected
}

func testValidRequest(baseURL, apiKey string, payload LPAddNotificationPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/lp-add", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	fmt.Printf("📨 Response: %+v\n", response)

	if response["status"] != "success" {
		return fmt.Errorf("expected success status, got %v", response["status"])
	}

	return nil
}
