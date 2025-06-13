package bot

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"sniper-bot/internal/wallet"
	"sniper-bot/pkg/eth"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Service represents the Telegram bot service
type Service struct {
	bot           *tgbotapi.BotAPI
	ethClient     *eth.Client
	walletManager *wallet.Manager
}

// NewService creates a new bot service
func NewService(walletManager *wallet.Manager) (*Service, error) {
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
	if len(parts) != 2 {
		return "Usage: /snipe <token_address> <bribe_in_ETH>"
	}

	tokenAddress := parts[0]
	bribeAmount := parts[1]

	// TODO: Implement snipe logic
	return fmt.Sprintf("Snipe request received:\nToken: %s\nBribe: %s ETH", tokenAddress, bribeAmount)
}
