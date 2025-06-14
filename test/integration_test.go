package test

import (
	"testing"

	"sniper-bot/internal/wallet"
	"sniper-bot/pkg/telegram"

	"github.com/ethereum/go-ethereum/common"
)

func TestWalletManager(t *testing.T) {
	// Skip this test since it requires a database connection
	t.Skip("Skipping wallet manager test - requires database connection")

	manager := wallet.NewManager(nil)

	// Test wallet creation
	wallet, err := manager.CreateWallet("test_user_1")
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	if wallet.UserID != "test_user_1" {
		t.Errorf("Expected user ID 'test_user_1', got '%s'", wallet.UserID)
	}

	if wallet.Address == (common.Address{}) {
		t.Error("Wallet address should not be empty")
	}

	// Test wallet retrieval
	retrievedWallet, err := manager.GetWallet("test_user_1")
	if err != nil {
		t.Fatalf("Failed to retrieve wallet: %v", err)
	}

	if retrievedWallet.Address != wallet.Address {
		t.Error("Retrieved wallet address doesn't match original")
	}

	// Test duplicate wallet creation
	_, err = manager.CreateWallet("test_user_1")
	if err == nil {
		t.Error("Expected error when creating duplicate wallet")
	}
}

func TestTelegramUtils(t *testing.T) {
	// Test address validation
	validAddr := "0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6"
	addr, err := telegram.ValidateAddress(validAddr)
	if err != nil {
		t.Fatalf("Failed to validate valid address: %v", err)
	}

	// Addresses are normalized to checksum format
	expectedAddr := "0x742d35Cc6634C0532925A3B8D4C9dB96C4B4d8B6"
	if addr.Hex() != expectedAddr {
		t.Errorf("Expected address %s, got %s", expectedAddr, addr.Hex())
	}

	// Test invalid address
	_, err = telegram.ValidateAddress("invalid_address")
	if err == nil {
		t.Error("Expected error for invalid address")
	}

	// Test amount validation
	amount, err := telegram.ValidateAmount("1.5")
	if err != nil {
		t.Fatalf("Failed to validate valid amount: %v", err)
	}

	if amount != 1.5 {
		t.Errorf("Expected amount 1.5, got %f", amount)
	}

	// Test invalid amount
	_, err = telegram.ValidateAmount("invalid")
	if err == nil {
		t.Error("Expected error for invalid amount")
	}

	// Test zero amount
	_, err = telegram.ValidateAmount("0")
	if err == nil {
		t.Error("Expected error for zero amount")
	}

	// Test command parsing
	command, args := telegram.ParseCommand("/snipe 0x123 1.5")
	if command != "snipe" {
		t.Errorf("Expected command 'snipe', got '%s'", command)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}

	if args[0] != "0x123" || args[1] != "1.5" {
		t.Errorf("Unexpected args: %v", args)
	}
}

func TestAddressFormatting(t *testing.T) {
	addr := common.HexToAddress("0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6")
	formatted := telegram.FormatAddress(addr)

	// Address formatting uses checksum format
	expected := "0x742d...d8B6"
	if formatted != expected {
		t.Errorf("Expected formatted address '%s', got '%s'", expected, formatted)
	}
}

func TestAmountFormatting(t *testing.T) {
	amount := 1.234567
	formatted := telegram.FormatAmount(amount)

	expected := "1.234567"
	if formatted != expected {
		t.Errorf("Expected formatted amount '%s', got '%s'", expected, formatted)
	}
}
