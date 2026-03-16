package core

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

type Transaction struct {
	ID      []byte
	From    string
	To      string
	Amount  int64
	Nonce   uint64
	Fee     int64
	Payload []byte
	Sig     []byte
}

func (tx *Transaction) HashTx() []byte {
	txCopy := Transaction{
		ID:      tx.ID,
		From:    tx.From,
		To:      tx.To,
		Amount:  tx.Amount,
		Nonce:   tx.Nonce,
		Fee:     tx.Fee,
		Payload: tx.Payload,
		// Sig is left zero
	}

	var buf bytes.Buffer
	buf.Write(txCopy.ID)
	buf.WriteString(txCopy.From)
	buf.WriteString(txCopy.To)
	buf.Write([]byte(strconv.FormatInt(txCopy.Amount, 10)))
	buf.Write([]byte(strconv.FormatUint(txCopy.Nonce, 10)))
	buf.Write([]byte(strconv.FormatInt(txCopy.Fee, 10)))
	buf.Write(txCopy.Payload)

	hash := sha256.Sum256(buf.Bytes())
	return hash[:]
}

func (tx *Transaction) Serialize() []byte {
	var buf bytes.Buffer
	buf.Write(tx.ID)
	buf.WriteString(tx.From)
	buf.WriteString(tx.To)
	buf.Write([]byte(strconv.FormatInt(tx.Amount, 10)))
	buf.Write([]byte(strconv.FormatUint(tx.Nonce, 10)))
	buf.Write([]byte(strconv.FormatInt(tx.Fee, 10)))
	buf.Write(tx.Payload)
	buf.Write(tx.Sig)
	return buf.Bytes()
}

func DeserializeTransaction(data []byte) (*Transaction, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("transaction data too short")
	}

	id := data[0:32]
	offset := 32

	fromEnd := bytes.IndexByte(data[offset:], 0)
	if fromEnd == -1 {
		return nil, fmt.Errorf("invalid From field")
	}
	from := string(data[offset : offset+fromEnd])
	offset += fromEnd + 1

	toEnd := bytes.IndexByte(data[offset:], 0)
	if toEnd == -1 {
		return nil, fmt.Errorf("invalid To field")
	}
	to := string(data[offset : offset+toEnd])
	offset += toEnd + 1

	if offset+8+8+8 > len(data) {
		return nil, fmt.Errorf("not enough data for amounts")
	}
	amount := int64FromBytes(data[offset : offset+8])
	offset += 8
	nonce := uint64FromBytes(data[offset : offset+8])
	offset += 8
	fee := int64FromBytes(data[offset : offset+8])
	offset += 8

	payloadEnd := bytes.IndexByte(data[offset:], 0)
	if payloadEnd == -1 {
		return nil, fmt.Errorf("invalid Payload field")
	}
	payload := data[offset : offset+payloadEnd]
	offset += payloadEnd + 1

	sig := data[offset:]

	return &Transaction{
		ID:      id,
		From:    from,
		To:      to,
		Amount:  amount,
		Nonce:   nonce,
		Fee:     fee,
		Payload: payload,
		Sig:     sig,
	}, nil
}

func int64FromBytes(b []byte) int64 {
	if len(b) != 8 {
		return 0
	}
	return int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
		int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
}

func uint64FromBytes(b []byte) uint64 {
	if len(b) != 8 {
		return 0
	}
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Validator     string
}

func (b *Block) CalculateHash() []byte {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.serializeTransactions(), timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	return hash[:]
}

func (b *Block) serializeTransactions() []byte {
	var transactions [][]byte
	for _, tx := range b.Transactions {
		txData := []byte{}
		txData = append(txData, tx.From...)
		txData = append(txData, tx.To...)
		txData = append(txData, []byte(strconv.FormatInt(tx.Amount, 10))...)
		txData = append(txData, []byte(strconv.FormatUint(tx.Nonce, 10))...)
		txData = append(txData, []byte(strconv.FormatInt(tx.Fee, 10))...)
		txData = append(txData, tx.Payload...)
		txData = append(txData, tx.Sig...)
		transactions = append(transactions, txData)
	}
	return bytes.Join(transactions, []byte{})
}

func NewBlock(transactions []*Transaction, prevBlockHash []byte, validator string) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
		Validator:     validator,
	}
	block.Hash = block.CalculateHash()
	return block
}

func NewGenesisBlock() *Block {
	tx := &Transaction{ID: []byte("genesis"), From: "", To: "0x0000000000000000000000000000000000000000", Amount: 50, Nonce: 0, Fee: 0, Payload: []byte("Genesis Transaction")}
	return NewBlock([]*Transaction{tx}, []byte{}, "genesis")
}
