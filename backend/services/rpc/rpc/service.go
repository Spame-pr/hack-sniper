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
	"os"
	"sniper-bot/pkg/config"
	"sniper-bot/services/bot/db"
	"sync"

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
	botAPIURL  string
}

// SnipeBid represents a sniper's bid for a token
type SnipeBid struct {
	UserID       string
	TokenAddress common.Address
	BribeAmount  *big.Int
	Wallet       common.Address
}

// LPAddNotificationPayload represents the payload sent to bot service
type LPAddNotificationPayload struct {
	TokenAddress   string `json:"tokenAddress"`
	CreatorAddress string `json:"creatorAddress"`
	TxCallData     string `json:"txCallData"`
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

	// Get API service configuration
	botAPIURL := os.Getenv("API_SERVICE_URL")
	if botAPIURL == "" {
		botAPIURL = "http://localhost:8080" // Default for local development
	}

	return &Service{
		config:     cfg,
		db:         database,
		baseClient: client,
		snipeBids:  make(map[string][]*SnipeBid),
		botAPIURL:  botAPIURL,
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
		http.Error(w, "Invalid transaction parameters", http.StatusBadRequest)
		return
	}

	if len(params) == 0 {
		http.Error(w, "Missing transaction data", http.StatusBadRequest)
		return
	}

	txCallData := params[0]

	// Decode transaction
	txData, err := hexutil.Decode(txCallData)
	if err != nil {
		http.Error(w, "Invalid transaction hex", http.StatusBadRequest)
		return
	}

	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txData); err != nil {
		http.Error(w, "Invalid transaction data", http.StatusBadRequest)
		return
	}

	// Check if this is an addLiquidityETH transaction
	if s.isAddLiquidityTransaction(tx) {
		token, err := s.extractTokenFromAddLiquidity(tx)
		if err != nil {
			log.Printf("Error extracting token from addLiquidity: %v", err)
		} else {
			// Extract the sender (token creator) from the transaction
			sender, err := s.extractSenderFromTransaction(tx)
			if err != nil {
				log.Printf("Error extracting sender from addLiquidity: %v", err)
			} else {
				log.Printf("🎯 ADD_LIQUIDITY transaction detected: %s", tx.Hash().Hex())
				log.Printf("   Token: %s", token.Hex())
				log.Printf("   Creator (Sender): %s", sender.Hex())

				if err := s.notifyBotService(token, sender, txCallData); err != nil {
					log.Printf("❌ Failed to notify bot service: %v", err)
				}
			}
		}
	}

	// Forward the transaction to Base
	s.forwardToBase(w, body, true)
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

func (s *Service) extractSenderFromTransaction(tx *types.Transaction) (common.Address, error) {
	// Get the chain ID from the base client
	chainID, err := s.baseClient.ChainID(context.Background())
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get chain ID: %v", err)
	}

	// Create the appropriate signer based on the transaction type
	var signer types.Signer

	// Check if it's a legacy transaction or EIP-155
	if tx.Type() == types.LegacyTxType {
		// For legacy transactions, use EIP155 signer
		signer = types.NewEIP155Signer(chainID)
	} else {
		// For EIP-1559 and other transaction types, use LatestSigner
		signer = types.LatestSigner(nil)
	}

	// Extract the sender using the signer
	sender, err := types.Sender(signer, tx)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to extract sender: %v", err)
	}

	return sender, nil
}

// notifyBotService sends LP_ADD notification to the bot service
func (s *Service) notifyBotService(tokenAddress, creatorAddress common.Address, txCallData string) error {
	// Prepare payload
	payload := LPAddNotificationPayload{
		TokenAddress:   tokenAddress.Hex(),
		CreatorAddress: creatorAddress.Hex(),
		TxCallData:     txCallData,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Create HTTP request
	url := s.botAPIURL + "/api/lp-add"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	cfg := config.Load()

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AuthKey)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bot service returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ Successfully notified bot service about LP_ADD")
	return nil
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
