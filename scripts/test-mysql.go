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

	// Test creating a snipe
	fmt.Println("🧪 Testing snipe creation...")
	snipe := &db.Snipe{
		UserID:       "test_user_123",
		TokenAddress: "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6",
		Amount:       "1.0",
		BribeAmount:  "0.1",
		Wallet:       "0x1234567890123456789012345678901234567890",
	}

	if err := database.CreateSnipe(snipe); err != nil {
		log.Fatalf("❌ Failed to create snipe: %v", err)
	}

	fmt.Printf("✅ Created snipe with ID: %d\n", snipe.ID)

	// Test retrieving snipes
	fmt.Println("🔍 Testing snipe retrieval...")
	snipes, err := database.GetSnipesByToken(snipe.TokenAddress)
	if err != nil {
		log.Fatalf("❌ Failed to retrieve snipes: %v", err)
	}

	fmt.Printf("✅ Retrieved %d snipe(s) for token %s\n", len(snipes), snipe.TokenAddress)

	// Test updating snipe status
	fmt.Println("📝 Testing snipe status update...")
	if err := database.UpdateSnipeStatus(snipe.ID, "completed"); err != nil {
		log.Fatalf("❌ Failed to update snipe status: %v", err)
	}

	fmt.Println("✅ Updated snipe status successfully!")

	fmt.Println("🎉 All MySQL tests passed!")
}
