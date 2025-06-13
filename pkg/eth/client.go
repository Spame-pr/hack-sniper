package eth

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client wraps the Ethereum client with additional functionality
type Client struct {
	*ethclient.Client
	chainID *big.Int
}

// NewClient creates a new ETH client
func NewClient(rpcURL string) (*Client, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:  client,
		chainID: chainID,
	}, nil
}

// GetChainID returns the chain ID
func (c *Client) GetChainID() *big.Int {
	return c.chainID
}

// CreateTransactOpts creates transaction options from a private key
func (c *Client) CreateTransactOpts(privateKey *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, c.chainID)
	if err != nil {
		return nil, err
	}
	return auth, nil
}

// GetBalance gets the ETH balance of an address
func (c *Client) GetBalance(ctx context.Context, address common.Address) (*big.Int, error) {
	return c.BalanceAt(ctx, address, nil)
}

// GetTokenBalance gets the token balance of an address
func (c *Client) GetTokenBalance(ctx context.Context, tokenAddress, holderAddress common.Address) (*big.Int, error) {
	// TODO: Implement ERC20 token balance lookup
	// This should call balanceOf(address) on the token contract
	return big.NewInt(0), nil
}

// SendTransaction sends a transaction to the network
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return c.Client.SendTransaction(ctx, tx)
}

// WaitForTransaction waits for a transaction to be mined
func (c *Client) WaitForTransaction(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return bind.WaitMined(ctx, c.Client, &types.Transaction{})
}

// EstimateGas estimates the gas required for a transaction
func (c *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return c.Client.EstimateGas(ctx, msg)
}

// GetNonce gets the nonce for an address
func (c *Client) GetNonce(ctx context.Context, address common.Address) (uint64, error) {
	return c.PendingNonceAt(ctx, address)
}

// SignTransaction signs a transaction with a private key
func (c *Client) SignTransaction(tx *types.Transaction, privateKey *ecdsa.PrivateKey) (*types.Transaction, error) {
	signer := types.NewEIP155Signer(c.chainID)
	return types.SignTx(tx, signer, privateKey)
}
