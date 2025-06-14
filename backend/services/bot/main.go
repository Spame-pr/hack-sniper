package main

import (
	"log"
	"os"
	"os/signal"
	"sniper-bot/pkg/config"
	"sniper-bot/pkg/eth"
	"sniper-bot/services/bot/api"
	"sniper-bot/services/bot/bot"
	"sniper-bot/services/bot/db"
	"sniper-bot/services/bot/wallet"
	"sync"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	cfg := config.Load()

	// Initialize database
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize wallet manager with database
	walletManager := wallet.NewManager(database)

	// Initialize ethereum client for balance checks
	ethClient, err := eth.NewClient(cfg.BaseRPCURL)
	if err != nil {
		log.Fatalf("Failed to create eth client: %v", err)
	}

	// Initialize bot service
	botService, err := bot.NewService(walletManager, database, ethClient)
	if err != nil {
		log.Fatalf("Failed to create bot service: %v", err)
	}

	// Initialize API service
	apiService, err := api.NewService(walletManager, database)
	if err != nil {
		log.Fatalf("Failed to create API service: %v", err)
	}

	// Use WaitGroup to manage both services
	var wg sync.WaitGroup

	// Start bot service
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := botService.Start(); err != nil {
			log.Printf("Bot service error: %v", err)
		}
	}()

	// Start API service
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiService.Start(); err != nil {
			log.Printf("API service error: %v", err)
		}
	}()

	log.Println("🚀 Bot and API services started successfully")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("📥 Shutdown signal received, stopping services...")

	// Stop services gracefully
	if err := botService.Stop(); err != nil {
		log.Printf("Error stopping bot service: %v", err)
	}

	if err := apiService.Stop(); err != nil {
		log.Printf("Error stopping API service: %v", err)
	}

	log.Println("✅ Services stopped successfully")
}
