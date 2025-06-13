package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(databaseURL string) (*DB, error) {
	db, err := sql.Open("mysql", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return &DB{db}, nil
}

// Wallet represents a user's wallet in the database
type Wallet struct {
	ID             int64
	TelegramUserID string
	WalletAddress  string
	PrivateKey     string
	CreatedAt      time.Time
}

// SnipeBid represents a sniper's bid in the database
type SnipeBid struct {
	ID           int64
	UserID       string
	TokenAddress string
	BribeAmount  string
	Wallet       string
	CreatedAt    time.Time
	Status       string
}

// CreateWallet creates a new wallet for a user
func (db *DB) CreateWallet(wallet *Wallet) error {
	query := `
		INSERT INTO wallets (telegram_user_id, wallet_address, private_key, created_at)
		VALUES (?, ?, ?, ?)
	`

	result, err := db.Exec(
		query,
		wallet.TelegramUserID,
		wallet.WalletAddress,
		wallet.PrivateKey,
		time.Now(),
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	wallet.ID = id
	return nil
}

// GetWalletByTelegramUserID gets a wallet by telegram user ID
func (db *DB) GetWalletByTelegramUserID(telegramUserID string) (*Wallet, error) {
	query := `
		SELECT id, telegram_user_id, wallet_address, private_key, created_at
		FROM wallets
		WHERE telegram_user_id = ?
	`

	wallet := &Wallet{}
	err := db.QueryRow(query, telegramUserID).Scan(
		&wallet.ID,
		&wallet.TelegramUserID,
		&wallet.WalletAddress,
		&wallet.PrivateKey,
		&wallet.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

// CreateSnipeBid creates a new snipe bid
func (db *DB) CreateSnipeBid(bid *SnipeBid) error {
	query := `
		INSERT INTO snipe_bids (user_id, token_address, bribe_amount, wallet, created_at, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.Exec(
		query,
		bid.UserID,
		bid.TokenAddress,
		bid.BribeAmount,
		bid.Wallet,
		time.Now(),
		"pending",
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	bid.ID = id
	return nil
}

// GetSnipeBidsByToken gets all snipe bids for a token
func (db *DB) GetSnipeBidsByToken(tokenAddress string) ([]*SnipeBid, error) {
	query := `
		SELECT id, user_id, token_address, bribe_amount, wallet, created_at, status
		FROM snipe_bids
		WHERE token_address = ? AND status = 'pending'
		ORDER BY CAST(bribe_amount AS DECIMAL(20,8)) DESC
	`

	rows, err := db.Query(query, tokenAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []*SnipeBid
	for rows.Next() {
		bid := &SnipeBid{}
		if err := rows.Scan(
			&bid.ID,
			&bid.UserID,
			&bid.TokenAddress,
			&bid.BribeAmount,
			&bid.Wallet,
			&bid.CreatedAt,
			&bid.Status,
		); err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}

	return bids, nil
}

// UpdateSnipeBidStatus updates the status of a snipe bid
func (db *DB) UpdateSnipeBidStatus(id int64, status string) error {
	query := `
		UPDATE snipe_bids
		SET status = ?
		WHERE id = ?
	`

	_, err := db.Exec(query, status, id)
	return err
}

// InitSchema initializes the database schema
func (db *DB) InitSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS wallets (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			telegram_user_id VARCHAR(255) NOT NULL UNIQUE,
			wallet_address VARCHAR(255) NOT NULL,
			private_key TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_wallets_telegram_user_id (telegram_user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

		CREATE TABLE IF NOT EXISTS snipe_bids (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			token_address VARCHAR(255) NOT NULL,
			bribe_amount VARCHAR(255) NOT NULL,
			wallet VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(50) NOT NULL,
			INDEX idx_snipe_bids_token_address (token_address),
			INDEX idx_snipe_bids_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	_, err := db.Exec(schema)
	return err
}
