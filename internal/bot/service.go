package bot

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"sniper-bot/internal/db"
	"sniper-bot/internal/wallet"
	"sniper-bot/pkg/eth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Service represents the Telegram bot service
type Service struct {
	bot           *tgbotapi.BotAPI
	ethClient     *eth.Client
	walletManager *wallet.Manager
	db            *db.DB
}

// NewService creates a new bot service
func NewService(walletManager *wallet.Manager, database *db.DB) (*Service, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %v", err)
	}

	// Use BASE_SEQUENCER_URL for RPC connection
	sequencerURL := os.Getenv("BASE_RPC_URL")
	if sequencerURL == "" {
		sequencerURL = "https://mainnet.base.org" // fallback
	}

	ethClient, err := eth.NewClient(sequencerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create eth client: %v", err)
	}

	return &Service{
		bot:           bot,
		walletManager: walletManager,
		ethClient:     ethClient,
		db:            database,
	}, nil
}

// Start starts the bot service
func (s *Service) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ParseMode = "HTML"

		switch update.Message.Command() {
		case "start":
			msg.Text = "Welcome to the Sniper Bot! Use /register to create your wallet."
		case "register":
			msg.Text = s.handleRegister(update.Message.From.ID)
		case "balance":
			msg.Text = s.handleBalance(update.Message.From.ID)
		case "snipe":
			msg.Text = s.handleSnipe(update.Message.From.ID, update.Message.CommandArguments())
		default:
			msg.Text = "Unknown command"
		}

		if _, err := s.bot.Send(msg); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}

	return nil
}

// Stop stops the bot service
func (s *Service) Stop() error {
	s.bot.StopReceivingUpdates()
	return nil
}

func (s *Service) handleRegister(userID int64) string {
	userIDStr := fmt.Sprintf("%d", userID)

	// Check if user already has a wallet
	existingWallet, err := s.walletManager.GetWallet(userIDStr)
	if err == nil {
		return fmt.Sprintf("You already have a wallet!\nAddress: %s", existingWallet.Address.Hex())
	}

	// Create new wallet
	wallet, err := s.walletManager.CreateWallet(userIDStr)
	if err != nil {
		if err.Error() == "user already has a wallet" {
			return "You already have a wallet! Use /balance to check your wallet details."
		}
		return fmt.Sprintf("Error creating wallet: %v", err)
	}

	return fmt.Sprintf("Wallet created successfully!\nAddress: %s\n\n⚠️ <b>Important:</b> This is your only wallet. Keep your address safe!", wallet.Address.Hex())
}

func (s *Service) handleBalance(userID int64) string {
	userIDStr := fmt.Sprintf("%d", userID)
	wallet, err := s.walletManager.GetWallet(userIDStr)
	if err != nil {
		return "Wallet not found. Please register first using /register"
	}

	balance, err := s.ethClient.GetBalance(context.Background(), wallet.Address)
	if err != nil {
		return fmt.Sprintf("Error getting balance: %v", err)
	}

	// Convert wei to ETH
	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))

	return fmt.Sprintf("Wallet address: %s\nBalance: %s ETH", wallet.Address.Hex(), ethBalance.Text('f', 6))
}

func (s *Service) handleSnipe(userID int64, args string) string {
	parts := strings.Fields(args)
	if len(parts) != 3 {
		return "Usage: /snipe <token_address> <amount_in_ETH> <bribe_in_ETH>"
	}

	tokenAddress := parts[0]
	amount := parts[1]
	bribeAmount := parts[2]

	userIDStr := fmt.Sprintf("%d", userID)

	// Check if user has a wallet
	userWallet, err := s.walletManager.GetWallet(userIDStr)
	if err != nil {
		return "❌ Wallet not found. Please register first using /register"
	}

	// Validate token address format
	if len(tokenAddress) != 42 || tokenAddress[:2] != "0x" {
		return "❌ Invalid token address format. Must be a valid Ethereum address (0x...)"
	}

	// Validate amount and bribe amount are positive numbers
	if !isValidAmount(amount) {
		return "❌ Invalid amount. Must be a positive number (e.g., 0.1, 1.5)"
	}

	if !isValidAmount(bribeAmount) {
		return "❌ Invalid bribe amount. Must be a positive number (e.g., 0.01, 0.1)"
	}

	// Create snipe record in database
	snipe := &db.Snipe{
		UserID:       userIDStr,
		TokenAddress: tokenAddress,
		Amount:       amount,
		BribeAmount:  bribeAmount,
		Wallet:       userWallet.Address.Hex(),
		Status:       "pending",
	}

	if err := s.db.CreateSnipe(snipe); err != nil {
		log.Printf("Failed to create snipe record: %v", err)
		return "❌ Failed to submit snipe request. Please try again."
	}

	return fmt.Sprintf("✅ Snipe request submitted successfully!\n\n"+
		"📋 <b>Details:</b>\n"+
		"🎯 Token: <code>%s</code>\n"+
		"💰 Amount: %s ETH\n"+
		"💸 Bribe: %s ETH\n"+
		"👛 Wallet: <code>%s</code>\n"+
		"🆔 Request ID: %d\n\n"+
		"⏳ Your request is now pending. You'll be included in the next bundle when liquidity is added for this token.",
		tokenAddress, amount, bribeAmount, userWallet.Address.Hex(), snipe.ID)
}

// isValidAmount checks if a string represents a valid positive number
func isValidAmount(amount string) bool {
	if amount == "" {
		return false
	}

	// Try to parse as float
	var f float64
	n, err := fmt.Sscanf(amount, "%f", &f)
	if err != nil || n != 1 {
		return false
	}

	// Must be positive
	return f > 0
}
