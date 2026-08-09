//nolint:testpackage
package dice

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"sealdice-core/utils/procs"
)

func newMilkyBuiltInTestAdapter(builtInMode string) *PlatformAdapterMilky {
	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			UserID: "QQ:10010",
		},
	}
	pa := &PlatformAdapterMilky{EndPoint: ep, BuiltInMode: builtInMode}
	ep.Adapter = pa
	return pa
}

const (
	milkyHelperEnv   = "GO_WANT_MILKY_HELPER_PROCESS"
	milkyHelperSleep = "GO_WANT_MILKY_HELPER_SLEEP"
)

func startMilkyBuiltInHelperProcess(t *testing.T, sleep bool) (*procs.Process, func()) {
	t.Helper()
	args := []string{"-test.run=TestMilkyBuiltInHelperProcess"}
	if sleep {
		args = append(args, "-test.v")
	}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), milkyHelperEnv+"=1")
	if sleep {
		cmd.Env = append(cmd.Env, milkyHelperSleep+"=1")
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper process: %v", err)
	}
	cleanup := func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}
	return &procs.Process{Cmd: cmd}, cleanup
}

func TestMilkyBuiltInHelperProcess(t *testing.T) {
	if os.Getenv(milkyHelperEnv) != "1" {
		return
	}
	if os.Getenv(milkyHelperSleep) == "1" {
		time.Sleep(time.Hour)
	}
}

func TestPrepareMilkyBuiltInConfigAssignsNewPortAndToken(t *testing.T) {
	dir := t.TempDir()
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	configFile := filepath.Join(dir, "appsettings.jsonc")

	if err := prepareMilkyBuiltInConfig(pa.EndPoint, configFile); err != nil {
		t.Fatalf("prepareMilkyBuiltInConfig() error = %v", err)
	}
	firstGateway := pa.WsGateway
	firstRest := pa.RestGateway
	firstToken := pa.Token
	config, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("read config file: %v", err)
	}
	if !strings.Contains(string(config), firstToken) {
		t.Fatalf("config file does not contain generated token")
	}

	if err := prepareMilkyBuiltInConfig(pa.EndPoint, configFile); err != nil {
		t.Fatalf("second prepareMilkyBuiltInConfig() error = %v", err)
	}
	if pa.WsGateway == firstGateway {
		t.Fatalf("expected a new WsGateway after re-preparing config, got %q", pa.WsGateway)
	}
	if pa.RestGateway == firstRest {
		t.Fatalf("expected a new RestGateway after re-preparing config, got %q", pa.RestGateway)
	}
	if pa.Token == firstToken {
		t.Fatalf("expected a new access token after re-preparing config")
	}
}

func TestStopMilkyBuiltInLockedWithoutProcess(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	if err := stopMilkyBuiltInLocked(pa); err != nil {
		t.Fatalf("stopMilkyBuiltInLocked() nil-process error = %v", err)
	}
}

func TestStopMilkyBuiltInLockedMissingDoneChannel(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	process, cleanup := startMilkyBuiltInHelperProcess(t, true)
	t.Cleanup(cleanup)
	pa.MilkyProcess = process

	err := stopMilkyBuiltInLocked(pa)
	if err == nil {
		t.Fatal("expected stopMilkyBuiltInLocked to report a missing exit notification")
	}
	if !strings.Contains(err.Error(), "缺少退出通知") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopMilkyBuiltInLockedWaitsForExitNotification(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	process, cleanup := startMilkyBuiltInHelperProcess(t, true)
	t.Cleanup(cleanup)
	done := make(chan struct{})
	pa.MilkyProcess = process
	pa.builtInProcessDone = done

	result := make(chan error, 1)
	go func() {
		result <- stopMilkyBuiltInLocked(pa)
	}()

	select {
	case err := <-result:
		t.Fatalf("stopMilkyBuiltInLocked returned before done was closed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(done)
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("stopMilkyBuiltInLocked() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stopMilkyBuiltInLocked did not return after done closed")
	}

	if pa.MilkyProcess != nil {
		t.Fatalf("expected MilkyProcess to be cleared after stop, got %p", pa.MilkyProcess)
	}
	if pa.builtInProcessDone != nil {
		t.Fatalf("expected builtInProcessDone to be cleared after stop")
	}
}

func TestWaitMilkyBuiltInKeepsNewerProcessState(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	oldProcess, cleanup := startMilkyBuiltInHelperProcess(t, false)
	t.Cleanup(cleanup)
	newProcess := &procs.Process{}
	pa.MilkyProcess = newProcess
	pa.builtInProcessDone = make(chan struct{})

	d := &Dice{Logger: zap.NewNop().Sugar()}
	done := make(chan struct{})
	waitMilkyBuiltIn(d, pa, oldProcess, done, nil, nil)

	if pa.MilkyProcess != newProcess {
		t.Fatalf("waitMilkyBuiltIn cleared a newer process: got %p want %p", pa.MilkyProcess, newProcess)
	}
	if pa.builtInProcessDone == nil {
		t.Fatal("waitMilkyBuiltIn cleared builtInProcessDone for a newer process")
	}
}

func TestWaitMilkyBuiltInClearsCurrentProcessState(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	process, cleanup := startMilkyBuiltInHelperProcess(t, false)
	t.Cleanup(cleanup)
	done := make(chan struct{})
	pa.MilkyProcess = process
	pa.builtInProcessDone = done

	d := &Dice{Logger: zap.NewNop().Sugar()}
	waitMilkyBuiltIn(d, pa, process, done, nil, nil)

	if pa.MilkyProcess != nil {
		t.Fatalf("expected MilkyProcess to be cleared, got %p", pa.MilkyProcess)
	}
	if pa.builtInProcessDone != nil {
		t.Fatal("expected builtInProcessDone to be cleared")
	}
}

func TestBuiltinMilkyClientKillClearsProcessState(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	process, cleanup := startMilkyBuiltInHelperProcess(t, true)
	t.Cleanup(cleanup)
	done := make(chan struct{})
	close(done)
	pa.MilkyProcess = process
	pa.builtInProcessDone = done

	BuiltinMilkyClientKill(&Dice{Logger: zap.NewNop().Sugar()}, pa.EndPoint)

	if pa.MilkyProcess != nil {
		t.Fatalf("BuiltinMilkyClientKill did not clear MilkyProcess")
	}
	if pa.builtInProcessDone != nil {
		t.Fatalf("BuiltinMilkyClientKill did not clear builtInProcessDone")
	}
}

func TestBuiltinMilkyClientKillWaitsForRunningProcess(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")
	process, cleanup := startMilkyBuiltInHelperProcess(t, true)
	t.Cleanup(cleanup)
	done := make(chan struct{})
	pa.MilkyProcess = process
	pa.builtInProcessDone = done

	d := &Dice{Logger: zap.NewNop().Sugar()}
	waitDone := make(chan struct{})
	go func() {
		waitMilkyBuiltIn(d, pa, process, done, nil, nil)
		close(waitDone)
	}()

	killDone := make(chan struct{})
	go func() {
		BuiltinMilkyClientKill(d, pa.EndPoint)
		close(killDone)
	}()

	select {
	case <-killDone:
	case <-time.After(3 * time.Second):
		t.Fatal("BuiltinMilkyClientKill did not return after killing the running process")
	}
	select {
	case <-waitDone:
	case <-time.After(3 * time.Second):
		t.Fatal("waitMilkyBuiltIn did not return after the process was killed")
	}

	if pa.MilkyProcess != nil {
		t.Fatalf("expected MilkyProcess to be cleared after kill")
	}
	if pa.builtInProcessDone != nil {
		t.Fatal("expected builtInProcessDone to be cleared after kill")
	}
}

func TestMilkyBuiltInProcessOperationsAreSerialized(t *testing.T) {
	pa := newMilkyBuiltInTestAdapter("lagrangeV2")

	const workerCount = 8
	var active int32
	seen := make(chan int32, workerCount)
	release := make(chan struct{})
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		pa.builtInProcessMu.Lock()
		current := atomic.AddInt32(&active, 1)
		seen <- current
		<-release
		atomic.AddInt32(&active, -1)
		pa.builtInProcessMu.Unlock()
	}

	for range workerCount {
		wg.Add(1)
		go worker()
	}
	close(release)
	wg.Wait()
	close(seen)

	count := 0
	for current := range seen {
		count++
		if current != 1 {
			t.Fatalf("expected at most one process operation at a time, saw %d concurrent", current)
		}
	}
	if count != workerCount {
		t.Fatalf("expected %d workers to run, got %d", workerCount, count)
	}
}

func TestPrepareMilkyBuiltInConfigUnsupportedMode(t *testing.T) {
	dir := t.TempDir()
	pa := newMilkyBuiltInTestAdapter("unsupported")

	err := prepareMilkyBuiltInConfig(pa.EndPoint, filepath.Join(dir, "config.json"))
	if err == nil {
		t.Fatal("expected prepareMilkyBuiltInConfig to reject an unsupported mode")
	}
	if !strings.Contains(err.Error(), "不支持的内置 Milky 模式") {
		t.Fatalf("unexpected error: %v", err)
	}
}
