package dice //nolint:testpackage

import (
	"testing"
	"time"

	"github.com/dop251/goja"
	"go.uber.org/zap"
)

type countingDiceSource struct {
	values []uint64
	index  int
}

func (s *countingDiceSource) Uint64() uint64 {
	if len(s.values) == 0 {
		return 0
	}
	if s.index >= len(s.values) {
		return s.values[len(s.values)-1]
	}
	v := s.values[s.index]
	s.index++
	return v
}

func newJSInitTestDice(t *testing.T) *Dice {
	t.Helper()
	d := &Dice{
		Logger: zap.NewNop().Sugar(),
		BaseConfig: BaseConfig{
			DataDir: t.TempDir(),
		},
		ImSession: &IMSession{
			ServiceAtNew: new(SyncMap[string, *GroupInfo]),
			EndPoints:    []*EndPointInfo{},
		},
		DirtyGroups:  new(SyncMap[string, int64]),
		AttrsManager: &AttrsManager{},
	}

	t.Cleanup(func() {
		if d.JsScriptCron != nil {
			<-d.JsScriptCron.Stop().Done()
		}
		if d.ExtLoopManager != nil {
			d.ExtLoopManager.SetLoop(nil)
		}
	})
	return d
}

func runOnTestJSLoop(t *testing.T, d *Dice, fn func(*goja.Runtime)) {
	t.Helper()
	if d.ExtLoopManager == nil || d.ExtLoopManager.GetWebLoop() == nil {
		t.Fatal("JS runtime was not initialized")
	}
	done := make(chan struct{})
	if !d.ExtLoopManager.GetWebLoop().RunOnLoop(func(vm *goja.Runtime) {
		defer close(done)
		fn(vm)
	}) {
		t.Fatal("JS event loop rejected the callback")
	}
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		t.Fatal("timed out waiting for the JS event loop")
	}
}

func TestJsInit_WhenExtLoopManagerNil_DoesNotPanic(t *testing.T) {
	d := newJSInitTestDice(t)
	// 模拟“调用 shutdown 后重启”：JsEnable=false 且 ExtLoopManager 尚未初始化。
	d.Config.JsEnable = false
	d.ExtLoopManager = nil

	d.JsInit()
	runOnTestJSLoop(t, d, func(_ *goja.Runtime) {})

	if d.ExtLoopManager == nil {
		t.Fatalf("expected ExtLoopManager to be initialized")
	}
	if !d.Config.JsEnable {
		t.Fatalf("expected JsEnable to be true after JsInit")
	}
}

func TestJsInit_BindsGojaRandSourceToGlobalRandSource(t *testing.T) {
	src := &countingDiceSource{values: []uint64{1 << 63}}
	installGlobalDiceSourceState(t, completeSourceMap(
		sourceOverride{mode: DiceRandomModePCG, src: src},
	), nil)
	if _, err := globalRandSource.SetActive(DiceRandomModePCG); err != nil {
		t.Fatalf("activate pcg source: %v", err)
	}
	d := newJSInitTestDice(t)
	d.Config = NewConfig(nil)
	d.Config.DiceRandomMode = string(DiceRandomModePCG)
	d.JsInit()

	var got float64
	var runErr error
	runOnTestJSLoop(t, d, func(vm *goja.Runtime) {
		value, err := vm.RunString("Math.random()")
		if err != nil {
			runErr = err
		} else {
			got = value.ToFloat()
		}
	})
	if runErr != nil {
		t.Fatalf("Math.random() error = %v", runErr)
	}

	const want = 0.5
	if got != want {
		t.Fatalf("Math.random() = %.16f, want %.16f", got, want)
	}

	if src.index != 1 {
		t.Fatalf("system dice source consumed %d values, want 1", src.index)
	}
}
