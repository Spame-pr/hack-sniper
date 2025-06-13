package rpc

import (
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

	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Forward non-eth_sendRawTransaction requests to Base
	if req.Method != "eth_sendRawTransaction" {
		s.forwardToBase(w, r)
		return
	}

	// Handle eth_sendRawTransaction
	var txHex string
	if err := json.Unmarshal(req.Params, &txHex); err != nil {
		http.Error(w, "Invalid transaction data", http.StatusBadRequest)
		return
	}

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

	// Check if this is an LP_ADD transaction
	if s.isLPAddTransaction(tx) {
		// TODO: Implement bundle creation and submission
		log.Printf("LP_ADD transaction detected: %s", tx.Hash().Hex())
	}

	// Forward the transaction to Base
	s.forwardToBase(w, r)
}

func (s *Service) isLPAddTransaction(tx *types.Transaction) bool {
	// TODO: Implement LP_ADD transaction detection
	// This should check if the transaction is adding liquidity to a DEX
	return false
}

func (s *Service) forwardToBase(w http.ResponseWriter, r *http.Request) {
	// Forward the request to Base
	resp, err := http.Post(s.config.BaseRPCURL, "application/json", r.Body)
	if err != nil {
		http.Error(w, "Failed to forward request to Base", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response: %v", err)
	}
}
