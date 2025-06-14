package bundle

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

	"sniper-bot/internal/dex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Manager handles the creation and submission of transaction bundles
type Manager struct {
	client         *ethclient.Client
	sniperContract *dex.SniperContract
}

// NewManager creates a new bundle manager
func NewManager(client *ethclient.Client, sniperContractAddr common.Address) (*Manager, error) {
	sniperContract, err := dex.NewSniperContract(client, sniperContractAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create sniper contract: %v", err)
	}

	return &Manager{
		client:         client,
		sniperContract: sniperContract,
	}, nil
}

// SnipeBid represents a sniper's bid for a token
type SnipeBid struct {
	UserID       string
	TokenAddress common.Address
	SwapAmount   *big.Int
	BribeAmount  *big.Int
	Wallet       common.Address
	PrivateKey   string // Base64 encoded private key
}

// CreateBundleTransactions creates transaction bundle from an LP_ADD transaction and snipe bids
func (m *Manager) CreateBundleTransactions(
	ctx context.Context,
	lpAddTx *types.Transaction,
	bids []*SnipeBid,
) ([]*types.Transaction, error) {
	// Sort bids by bribe amount (descending)
	sort.Slice(bids, func(i, j int) bool {
		return bids[i].BribeAmount.Cmp(bids[j].BribeAmount) > 0
	})

	// Extract token creator from LP_ADD transaction
	creator, err := m.sniperContract.GetCreatorFromLPAddTx(lpAddTx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract creator from LP_ADD tx: %v", err)
	}

	// Extract token address from LP_ADD transaction
	token, err := m.extractTokenFromLPAdd(lpAddTx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract token from LP_ADD tx: %v", err)
	}

	// Get base gas price from LP_ADD transaction
	baseGasPrice := lpAddTx.GasPrice()
	if baseGasPrice == nil {
		baseGasPrice, err = m.client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get gas price: %v", err)
		}
	}

	var transactions []*types.Transaction

	// Add the original LP_ADD transaction first
	transactions = append(transactions, lpAddTx)

	// Create snipe transactions for each bid
	for i, bid := range bids {
		// Calculate deadline (5 minutes from now)
		deadline := big.NewInt(time.Now().Add(5 * time.Minute).Unix())

		// Calculate minimum amount out (can be improved with price calculation)
		amountOutMin := big.NewInt(1) // Minimum 1 wei of tokens

		// Get nonce for the sniper
		nonce, err := m.client.PendingNonceAt(ctx, bid.Wallet)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce for sniper %s: %v", bid.Wallet.Hex(), err)
		}

		// Calculate gas price (decreasing for proper ordering)
		// Each subsequent transaction should have slightly lower gas price
		gasPrice := new(big.Int).Sub(baseGasPrice, big.NewInt(int64(i+1)))
		if gasPrice.Cmp(big.NewInt(1000000000)) < 0 { // Minimum 1 gwei
			gasPrice = big.NewInt(1000000000)
		}

		// Create snipe transaction
		snipeTx, err := m.sniperContract.CreateSnipeTransaction(
			ctx,
			bid.Wallet,
			token,
			creator,
			bid.SwapAmount,
			bid.BribeAmount,
			amountOutMin,
			deadline,
			gasPrice,
			nonce,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create snipe transaction for %s: %v", bid.Wallet.Hex(), err)
		}

		transactions = append(transactions, snipeTx)
	}

	return transactions, nil
}

// SubmitBundle submits a transaction bundle to the network
func (m *Manager) SubmitBundle(ctx context.Context, transactions []*types.Transaction) error {
	// Submit transactions in order
	for i, tx := range transactions {
		err := m.client.SendTransaction(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to submit transaction %d: %v", i, err)
		}
	}

	return nil
}

// GetBundleGasPrice calculates the gas price for a bundle transaction
func (m *Manager) GetBundleGasPrice(baseGasPrice *big.Int, position int) *big.Int {
	// Each subsequent transaction in the bundle should have a slightly lower gas price
	// to ensure proper ordering
	decrease := big.NewInt(int64(position))
	result := new(big.Int).Sub(baseGasPrice, decrease)

	// Ensure minimum gas price of 1 gwei
	minGasPrice := big.NewInt(1000000000)
	if result.Cmp(minGasPrice) < 0 {
		return minGasPrice
	}

	return result
}

// extractTokenFromLPAdd extracts the token address from an addLiquidityETH transaction
func (m *Manager) extractTokenFromLPAdd(tx *types.Transaction) (common.Address, error) {
	if len(tx.Data()) < 36 { // 4 bytes selector + 32 bytes token address
		return common.Address{}, fmt.Errorf("insufficient data length")
	}

	// Skip the 4-byte selector
	data := tx.Data()[4:]

	// Extract token address (first parameter, last 20 bytes of first 32 bytes)
	token := common.BytesToAddress(data[12:32])

	return token, nil
}

// EstimateBundleGas estimates the total gas required for a bundle
func (m *Manager) EstimateBundleGas(
	ctx context.Context,
	lpAddTx *types.Transaction,
	bids []*SnipeBid,
) (uint64, error) {
	totalGas := lpAddTx.Gas()

	// Extract token and creator for gas estimation
	creator, err := m.sniperContract.GetCreatorFromLPAddTx(lpAddTx)
	if err != nil {
		return 0, err
	}

	token, err := m.extractTokenFromLPAdd(lpAddTx)
	if err != nil {
		return 0, err
	}

	// Estimate gas for each snipe
	for _, bid := range bids {
		deadline := big.NewInt(time.Now().Add(5 * time.Minute).Unix())
		amountOutMin := big.NewInt(1)

		gas, err := m.sniperContract.EstimateGasForSnipe(
			ctx,
			bid.Wallet,
			token,
			creator,
			bid.SwapAmount,
			bid.BribeAmount,
			amountOutMin,
			deadline,
		)
		if err != nil {
			// Use a conservative estimate if estimation fails
			gas = 300000
		}

		totalGas += gas
	}

	return totalGas, nil
}

// GetSniperContract returns the sniper contract instance
func (m *Manager) GetSniperContract() *dex.SniperContract {
	return m.sniperContract
}
