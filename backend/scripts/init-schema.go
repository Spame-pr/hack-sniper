package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Get database URL from command line argument or environment
	var databaseURL string
	if len(os.Args) > 1 {
		databaseURL = os.Args[1]
	} else {
		databaseURL = os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			fmt.Println("Usage: go run scripts/init-schema.go [DATABASE_URL]")
			fmt.Println("Or set DATABASE_URL environment variable")
			fmt.Println("")
			fmt.Println("Example:")
			fmt.Println("  go run scripts/init-schema.go 'root:password@tcp(localhost:3306)/sniper_bot?charset=utf8mb4&parseTime=True&loc=Local'")
			os.Exit(1)
		}
	}

	fmt.Println("🔗 Connecting to MySQL...")
	fmt.Printf("Database URL: %s\n", databaseURL)

	// Connect to database
	db, err := sql.Open("mysql", databaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to connect to MySQL: %v", err)
	}

	fmt.Println("✅ Successfully connected to MySQL!")

	// Initialize schema
	fmt.Println("📋 Initializing database schema...")

	// Create wallets table
	walletsSchema := `
		CREATE TABLE IF NOT EXISTS wallets (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			telegram_user_id VARCHAR(255) NOT NULL UNIQUE,
			wallet_address VARCHAR(255) NOT NULL,
			private_key TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_wallets_telegram_user_id (telegram_user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

	if _, err := db.Exec(walletsSchema); err != nil {
		log.Fatalf("❌ Failed to create wallets table: %v", err)
	}
	fmt.Println("✅ Created wallets table")

	// Create snipes table
	snipeBidsSchema := `
		CREATE TABLE IF NOT EXISTS snipes (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			token_address VARCHAR(255) NOT NULL,
		    amount VARCHAR(255) NOT NULL,
			bribe_amount VARCHAR(255) NOT NULL,
			wallet VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(50) NOT NULL,
			INDEX idx_snipes_token_address (token_address),
			INDEX idx_snipes_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

	if _, err := db.Exec(snipeBidsSchema); err != nil {
		log.Fatalf("❌ Failed to create snipes table: %v", err)
	}
	fmt.Println("✅ Created snipes table")

	fmt.Println("✅ Database schema initialized successfully!")

	// Verify tables were created
	fmt.Println("🔍 Verifying tables...")

	tables := []string{"wallets", "snipes"}
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			log.Fatalf("❌ Failed to verify table %s: %v", table, err)
		}
		fmt.Printf("✅ Table '%s' exists and is accessible (current rows: %d)\n", table, count)
	}

	fmt.Println("")
	fmt.Println("🎉 Migration completed successfully!")
	fmt.Println("Your database is ready to use with the sniper bot.")
}
