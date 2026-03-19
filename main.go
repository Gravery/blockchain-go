// cmd/main.go
package main

import (
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/gravery/go-pos-blockchain/internal/consensus"
	"github.com/gravery/go-pos-blockchain/internal/core"
	"github.com/gravery/go-pos-blockchain/internal/wallet"
)

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	node, err := NewNode()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize node")
	}
	log.Info().Str("node_address", node.Wallet.GetAddress()).Msg("Node initialized successfully")

	go node.runBlockProduction(10 * time.Second)

	http.Handle("/blocks", withCORS(http.HandlerFunc(node.handleGetBlocks)))
	http.Handle("/transactions", withCORS(http.HandlerFunc(node.handleAddTransaction)))
	http.Handle("/wallets", withCORS(http.HandlerFunc(node.handleCreateWallet)))
	http.Handle("/wallets/", withCORS(http.HandlerFunc(node.handleGetBalance)))
	http.Handle("/validators", withCORS(http.HandlerFunc(node.handleGetValidators)))
	http.Handle("/validator/register", withCORS(http.HandlerFunc(node.handleRegisterValidator)))
	http.Handle("/validator/unregister", withCORS(http.HandlerFunc(node.handleUnregisterValidator)))
	http.Handle("/dashboard", withCORS(http.HandlerFunc(node.handleDashboard)))
	http.Handle("/health", withCORS(http.HandlerFunc(node.handleHealth)))
	http.Handle("/blocks/latest", withCORS(http.HandlerFunc(node.handleGetBlockByHash)))
	http.Handle("/blocks/", withCORS(http.HandlerFunc(node.handleGetBlockByHash)))
	http.Handle("/seed", withCORS(http.HandlerFunc(node.handleSeedData)))

	port := os.Getenv("BLOCKCHAIN_API_PORT")
	if port == "" {
		port = "8080"
	}
	log.Info().Str("port", port).Msg("Starting HTTP API server...")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal().Err(err).Msg("Failed to start HTTP API server")
	}
}

type Node struct {
	Blockchain *core.Blockchain
	Mempool    *core.Mempool
	Wallet     *wallet.Wallet
}

func (n *Node) handleSeedData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bc := n.Blockchain
	if bc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	wa, err := wallet.NewWallet()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	wb, err := wallet.NewWallet()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	bc.AddPublicKey(wa.GetAddress(), wa.PublicKey)
	bc.AddPublicKey(wb.GetAddress(), wb.PublicKey)
	bc.Balances[wa.GetAddress()] = 100
	bc.Balances[wb.GetAddress()] = 0
	bc.Nonces[wa.GetAddress()] = 0
	bc.Nonces[wb.GetAddress()] = 0

	additional, _ := wallet.NewWallet()
	bc.AddPublicKey(additional.GetAddress(), additional.PublicKey)
	bc.Balances[additional.GetAddress()] = int64(5 + mathrand.Intn(20))
	bc.Nonces[additional.GetAddress()] = 0
	seedTx := &core.Transaction{
		ID:      []byte("seed_tx"),
		From:    wa.GetAddress(),
		To:      additional.GetAddress(),
		Amount:  5,
		Nonce:   0,
		Fee:     1,
		Payload: []byte("seed transfer"),
	}
	sig, err := wa.Sign(seedTx.HashTx())
	if err == nil {
		seedTx.Sig = sig
		block := core.NewBlock([]*core.Transaction{seedTx}, bc.GetLastBlock().Hash, wa.GetAddress())
		_ = bc.AddBlock(block)
	}

	resp := map[string]interface{}{
		"walletA": map[string]string{
			"address":   wa.GetAddress(),
			"publicKey": fmt.Sprintf("0x%x", wa.PublicKey),
		},
		"walletB": map[string]string{
			"address":   wb.GetAddress(),
			"publicKey": fmt.Sprintf("0x%x", wb.PublicKey),
		},
		"balances": bc.Balances,
		"nonces":   bc.Nonces,
		"seedTransaction": map[string]interface{}{
			"id":      string(seedTx.ID),
			"from":    seedTx.From,
			"to":      seedTx.To,
			"amount":  seedTx.Amount,
			"nonce":   seedTx.Nonce,
			"fee":     seedTx.Fee,
			"payload": string(seedTx.Payload),
			"sig":     fmt.Sprintf("%x", seedTx.Sig),
		},
		"lastBlockHash": fmt.Sprintf("%x", bc.GetLastBlock().Hash),
		"blockHeight":   len(bc.Blocks),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (n *Node) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bc := n.Blockchain
	if bc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	last := bc.GetLastBlock()
	var lastHash string
	if last != nil {
		lastHash = fmt.Sprintf("%x", last.Hash)
	}
	recent := []map[string]interface{}{}
	max := 5
	for i := len(bc.Blocks) - 1; i >= 0 && len(recent) < max; i-- {
		b := bc.Blocks[i]
		for _, tx := range b.Transactions {
			recent = append(recent, map[string]interface{}{
				"id":      string(tx.ID),
				"from":    tx.From,
				"to":      tx.To,
				"amount":  tx.Amount,
				"nonce":   tx.Nonce,
				"payload": string(tx.Payload),
			})
		}
	}
	walletCount := len(bc.PubKeys)
	validatorCount := len(bc.ValidatorSet.GetValidators())

	payload := map[string]interface{}{
		"blockHeight":        len(bc.Blocks),
		"totalSupply":        bc.GetTotalSupply(),
		"lastBlockHash":      lastHash,
		"walletCount":        walletCount,
		"validatorCount":     validatorCount,
		"recentTransactions": recent,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (n *Node) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	status := map[string]interface{}{
		"uptime":      "unknown",
		"version":     "1.0.0",
		"blockHeight": len(n.Blockchain.Blocks),
		"status":      "ok",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (n *Node) handleGetBlockByHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid path"})
		return
	}
	hashOrKeyword := parts[len(parts)-1]
	if hashOrKeyword == "latest" {
		last := n.Blockchain.GetLastBlock()
		if last == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "no blocks"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(last)
		return
	}
	hashHex := hashOrKeyword
	for _, b := range n.Blockchain.Blocks {
		if fmt.Sprintf("%x", b.Hash) == hashHex {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(b)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "block not found"})
}

func (n *Node) handleAddRawTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var tx core.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := n.Mempool.Add(&tx); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "transaction added"})
}

func NewNode() (*Node, error) {
	dataDir := "./blockchain_data"

	bc := core.NewBlockchain(dataDir)

	nodeWallet, err := wallet.NewWallet()
	if err != nil {
		return nil, err
	}

	bc.AddValidator(nodeWallet.GetAddress(), 25)
	bc.AddPublicKey(nodeWallet.GetAddress(), nodeWallet.PublicKey)

	if bc != nil {
		bc.AddPublicKey(nodeWallet.GetAddress(), nodeWallet.PublicKey)
	}
	return &Node{
		Blockchain: bc,
		Mempool:    core.NewMempool(),
		Wallet:     nodeWallet,
	}, nil
}

func (n *Node) runBlockProduction(blockTime time.Duration) {
	log.Info().Str("interval", blockTime.String()).Msg("Starting block production")
	ticker := time.NewTicker(blockTime)
	defer ticker.Stop()

	for range ticker.C {
		lastBlock := n.Blockchain.GetLastBlock()
		if lastBlock == nil {
			log.Error().Msg("Blockchain is empty, skipping block production")
			continue
		}
		validatorAddress := consensus.SelectValidator(lastBlock, n.Blockchain.ValidatorSet)

		log.Info().Str("chosen_validator", validatorAddress).Msg("Validator selection for next round")

		if validatorAddress == n.Wallet.GetAddress() {
			log.Info().Msg("✅ Our node was chosen to forge the next block!")

			transactions := n.Mempool.GetPendingTransactions(10)
			if len(transactions) == 0 {
				log.Warn().Msg("No transactions in mempool, forging empty block.")
			}

			blockHeight := int64(len(n.Blockchain.Blocks))

			rewardTx := &core.Transaction{
				ID:      []byte(fmt.Sprintf("reward_%d", time.Now().UnixNano())),
				From:    "",
				To:      validatorAddress,
				Amount:  n.Blockchain.CalculateBlockReward(blockHeight),
				Fee:     0,
				Nonce:   0,
				Payload: []byte("block reward"),
			}

			rewardTxSlice := []*core.Transaction{rewardTx}
			allTransactions := append(transactions, rewardTxSlice...)

			newBlock := core.NewBlock(allTransactions, lastBlock.Hash, validatorAddress)

			n.Blockchain.AddBlock(newBlock)
			log.Info().Int("tx_count", len(transactions)).Str("hash", fmt.Sprintf("%x", newBlock.Hash)).Msg("New block forged and added to chain!")
		}
	}
}

func (n *Node) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(n.Blockchain.Blocks); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error().Err(err).Msg("Falha ao serializar blocos")
	}
}

func (n *Node) handleGetValidators(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	validators := n.Blockchain.ValidatorSet.GetValidators()
	if err := json.NewEncoder(w).Encode(validators); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error().Err(err).Msg("Falha ao serializar validadores")
	}
}

func (n *Node) handleRegisterValidator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var reqData struct {
		Address string `json:"address"`
		Stake   int64  `json:"stake"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error().Err(err).Msg("Falha ao decodificar corpo da requisição de registro de validador")
		return
	}

	if reqData.Address == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "endereço não pode estar vazio"})
		return
	}

	if reqData.Stake <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "stake deve ser maior que zero"})
		return
	}

	n.Blockchain.AddValidator(reqData.Address, reqData.Stake)

	log.Info().Str("address", reqData.Address).Int64("stake", reqData.Stake).Msg("Novo validador registrado")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "validator registered"})
}

func (n *Node) handleUnregisterValidator(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var reqData struct {
		Address string `json:"address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error().Err(err).Msg("Falha ao decodificar corpo da requisição de remoção de validador")
		return
	}

	if reqData.Address == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "address cannot be empty"})
		return
	}

	n.Blockchain.RemoveValidator(reqData.Address)

	log.Info().Str("address", reqData.Address).Msg("Validator removed")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "validator unregistered"})
}

func (n *Node) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		newWallet, err := wallet.NewWallet()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Msg("Falha ao criar nova carteira")
			return
		}

		address := newWallet.GetAddress()
		if n.Blockchain != nil {
			n.Blockchain.AddPublicKey(address, newWallet.PublicKey)
		}
		log.Info().Str("address", address).Msg("Nova carteira criada")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"address": address,
			"status":  "wallet created",
		})
	case http.MethodGet:
		if n.Blockchain == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		entries := []map[string]interface{}{}
		for addr, pub := range n.Blockchain.PubKeys {
			bal := n.Blockchain.Balances[addr]
			nonce := n.Blockchain.Nonces[addr]
			entries = append(entries, map[string]interface{}{
				"address":   addr,
				"balance":   bal,
				"nonce":     nonce,
				"publicKey": fmt.Sprintf("0x%x", pub),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"wallets": entries})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (n *Node) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("handleGetBalance called")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[2] == "" || parts[3] != "balance" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid URL format, expected /wallets/{address}/balance"})
		return
	}
	address := parts[2]

	balance := n.Blockchain.Balances[address]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"address": address,
		"balance": balance,
		"status":  "success",
	})
}

func (n *Node) handleAddTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var txData struct {
		To      string `json:"to"`
		Amount  int64  `json:"amount"`
		Payload string `json:"payload,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&txData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error().Err(err).Msg("Failed to decode transaction request body")
		return
	}

	nonce := n.Blockchain.GetNextNonce(n.Wallet.GetAddress())
	tx := &core.Transaction{
		ID:      []byte(fmt.Sprintf("tx_%d", time.Now().UnixNano())),
		From:    n.Wallet.GetAddress(),
		To:      txData.To,
		Amount:  txData.Amount,
		Fee:     1,
		Nonce:   nonce,
		Payload: []byte(txData.Payload),
	}

	signature, err := n.Wallet.Sign(tx.HashTx())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to sign transaction")
		return
	}
	tx.Sig = signature

	n.Mempool.Add(tx)

	log.Info().Str("tx_id", string(tx.ID)).Msg("New transaction added to mempool")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "transaction added"})
}
