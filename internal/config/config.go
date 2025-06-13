package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the application
type Config struct {
	// Telegram Bot
	TelegramBotToken string

	// Base Network
	BaseRPCURL string
	BaseWSURL  string

	// Database (MySQL)
	DatabaseURL string

	// Service
	BotPort int
	RPCPort int

	// DEX
	UniswapV2Router  string
	UniswapV2Factory string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		BaseRPCURL:       os.Getenv("BASE_RPC_URL"),
		BaseWSURL:        os.Getenv("BASE_WS_URL"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		UniswapV2Router:  os.Getenv("UNISWAP_V2_ROUTER"),
		UniswapV2Factory: os.Getenv("UNISWAP_V2_FACTORY"),
	}

	// Set default values
	if config.BaseRPCURL == "" {
		config.BaseRPCURL = "https://mainnet.base.org"
	}
	if config.BaseWSURL == "" {
		config.BaseWSURL = "wss://mainnet.base.org"
	}
	if config.DatabaseURL == "" {
		config.DatabaseURL = "root:admin@tcp(localhost:3306)/sniper_bot?charset=utf8mb4&parseTime=True&loc=Local"
	}
	if config.UniswapV2Router == "" {
		config.UniswapV2Router = "0x4752ba5dbc23f44d87826276bf6fd6b1c372ad24"
	}
	if config.UniswapV2Factory == "" {
		config.UniswapV2Factory = "0x8909dc15e40173ff4699343b6eb8132c65e18ec6"
	}

	// Validate required fields
	if config.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	return config, nil
}
