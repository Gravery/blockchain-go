package consensus

import (
	"testing"

	"github.com/gravery/go-pos-blockchain/internal/core"
)

func TestSelectValidatorIsDeterministic(t *testing.T) {
	validators := make(map[string]int64)
	validators["addr1"] = 10
	validators["addr2"] = 20
	validatorSet := &core.ValidatorSet{Validators: validators}

	lastBlock := core.NewGenesisBlock()

	winner1 := SelectValidator(lastBlock, validatorSet)
	winner2 := SelectValidator(lastBlock, validatorSet)

	if winner1 == "" {
		t.Fatal("A seleção do validador não deveria retornar vazio")
	}

	if winner1 != winner2 {
		t.Fatalf("A seleção do validador não é determinística! Vencedor 1: %s, Vencedor 2: %s", winner1, winner2)
	}

	t.Logf("Vencedor selecionado deterministicamente: %s", winner1)
}
