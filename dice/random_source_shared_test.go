package dice //nolint:testpackage

import (
	"errors"
	"fmt"
	randv2 "math/rand/v2"
	"strings"
	"testing"

	"github.com/dop251/goja"
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

func snapshotGlobalDiceSourceState() *randcore.GlobalRand {
	return globalRandSource
}

func restoreGlobalDiceSourceState(owner *randcore.GlobalRand) {
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

func TestWeightedBusinessCompatibility(t *testing.T) {
	// Golden sequences were recorded through the original business entry points
	// before replacing weightedrand. Also detect any extra random consumption.
	for _, name := range []string{"template", "reply", "deck", "names", "js"} {
		t.Run(name, func(t *testing.T) {
			src := newPCGDiceSource(41)
			var text strings.Builder
			want := "ccacabbaccacbcaaacca"
			switch name {
			case "template":
				d := &Dice{TextMapRaw: TextTemplateWithWeightDict{"test": {"g": {{"zero", 0}, {"a", 3}, {"b", 1}, {"c", 3}}}}}
				d.GenerateTextMap()
				for range 20 {
					text.WriteString(pickChooserWithSource(d.TextMap["test:g"], src))
				}
			case "reply":
				items := TextTemplateItemList{{"zero", 0}, {"a", float64(3)}, {"b", 1}, {"c", 3}}
				pool := items.toRandomPool()
				for range 20 {
					text.WriteString(pickChooserWithSource(pool, src))
				}
			case "deck":
				pool := DeckToRandomPool([]string{"::0::zero", "::3::a", "b", "::3::c"})
				for range 20 {
					text.WriteString(pickChooserWithSource(pool, src))
				}
			case "names":
				installGlobalDiceSourceState(t, completeSourceMap(sourceOverride{mode: DiceRandomModePCG, src: src}), nil)
				ng := &NamesGenerator{NamesInfo: map[string]map[string][]string{"s": {"name": {"zero", "a", "b", "c"}, "weight": {"0", "3", "1", "3"}, "alias": {"Z", "A", "B", "C"}}}}
				for range 20 {
					text.WriteString(ng.NameGenerate("{s:name@s:weight}{s:alias#s:name.index}"))
				}
				want = "cCcCaAcCaAbBbBaAcCcCaAcCbBcCaAaAaAcCcCaA"
			case "js":
				vm := goja.New()
				if err := vm.Set("pool", DeckToRandomPool([]string{"::0::zero", "::3::a", "b", "::3::c"})); err != nil {
					t.Fatal(err)
				}
				if err := vm.Set("random", randv2.New(src)); err != nil {
					t.Fatal(err)
				}
				value, err := vm.RunString(`Array.from({length: 20}, () => pool.PickWith(random)).join("")`)
				if err != nil {
					t.Fatal(err)
				}
				text.WriteString(value.String())
				if err := vm.Set("single", DeckToRandomPool([]string{"only"})); err != nil {
					t.Fatal(err)
				}
				if value, err := vm.RunString(`single.Pick()`); err != nil || value.String() != "only" {
					t.Fatalf("JS Pick: %v, %v", value, err)
				}
			}
			if got := text.String(); got != want {
				t.Fatalf("draw sequence changed: %q != %q", got, want)
			}
			if got := src.Uint64(); got != 6450427785793706784 {
				t.Fatalf("random consumption changed: next = %d", got)
			}
		})
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

	if _, err := globalRandSource.SetActive(DiceRandomModePCG); err != nil {
		t.Fatalf("SetActive(pcg): %v", err)
	}

	got := globalRandSource.ReportGetText("", 20)
	expected := ds.Roll(&countingDiceSource{values: []uint64{0}}, ds.IntType(20), 0)
	if !strings.Contains(got, fmt.Sprintf("pcg: 出目=%d", expected)) {
		t.Fatalf("expected current pcg source in get text, got %q", got)
	}
	for _, line := range []string{"nist: 出目=", "crng: 出目=", "hybrid: 出目="} {
		if strings.Contains(got, line) {
			t.Fatalf("default get should not test other sources, got %q", got)
		}
	}
	if !strings.Contains(got, "gm: 未启用") {
		t.Fatalf("expected gm disabled hint in get text, got %q", got)
	}

	got = globalRandSource.ReportGetText(DiceRandomModeNIST, 20)
	expected = ds.Roll(&countingDiceSource{values: []uint64{2}}, ds.IntType(20), 0)
	if !strings.Contains(got, fmt.Sprintf("nist: 出目=%d", expected)) {
		t.Fatalf("expected explicit nist source in get text, got %q", got)
	}

	got = globalRandSource.ReportAllGetText(20)
	for _, line := range []string{"pcg: 出目=", "nist: 出目=", "crng: 出目=", "hybrid: 出目=", "gm: 未启用"} {
		if !strings.Contains(got, line) {
			t.Fatalf("expected %q in all get text, got %q", line, got)
		}
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
	if !strings.Contains(got, "当前随机模式: 国密 SM3") {
		t.Fatalf("expected configured mode in status text, got %q", got)
	}
	if !strings.Contains(got, "当前生效模式: "+randcore.ModeSpecFor(mode).Label) {
		t.Fatalf("expected fallback mode in status text, got %q", got)
	}
	if !strings.Contains(got, "gm init failed") {
		t.Fatalf("expected fallback reason in status text, got %q", got)
	}
}
