package core

import (
	"os"
	"testing"

	"github.com/gravery/go-pos-blockchain/internal/wallet"
)

// Minimal, deterministic tests for core blockchain behavior

func TestBlockchainBasicFunctionality(t *testing.T) {
	dir, err := os.MkdirTemp("", "blockchain_basic_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	bc := NewBlockchain(dir)
	if len(bc.Blocks) == 0 {
		t.Fatal("Genesis block not created")
	}
	// Genesis should mint 50 to genesis address (0x0..0)
	zero := "0x0000000000000000000000000000000000000000"
	if bc.Balances[zero] != 50 {
		t.Fatalf("Genesis balance expected 50, got %d", bc.Balances[zero])
	}
	if bc.GetTotalSupply() != 50 {
		t.Fatalf("Initial total supply should be 50, got %d", bc.GetTotalSupply())
	}
}

func TestBlockchainStateUpdates(t *testing.T) {
	dir, err := os.MkdirTemp("", "blockchain_state_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	bc := NewBlockchain(dir)

	// Create two wallets
	w1, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("wallet1 error: %v", err)
	}
	w2, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("wallet2 error: %v", err)
	}

	addr1 := w1.GetAddress()
	addr2 := w2.GetAddress()
	// Register public keys for the test wallets to enable signature verification in tests
	bc.AddPublicKey(addr1, w1.PublicKey)
	bc.AddPublicKey(addr2, w2.PublicKey)
	// Seed balances for testing
	bc.Balances[addr1] = 100
	bc.Balances[addr2] = 0
	bc.Nonces[addr1] = 0
	bc.Nonces[addr2] = 0

	// Create and sign a transfer from addr1 to addr2
	tx := &Transaction{
		ID:      []byte("tx1"),
		From:    addr1,
		To:      addr2,
		Amount:  10,
		Fee:     1,
		Nonce:   0,
		Payload: []byte("transfer"),
	}
	sig, err := w1.Sign(tx.HashTx())
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	tx.Sig = sig

	block := NewBlock([]*Transaction{tx}, bc.GetLastBlock().Hash, addr1)
	if err := bc.AddBlock(block); err != nil {
		t.Fatalf("failed to add block: %v", err)
	}

	// Validate state changes
	if bc.Balances[addr1] != 100-11 {
		t.Fatalf("addr1 balance incorrect: got %d want %d", bc.Balances[addr1], 100-11)
	}
	if bc.Balances[addr2] != 10 {
		t.Fatalf("addr2 balance incorrect: got %d want %d", bc.Balances[addr2], 10)
	}
	if bc.Nonces[addr1] != 1 {
		t.Fatalf("addr1 nonce should be 1, got %d", bc.Nonces[addr1])
	}
}

func TestBlockchainPersistency(t *testing.T) {
	dir, err := os.MkdirTemp("", "blockchain_persist_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	bc1 := NewBlockchain(dir)
	if err := bc1.SaveToDisk(); err != nil {
		t.Fatalf("save error: %v", err)
	}

	bc2 := NewBlockchain(dir)
	if bc1.GetTotalSupply() != bc2.GetTotalSupply() {
		t.Fatalf("supply mismatch: %d != %d", bc1.GetTotalSupply(), bc2.GetTotalSupply())
	}
	if len(bc1.Blocks) != len(bc2.Blocks) {
		t.Fatalf("block count mismatch after load: %d != %d", len(bc1.Blocks), len(bc2.Blocks))
	}
}

func TestCalculateBlockReward(t *testing.T) {
	// reuse existing logic from previous tests for consistency
	dir := "./tmp"
	bc := NewBlockchain(dir)
	if bc.CalculateBlockReward(0) != 50 {
		t.Fatalf("expected 50 at height 0, got %d", bc.CalculateBlockReward(0))
	}
}

func TestTransactionSigning(t *testing.T) {
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("wallet error: %v", err)
	}
	tx := &Transaction{ID: []byte("tx1"), From: w.GetAddress(), To: "0x1111111111111111111111111111111111111111", Amount: 1, Nonce: 0, Fee: 1, Payload: []byte("ptx"), Sig: nil}
	sig, err := w.Sign(tx.HashTx())
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	tx.Sig = sig
	if len(tx.Sig) == 0 {
		t.Fatalf("signature missing")
	}
}
