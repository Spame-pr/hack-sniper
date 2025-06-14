package main

import (
	"log"
	"os"
	"os/signal"
	"sniper-bot/pkg/config"
	"sniper-bot/services/bot/db"
	"sniper-bot/services/rpc/rpc"
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

	// Initialize RPC service
	rpcService, err := rpc.NewService(cfg, database)
	if err != nil {
		log.Fatalf("Failed to create RPC service: %v", err)
	}

	// Start RPC service
	go func() {
		if err := rpcService.Start(); err != nil {
			log.Fatalf("RPC service error: %v", err)
		}
	}()

	log.Println("RPC service started successfully")

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down RPC service...")
	if err := rpcService.Stop(); err != nil {
		log.Printf("Error stopping RPC service: %v", err)
	}
}
