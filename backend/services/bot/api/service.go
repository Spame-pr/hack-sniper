package api

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	rpcUrl := cfg.BaseRPCURL

	ethClient, err := eth.NewClient(rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create eth client: %v", err)
	}

	// Initialize bundle manager with sniper contract address
	sniperContractAddr := common.HexToAddress(cfg.SniperContract)

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
	bundleTxs, err := s.createBundleTransactions(ctx, bundleBids, notification)
	if err != nil {
		log.Printf("❌ Failed to create bundle transactions: %v", err)
		return
	}

	log.Printf("📦 Created bundle with %d transactions (1 LP_ADD + %d snipes)", len(bundleTxs)+1, len(bundleBids))

	// Submit bundle to Base sequencer
	s.submitBundle(ctx, notification.TxCallData, bundleTxs)

	// Update snipe statuses to 'submitted'
	for _, snipe := range snipes {
		if err := s.db.UpdateSnipeStatus(snipe.ID, "submitted"); err != nil {
			log.Printf("⚠️ Failed to update snipe status for ID %d: %v", snipe.ID, err)
		}
	}

	log.Printf("✅ Bundle submitted successfully for token %s with %d snipes", notification.TokenAddress, len(snipes))
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
func (s *Service) createBundleTransactions(ctx context.Context, bids []*bundle.SnipeBid, notification LPAddNotification) ([]*types.Transaction, error) {
	var transactions []*types.Transaction

	// Get base fee for EIP-1559 transactions
	latestBlock, err := s.ethClient.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %v", err)
	}

	baseFee := latestBlock.BaseFee
	if baseFee == nil {
		// Fallback to legacy gas price if base fee not available
		legacyGasPrice, err := s.ethClient.Client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get gas price: %v", err)
		}
		baseFee = legacyGasPrice
	}

	// Set initial max priority fee per gas (tip to miners/validators)
	maxPriorityFeePerGas := big.NewInt(2000000) // 2 gwei tip

	// Calculate initial max fee per gas = base fee + priority fee + buffer
	buffer := big.NewInt(1000000) // 1 gwei buffer
	initialMaxFeePerGas := new(big.Int).Add(baseFee, maxPriorityFeePerGas)
	initialMaxFeePerGas.Add(initialMaxFeePerGas, buffer)

	// Cap max fee at reasonable level for Base network (20 gwei)
	maxGasPriceGwei := big.NewInt(20)
	maxGasPrice := new(big.Int).Mul(maxGasPriceGwei, big.NewInt(1e9))
	if initialMaxFeePerGas.Cmp(maxGasPrice) > 0 {
		initialMaxFeePerGas = maxGasPrice
	}

	// Debug gas price information
	baseFeeGwei := new(big.Float).Quo(new(big.Float).SetInt(baseFee), big.NewFloat(1e9))
	maxFeeGwei := new(big.Float).Quo(new(big.Float).SetInt(initialMaxFeePerGas), big.NewFloat(1e9))
	priorityFeeGwei := new(big.Float).Quo(new(big.Float).SetInt(maxPriorityFeePerGas), big.NewFloat(1e9))

	fmt.Printf("💰 EIP-1559 Gas Price Debug:\n")
	fmt.Printf("   Base Fee: %s wei (%s gwei)\n", baseFee.String(), baseFeeGwei.Text('f', 2))
	fmt.Printf("   Initial Max Fee: %s wei (%s gwei)\n", initialMaxFeePerGas.String(), maxFeeGwei.Text('f', 2))
	fmt.Printf("   Priority Fee: %s wei (%s gwei)\n", maxPriorityFeePerGas.String(), priorityFeeGwei.Text('f', 2))

	// Create snipe transactions with decreasing max fee per gas (sorted by bribe size)
	for i, bid := range bids {
		// Calculate max fee per gas: each subsequent tx has maxFeePerGas = previous - 1 wei
		// This ensures strict ordering based on bribe size for Base sequencer
		maxFeePerGas := new(big.Int).Sub(initialMaxFeePerGas, big.NewInt(int64(i)))

		// Ensure minimum fee (at least base fee + priority fee)
		minMaxFee := new(big.Int).Add(baseFee, maxPriorityFeePerGas)
		if maxFeePerGas.Cmp(minMaxFee) < 0 {
			maxFeePerGas = minMaxFee
		}

		// Debug gas price for this transaction
		maxFeeGwei := new(big.Float).Quo(new(big.Float).SetInt(maxFeePerGas), big.NewFloat(1e9))
		bribeETH := new(big.Float).Quo(new(big.Float).SetInt(bid.BribeAmount), big.NewFloat(1e18))
		fmt.Printf("   Tx %d (Bribe: %s ETH) Max Fee: %s gwei\n", i+1, bribeETH.Text('f', 4), maxFeeGwei.Text('f', 2))

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

		// Create snipe transaction call data
		snipeTx, err := sniperContract.CreateSnipeTransaction(
			ctx,
			bid.Wallet,
			bid.TokenAddress,
			creatorAddr,
			bid.SwapAmount,
			bid.BribeAmount,
			amountOutMin,
			deadline,
			maxFeePerGas, // Pass maxFeePerGas instead of legacy gasPrice
			nonce,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create snipe transaction for %s: %v", bid.Wallet.Hex(), err)
		}

		// Create EIP-1559 transaction (v2)
		dynamicTx := &types.DynamicFeeTx{
			ChainID:   big.NewInt(8453), // Base mainnet chain ID
			Nonce:     nonce,
			GasTipCap: maxPriorityFeePerGas,
			GasFeeCap: maxFeePerGas,
			Gas:       snipeTx.Gas(),
			To:        snipeTx.To(),
			Value:     snipeTx.Value(),
			Data:      snipeTx.Data(),
		}

		// Convert to Transaction type
		eip1559Tx := types.NewTx(dynamicTx)

		// Sign the transaction with the user's private key
		privateKeyHex := bid.PrivateKey
		if privateKeyHex == "" {
			return nil, fmt.Errorf("private key not found for wallet %s", bid.Wallet.Hex())
		}

		// Remove 0x prefix if present
		if strings.HasPrefix(privateKeyHex, "0x") {
			privateKeyHex = privateKeyHex[2:]
		}

		privateKeyBytes, err := hex.DecodeString(privateKeyHex)
		if err != nil {
			return nil, fmt.Errorf("failed to decode private key for %s: %v", bid.Wallet.Hex(), err)
		}

		privateKey, err := crypto.ToECDSA(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key for %s: %v", bid.Wallet.Hex(), err)
		}

		// Get chain ID for signing
		chainID, err := s.ethClient.Client.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get chain ID: %v", err)
		}

		// Sign EIP-1559 transaction with London signer
		signedTx, err := types.SignTx(eip1559Tx, types.NewLondonSigner(chainID), privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign EIP-1559 transaction for %s: %v", bid.Wallet.Hex(), err)
		}

		log.Printf("✅ EIP-1559 transaction signed for wallet %s (Bribe: %s ETH)",
			bid.Wallet.Hex()[:10]+"...",
			bribeETH.Text('f', 4))

		transactions = append(transactions, signedTx)
	}

	log.Printf("📦 Created %d EIP-1559 transactions sorted by bribe size (highest to lowest)", len(transactions))
	return transactions, nil
}

func (s *Service) submitBundle(ctx context.Context, addLiqRawTx string, transactions []*types.Transaction) {
	err := s.submitTx(ctx, addLiqRawTx)
	if err != nil {
		log.Printf("failed to submit add liq transaction: %v", err)
	}

	for _, tx := range transactions {
		// Convert transaction to raw hex string
		rawTx, err := tx.MarshalBinary()
		if err != nil {
			log.Printf("failed to submit transaction: %v; hash: %s", err, tx.Hash().Hex())
			continue
		}
		rawTxHex := "0x" + hex.EncodeToString(rawTx)
		err = s.submitTx(ctx, rawTxHex)
		if err != nil {
			log.Printf("failed to submit transaction: %v; hash: %s", err, tx.Hash().Hex())
		}
	}
}

func (s *Service) submitTx(ctx context.Context, rawTxHex string) error {

	// Create eth_sendRawTransaction request
	type RawTxRequest struct {
		JSONRPC string   `json:"jsonrpc"`
		Method  string   `json:"method"`
		Params  []string `json:"params"`
		ID      int      `json:"id"`
	}

	txReq := RawTxRequest{
		JSONRPC: "2.0",
		Method:  "eth_sendRawTransaction",
		Params:  []string{rawTxHex},
		ID:      1,
	}

	// Marshal request
	reqBody, err := json.Marshal(txReq)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction request: %v", err)
	}

	resp, err := http.Post(s.config.BaseSequencerRPCURL, "application/json", bytes.NewBuffer(reqBody))

	if err != nil {
		return fmt.Errorf("failed to submit transaction: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("transaction submission failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response to get transaction hash
	type RawTxResponse struct {
		JSONRPC string      `json:"jsonrpc"`
		Result  string      `json:"result"`
		Error   interface{} `json:"error"`
		ID      int         `json:"id"`
	}

	var txResp RawTxResponse
	if err := json.Unmarshal(respBody, &txResp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if txResp.Error != nil {
		return fmt.Errorf("transaction failed: %v", txResp.Error)
	}

	log.Printf("Transaction submitted successfully; Hash: %s", txResp.Result)

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
