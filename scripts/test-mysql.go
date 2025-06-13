package main

import (
	"fmt"
	"log"
	"os"

	"sniper-bot/internal/db"
)

func main() {
	// Get database URL from environment or use default
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "sniper_user:sniper_password@tcp(localhost:3306)/sniper_bot?charset=utf8mb4&parseTime=True&loc=Local"
	}

	fmt.Println("🔗 Testing MySQL connection...")
	fmt.Printf("Database URL: %s\n", databaseURL)

	// Connect to database
	database, err := db.New(databaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to MySQL: %v", err)
	}
	defer database.Close()

	fmt.Println("✅ Successfully connected to MySQL!")

	// Initialize schema
	fmt.Println("📋 Initializing database schema...")
	if err := database.InitSchema(); err != nil {
		log.Fatalf("❌ Failed to initialize schema: %v", err)
	}

	fmt.Println("✅ Database schema initialized successfully!")

	// Test creating a snipe bid
	fmt.Println("🧪 Testing snipe bid creation...")
	bid := &db.SnipeBid{
		UserID:       "test_user_123",
		TokenAddress: "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
		BribeAmount:  "0.1",
		Wallet:       "0x1234567890123456789012345678901234567890",
	}

	if err := database.CreateSnipeBid(bid); err != nil {
		log.Fatalf("❌ Failed to create snipe bid: %v", err)
	}

	fmt.Printf("✅ Created snipe bid with ID: %d\n", bid.ID)

	// Test retrieving snipe bids
	fmt.Println("🔍 Testing snipe bid retrieval...")
	bids, err := database.GetSnipeBidsByToken(bid.TokenAddress)
	if err != nil {
		log.Fatalf("❌ Failed to retrieve snipe bids: %v", err)
	}

	fmt.Printf("✅ Retrieved %d snipe bid(s) for token %s\n", len(bids), bid.TokenAddress)

	// Test updating snipe bid status
	fmt.Println("📝 Testing snipe bid status update...")
	if err := database.UpdateSnipeBidStatus(bid.ID, "completed"); err != nil {
		log.Fatalf("❌ Failed to update snipe bid status: %v", err)
	}

	fmt.Println("✅ Updated snipe bid status successfully!")

	fmt.Println("🎉 All MySQL tests passed!")
}
