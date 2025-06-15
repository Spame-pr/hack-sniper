package config

import (
	"os"
)

// Config holds all configuration for the application
type Config struct {
	// Telegram Bot
	TelegramBotToken string

	// Base Network
	BaseRPCURL          string
	BaseSequencerRPCURL string
	BaseWSURL           string

	// Database (MySQL)
	DatabaseURL string

	// Service
	BotPort int
	RPCPort int

	// DEX
	UniswapV2Router  string
	UniswapV2Factory string

	// Sniper contract
	SniperContract string

	// Auth
	AuthKey string
}

// Load loads configuration from environment variables
func Load() *Config {
	config := &Config{
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		BaseRPCURL:          os.Getenv("BASE_RPC_URL"),
		BaseSequencerRPCURL: os.Getenv("BASE_SEQUENCER_URL"),
		BaseWSURL:           os.Getenv("BASE_WS_URL"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		UniswapV2Router:     os.Getenv("UNISWAP_V2_ROUTER"),
		UniswapV2Factory:    os.Getenv("UNISWAP_V2_FACTORY"),
		AuthKey:             os.Getenv("AUTH_KEY"),
		SniperContract:      "0xa71940cb90C8F3634DD3AB6a992D0EFF056Db48d",
	}

	if config.DatabaseURL == "" {
		config.DatabaseURL = "root:admin@tcp(localhost:3306)/sniper?charset=utf8mb4&parseTime=True&loc=Local"
	}

	return config
}
