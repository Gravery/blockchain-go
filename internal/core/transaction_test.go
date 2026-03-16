package core

import (
	"testing"

	"github.com/gravery/go-pos-blockchain/internal/wallet"
)

func TestTransactionCreation(t *testing.T) {
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	tx := &Transaction{
		ID:      []byte("test_tx_1"),
		From:    w.GetAddress(),
		To:      "0x742d35Cc6634C0532925a3b8D4C0532950532950",
		Amount:  100,
		Fee:     1,
		Nonce:   1,
		Payload: []byte("Test transaction"),
	}

	signature, err := w.Sign(tx.HashTx())
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}
	tx.Sig = signature

	if err := ValidateTransaction(tx); err != nil {
		t.Fatalf("Invalid transaction: %v", err)
	}

	if tx.From != w.GetAddress() {
		t.Errorf("Sender address incorrect. Expected: %s, Got: %s", w.GetAddress(), tx.From)
	}
	if tx.To != "0x742d35Cc6634C0532925a3b8D4C0532950532950" {
		t.Errorf("Recipient address incorrect")
	}
	if tx.Amount != 100 {
		t.Errorf("Amount incorrect. Expected: 100, Got: %d", tx.Amount)
	}
	if tx.Fee != 1 {
		t.Errorf("Fee incorrect. Expected: 1, Got: %d", tx.Fee)
	}
	if tx.Nonce != 1 {
		t.Errorf("Nonce incorrect. Expected: 1, Got: %d", tx.Nonce)
	}
	if string(tx.Payload) != "Test transaction" {
		t.Errorf("Payload incorrect")
	}
	if len(tx.Sig) == 0 {
		t.Errorf("Signature was not generated")
	}

	t.Logf("Transaction created and signed successfully. ID: %x", tx.ID)
}
