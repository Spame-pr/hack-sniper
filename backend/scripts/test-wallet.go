package main

import (
	"fmt"
	"log"

	"sniper-bot/internal/config"
	"sniper-bot/internal/db"
	"sniper-bot/internal/wallet"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize database schema
	if err := database.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	fmt.Println("✅ Database connection successful")
	fmt.Println("✅ Database schema initialized")

	// Initialize wallet manager
	walletManager := wallet.NewManager(database)

	// Test wallet creation
	testUserID := "123456789"

	fmt.Printf("Creating wallet for user %s...\n", testUserID)
	wallet1, err := walletManager.CreateWallet(testUserID)
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	fmt.Printf("✅ Wallet created successfully!\n")
	fmt.Printf("   Address: %s\n", wallet1.Address.Hex())
	fmt.Printf("   User ID: %s\n", wallet1.UserID)

	// Test retrieving the wallet
	fmt.Printf("Retrieving wallet for user %s...\n", testUserID)
	wallet2, err := walletManager.GetWallet(testUserID)
	if err != nil {
		log.Fatalf("Failed to get wallet: %v", err)
	}

	fmt.Printf("✅ Wallet retrieved successfully!\n")
	fmt.Printf("   Address: %s\n", wallet2.Address.Hex())
	fmt.Printf("   User ID: %s\n", wallet2.UserID)

	// Verify addresses match
	if wallet1.Address.Hex() != wallet2.Address.Hex() {
		log.Fatalf("❌ Address mismatch: %s != %s", wallet1.Address.Hex(), wallet2.Address.Hex())
	}

	fmt.Println("✅ Address verification passed")

	// Test duplicate wallet creation (should fail)
	fmt.Printf("Attempting to create duplicate wallet for user %s...\n", testUserID)
	_, err = walletManager.CreateWallet(testUserID)
	if err == nil {
		log.Fatalf("❌ Expected error for duplicate wallet creation, but got none")
	}

	fmt.Printf("✅ Duplicate wallet creation properly rejected: %v\n", err)

	// Test wallet for non-existent user
	fmt.Println("Testing wallet retrieval for non-existent user...")
	_, err = walletManager.GetWallet("999999999")
	if err == nil {
		log.Fatalf("❌ Expected error for non-existent user, but got none")
	}

	fmt.Printf("✅ Non-existent user properly handled: %v\n", err)

	fmt.Println("\n🎉 All wallet tests passed!")
}
