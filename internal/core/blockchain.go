package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/gravery/go-pos-blockchain/internal/wallet"
)

type ValidatorSet struct {
	Validators map[string]int64
	mu         sync.RWMutex `json:"-"`
}

func (vs *ValidatorSet) GetValidators() map[string]int64 {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	validatorsCopy := make(map[string]int64)
	for addr, stake := range vs.Validators {
		validatorsCopy[addr] = stake
	}
	return validatorsCopy
}

type Blockchain struct {
	Blocks       []*Block
	ValidatorSet *ValidatorSet
	mu           sync.RWMutex `json:"-"`

	dataDir     string
	TotalSupply int64

	blockReward     int64
	halvingInterval int64
	maxSupply       int64

	Nonces   map[string]uint64
	Balances map[string]int64
	PubKeys  map[string][]byte
}

func NewBlockchain(dataDir string) *Blockchain {
	blockReward := int64(50)
	halvingInterval := int64(210000)
	maxSupply := int64(21000000)

	validators := make(map[string]int64)
	validators["validator_address_1"] = 100
	validators["validator_address_2"] = 50
	validators["validator_address_3"] = 75

	bc := &Blockchain{
		ValidatorSet:    &ValidatorSet{Validators: validators},
		dataDir:         dataDir,
		blockReward:     blockReward,
		halvingInterval: halvingInterval,
		maxSupply:       maxSupply,
		TotalSupply:     0,
		Nonces:          make(map[string]uint64),
		Balances:        make(map[string]int64),
		PubKeys:         make(map[string][]byte),
	}

	if err := bc.LoadFromDisk(); err != nil {
		genesis := NewGenesisBlock()
		if err := bc.updateStateWithBlock(genesis); err != nil {
			panic(fmt.Sprintf("Failed to process genesis block: %v", err))
		}
		bc.Blocks = []*Block{genesis}
	} else {
		if len(bc.Blocks) == 0 {
			genesis := NewGenesisBlock()
			if err := bc.updateStateWithBlock(genesis); err != nil {
				panic(fmt.Sprintf("Failed to process genesis block: %v", err))
			}
			bc.Blocks = []*Block{genesis}
		}
	}

	return bc
}

func (bc *Blockchain) AddPublicKey(address string, pubKey []byte) {
	if bc == nil {
		return
	}
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if bc.PubKeys == nil {
		bc.PubKeys = make(map[string][]byte)
	}
	bc.PubKeys[address] = pubKey
}

func (bc *Blockchain) AddBlock(block *Block) error {
	if err := bc.updateStateWithBlock(block); err != nil {
		return err
	}

	bc.mu.Lock()
	bc.Blocks = append(bc.Blocks, block)
	bc.mu.Unlock()

	if err := bc.SaveToDisk(); err != nil {
		return err
	}
	return nil
}

func (bc *Blockchain) updateStateWithBlock(block *Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	var newlyCreatedCoins int64

	for _, tx := range block.Transactions {
		if tx.From == "" {
			if tx.Amount <= 0 {
				return fmt.Errorf("system transaction invalid: amount must be positive")
			}
			newlyCreatedCoins += tx.Amount
			bc.Balances[tx.To] += tx.Amount
		} else {

			if len(tx.Sig) == 0 {
				return fmt.Errorf("transaction invalid: missing signature")
			}
			pubKey, ok := bc.PubKeys[tx.From]
			if !ok {
				return fmt.Errorf("unknown sender address: %s", tx.From)
			}
			if !wallet.VerifySignature(pubKey, tx.HashTx(), tx.Sig) {
				return fmt.Errorf("invalid signature for address %s", tx.From)
			}

			if tx.Amount < 0 {
				return fmt.Errorf("transaction invalid: negative amount")
			}
			totalRequired := tx.Amount + tx.Fee
			if bc.Balances[tx.From] < totalRequired {
				return fmt.Errorf("insufficient balance for address %s: have %d, need %d", tx.From, bc.Balances[tx.From], totalRequired)
			}
			bc.Balances[tx.From] -= totalRequired
			bc.Balances[tx.To] += tx.Amount

			expectedNonce := bc.Nonces[tx.From]
			if tx.Nonce != expectedNonce {
				return fmt.Errorf("invalid nonce for address %s: expected %d, got %d", tx.From, expectedNonce, tx.Nonce)
			}
			bc.Nonces[tx.From] = expectedNonce + 1
		}
	}

	bc.TotalSupply += newlyCreatedCoins
	return nil
}

func (bc *Blockchain) CalculateBlockReward(height int64) int64 {
	halvings := height / bc.halvingInterval

	if halvings >= 64 {
		return 0
	}

	reward := bc.blockReward >> halvings
	return reward
}

func (bc *Blockchain) Slash(address string, percentage float64) {
	if bc.ValidatorSet == nil {
		return
	}

	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}

	bc.ValidatorSet.mu.Lock()
	defer bc.ValidatorSet.mu.Unlock()

	if stake, exists := bc.ValidatorSet.Validators[address]; exists {
		slashAmount := int64(float64(stake) * percentage / 100)

		if slashAmount >= stake {
			delete(bc.ValidatorSet.Validators, address)
		} else {
			newStake := stake - slashAmount
			if newStake == 0 {
				delete(bc.ValidatorSet.Validators, address)
			} else {
				bc.ValidatorSet.Validators[address] = newStake
			}
		}

	}
}

func (bc *Blockchain) UpdateValidatorStake(address string, stake int64) {
	if bc.ValidatorSet == nil {
		return
	}

	bc.ValidatorSet.mu.Lock()
	defer bc.ValidatorSet.mu.Unlock()

	if stake < 0 {
		stake = 0
	}

	if stake == 0 {
		delete(bc.ValidatorSet.Validators, address)
	} else {
		bc.ValidatorSet.Validators[address] = stake
	}
}

func (bc *Blockchain) AddValidator(address string, stake int64) {
	if bc.ValidatorSet == nil {
		return
	}

	bc.ValidatorSet.mu.Lock()
	defer bc.ValidatorSet.mu.Unlock()

	if stake > 0 {
		bc.ValidatorSet.Validators[address] = stake
	}
}

func (bc *Blockchain) RemoveValidator(address string) {
	if bc.ValidatorSet == nil {
		return
	}

	bc.ValidatorSet.mu.Lock()
	defer bc.ValidatorSet.mu.Unlock()

	delete(bc.ValidatorSet.Validators, address)
}

func (bc *Blockchain) GetLastBlock() *Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) GetTotalSupply() int64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.TotalSupply
}

func (bc *Blockchain) GetNextNonce(address string) uint64 {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	nonce := bc.Nonces[address]
	bc.Nonces[address]++
	return nonce
}

func (bc *Blockchain) SaveToDisk() error {
	if bc.dataDir == "" {
		return nil
	}

	if err := os.MkdirAll(bc.dataDir, 0755); err != nil {
		return err
	}

	blockchainFile := filepath.Join(bc.dataDir, "blockchain.json")
	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(blockchainFile, data, 0644)
}

func (bc *Blockchain) LoadFromDisk() error {
	if bc.dataDir == "" {
		return nil
	}

	blockchainFile := filepath.Join(bc.dataDir, "blockchain.json")

	if _, err := os.Stat(blockchainFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(blockchainFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, bc); err != nil {
		return err
	}

	return nil
}
