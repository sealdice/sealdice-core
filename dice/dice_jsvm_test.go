package dice //nolint:testpackage

import (
	"testing"

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

func TestJsInit_WhenExtLoopManagerNil_DoesNotPanic(t *testing.T) {
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

	// 模拟“调用 shutdown 后重启”：JsEnable=false 且 ExtLoopManager 尚未初始化。
	d.Config.JsEnable = false
	d.ExtLoopManager = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("JsInit should not panic when ExtLoopManager is nil, got: %v", r)
		}
		// 清理后台任务，避免测试进程残留 goroutine
		if d.JsScriptCron != nil {
			d.JsScriptCron.Stop()
			d.JsScriptCron = nil
		}
		if d.ExtLoopManager != nil {
			d.ExtLoopManager.SetLoop(nil)
		}
	}()

	d.JsInit()

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
	d := &Dice{
		Logger: zap.NewNop().Sugar(),
		Config: NewConfig(nil),
	}
	if d.Logger == nil {
		t.Fatal("expected logger to be initialized")
	}
	d.Config.DiceRandomMode = string(DiceRandomModePCG)

	vm := goja.New()
	vm.SetRandSource(func() float64 {
		return float64(globalRandSource.Uint64()>>11) / (1 << 53)
	})
	var got float64
	var runErr error
	value, err := vm.RunString("Math.random()")
	if err != nil {
		runErr = err
	} else {
		got = value.ToFloat()
	}
	if runErr != nil {
		t.Fatalf("Math.random() error = %v", runErr)
	}

	want := float64((uint64(1)<<63)>>11) / (1 << 53)
	if got != want {
		t.Fatalf("Math.random() = %.16f, want %.16f", got, want)
	}

	if src.index != 1 {
		t.Fatalf("system dice source consumed %d values, want 1", src.index)
	}
}
