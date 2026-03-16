// cmd/main.go
package main

import (
	"encoding/json"
	"fmt"
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

	http.HandleFunc("/blocks", node.handleGetBlocks)
	http.HandleFunc("/transactions", node.handleAddTransaction)
	http.HandleFunc("/wallets", node.handleCreateWallet)
	http.HandleFunc("/wallets/", node.handleGetBalance)

	port := "8080"
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

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
