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
	CreatedAt      string
}

// Snipe represents a sniper's bid in the database
type Snipe struct {
	ID           int64
	UserID       string
	TokenAddress string
	Amount       string
	BribeAmount  string
	Wallet       string
	CreatedAt    string
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

// CreateSnipe creates a new snipe
func (db *DB) CreateSnipe(snipe *Snipe) error {
	query := `
		INSERT INTO snipes (user_id, token_address, amount, bribe_amount, wallet, created_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.Exec(
		query,
		snipe.UserID,
		snipe.TokenAddress,
		snipe.Amount,
		snipe.BribeAmount,
		snipe.Wallet,
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

	snipe.ID = id
	return nil
}

// GetSnipesByToken gets all snipes for a token
func (db *DB) GetSnipesByToken(tokenAddress string) ([]*Snipe, error) {
	query := `
		SELECT id, user_id, token_address, amount, bribe_amount, wallet, created_at, status
		FROM snipes
		WHERE token_address = ? AND status = 'pending'
		ORDER BY CAST(bribe_amount AS DECIMAL(20,8)) DESC
	`

	rows, err := db.Query(query, tokenAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snipes []*Snipe
	for rows.Next() {
		snipe := &Snipe{}
		if err := rows.Scan(
			&snipe.ID,
			&snipe.UserID,
			&snipe.TokenAddress,
			&snipe.Amount,
			&snipe.BribeAmount,
			&snipe.Wallet,
			&snipe.CreatedAt,
			&snipe.Status,
		); err != nil {
			return nil, err
		}
		snipes = append(snipes, snipe)
	}

	return snipes, nil
}

// UpdateSnipeStatus updates the status of a snipe
func (db *DB) UpdateSnipeStatus(id int64, status string) error {
	query := `
		UPDATE snipes
		SET status = ?
		WHERE id = ?
	`

	_, err := db.Exec(query, status, id)
	return err
}
