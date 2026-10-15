package random_test

import (
	"testing"

	ds "github.com/sealdice/dicescript"

	randcore "sealdice-core/utils/random"
)

type fixedDiceSource struct {
	value uint64
}

func (s *fixedDiceSource) Uint64() uint64 {
	return s.value
}

func TestGlobalOwnerSetActiveFallsBackToAnyAvailableSource(t *testing.T) {
	owner := randcore.NewEmptyGlobalOwner()
	owner.RegisterSource(randcore.ModeGM, &fixedDiceSource{value: 7})

	mode, err := owner.SetActive(randcore.ModeNIST)
	if err == nil {
		t.Fatal("expected SetActive to report fallback error")
	}
	if mode != randcore.ModeGM {
		t.Fatalf("SetActive(nist) fallback mode = %s, want gm", mode)
	}
	if got := owner.Uint64(); got != 7 {
		t.Fatalf("Uint64() = %d, want 7", got)
	}
}

var _ ds.DiceSource = (*fixedDiceSource)(nil)
