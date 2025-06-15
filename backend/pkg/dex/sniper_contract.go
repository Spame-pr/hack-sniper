package dex

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SniperContract represents the custom sniper contract
type SniperContract struct {
	client   *ethclient.Client
	contract *bind.BoundContract
	address  common.Address
	chainID  *big.Int
}

// SnipeData represents the parameters for a snipe
type SnipeData struct {
	Token        common.Address
	Creator      common.Address
	SwapAmount   *big.Int
	BribeAmount  *big.Int
	AmountOutMin *big.Int
	Deadline     *big.Int
}

// SniperContractABI is the ABI for the sniper contract
const SniperContractABI = `[
	{
		"inputs": [
			{"internalType": "address", "name": "_router", "type": "address"}
		],
		"stateMutability": "nonpayable",
		"type": "constructor"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "internalType": "address", "name": "sniper", "type": "address"},
			{"indexed": true, "internalType": "address", "name": "token", "type": "address"},
			{"indexed": true, "internalType": "address", "name": "creator", "type": "address"},
			{"indexed": false, "internalType": "uint256", "name": "swapAmount", "type": "uint256"},
			{"indexed": false, "internalType": "uint256", "name": "bribeAmount", "type": "uint256"},
			{"indexed": false, "internalType": "uint256", "name": "tokensReceived", "type": "uint256"}
		],
		"name": "SnipeExecuted",
		"type": "event"
	},
	{
		"inputs": [
			{"internalType": "address", "name": "token", "type": "address"},
			{"internalType": "address payable", "name": "creator", "type": "address"},
			{"internalType": "uint256", "name": "amountOutMin", "type": "uint256"},
			{"internalType": "uint256", "name": "deadline", "type": "uint256"},
			{"internalType": "uint256", "name": "bribeAmount", "type": "uint256"}
		],
		"name": "snipeWithBribe",
		"outputs": [],
		"stateMutability": "payable",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "emergencyWithdraw",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{"internalType": "address", "name": "token", "type": "address"}
		],
		"name": "withdrawToken",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"stateMutability": "payable",
		"type": "receive"
	}
]`

// NewSniperContract creates a new sniper contract instance
func NewSniperContract(client *ethclient.Client, address common.Address) (*SniperContract, error) {
	parsed, err := abi.JSON(strings.NewReader(SniperContractABI))
	if err != nil {
		return nil, err
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	contract := bind.NewBoundContract(address, parsed, client, client, client)
	return &SniperContract{
		client:   client,
		contract: contract,
		address:  address,
		chainID:  chainID,
	}, nil
}

// SnipeWithBribe executes a single snipe with bribe
func (s *SniperContract) SnipeWithBribe(
	ctx context.Context,
	privateKey *ecdsa.PrivateKey,
	token common.Address,
	creator common.Address,
	swapAmount *big.Int,
	bribeAmount *big.Int,
	amountOutMin *big.Int,
	deadline *big.Int,
) (*types.Transaction, error) {
	// Create transaction options
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, s.chainID)
	if err != nil {
		return nil, err
	}

	// Set the total value (swap amount + bribe amount)
	totalValue := new(big.Int).Add(swapAmount, bribeAmount)
	auth.Value = totalValue

	// Estimate gas
	auth.GasLimit = 300000 // Conservative estimate

	// Get current gas price
	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	auth.GasPrice = gasPrice

	// Execute the transaction
	return s.contract.Transact(auth, "snipeWithBribe",
		token,
		creator,
		amountOutMin,
		deadline,
		bribeAmount,
	)
}

// CreateSnipeTransaction creates a snipe transaction without executing it
func (s *SniperContract) CreateSnipeTransaction(
	ctx context.Context,
	from common.Address,
	token common.Address,
	creator common.Address,
	swapAmount *big.Int,
	bribeAmount *big.Int,
	amountOutMin *big.Int,
	deadline *big.Int,
	gasPrice *big.Int,
	nonce uint64,
) (*types.Transaction, error) {
	// Parse the ABI
	parsed, err := abi.JSON(strings.NewReader(SniperContractABI))
	if err != nil {
		return nil, err
	}

	// Pack the function call data
	data, err := parsed.Pack("snipeWithBribe",
		token,
		creator,
		amountOutMin,
		deadline,
		bribeAmount,
	)
	if err != nil {
		return nil, err
	}

	// Calculate total value
	totalValue := new(big.Int).Add(swapAmount, bribeAmount)

	// Create the transaction
	return types.NewTransaction(
		nonce,
		s.address,
		totalValue,
		300000, // Gas limit
		gasPrice,
		data,
	), nil
}

// GetCreatorFromLPAddTx extracts the token creator from an LP_ADD transaction
func (s *SniperContract) GetCreatorFromLPAddTx(tx *types.Transaction) (common.Address, error) {
	// For LP_ADD transactions, the creator is typically the tx.origin or from address
	// This depends on how the transaction was sent

	// Get the from address using the transaction's signature
	from, err := types.Sender(types.NewEIP155Signer(s.chainID), tx)
	if err != nil {
		return common.Address{}, err
	}

	return from, nil
}

// EstimateGasForSnipe estimates gas required for a snipe transaction
func (s *SniperContract) EstimateGasForSnipe(
	ctx context.Context,
	from common.Address,
	token common.Address,
	creator common.Address,
	swapAmount *big.Int,
	bribeAmount *big.Int,
	amountOutMin *big.Int,
	deadline *big.Int,
) (uint64, error) {
	// Parse the ABI
	parsed, err := abi.JSON(strings.NewReader(SniperContractABI))
	if err != nil {
		return 0, err
	}

	// Pack the function call data
	data, err := parsed.Pack("snipeWithBribe",
		token,
		creator,
		amountOutMin,
		deadline,
		bribeAmount,
	)
	if err != nil {
		return 0, err
	}

	// Calculate total value
	totalValue := new(big.Int).Add(swapAmount, bribeAmount)

	// Create call message
	msg := ethereum.CallMsg{
		From:  from,
		To:    &s.address,
		Value: totalValue,
		Data:  data,
	}

	// Estimate gas
	return s.client.EstimateGas(ctx, msg)
}
