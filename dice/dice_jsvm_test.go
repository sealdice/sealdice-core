package dice //nolint:testpackage

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"sealdice-core/utils/dboperator/gormkv"
	"sealdice-core/utils/dboperator/gormkv/codec"
)

type countingDiceSource struct {
	values []uint64
	index  int
}

func TestPluginKVCompressionOptions(t *testing.T) {
	options := pluginKVCompressionOptions()
	if !options.Enabled || options.Level != 1 || options.MinSize != 4*1024 {
		t.Fatalf("plugin KV compression options = %#v", options)
	}
	compressor, err := codec.New(options)
	if err != nil {
		t.Fatal(err)
	}
	value := []byte(strings.Repeat("plugin-json-value", 1024))
	encoded, kind := compressor.CompressWithKind(value, store.ValueKindJSON)
	if kind != codec.CodecZstd || len(encoded) >= len(value) {
		t.Fatalf("runtime compression codec=%d encoded=%d raw=%d", kind, len(encoded), len(value))
	}
}

func TestRegisterPluginKVExpirationCleanup(t *testing.T) {
	dbOperator, err := newMockDatabaseOperator(filepath.Join(t.TempDir(), "plugins.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dbOperator.Close)
	backend, err := gormkv.NewBackend(gormkv.BackendOptions{DB: dbOperator.db})
	if err != nil {
		t.Fatal(err)
	}
	d := &Dice{
		Cron:            cron.New(),
		Logger:          zap.NewNop().Sugar(),
		PluginKVBackend: backend,
	}
	if err := d.registerPluginKVExpirationCleanup(); err != nil {
		t.Fatal(err)
	}
	if entries := d.Cron.Entries(); len(entries) != 1 {
		t.Fatalf("expiration cleanup entries = %d", len(entries))
	}
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

func TestJsInit_RegistersExtendedWebAPIs(t *testing.T) {
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

	d.JsInit()
	t.Cleanup(func() {
		if d.JsScriptCron != nil {
			d.JsScriptCron.Stop()
		}
		if d.ExtLoopManager != nil {
			d.ExtLoopManager.SetLoop(nil)
		}
	})

	values := make(chan bool, 1)
	if ok := d.ExtLoopManager.GetWebLoop().RunOnLoop(func(vm *goja.Runtime) {
		value, err := vm.RunString(`(() => {
			const controller = new AbortController();
			controller.abort("test");
			return typeof fetch === "function" &&
				typeof AbortController === "function" &&
				controller.signal.aborted === true &&
				typeof crypto.randomUUID === "function" &&
				typeof WebSocket === "function" &&
				typeof require("fetch").Response === "function" &&
				typeof require("abort").AbortSignal === "object" &&
				typeof require("crypto").randomUUID === "function";
		})()`)
		if err != nil {
			t.Errorf("inspect globals: %v", err)
			return
		}
		values <- value.ToBoolean()
	}); !ok {
		t.Fatal("failed to schedule runtime inspection")
	}

	if !<-values {
		t.Fatal("extended Web APIs are not fully enabled")
	}
}

func TestJsExtRegisterInjectsStorageNamespaces(t *testing.T) {
	dbOperator, err := newMockDatabaseOperator(filepath.Join(t.TempDir(), "plugins.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dbOperator.Close)
	backend, err := gormkv.NewBackend(gormkv.BackendOptions{DB: dbOperator.db})
	if err != nil {
		t.Fatal(err)
	}
	if err = backend.AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	repo, err := backend.Namespace("storage-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Put(t.Context(), "migrated", []byte("legacy-value"), store.PutOptions{
		ValueKind: store.ValueKindText,
	}); err != nil {
		t.Fatal(err)
	}

	d := &Dice{
		Logger: zap.NewNop().Sugar(),
		BaseConfig: BaseConfig{
			Name:    "storage-test",
			DataDir: t.TempDir(),
		},
		DBOperator:      dbOperator,
		PluginKVBackend: backend,
		ExtRegistry:     new(SyncMap[string, *ExtInfo]),
		ImSession: &IMSession{
			ServiceAtNew: new(SyncMap[string, *GroupInfo]),
			EndPoints:    []*EndPointInfo{},
		},
		DirtyGroups:  new(SyncMap[string, int64]),
		AttrsManager: &AttrsManager{},
	}
	d.JsInit()
	t.Cleanup(func() {
		if d.JsScriptCron != nil {
			d.JsScriptCron.Stop()
		}
		if d.ExtLoopManager != nil {
			d.ExtLoopManager.SetLoop(nil)
		}
	})

	type registrationResult struct {
		seen     string
		original *ExtInfo
	}
	result := make(chan registrationResult, 1)
	if ok := d.ExtLoopManager.GetWebLoop().RunOnLoop(func(vm *goja.Runtime) {
		value, runErr := vm.RunString(`(() => {
			const ext = seal.ext.new("storage-test", "author", "1");
			globalThis.__storageTestExt = ext;
			let seen = "";
			ext.onLoad = function() {
				seen = [typeof ext.storage, typeof ext.storage.kv,
					typeof ext.storage.synckv, typeof ext.storage.kv.put,
					typeof ext.storage.synckv.put, typeof ext.storageGet,
					typeof ext.storageSet].join(",");
				seen += "|" + ext.storageGet("migrated");
				ext.storageSet("legacy-write", "written-value");
				ext.storage.synckv.put("key", "value");
			};
			seal.ext.register(ext);
			return seen;
		})()`)
		if runErr != nil {
			t.Errorf("register storage extension: %v", runErr)
			return
		}
		original, _ := vm.Get("__storageTestExt").Export().(*ExtInfo)
		result <- registrationResult{seen: value.String(), original: original}
	}); !ok {
		t.Fatal("failed to schedule runtime registration")
	}

	got := <-result
	if got.seen != "object,object,object,function,function,function,function|legacy-value" {
		t.Fatalf("storage namespaces = %q", got.seen)
	}
	registered, ok := d.JsExtRegistry.Load("storage-test")
	if !ok || registered == nil || registered != got.original {
		t.Fatalf("registered extension must be the object returned by seal.ext.new: registered=%p original=%p", registered, got.original)
	}

	for key, want := range map[string]string{"key": "value", "legacy-write": "written-value"} {
		record, found, err := repo.Get(t.Context(), key)
		if err != nil || !found || string(record.Value) != want {
			t.Fatalf("key %q = %#v found=%v err=%v", key, record, found, err)
		}
	}
}
