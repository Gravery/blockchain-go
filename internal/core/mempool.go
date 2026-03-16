package core

import (
	"errors"
	"sync"
)

// Mempool is a pool for pending transactions.
type Mempool struct {
	Transactions []*Transaction
	mu           sync.RWMutex
}

func NewMempool() *Mempool {
	return &Mempool{
		Transactions: []*Transaction{},
	}
}

func (m *Mempool) Add(tx *Transaction) error {
	if err := ValidateTransaction(tx); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.Transactions = append(m.Transactions, tx)
	return nil
}

func (m *Mempool) GetPendingTransactions(maxTransactions int) []*Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := len(m.Transactions)
	if count == 0 {
		return []*Transaction{}
	}

	if count > maxTransactions {
		count = maxTransactions
	}

	pending := m.Transactions[:count]
	m.Transactions = m.Transactions[count:]

	return pending
}

func ValidateTransaction(tx *Transaction) error {
	if tx.From == "" {
		if tx.To == "" {
			return errors.New("system transaction invalid: recipient empty")
		}
		if tx.Amount <= 0 {
			return errors.New("system transaction invalid: amount must be positive")
		}
		if tx.Fee != 0 {
			return errors.New("system transaction invalid: fee must be zero")
		}
		if len(tx.Sig) == 0 {
		}
		return nil
	}

	if tx.To == "" {
		return errors.New("transaction invalid: recipient empty")
	}
	if tx.Amount < 0 {
		return errors.New("transaction invalid: negative amount")
	}
	if tx.Fee < 0 {
		return errors.New("transaction invalid: negative fee")
	}
	if len(tx.Sig) == 0 {
		return errors.New("transaction invalid: missing signature")
	}
	if len(tx.ID) == 0 {
		return errors.New("transaction invalid: missing ID")
	}

	return nil
}
