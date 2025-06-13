package wallet

import (
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"fmt"

	"sniper-bot/internal/db"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Manager handles the creation and management of sniper wallets
type Manager struct {
	db *db.DB
}

// Wallet represents a sniper's wallet
type Wallet struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
	UserID     string
}

// NewManager creates a new wallet manager
func NewManager(database *db.DB) *Manager {
	return &Manager{
		db: database,
	}
}

// CreateWallet creates a new wallet for a sniper
func (m *Manager) CreateWallet(userID string) (*Wallet, error) {
	// Check if user already has a wallet
	existingWallet, err := m.db.GetWalletByTelegramUserID(userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing wallet: %v", err)
	}
	if existingWallet != nil {
		return nil, errors.New("user already has a wallet")
	}

	// Generate new private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	// Get the address
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Store in database
	dbWallet := &db.Wallet{
		TelegramUserID: userID,
		WalletAddress:  address.Hex(),
		PrivateKey:     fmt.Sprintf("%x", crypto.FromECDSA(privateKey)),
	}

	if err := m.db.CreateWallet(dbWallet); err != nil {
		return nil, fmt.Errorf("failed to store wallet in database: %v", err)
	}

	wallet := &Wallet{
		Address:    address,
		PrivateKey: privateKey,
		UserID:     userID,
	}

	return wallet, nil
}

// GetWallet retrieves a wallet for a user
func (m *Manager) GetWallet(userID string) (*Wallet, error) {
	dbWallet, err := m.db.GetWalletByTelegramUserID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("wallet not found")
		}
		return nil, fmt.Errorf("failed to get wallet from database: %v", err)
	}

	// Parse private key from hex string
	privateKey, err := crypto.HexToECDSA(dbWallet.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	wallet := &Wallet{
		Address:    common.HexToAddress(dbWallet.WalletAddress),
		PrivateKey: privateKey,
		UserID:     userID,
	}

	return wallet, nil
}

// GetAddress returns the Ethereum address for a user
func (m *Manager) GetAddress(userID string) (common.Address, error) {
	wallet, err := m.GetWallet(userID)
	if err != nil {
		return common.Address{}, err
	}
	return wallet.Address, nil
}
