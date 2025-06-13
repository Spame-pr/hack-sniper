# Database Migrations

This directory contains database migration files for the sniper bot project.

## Migration Files

Migration files follow the naming convention: `{version}_{description}.sql`

- `001_initial_schema.sql` - Creates the initial database schema with wallets and snipe_bids tables

## Running Migrations

### Using Make Commands (Recommended)

```bash
# Run all pending migrations
make migrate

# Check migration status
make migrate-status
```

### Using the Migration Tool Directly

```bash
# Build the migration tool
make build-migrate

# Run migrations
./bin/migrate up

# Check status
./bin/migrate status
```

### Using Go Run

```bash
# Run migrations
go run cmd/migrate/main.go up

# Check status
go run cmd/migrate/main.go status
```

## Environment Variables

Make sure your `.env` file contains the `DATABASE_URL` variable:

```env
DATABASE_URL=user:password@tcp(localhost:3306)/sniper_bot?parseTime=true
```

## Creating New Migrations

1. Create a new SQL file in the `migrations/` directory
2. Use the next sequential number (e.g., `002_add_new_table.sql`)
3. Write your SQL DDL statements
4. Run `make migrate` to apply the new migration

## Migration Tracking

The system automatically creates a `migrations` table to track which migrations have been applied. This ensures migrations are only run once and in the correct order.

## Example Migration File

```sql
-- Migration: 002_add_user_settings
-- Description: Add user settings table

CREATE TABLE IF NOT EXISTS user_settings (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    telegram_user_id VARCHAR(255) NOT NULL UNIQUE,
    max_bribe_amount VARCHAR(255) NOT NULL DEFAULT '0.1',
    auto_snipe_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_settings_telegram_user_id (telegram_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
``` 