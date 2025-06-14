package api

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"sniper-bot/pkg/config"
	"sniper-bot/services/bot/bundle"
	"sniper-bot/services/bot/db"
	"sniper-bot/services/bot/wallet"
	"sort"
	"strconv"
	"strings"
	"time"

	"sniper-bot/pkg/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Service represents the API service
type Service struct {
	ethClient     *eth.Client
	walletManager *wallet.Manager
	db            *db.DB
	httpServer    *http.Server
	apiKey        string
	bundleManager *bundle.Manager
	config        *config.Config
}

// LPAddNotification represents the payload for LP_ADD notifications
type LPAddNotification struct {
	TokenAddress   string `json:"tokenAddress"`
	CreatorAddress string `json:"creatorAddress"`
	TxCallData     string `json:"txCallData"`
}

// BundleSubmissionRequest represents the request to submit a bundle to Base sequencer
type BundleSubmissionRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []struct {
		Txs             []string `json:"txs"`
		TrustedBuilders []string `json:"trustedBuilders"`
		BlockNumber     string   `json:"blockNumber"`
	} `json:"params"`
	ID int `json:"id"`
}

// NewService creates a new API service
func NewService(walletManager *wallet.Manager, database *db.DB) (*Service, error) {
	cfg := config.Load()

	// Get API key for authentication
	apiKey := os.Getenv("AUTH_KEY")
	if apiKey == "" {
		panic("AUTH_KEY environment variable is required")
	}

	// Use BASE_SEQUENCER_URL for RPC connection
	sequencerURL := cfg.BaseSequencerRPCURL

	ethClient, err := eth.NewClient(sequencerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create eth client: %v", err)
	}

	// Initialize bundle manager with sniper contract address
	sniperContractAddr := common.HexToAddress(os.Getenv("SNIPER_CONTRACT_ADDRESS"))
	if sniperContractAddr == (common.Address{}) {
		log.Printf("⚠️ SNIPER_CONTRACT_ADDRESS not set, bundle functionality will be limited")
	}

	bundleManager, err := bundle.NewManager(ethClient.Client, sniperContractAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle manager: %v", err)
	}

	return &Service{
		walletManager: walletManager,
		ethClient:     ethClient,
		db:            database,
		apiKey:        apiKey,
		bundleManager: bundleManager,
		config:        cfg,
	}, nil
}

// Start starts the API service
func (s *Service) Start() error {
	return s.startHTTPServer()
}

// startHTTPServer starts the HTTP server for receiving LP_ADD notifications
func (s *Service) startHTTPServer() error {
	mux := http.NewServeMux()

	// Add the LP_ADD notification endpoint
	mux.HandleFunc("/api/lp-add", s.handleLPAddNotification)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Get port from environment or use default
	port := os.Getenv("API_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("🌐 Starting API HTTP server on port %s", port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("API HTTP server error: %v", err)
	}

	return nil
}

// handleLPAddNotification handles LP_ADD notifications from the RPC service
func (s *Service) handleLPAddNotification(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	authHeader := r.Header.Get("Authorization")
	expectedAuth := "Bearer " + s.apiKey
	if authHeader != expectedAuth {
		log.Printf("🚨 Unauthorized LP_ADD request from %s", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON payload
	var notification LPAddNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		log.Printf("❌ Failed to parse LP_ADD notification: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Log the received data
	log.Printf("📨 LP_ADD Notification received:")
	log.Printf("   🎯 Token Address: %s", notification.TokenAddress)
	log.Printf("   👤 Creator Address: %s", notification.CreatorAddress)
	log.Printf("   📝 TX Call Data: %s", notification.TxCallData)
	log.Printf("   🌐 From: %s", r.RemoteAddr)

	// Process the LP_ADD notification and create bundle
	go s.processLPAddAndCreateBundle(notification)

	// Respond with success immediately
	response := map[string]interface{}{
		"status":  "success",
		"message": "LP_ADD notification received and processing started",
		"data": map[string]string{
			"tokenAddress":   notification.TokenAddress,
			"creatorAddress": notification.CreatorAddress,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// processLPAddAndCreateBundle processes the LP_ADD notification and creates a bundle
func (s *Service) processLPAddAndCreateBundle(notification LPAddNotification) {
	ctx := context.Background()

	log.Printf("🔄 Processing LP_ADD for token %s", notification.TokenAddress)

	// Get pending snipes for this token
	snipes, err := s.db.GetSnipesByToken(notification.TokenAddress)
	if err != nil {
		log.Printf("❌ Failed to get snipes for token %s: %v", notification.TokenAddress, err)
		return
	}

	if len(snipes) == 0 {
		log.Printf("ℹ️ No pending snipes found for token %s", notification.TokenAddress)
		return
	}

	log.Printf("📊 Found %d pending snipes for token %s", len(snipes), notification.TokenAddress)

	// Reconstruct LP_ADD transaction from call data
	lpAddTx, err := s.reconstructLPAddTransaction(notification)
	if err != nil {
		log.Printf("❌ Failed to reconstruct LP_ADD transaction: %v", err)
		return
	}

	// Convert database snipes to bundle format
	bundleBids, err := s.convertSnipesToBundleBids(snipes)
	if err != nil {
		log.Printf("❌ Failed to convert snipes to bundle bids: %v", err)
		return
	}

	// Sort bids by bribe amount (descending) - highest bribes first
	sort.Slice(bundleBids, func(i, j int) bool {
		return bundleBids[i].BribeAmount.Cmp(bundleBids[j].BribeAmount) > 0
	})

	log.Printf("💰 Sorted %d snipes by bribe amount (highest first)", len(bundleBids))
	for i, bid := range bundleBids {
		bribeETH := new(big.Float).Quo(new(big.Float).SetInt(bid.BribeAmount), big.NewFloat(1e18))
		log.Printf("   %d. Wallet %s: %s ETH bribe", i+1, bid.Wallet.Hex()[:10]+"...", bribeETH.Text('f', 4))
	}

	// Create bundle transactions
	bundleTxs, err := s.createBundleTransactions(ctx, lpAddTx, bundleBids, notification)
	if err != nil {
		log.Printf("❌ Failed to create bundle transactions: %v", err)
		return
	}

	log.Printf("📦 Created bundle with %d transactions (1 LP_ADD + %d snipes)", len(bundleTxs), len(bundleBids))

	// Submit bundle to Base sequencer
	if err := s.submitBundleToSequencer(ctx, bundleTxs); err != nil {
		log.Printf("❌ Failed to submit bundle to sequencer: %v", err)
		return
	}

	// Update snipe statuses to 'submitted'
	for _, snipe := range snipes {
		if err := s.db.UpdateSnipeStatus(snipe.ID, "submitted"); err != nil {
			log.Printf("⚠️ Failed to update snipe status for ID %d: %v", snipe.ID, err)
		}
	}

	log.Printf("✅ Bundle submitted successfully for token %s with %d snipes", notification.TokenAddress, len(snipes))
}

// reconstructLPAddTransaction reconstructs the LP_ADD transaction from call data
func (s *Service) reconstructLPAddTransaction(notification LPAddNotification) (*types.Transaction, error) {
	// Parse call data
	callData := notification.TxCallData
	if !strings.HasPrefix(callData, "0x") {
		callData = "0x" + callData
	}

	data, err := hex.DecodeString(callData[2:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode call data: %v", err)
	}

	// Get current gas price and nonce for the creator
	creatorAddr := common.HexToAddress(notification.CreatorAddress)
	gasPrice, err := s.ethClient.Client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %v", err)
	}

	nonce, err := s.ethClient.Client.PendingNonceAt(context.Background(), creatorAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}

	// Create the transaction (this is a simplified reconstruction)
	// In a real implementation, you'd need to extract more details from the call data
	routerAddr := common.HexToAddress(s.config.UniswapV2Router)

	// Estimate ETH value from call data (this is simplified - you'd parse the actual parameters)
	ethValue := big.NewInt(1000000000000000) // 0.001 ETH as default

	tx := types.NewTransaction(
		nonce,
		routerAddr,
		ethValue,
		300000, // Gas limit
		gasPrice,
		data,
	)

	return tx, nil
}

// convertSnipesToBundleBids converts database snipes to bundle bid format
func (s *Service) convertSnipesToBundleBids(snipes []*db.Snipe) ([]*bundle.SnipeBid, error) {
	var bundleBids []*bundle.SnipeBid

	for _, snipe := range snipes {
		// Parse amounts
		swapAmount, err := s.parseETHAmount(snipe.Amount)
		if err != nil {
			log.Printf("⚠️ Failed to parse swap amount for snipe %d: %v", snipe.ID, err)
			continue
		}

		bribeAmount, err := s.parseETHAmount(snipe.BribeAmount)
		if err != nil {
			log.Printf("⚠️ Failed to parse bribe amount for snipe %d: %v", snipe.ID, err)
			continue
		}

		// Get wallet private key
		wallet, err := s.walletManager.GetWallet(snipe.UserID)
		if err != nil {
			log.Printf("⚠️ Failed to get wallet for user %s: %v", snipe.UserID, err)
			continue
		}

		bundleBid := &bundle.SnipeBid{
			UserID:       snipe.UserID,
			TokenAddress: common.HexToAddress(snipe.TokenAddress),
			SwapAmount:   swapAmount,
			BribeAmount:  bribeAmount,
			Wallet:       wallet.Address,
			PrivateKey:   hex.EncodeToString(crypto.FromECDSA(wallet.PrivateKey)),
		}

		bundleBids = append(bundleBids, bundleBid)
	}

	return bundleBids, nil
}

// parseETHAmount parses ETH amount string to wei
func (s *Service) parseETHAmount(amountStr string) (*big.Int, error) {
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, err
	}

	// Convert to wei
	weiFloat := new(big.Float).Mul(big.NewFloat(amount), big.NewFloat(1e18))
	wei, _ := weiFloat.Int(nil)

	return wei, nil
}

// createBundleTransactions creates the bundle transactions with proper gas pricing
func (s *Service) createBundleTransactions(ctx context.Context, lpAddTx *types.Transaction, bids []*bundle.SnipeBid, notification LPAddNotification) ([]*types.Transaction, error) {
	var transactions []*types.Transaction

	// Add the LP_ADD transaction first
	transactions = append(transactions, lpAddTx)

	// Get base gas price from LP_ADD transaction
	baseGasPrice := lpAddTx.GasPrice()
	if baseGasPrice == nil {
		var err error
		baseGasPrice, err = s.ethClient.Client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get gas price: %v", err)
		}
	}

	// Create snipe transactions with decreasing gas prices
	for i, bid := range bids {
		// Calculate gas price (each subsequent tx has gasPrice = previous - 1 wei)
		gasPrice := new(big.Int).Sub(baseGasPrice, big.NewInt(int64(i+1)))
		minGasPrice := big.NewInt(1000000000) // 1 gwei minimum
		if gasPrice.Cmp(minGasPrice) < 0 {
			gasPrice = minGasPrice
		}

		// Note: Private key not needed for CreateSnipeTransaction as it only creates unsigned tx

		// Get nonce for the sniper
		nonce, err := s.ethClient.Client.PendingNonceAt(ctx, bid.Wallet)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce for sniper %s: %v", bid.Wallet.Hex(), err)
		}

		// Extract creator address from notification
		creatorAddr := common.HexToAddress(notification.CreatorAddress)
		deadline := big.NewInt(time.Now().Add(5 * time.Minute).Unix())
		amountOutMin := big.NewInt(1) // Minimum 1 wei of tokens (unlimited slippage)

		// Get sniper contract from bundle manager
		sniperContract := s.bundleManager.GetSniperContract()

		snipeTx, err := sniperContract.CreateSnipeTransaction(
			ctx,
			bid.Wallet,
			bid.TokenAddress,
			creatorAddr,
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

// submitBundleToSequencer submits the bundle to Base sequencer
func (s *Service) submitBundleToSequencer(ctx context.Context, transactions []*types.Transaction) error {
	// Convert transactions to raw hex strings
	var rawTxs []string
	for _, tx := range transactions {
		rawTx, err := tx.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal transaction: %v", err)
		}
		rawTxs = append(rawTxs, "0x"+hex.EncodeToString(rawTx))
	}

	// Get current block number
	blockNumber, err := s.ethClient.Client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block number: %v", err)
	}

	// Target next block
	targetBlock := fmt.Sprintf("0x%x", blockNumber+1)

	// Create bundle submission request
	bundleReq := BundleSubmissionRequest{
		JSONRPC: "2.0",
		Method:  "eth_sendBundle",
		Params: []struct {
			Txs             []string `json:"txs"`
			TrustedBuilders []string `json:"trustedBuilders"`
			BlockNumber     string   `json:"blockNumber"`
		}{
			{
				Txs: rawTxs,
				TrustedBuilders: []string{
					"titan",
					"beaver",
					"rsync",
				},
				BlockNumber: targetBlock,
			},
		},
		ID: 1,
	}

	// Marshal request
	reqBody, err := json.Marshal(bundleReq)
	if err != nil {
		return fmt.Errorf("failed to marshal bundle request: %v", err)
	}

	// Get sequencer URL with API key
	sequencerURL := s.config.BaseSequencerRPCURL
	apiKey := os.Getenv("BASE_SEQUENCER_API_KEY")
	if apiKey != "" {
		sequencerURL += "?api_key=" + apiKey
	}

	// Submit to sequencer
	resp, err := http.Post(sequencerURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to submit bundle: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bundle submission failed with status %d", resp.StatusCode)
	}

	log.Printf("📦 Bundle submitted to Base sequencer successfully")
	log.Printf("   📊 Transactions: %d", len(rawTxs))
	log.Printf("   🎯 Target Block: %s", targetBlock)
	log.Printf("   🔗 Sequencer: %s", sequencerURL)

	return nil
}

// Stop stops the API service
func (s *Service) Stop() error {
	// Gracefully shutdown HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}
