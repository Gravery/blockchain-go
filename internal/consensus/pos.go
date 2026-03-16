package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"sort"

	"github.com/gravery/go-pos-blockchain/internal/core"
)

func SelectValidator(lastBlock *core.Block, validatorSet *core.ValidatorSet) string {
	currentValidators := validatorSet.GetValidators()

	if len(currentValidators) == 0 {
		return ""
	}

	h := sha256.Sum256(lastBlock.Hash)
	seed := int64(binary.BigEndian.Uint64(h[:8]))
	r := rand.New(rand.NewSource(seed))

	var weightedValidators []string
	var totalStake int64 = 0

	var addresses []string
	for addr := range currentValidators {
		addresses = append(addresses, addr)
	}
	sort.Strings(addresses)

	for _, addr := range addresses {
		stake := currentValidators[addr]
		if stake <= 0 {
			continue
		}
		totalStake += stake
		for i := int64(0); i < stake; i++ {
			weightedValidators = append(weightedValidators, addr)
		}
	}

	if totalStake == 0 {
		return ""
	}

	winnerIndex := r.Intn(len(weightedValidators))
	return weightedValidators[winnerIndex]
}
