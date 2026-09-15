//nolint:testpackage
package dice

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
	"go.uber.org/zap"
	"sealdice-core/utils/procs"
)

func assertMilkyState(t *testing.T, pa *PlatformAdapterMilky, want EndpointState) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for {
		pa.lifecycleMu.Lock()
		got := pa.EndPoint.State
		pa.lifecycleMu.Unlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("state = %d, want %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestMilkyProcessSetupFailureCompletesDone(t *testing.T) {
	pa := &PlatformAdapterMilky{EndPoint: &EndPointInfo{}}
	generation, done := pa.beginMilkyProcess()
	if !pa.abortMilkyProcessSetup(generation, done) {
		t.Fatal("current setup failure was ignored")
	}
	select {
	case <-done:
	default:
		t.Fatal("failed setup left completion channel open")
	}
	assertMilkyState(t, pa, StateConnectionFailed)
	if pa.BuiltInLoginState != MilkyLoginStateFailed || pa.processDone != nil || pa.isCurrentMilkyProcess(generation) {
		t.Fatal("failed setup retained pending login or process state")
	}
	if !pa.EndPoint.Enable {
		t.Fatal("setup failure changed the user's enable intent")
	}
}

func TestMilkyOldSetupFailurePreservesNewAttempt(t *testing.T) {
	pa := &PlatformAdapterMilky{EndPoint: &EndPointInfo{}}
	oldGeneration, oldDone := pa.beginMilkyProcess()
	currentGeneration, currentDone := pa.beginMilkyProcess()
	defer pa.abortMilkyProcessSetup(currentGeneration, currentDone)
	if pa.abortMilkyProcessSetup(oldGeneration, oldDone) {
		t.Fatal("old setup failure changed current state")
	}
	select {
	case <-oldDone:
	default:
		t.Fatal("obsolete setup did not close its own completion channel")
	}
	select {
	case <-currentDone:
		t.Fatal("obsolete setup closed the new completion channel")
	default:
	}
	assertMilkyState(t, pa, StateConnecting)
	if pa.processDone != currentDone || pa.BuiltInLoginState != MilkyLoginStateInit || !pa.isCurrentMilkyProcess(currentGeneration) {
		t.Fatal("obsolete setup changed current process metadata")
	}
}

func newMilkyStartupTestAdapter(t *testing.T, mode string) (*Dice, *PlatformAdapterMilky) {
	t.Helper()
	d := &Dice{
		BaseConfig:   BaseConfig{Name: "default", DataDir: t.TempDir()},
		Logger:       zap.NewNop().Sugar(),
		AttrsManager: &AttrsManager{},
	}
	d.ImSession = &IMSession{Parent: d}
	ep := NewMilkyConnItem(AddMilkyEcho{BuiltInMode: mode})
	ep.UserID = "QQ:10010"
	ep.BindRuntime(d.ImSession)
	return d, ep.Adapter.(*PlatformAdapterMilky)
}

func TestMilkyBuiltinSetupFailures(t *testing.T) {
	for _, failure := range []string{"unsupported mode", "work directory", "config file"} {
		t.Run(failure, func(t *testing.T) {
			d, pa := newMilkyStartupTestAdapter(t, "yogurt")
			workDir := filepath.Join(d.BaseConfig.DataDir, pa.EndPoint.RelWorkDir)
			switch failure {
			case "unsupported mode":
				pa.BuiltInMode = "unsupported"
			case "work directory":
				if err := os.MkdirAll(filepath.Dir(workDir), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(workDir, []byte("blocked"), 0o644); err != nil {
					t.Fatal(err)
				}
			case "config file":
				// 已有目录占用配置文件名，稳定触发 WriteFile 失败。
				if err := os.MkdirAll(filepath.Join(workDir, "config.json"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			ServeMilkyBuiltIn(d, pa.EndPoint)
			assertMilkyState(t, pa, StateConnectionFailed)
			if pa.BuiltInLoginState != MilkyLoginStateFailed || pa.processDone != nil || pa.MilkyProcess != nil {
				t.Fatal("builtin setup failure retained unfinished process metadata")
			}
		})
	}
}

func TestMilkyBuiltinStartFailureOwnsDone(t *testing.T) {
	t.Chdir(t.TempDir()) // 独立目录没有内置客户端，稳定触发 p.Start 失败。
	for _, mode := range []string{"yogurt", "lagrangeV2"} {
		t.Run(mode, func(t *testing.T) {
			d, pa := newMilkyStartupTestAdapter(t, mode)
			ServeMilkyBuiltIn(d, pa.EndPoint)
			pa.lifecycleMu.Lock()
			done := pa.processDone
			pa.lifecycleMu.Unlock()
			if done == nil {
				t.Fatal("startup did not hand off completion channel")
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("failed child start did not complete")
			}
			assertMilkyState(t, pa, StateConnectionFailed)
			pa.lifecycleMu.Lock()
			failed := pa.BuiltInLoginState == MilkyLoginStateFailed
			pa.lifecycleMu.Unlock()
			if !failed {
				t.Fatal("failed child start retained login state")
			}
		})
	}
}

func TestMilkyInvalidGatewayEntryPointsReportFailure(t *testing.T) {
	for _, entry := range []string{"Serve", "DoRelogin", "SetEnable"} {
		t.Run(entry, func(t *testing.T) {
			_, pa := newMilkyStartupTestAdapter(t, "")
			pa.WsGateway = "invalid-websocket-gateway"
			pa.EndPoint.State = StateConnected
			pa.EndPoint.Enable = true
			switch entry {
			case "Serve":
				if pa.Serve() == 0 {
					t.Fatal("invalid gateway reported success")
				}
			case "DoRelogin":
				if pa.DoRelogin() {
					t.Fatal("invalid gateway relogin reported success")
				}
			case "SetEnable":
				pa.SetEnable(true)
			}
			assertMilkyState(t, pa, StateConnectionFailed)
			if pa.EndPoint.Enable {
				t.Fatal("failed session remained enabled")
			}
		})
	}
}

func TestMilkyProcessExitAndOldProcessIsolation(t *testing.T) {
	for _, mode := range []string{"yogurt", "lagrangeV2"} {
		t.Run(mode, func(t *testing.T) {
			pa := &PlatformAdapterMilky{EndPoint: &EndPointInfo{}, BuiltInMode: mode}
			generation, _ := pa.beginMilkyProcess()
			old := procs.NewProcess("old")
			pa.registerMilkyProcess(generation, old)
			pa.finishMilkyProcess(generation, old)
			assertMilkyState(t, pa, StateDisconnected)
			generation, _ = pa.beginMilkyProcess()
			pa.registerMilkyProcess(generation, old)
			currentGeneration, _ := pa.beginMilkyProcess()
			current := procs.NewProcess("current")
			pa.registerMilkyProcess(currentGeneration, current)
			pa.EndPoint.State = StateConnected
			pa.finishMilkyProcess(generation, old)
			assertMilkyState(t, pa, StateConnected)
			if pa.MilkyProcess != current {
				t.Fatal("old exit cleared the new process")
			}
			if pa.startMilkySession(&milky.Session{}, generation) {
				t.Fatal("old process started a new session")
			}
		})
	}
}
