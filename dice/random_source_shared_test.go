package dice //nolint:testpackage

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	ds "github.com/sealdice/dicescript"

	randcore "sealdice-core/utils/random"
)

type sourceOverride struct {
	mode DiceRandomMode
	src  ds.DiceSource
}

type errorOverride struct {
	mode DiceRandomMode
	err  error
}

func completeSourceMap(overrides ...sourceOverride) map[DiceRandomMode]ds.DiceSource {
	result := map[DiceRandomMode]ds.DiceSource{
		DiceRandomModePCG:    nil,
		DiceRandomModeGM:     nil,
		DiceRandomModeNIST:   nil,
		DiceRandomModeCRNG:   nil,
		DiceRandomModeHybrid: nil,
	}
	for _, item := range overrides {
		result[item.mode] = item.src
	}
	return result
}

func completeErrorMap(overrides ...errorOverride) map[DiceRandomMode]error {
	result := map[DiceRandomMode]error{
		DiceRandomModePCG:    nil,
		DiceRandomModeGM:     nil,
		DiceRandomModeNIST:   nil,
		DiceRandomModeCRNG:   nil,
		DiceRandomModeHybrid: nil,
	}
	for _, item := range overrides {
		result[item.mode] = item.err
	}
	return result
}

func snapshotGlobalDiceSourceState() *randcore.GlobalOwner {
	return globalRandSource
}

func restoreGlobalDiceSourceState(owner *randcore.GlobalOwner) {
	globalRandSource = owner
}

func installGlobalDiceSourceState(t *testing.T, sources map[DiceRandomMode]ds.DiceSource, errs map[DiceRandomMode]error) {
	t.Helper()

	prev := snapshotGlobalDiceSourceState()
	t.Cleanup(func() {
		restoreGlobalDiceSourceState(prev)
	})

	owner := randcore.NewEmptyGlobalOwner()
	for mode, src := range sources {
		if src != nil {
			owner.RegisterSource(mode, src)
		}
	}
	for mode, err := range errs {
		if err != nil {
			owner.RegisterSourceError(mode, err)
		}
	}
	if err := owner.RegisterHybridSource(); err != nil {
		owner.RegisterSourceError(DiceRandomModeHybrid, err)
	}

	globalRandSource = owner
}

func TestGlobalRandSourceUsesSingleSharedOwner(t *testing.T) {
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{1}}},
		sourceOverride{mode: DiceRandomModeGM, src: &countingDiceSource{values: []uint64{2}}},
	), nil)

	ctx := &MsgContext{Dice: &Dice{}}
	if got := ctx.getDiceSource(); got != globalRandSource {
		t.Fatalf("getDiceSource() = %T, want globalRandSource", got)
	}
	if ctx._v1Rand != globalRandSource {
		t.Fatalf("expected _v1Rand to cache globalRandSource")
	}

	if mode, err := globalRandSource.SetActive(DiceRandomModePCG); err != nil || mode != DiceRandomModePCG {
		t.Fatalf("SetActive(pcg) = (%s, %v), want (pcg, nil)", mode, err)
	}
	if got := globalRandSource.Uint64(); got != 1 {
		t.Fatalf("pcg Uint64() = %d, want 1", got)
	}

	if mode, err := globalRandSource.SetActive(DiceRandomModeGM); err != nil || mode != DiceRandomModeGM {
		t.Fatalf("SetActive(gm) = (%s, %v), want (gm, nil)", mode, err)
	}
	if got := globalRandSource.Uint64(); got != 2 {
		t.Fatalf("gm Uint64() = %d, want 2", got)
	}
	if current := globalRandSource.CurrentMode(); current != DiceRandomModeGM {
		t.Fatalf("CurrentMode() = %s, want gm", current)
	}
}

func TestGlobalRandSourceHybridUsesRegisteredSources(t *testing.T) {
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{1}}},
		sourceOverride{mode: DiceRandomModeGM, src: &countingDiceSource{values: []uint64{2}}},
		sourceOverride{mode: DiceRandomModeNIST, src: &countingDiceSource{values: []uint64{4}}},
		sourceOverride{mode: DiceRandomModeCRNG, src: &countingDiceSource{values: []uint64{8}}},
	), nil)

	if mode, err := globalRandSource.SetActive(DiceRandomModeHybrid); err != nil || mode != DiceRandomModeHybrid {
		t.Fatalf("SetActive(hybrid) = (%s, %v), want (hybrid, nil)", mode, err)
	}
	if got := globalRandSource.Uint64(); got != 1^2^4^8 {
		t.Fatalf("hybrid Uint64() = %d, want %d", got, 1^2^4^8)
	}
}

func TestGlobalRandSourceReportGetText(t *testing.T) {
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{0}}},
		sourceOverride{mode: DiceRandomModeNIST, src: &countingDiceSource{values: []uint64{2}}},
		sourceOverride{mode: DiceRandomModeCRNG, src: &countingDiceSource{values: []uint64{3}}},
	), completeErrorMap(
		errorOverride{mode: DiceRandomModeGM, err: errors.New("gm init failed")},
	))

	got := globalRandSource.ReportGetText(20)
	for mode, raw := range map[DiceRandomMode]uint64{
		DiceRandomModePCG:  0,
		DiceRandomModeNIST: 2,
		DiceRandomModeCRNG: 3,
	} {
		expected := ds.Roll(&countingDiceSource{values: []uint64{raw}}, ds.IntType(20), 0)
		line := fmt.Sprintf("%s: 出目=%d", mode, expected)
		if !strings.Contains(got, line) {
			t.Fatalf("expected %q in get text, got %q", line, got)
		}
	}
	if !strings.Contains(got, "gm: 不可用") || !strings.Contains(got, "gm init failed") {
		t.Fatalf("expected gm unavailability in get text, got %q", got)
	}
	if !strings.Contains(got, "hybrid: 出目=") {
		t.Fatalf("expected hybrid line in get text, got %q", got)
	}
}

func TestGlobalRandSourceReportStatusText(t *testing.T) {
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{1}}},
	), completeErrorMap(
		errorOverride{mode: DiceRandomModeGM, err: errors.New("gm init failed")},
	))

	mode, err := globalRandSource.SetActive(DiceRandomModeGM)
	if err == nil {
		t.Fatal("expected SetActive(gm) to return fallback error")
	}
	if mode == DiceRandomModeGM {
		t.Fatal("expected SetActive(gm) to fall back to another available mode")
	}

	got := globalRandSource.ReportStatusText(DiceRandomModeGM)
	if !strings.Contains(got, "当前随机模式: GM 国密") {
		t.Fatalf("expected configured mode in status text, got %q", got)
	}
	if !strings.Contains(got, "当前生效模式: "+randcore.ModeSpecFor(mode).Label) {
		t.Fatalf("expected fallback mode in status text, got %q", got)
	}
	if !strings.Contains(got, "gm init failed") {
		t.Fatalf("expected fallback reason in status text, got %q", got)
	}
}
