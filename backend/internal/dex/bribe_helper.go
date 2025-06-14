package dex

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BribeHelper handles bribe transactions without custom contracts
type BribeHelper struct {
	client  *ethclient.Client
	chainID *big.Int
}

// NewBribeHelper creates a new bribe helper
func NewBribeHelper(client *ethclient.Client) (*BribeHelper, error) {
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	return &BribeHelper{
		client:  client,
		chainID: chainID,
	}, nil
}

// CreateSwapWithBribeTxs creates both a swap transaction and a bribe transaction
func (b *BribeHelper) CreateSwapWithBribeTxs(
	ctx context.Context,
	privateKey *ecdsa.PrivateKey,
	routerAddress common.Address,
	token common.Address,
	creator common.Address,
	swapAmount *big.Int,
	bribeAmount *big.Int,
	amountOutMin *big.Int,
	deadline *big.Int,
	baseGasPrice *big.Int,
) ([]*types.Transaction, error) {
	from := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Get nonce for the first transaction
	nonce, err := b.client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, err
	}

	var transactions []*types.Transaction

	// 1. Create swap transaction
	swapTx, err := b.createSwapTransaction(
		routerAddress,
		token,
		swapAmount,
		amountOutMin,
		deadline,
		from,
		nonce,
		baseGasPrice,
	)
	if err != nil {
		return nil, err
	}

	// Sign swap transaction
	signedSwapTx, err := types.SignTx(swapTx, types.NewEIP155Signer(b.chainID), privateKey)
	if err != nil {
		return nil, err
	}
	transactions = append(transactions, signedSwapTx)

	// 2. Create bribe transaction (higher gas price for priority)
	bribeGasPrice := new(big.Int).Add(baseGasPrice, big.NewInt(1000000000)) // +1 gwei
	bribeTx, err := b.createBribeTransaction(
		creator,
		bribeAmount,
		from,
		nonce+1,
		bribeGasPrice,
	)
	if err != nil {
		return nil, err
	}

	// Sign bribe transaction
	signedBribeTx, err := types.SignTx(bribeTx, types.NewEIP155Signer(b.chainID), privateKey)
	if err != nil {
		return nil, err
	}
	transactions = append(transactions, signedBribeTx)

	return transactions, nil
}

// createSwapTransaction creates a swap transaction
func (b *BribeHelper) createSwapTransaction(
	routerAddress common.Address,
	token common.Address,
	swapAmount *big.Int,
	amountOutMin *big.Int,
	deadline *big.Int,
	from common.Address,
	nonce uint64,
	gasPrice *big.Int,
) (*types.Transaction, error) {
	// Create the swap function call data
	// swapExactETHForTokens(uint amountOutMin, address[] path, address to, uint deadline)

	// Function selector for swapExactETHForTokens: 0x7ff36ab5
	selector := []byte{0x7f, 0xf3, 0x6a, 0xb5}

	// Encode parameters
	data := make([]byte, 4)
	copy(data, selector)

	// For simplicity, we'll use a basic encoding
	// In production, you should use the ABI encoder

	// Pack amountOutMin (32 bytes)
	amountOutMinBytes := make([]byte, 32)
	amountOutMin.FillBytes(amountOutMinBytes)
	data = append(data, amountOutMinBytes...)

	// Pack path offset (32 bytes) - offset to path array
	pathOffset := make([]byte, 32)
	pathOffset[31] = 0x80 // Offset to path (128 bytes from start)
	data = append(data, pathOffset...)

	// Pack to address (32 bytes)
	toBytes := make([]byte, 32)
	copy(toBytes[12:], from.Bytes())
	data = append(data, toBytes...)

	// Pack deadline (32 bytes)
	deadlineBytes := make([]byte, 32)
	deadline.FillBytes(deadlineBytes)
	data = append(data, deadlineBytes...)

	// Pack path array
	// Array length (2)
	pathLengthBytes := make([]byte, 32)
	pathLengthBytes[31] = 2
	data = append(data, pathLengthBytes...)

	// WETH address (Base)
	wethBytes := make([]byte, 32)
	weth := common.HexToAddress("0x4200000000000000000000000000000000000006")
	copy(wethBytes[12:], weth.Bytes())
	data = append(data, wethBytes...)

	// Token address
	tokenBytes := make([]byte, 32)
	copy(tokenBytes[12:], token.Bytes())
	data = append(data, tokenBytes...)

	return types.NewTransaction(
		nonce,
		routerAddress,
		swapAmount,
		250000, // Gas limit for swap
		gasPrice,
		data,
	), nil
}

// createBribeTransaction creates a simple ETH transfer transaction for the bribe
func (b *BribeHelper) createBribeTransaction(
	creator common.Address,
	bribeAmount *big.Int,
	from common.Address,
	nonce uint64,
	gasPrice *big.Int,
) (*types.Transaction, error) {
	return types.NewTransaction(
		nonce,
		creator,
		bribeAmount,
		21000, // Standard ETH transfer gas limit
		gasPrice,
		nil, // No data for simple transfer
	), nil
}

// EstimateSwapWithBribeGas estimates gas for both swap and bribe transactions
func (b *BribeHelper) EstimateSwapWithBribeGas(
	ctx context.Context,
	routerAddress common.Address,
	token common.Address,
	swapAmount *big.Int,
	amountOutMin *big.Int,
	deadline *big.Int,
	from common.Address,
) (uint64, error) {
	// Swap transaction gas estimate
	swapTx, err := b.createSwapTransaction(
		routerAddress,
		token,
		swapAmount,
		amountOutMin,
		deadline,
		from,
		0,             // Dummy nonce
		big.NewInt(1), // Dummy gas price
	)
	if err != nil {
		return 0, err
	}

	swapGas, err := b.client.EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    &routerAddress,
		Value: swapAmount,
		Data:  swapTx.Data(),
	})
	if err != nil {
		swapGas = 250000 // Fallback estimate
	}

	// Bribe transaction gas is always 21000 for simple ETH transfer
	bribeGas := uint64(21000)

	return swapGas + bribeGas, nil
}
