package dice //nolint:testpackage

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	ds "github.com/sealdice/dicescript"
)

type globalDiceSourceState struct {
	ready   bool
	sources map[DiceRandomMode]ds.DiceSource
	errs    map[DiceRandomMode]error
}

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

func snapshotGlobalDiceSourceState() globalDiceSourceState {
	globalDiceSourceMu.Lock()
	defer globalDiceSourceMu.Unlock()

	state := globalDiceSourceState{
		ready:   globalDiceSourcesReady,
		sources: map[DiceRandomMode]ds.DiceSource{},
		errs:    map[DiceRandomMode]error{},
	}
	for mode, src := range globalDiceSources {
		state.sources[mode] = src
	}
	for mode, err := range globalDiceSourceErrors {
		state.errs[mode] = err
	}
	return state
}

func restoreGlobalDiceSourceState(state globalDiceSourceState) {
	globalDiceSourceMu.Lock()
	defer globalDiceSourceMu.Unlock()

	globalDiceSourcesReady = state.ready
	globalDiceSources = map[DiceRandomMode]ds.DiceSource{}
	globalDiceSourceErrors = map[DiceRandomMode]error{}
	for mode, src := range state.sources {
		globalDiceSources[mode] = src
	}
	for mode, err := range state.errs {
		globalDiceSourceErrors[mode] = err
	}
}

func installGlobalDiceSourceState(t *testing.T, sources map[DiceRandomMode]ds.DiceSource, errs map[DiceRandomMode]error) {
	t.Helper()

	prev := snapshotGlobalDiceSourceState()
	t.Cleanup(func() {
		restoreGlobalDiceSourceState(prev)
	})

	globalDiceSourceMu.Lock()
	defer globalDiceSourceMu.Unlock()

	globalDiceSourcesReady = true
	globalDiceSources = map[DiceRandomMode]ds.DiceSource{}
	globalDiceSourceErrors = map[DiceRandomMode]error{}
	for mode, src := range sources {
		globalDiceSources[mode] = src
	}
	for mode, err := range errs {
		globalDiceSourceErrors[mode] = err
	}
}

func TestGetDiceSource_UsesPackageGlobalSharedSource(t *testing.T) {
	sharedPCG := &countingDiceSource{values: []uint64{1}}
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: sharedPCG},
	), nil)

	d1 := &Dice{Config: NewConfig(nil)}
	d1.Config.DiceRandomMode = string(DiceRandomModePCG)
	d2 := &Dice{Config: NewConfig(nil)}
	d2.Config.DiceRandomMode = string(DiceRandomModePCG)

	ctx1 := &MsgContext{Dice: d1}
	ctx2 := &MsgContext{Dice: d2}

	got1 := ctx1.getDiceSource()
	got2 := ctx2.getDiceSource()
	if got1 != sharedPCG || got2 != sharedPCG {
		t.Fatalf("expected both contexts to use shared PCG source, got %T / %T", got1, got2)
	}
	if got1 != got2 {
		t.Fatalf("expected shared source pointer, got %p / %p", got1, got2)
	}
	if ctx1._v1Rand != sharedPCG || ctx2._v1Rand != sharedPCG {
		t.Fatalf("expected _v1Rand to follow shared source")
	}
	if ctx1.diceRandSrc != nil || ctx2.diceRandSrc != nil {
		t.Fatalf("expected ordinary getDiceSource not to populate ctx.diceRandSrc cache")
	}
}

func TestGetSystemDiceSource_ModeSwitchReusesGlobalSource(t *testing.T) {
	sharedPCG := &countingDiceSource{values: []uint64{1}}
	sharedGM := &countingDiceSource{values: []uint64{2}}
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: sharedPCG},
		sourceOverride{mode: DiceRandomModeGM, src: sharedGM},
	), nil)

	d := &Dice{Config: NewConfig(nil)}
	d.Config.DiceRandomMode = string(DiceRandomModePCG)
	src1 := d.getSystemDiceSource()

	d.Config.DiceRandomMode = string(DiceRandomModeGM)
	src2 := d.getSystemDiceSource()

	d.Config.DiceRandomMode = string(DiceRandomModePCG)
	src3 := d.getSystemDiceSource()

	if src1 != sharedPCG || src2 != sharedGM || src3 != sharedPCG {
		t.Fatalf("unexpected mode switch sources: %p %p %p", src1, src2, src3)
	}
	if src1 != src3 {
		t.Fatalf("expected switching back to pcg to reuse original shared source")
	}
}

func TestGetSystemDiceSource_HybridXorsAvailableSources(t *testing.T) {
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{1}}},
		sourceOverride{mode: DiceRandomModeGM, src: &countingDiceSource{values: []uint64{2}}},
		sourceOverride{mode: DiceRandomModeNIST, src: &countingDiceSource{values: []uint64{4}}},
		sourceOverride{mode: DiceRandomModeCRNG, src: &countingDiceSource{values: []uint64{8}}},
	), nil)

	d := &Dice{Config: NewConfig(nil)}
	d.Config.DiceRandomMode = string(DiceRandomModeHybrid)

	src := d.getSystemDiceSource()
	if src == nil {
		t.Fatal("expected hybrid source")
	}
	if got := src.Uint64(); got != 1^2^4^8 {
		t.Fatalf("hybrid source xor = %d, want %d", got, 1^2^4^8)
	}
}

func TestGetDiceSource_ExplicitOverrideStillWins(t *testing.T) {
	sharedPCG := &countingDiceSource{values: []uint64{1}}
	override := &countingDiceSource{values: []uint64{2}}
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: sharedPCG},
	), nil)

	d := &Dice{Config: NewConfig(nil)}
	d.Config.DiceRandomMode = string(DiceRandomModePCG)
	ctx := &MsgContext{Dice: d, diceRandSrc: override}

	got := ctx.getDiceSource()
	if got != override {
		t.Fatalf("expected explicit override source, got %p", got)
	}
	if ctx._v1Rand != override {
		t.Fatalf("expected _v1Rand to follow override source")
	}
}

func TestRandalgoRejectsUnavailableModeAndReportsEffectiveFallback(t *testing.T) {
	sharedPCG := &countingDiceSource{values: []uint64{1}}
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: sharedPCG},
	), completeErrorMap(
		errorOverride{mode: DiceRandomModeGM, err: errors.New("gm init failed")},
	))

	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ep.Platform = "TG"
	ctx, msg := newQuitCommandTestContext(t, d, ep, "TG:9201", "TG-Group:2201", "RandomGroup")

	result := d.CmdMap["randalgo"].Solve(ctx, msg, &CmdArgs{Args: []string{"set", "gm"}})
	if !result.Matched || !result.Solved {
		t.Fatalf("unexpected set result: %#v", result)
	}

	reply, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("expected unavailable-mode reply")
	}
	if !strings.Contains(reply, "不可用") || !strings.Contains(reply, "gm init failed") {
		t.Fatalf("unexpected unavailable-mode reply: %q", reply)
	}
	if d.Config.DiceRandomMode == string(DiceRandomModeGM) {
		t.Fatalf("expected rejected mode not to be persisted into config")
	}

	d.Config.DiceRandomMode = string(DiceRandomModeGM)
	result = d.CmdMap["randalgo"].Solve(ctx, msg, &CmdArgs{})
	if !result.Matched || !result.Solved {
		t.Fatalf("unexpected query result: %#v", result)
	}

	reply, ok = adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("expected fallback status reply")
	}
	if !strings.Contains(reply, "当前随机模式: GM 国密") {
		t.Fatalf("expected configured mode in fallback reply, got %q", reply)
	}
	if !strings.Contains(reply, "当前生效模式: 默认 PCG") {
		t.Fatalf("expected effective fallback mode in reply, got %q", reply)
	}
}

func TestRandalgoGetReportsEachModeWithCustomPoints(t *testing.T) {
	points := int64(20)
	sources := completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{0}}},
		sourceOverride{mode: DiceRandomModeGM, src: &countingDiceSource{values: []uint64{1}}},
		sourceOverride{mode: DiceRandomModeNIST, src: &countingDiceSource{values: []uint64{2}}},
		sourceOverride{mode: DiceRandomModeCRNG, src: &countingDiceSource{values: []uint64{3}}},
	)
	installGlobalDiceSourceState(t, sources, nil)

	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ep.Platform = "TG"
	ctx, msg := newQuitCommandTestContext(t, d, ep, "TG:9202", "TG-Group:2202", "RandomGroup")

	result := d.CmdMap["randalgo"].Solve(ctx, msg, &CmdArgs{Args: []string{"get", "20"}})
	if !result.Matched || !result.Solved {
		t.Fatalf("unexpected get result: %#v", result)
	}

	reply, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("expected get reply")
	}
	if !strings.Contains(reply, "D20") {
		t.Fatalf("expected points in get reply, got %q", reply)
	}

	for mode, src := range sources {
		if src == nil {
			continue
		}
		expected := ds.Roll(src, ds.IntType(points), 0)
		line := fmt.Sprintf("%s: 出目=%d", mode, expected)
		if !strings.Contains(reply, line) {
			t.Fatalf("expected line %q in get reply: %q", line, reply)
		}
		if !strings.Contains(reply, string(mode)+": 出目=") || !strings.Contains(reply, "耗时=") {
			t.Fatalf("expected timing output for %s in reply: %q", mode, reply)
		}
	}
	if !strings.Contains(reply, "hybrid: 出目=") {
		t.Fatalf("expected hybrid line in get reply: %q", reply)
	}
}

func TestRandalgoGetReportsUnavailableModeWithoutBlockingOthers(t *testing.T) {
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{0}}},
		sourceOverride{mode: DiceRandomModeNIST, src: &countingDiceSource{values: []uint64{2}}},
		sourceOverride{mode: DiceRandomModeCRNG, src: &countingDiceSource{values: []uint64{3}}},
	), completeErrorMap(
		errorOverride{mode: DiceRandomModeGM, err: errors.New("gm init failed")},
	))

	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ep.Platform = "TG"
	ctx, msg := newQuitCommandTestContext(t, d, ep, "TG:9203", "TG-Group:2203", "RandomGroup")

	result := d.CmdMap["randalgo"].Solve(ctx, msg, &CmdArgs{Args: []string{"get"}})
	if !result.Matched || !result.Solved {
		t.Fatalf("unexpected get result: %#v", result)
	}

	reply, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("expected get reply")
	}
	if !strings.Contains(reply, "pcg: 出目=") || !strings.Contains(reply, "nist: 出目=") || !strings.Contains(reply, "crng: 出目=") {
		t.Fatalf("expected available modes in get reply: %q", reply)
	}
	if !strings.Contains(reply, "hybrid: 出目=") {
		t.Fatalf("expected hybrid mode in get reply: %q", reply)
	}
	if !strings.Contains(reply, "gm: 不可用") || !strings.Contains(reply, "gm init failed") {
		t.Fatalf("expected unavailable gm in get reply: %q", reply)
	}
}

func TestRandalgoQuery_HybridShowsAvailableSources(t *testing.T) {
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: &countingDiceSource{values: []uint64{1}}},
		sourceOverride{mode: DiceRandomModeNIST, src: &countingDiceSource{values: []uint64{4}}},
		sourceOverride{mode: DiceRandomModeCRNG, src: &countingDiceSource{values: []uint64{8}}},
	), completeErrorMap(
		errorOverride{mode: DiceRandomModeGM, err: errors.New("gm init failed")},
	))

	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	d.Config.DiceRandomMode = string(DiceRandomModeHybrid)
	ep.Platform = "TG"
	ctx, msg := newQuitCommandTestContext(t, d, ep, "TG:9204", "TG-Group:2204", "RandomGroup")

	result := d.CmdMap["randalgo"].Solve(ctx, msg, &CmdArgs{})
	if !result.Matched || !result.Solved {
		t.Fatalf("unexpected query result: %#v", result)
	}

	reply, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("expected hybrid query reply")
	}
	if !strings.Contains(reply, "当前随机模式: Hybrid 混合") {
		t.Fatalf("expected hybrid mode label in reply: %q", reply)
	}
	if !strings.Contains(reply, "当前混合源包含: 默认 PCG, NIST, 系统级 CRNG") {
		t.Fatalf("expected hybrid available source summary in reply: %q", reply)
	}
}
