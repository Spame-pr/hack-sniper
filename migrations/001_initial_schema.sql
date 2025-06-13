-- Migration: 001_initial_schema
-- Description: Create initial tables for wallets and snipe bids

-- Create wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    telegram_user_id VARCHAR(255) NOT NULL UNIQUE,
    wallet_address VARCHAR(255) NOT NULL,
    private_key TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_wallets_telegram_user_id (telegram_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create snipe_bids table
CREATE TABLE IF NOT EXISTS snipe_bids (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token_address VARCHAR(255) NOT NULL,
    bribe_amount VARCHAR(255) NOT NULL,
    wallet VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    INDEX idx_snipe_bids_token_address (token_address),
    INDEX idx_snipe_bids_status (status),
    INDEX idx_snipe_bids_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Create migrations table to track applied migrations
CREATE TABLE IF NOT EXISTS migrations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    version VARCHAR(255) NOT NULL UNIQUE,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci; 