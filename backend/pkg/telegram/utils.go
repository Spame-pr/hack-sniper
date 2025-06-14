package telegram

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ValidateAddress validates an Ethereum address
func ValidateAddress(address string) (common.Address, error) {
	if !common.IsHexAddress(address) {
		return common.Address{}, fmt.Errorf("invalid Ethereum address format")
	}
	return common.HexToAddress(address), nil
}

// ValidateAmount validates and parses an ETH amount
func ValidateAmount(amount string) (float64, error) {
	amt, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format")
	}
	if amt <= 0 {
		return 0, fmt.Errorf("amount must be greater than 0")
	}
	return amt, nil
}

// FormatAddress formats an Ethereum address for display
func FormatAddress(address common.Address) string {
	addr := address.Hex()
	if len(addr) > 10 {
		return fmt.Sprintf("%s...%s", addr[:6], addr[len(addr)-4:])
	}
	return addr
}

// FormatAmount formats an amount for display
func FormatAmount(amount float64) string {
	return fmt.Sprintf("%.6f", amount)
}

// ParseCommand parses a command and its arguments
func ParseCommand(text string) (string, []string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", nil
	}

	command := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	return command, args
}

// EscapeMarkdown escapes special characters for Telegram markdown
func EscapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// CreateKeyboard creates an inline keyboard markup
func CreateKeyboard(buttons [][]string) string {
	// This is a simplified version - in a real implementation,
	// you'd use the Telegram Bot API's InlineKeyboardMarkup
	return ""
}
