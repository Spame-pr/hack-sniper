package dex

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// UniswapV2RouterContract represents the Uniswap V2 Router contract
type UniswapV2RouterContract struct {
	*UniswapV2Router
	contract *bind.BoundContract
}

// NewUniswapV2RouterContract creates a new Uniswap V2 Router contract
func NewUniswapV2RouterContract(client *ethclient.Client, address common.Address) (*UniswapV2RouterContract, error) {
	parsed, err := abi.JSON(strings.NewReader(UniswapV2RouterABI))
	if err != nil {
		return nil, err
	}

	contract := bind.NewBoundContract(address, parsed, client, client, client)
	return &UniswapV2RouterContract{
		UniswapV2Router: NewUniswapV2Router(client, address, common.Address{}),
		contract:        contract,
	}, nil
}

// AddLiquidity adds liquidity to a token pair
func (r *UniswapV2RouterContract) AddLiquidity(
	opts *bind.TransactOpts,
	tokenA,
	tokenB common.Address,
	amountADesired,
	amountBDesired,
	amountAMin,
	amountBMin *big.Int,
	to common.Address,
	deadline *big.Int,
) (*types.Transaction, error) {
	return r.contract.Transact(opts, "addLiquidity",
		tokenA,
		tokenB,
		amountADesired,
		amountBDesired,
		amountAMin,
		amountBMin,
		to,
		deadline,
	)
}

// SwapExactETHForTokens swaps ETH for tokens
func (r *UniswapV2RouterContract) SwapExactETHForTokens(
	opts *bind.TransactOpts,
	amountOutMin *big.Int,
	path []common.Address,
	to common.Address,
	deadline *big.Int,
) (*types.Transaction, error) {
	return r.contract.Transact(opts, "swapExactETHForTokens",
		amountOutMin,
		path,
		to,
		deadline,
	)
}

// UniswapV2FactoryContract represents the Uniswap V2 Factory contract
type UniswapV2FactoryContract struct {
	*UniswapV2Factory
	contract *bind.BoundContract
}

// NewUniswapV2FactoryContract creates a new Uniswap V2 Factory contract
func NewUniswapV2FactoryContract(client *ethclient.Client, address common.Address) (*UniswapV2FactoryContract, error) {
	parsed, err := abi.JSON(strings.NewReader(UniswapV2FactoryABI))
	if err != nil {
		return nil, err
	}

	contract := bind.NewBoundContract(address, parsed, client, client, client)
	return &UniswapV2FactoryContract{
		UniswapV2Factory: NewUniswapV2Factory(client, address),
		contract:         contract,
	}, nil
}

// GetPair gets the pair address for two tokens
func (f *UniswapV2FactoryContract) GetPair(
	opts *bind.CallOpts,
	tokenA,
	tokenB common.Address,
) (common.Address, error) {
	var result []interface{}
	err := f.contract.Call(opts, &result, "getPair", tokenA, tokenB)
	if err != nil {
		return common.Address{}, err
	}
	return result[0].(common.Address), nil
}

// UniswapV2PairContract represents the Uniswap V2 Pair contract
type UniswapV2PairContract struct {
	*UniswapV2Pair
	contract *bind.BoundContract
}

// NewUniswapV2PairContract creates a new Uniswap V2 Pair contract
func NewUniswapV2PairContract(client *ethclient.Client, address common.Address) (*UniswapV2PairContract, error) {
	parsed, err := abi.JSON(strings.NewReader(UniswapV2PairABI))
	if err != nil {
		return nil, err
	}

	contract := bind.NewBoundContract(address, parsed, client, client, client)
	return &UniswapV2PairContract{
		UniswapV2Pair: NewUniswapV2Pair(client, address),
		contract:      contract,
	}, nil
}

// GetReserves gets the current reserves of the pair
func (p *UniswapV2PairContract) GetReserves(opts *bind.CallOpts) (*big.Int, *big.Int, error) {
	var result []interface{}
	err := p.contract.Call(opts, &result, "getReserves")
	if err != nil {
		return nil, nil, err
	}
	return result[0].(*big.Int), result[1].(*big.Int), nil
}

// GetToken0 gets the first token in the pair
func (p *UniswapV2PairContract) GetToken0(opts *bind.CallOpts) (common.Address, error) {
	var result []interface{}
	err := p.contract.Call(opts, &result, "token0")
	if err != nil {
		return common.Address{}, err
	}
	return result[0].(common.Address), nil
}

// GetToken1 gets the second token in the pair
func (p *UniswapV2PairContract) GetToken1(opts *bind.CallOpts) (common.Address, error) {
	var result []interface{}
	err := p.contract.Call(opts, &result, "token1")
	if err != nil {
		return common.Address{}, err
	}
	return result[0].(common.Address), nil
}
