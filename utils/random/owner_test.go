package random_test

import (
	"errors"
	"strings"
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

func TestGlobalOwnerGMDisabledByDefault(t *testing.T) {
	owner := randcore.NewEmptyGlobalOwner()
	owner.RegisterSource(randcore.ModePCG, &fixedDiceSource{value: 1})

	if owner.GMEnabled() {
		t.Fatal("expected GM source to be disabled by default")
	}
	if err := owner.InitError(randcore.ModeGM); !errors.Is(err, randcore.ErrGMDisabled) {
		t.Fatalf("InitError(gm) = %v, want ErrGMDisabled", err)
	}
	if got := owner.ReportGetText(randcore.ModeGM, 20); !strings.Contains(got, "未启用") {
		t.Fatalf("expected gm disabled hint, got %q", got)
	}
}

func TestNewGlobalOwnerSkipsGMByDefault(t *testing.T) {
	owner := randcore.NewGlobalOwner(nil)
	if owner.GMEnabled() {
		t.Fatal("expected new global owner to skip GM source")
	}
	if err := owner.InitError(randcore.ModeGM); !errors.Is(err, randcore.ErrGMDisabled) {
		t.Fatalf("InitError(gm) = %v, want ErrGMDisabled", err)
	}
}

func TestGlobalOwnerRegisterGMSourceEnablesGM(t *testing.T) {
	owner := randcore.NewEmptyGlobalOwner()
	owner.RegisterSource(randcore.ModePCG, &fixedDiceSource{value: 1})
	owner.RegisterSource(randcore.ModeGM, &fixedDiceSource{value: 2})

	if !owner.GMEnabled() {
		t.Fatal("expected GM source to be enabled after RegisterSource")
	}
	if err := owner.InitError(randcore.ModeGM); err != nil {
		t.Fatalf("InitError(gm) = %v, want nil", err)
	}
}

var _ ds.DiceSource = (*fixedDiceSource)(nil)
