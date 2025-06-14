package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"sniper-bot/internal/bot"
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

	cfg := config.Load()

	// Initialize database
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize wallet manager with database
	walletManager := wallet.NewManager(database)

	// Initialize bot service with database
	botService, err := bot.NewService(walletManager, database)
	if err != nil {
		log.Fatalf("Failed to create bot service: %v", err)
	}

	// Start bot service
	go func() {
		if err := botService.Start(); err != nil {
			log.Fatalf("Bot service error: %v", err)
		}
	}()

	log.Println("Bot started successfully")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down bot...")
	if err := botService.Stop(); err != nil {
		log.Printf("Error stopping bot: %v", err)
	}
}
