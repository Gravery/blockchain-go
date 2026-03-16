package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() (*Wallet, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	return &Wallet{PrivateKey: *privateKey, PublicKey: publicKey}, nil
}

func (w *Wallet) GetAddress() string {
	pubKeyHash := sha3.New256()
	pubKeyHash.Write(w.PublicKey[1:])
	hashedKey := pubKeyHash.Sum(nil)
	address := hashedKey[len(hashedKey)-20:]
	return fmt.Sprintf("0x%x", address)
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	signature, err := ecdsa.SignASN1(rand.Reader, &w.PrivateKey, data)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func VerifySignature(publicKey, data, signature []byte) bool {
	x := new(big.Int)
	y := new(big.Int)
	keyLen := len(publicKey)
	x.SetBytes(publicKey[:(keyLen / 2)])
	y.SetBytes(publicKey[(keyLen / 2):])

	rawPubKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

	return ecdsa.VerifyASN1(&rawPubKey, data, signature)
}
