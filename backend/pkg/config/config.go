package config

import (
	"os"
)

// getEnvWithFallback tries the primary env var first, then falls back to secondary
func getEnvWithFallback(primary, secondary string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(secondary)
}

// Config holds all configuration for the application
type Config struct {
	// Telegram Bot
	TelegramBotToken string

	// Base Network
	BaseRPCURL          string
	BaseSequencerRPCURL string
	BaseWSURL           string
	BaseKolibrioRpcURL  string

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
		BaseKolibrioRpcURL:  os.Getenv("KOLIBRIO_BASE_RPC"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		UniswapV2Router:     os.Getenv("UNISWAP_V2_ROUTER"),
		UniswapV2Factory:    os.Getenv("UNISWAP_V2_FACTORY"),
		AuthKey:             os.Getenv("AUTH_KEY"),
		SniperContract:      "0xa71940cb90C8F3634DD3AB6a992D0EFF056Db48d",
	}

	if config.DatabaseURL == "" {
		config.DatabaseURL = "root:admin@tcp(localhost:3306)/sniper?charset=utf8mb4&parseTime=True&loc=Local"
	}
	if config.UniswapV2Router == "" {
		config.UniswapV2Router = "0x1689E7B1F10000AE47eBfE339a4f69dECd19F602" // Base Sepolia Router
	}
	if config.UniswapV2Factory == "" {
		config.UniswapV2Factory = "0x7Ae58f10f7849cA6F5fB71b7f45CB416c9204b1e" // Base Sepolia Factory
	}

	return config
}
