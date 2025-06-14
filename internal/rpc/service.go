package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"sync"

	"sniper-bot/internal/config"
	"sniper-bot/internal/db"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Service represents the RPC proxy service
type Service struct {
	config     *config.Config
	db         *db.DB
	baseClient *ethclient.Client
	server     *http.Server
	mu         sync.RWMutex
	snipeBids  map[string][]*SnipeBid // map[tokenAddress][]*SnipeBid
}

// SnipeBid represents a sniper's bid for a token
type SnipeBid struct {
	UserID       string
	TokenAddress common.Address
	BribeAmount  *big.Int
	Wallet       common.Address
}

// Function selectors for Uniswap V2
var (
	// createPair(address,address) -> bytes4(keccak256("createPair(address,address)"))
	createPairSelector = crypto.Keccak256([]byte("createPair(address,address)"))[:4]
	// addLiquidityETH(address,uint256,uint256,uint256,address,uint256) -> bytes4(keccak256("addLiquidityETH(address,uint256,uint256,uint256,address,uint256)"))
	addLiquidityETHSelector = crypto.Keccak256([]byte("addLiquidityETH(address,uint256,uint256,uint256,address,uint256)"))[:4]
)

// NewService creates a new RPC service
func NewService(cfg *config.Config, database *db.DB) (*Service, error) {
	client, err := ethclient.Dial(cfg.BaseRPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Base: %v", err)
	}

	return &Service{
		config:     cfg,
		db:         database,
		baseClient: client,
		snipeBids:  make(map[string][]*SnipeBid),
	}, nil
}

// Start starts the RPC service
func (s *Service) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRPC)

	s.server = &http.Server{
		Addr:    ":8545",
		Handler: mux,
	}

	log.Printf("Starting RPC proxy on :8545")
	return s.server.ListenAndServe()
}

// Stop stops the RPC service
func (s *Service) Stop() error {
	return s.server.Shutdown(context.Background())
}

// AddSnipeBid adds a new snipe bid
func (s *Service) AddSnipeBid(bid *SnipeBid) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenAddr := bid.TokenAddress.Hex()
	s.snipeBids[tokenAddr] = append(s.snipeBids[tokenAddr], bid)
}

func (s *Service) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Forward non-eth_sendRawTransaction requests to Base
	if req.Method != "eth_sendRawTransaction" {
		s.forwardToBase(w, body, false)
		return
	}

	// Handle eth_sendRawTransaction
	var params []string
	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Printf("Error unmarshaling params: %v", err)
		http.Error(w, "Invalid transaction parameters", http.StatusBadRequest)
		return
	}

	if len(params) == 0 {
		http.Error(w, "Missing transaction data", http.StatusBadRequest)
		return
	}

	txHex := params[0]
	log.Printf("Received transaction: %s", txHex)

	// Decode transaction
	txData, err := hexutil.Decode(txHex)
	if err != nil {
		http.Error(w, "Invalid transaction hex", http.StatusBadRequest)
		return
	}

	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txData); err != nil {
		http.Error(w, "Invalid transaction data", http.StatusBadRequest)
		return
	}
	fmt.Printf("Transaction: %s\n", tx.Hash().Hex())

	// Check if this is a createPair transaction
	if s.isCreatePairTransaction(tx) {
		tokenA, tokenB, err := s.extractTokensFromCreatePair(tx)
		if err != nil {
			log.Printf("Error extracting tokens from createPair: %v", err)
		} else {
			log.Printf("🎯 CREATE_PAIR transaction detected: %s", tx.Hash().Hex())
			log.Printf("   TokenA: %s", tokenA.Hex())
			log.Printf("   TokenB: %s", tokenB.Hex())

			// TODO: Store this information for sniping
		}
	}

	// Check if this is an addLiquidityETH transaction
	if s.isAddLiquidityTransaction(tx) {
		token, err := s.extractTokenFromAddLiquidity(tx)
		if err != nil {
			log.Printf("Error extracting token from addLiquidity: %v", err)
		} else {
			log.Printf("🎯 ADD_LIQUIDITY transaction detected: %s", tx.Hash().Hex())
			log.Printf("   Token: %s", token.Hex())

			// TODO: This is where you'd trigger sniping logic
		}
	}

	return

	// Forward the transaction to Base
	s.forwardToBase(w, body, true)
}

func (s *Service) isCreatePairTransaction(tx *types.Transaction) bool {
	// Check if transaction has data
	if len(tx.Data()) < 4 {
		return false
	}

	// Check if the transaction is sent to the factory address
	factoryAddr := common.HexToAddress(s.config.UniswapV2Factory)
	if tx.To() == nil || *tx.To() != factoryAddr {
		return false
	}

	// Check if the function selector matches createPair
	selector := tx.Data()[:4]
	return bytes.Equal(selector, createPairSelector)
}

func (s *Service) isAddLiquidityTransaction(tx *types.Transaction) bool {
	// Check if transaction has data
	if len(tx.Data()) < 4 {
		return false
	}

	// Check if the transaction is sent to the router address
	routerAddr := common.HexToAddress(s.config.UniswapV2Router)
	if tx.To() == nil || *tx.To() != routerAddr {
		return false
	}

	// Check if the function selector matches addLiquidityETH
	selector := tx.Data()[:4]
	return bytes.Equal(selector, addLiquidityETHSelector)
}

func (s *Service) extractTokensFromCreatePair(tx *types.Transaction) (tokenA, tokenB common.Address, err error) {
	if len(tx.Data()) < 68 { // 4 bytes selector + 32 bytes tokenA + 32 bytes tokenB
		return common.Address{}, common.Address{}, fmt.Errorf("insufficient data length")
	}

	// Skip the 4-byte selector
	data := tx.Data()[4:]

	// Extract tokenA (first 32 bytes, but address is in the last 20 bytes)
	tokenA = common.BytesToAddress(data[12:32])

	// Extract tokenB (second 32 bytes, but address is in the last 20 bytes)
	tokenB = common.BytesToAddress(data[44:64])

	return tokenA, tokenB, nil
}

func (s *Service) extractTokenFromAddLiquidity(tx *types.Transaction) (token common.Address, err error) {
	if len(tx.Data()) < 36 { // 4 bytes selector + 32 bytes token address
		return common.Address{}, fmt.Errorf("insufficient data length")
	}

	// Skip the 4-byte selector
	data := tx.Data()[4:]

	// Extract token address (first parameter, last 20 bytes of first 32 bytes)
	token = common.BytesToAddress(data[12:32])

	return token, nil
}

func (s *Service) forwardToBase(w http.ResponseWriter, requestBody []byte, isToSequencer bool) {
	// Forward the request to Base

	rpcURL := s.config.BaseRPCURL
	if isToSequencer {
		rpcURL = s.config.BaseSequencerRPCURL
	}
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		log.Printf("Error forwarding to Base: %v", err)
		http.Error(w, "Failed to forward request to Base", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response: %v", err)
	}
}
