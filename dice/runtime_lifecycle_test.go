package dice //nolint:testpackage // Tests exercise unexported Runtime task coordination.

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"gorm.io/gorm"

	"sealdice-core/utils/constant"
)

type lifecycleTestOperator struct {
	closed atomic.Bool
}

func TestDiceManagerStopRetriesAfterSharedPhaseTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	operator := &lifecycleTestOperator{}
	manager := &DiceManager{Operator: operator}
	manager.SetRuntimeContext(ctx, cancel)
	release := make(chan struct{})
	manager.goRuntime(func(context.Context) { <-release })

	firstCtx, firstCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer firstCancel()
	firstErr := manager.Stop(firstCtx)
	if !errors.Is(firstErr, context.DeadlineExceeded) {
		t.Fatalf("first Stop() error = %v, want deadline exceeded", firstErr)
	}
	if operator.closed.Load() {
		t.Fatal("Finalize ran before Quiesce completed")
	}
	close(release)

	secondCtx, secondCancel := context.WithTimeout(context.Background(), time.Second)
	defer secondCancel()
	if err := manager.Stop(secondCtx); err != nil {
		t.Fatalf("second Stop() should wait for the shared phase result, got: %v", err)
	}
	if !operator.closed.Load() {
		t.Fatal("Finalize did not run after the shared Quiesce completed")
	}
}

func TestLifecyclePhaseRetriesAfterCallerTimeout(t *testing.T) {
	var phase lifecyclePhase
	release := make(chan struct{})
	firstCtx, firstCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer firstCancel()
	if err := phase.run(firstCtx, func(context.Context) error {
		<-release
		return nil
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first run error = %v, want deadline exceeded", err)
	}
	close(release)

	secondCtx, secondCancel := context.WithTimeout(context.Background(), time.Second)
	defer secondCancel()
	if err := phase.run(secondCtx, func(context.Context) error {
		t.Fatal("lifecycle work ran more than once")
		return nil
	}); err != nil {
		t.Fatalf("shared phase result = %v, want nil after work completed", err)
	}
}

func TestLifecyclePhaseKeepsReportingTimeoutWhileWorkIsStillRunning(t *testing.T) {
	var phase lifecyclePhase
	release := make(chan struct{})
	firstCtx, firstCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer firstCancel()
	if err := phase.run(firstCtx, func(context.Context) error {
		<-release
		return nil
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("first run error = %v, want deadline exceeded", err)
	}

	secondCtx, secondCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer secondCancel()
	if err := phase.run(secondCtx, func(context.Context) error {
		t.Fatal("lifecycle work ran more than once")
		return nil
	}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second run error = %v, want deadline exceeded while work is unfinished", err)
	}
	close(release)

	finalCtx, finalCancel := context.WithTimeout(context.Background(), time.Second)
	defer finalCancel()
	if err := phase.run(finalCtx, func(context.Context) error {
		t.Fatal("lifecycle work ran more than once")
		return nil
	}); err != nil {
		t.Fatalf("final run error = %v, want nil after work completed", err)
	}
}

func TestFinalizeKeepsDatabaseOpenWhenJSLoopDoesNotStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	operator := &lifecycleTestOperator{}
	manager := &DiceManager{Operator: operator}
	manager.SetRuntimeContext(ctx, cancel)
	loopDone := make(chan struct{})
	instance := &Dice{Parent: manager, ExtLoopManager: NewJsLoopManager(), jsLoopDone: loopDone}
	manager.Dice = []*Dice{instance}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stopCancel()
	err := manager.Stop(stopCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop() error = %v, want deadline exceeded", err)
	}
	if operator.closed.Load() {
		t.Fatal("database was closed while the JS Runtime could still be running")
	}
	close(loopDone)
}

func TestValidateDiceConfigNames(t *testing.T) {
	valid := []BaseConfig{{Name: "default"}, {Name: "测试 骰"}, {Name: ".private"}}
	if err := ValidateDiceConfigNames(valid); err != nil {
		t.Fatalf("valid Dice names rejected: %v", err)
	}

	alwaysInvalid := [][]BaseConfig{
		{{Name: ""}},
		{{Name: "."}},
		{{Name: ".."}},
		{{Name: "../escape"}},
		{{Name: `folder\\child`}},
		{{Name: "bad\x00name"}},
	}
	for _, configs := range alwaysInvalid {
		if err := ValidateDiceConfigNames(configs); err == nil {
			t.Fatalf("unsafe Dice names accepted: %#v", configs)
		}
	}

	portableInvalid := [][]BaseConfig{
		{{Name: "C:drive"}},
		{{Name: "trailing."}},
		{{Name: "trailing "}},
		{{Name: "CON"}},
		{{Name: "con.txt"}},
		{{Name: "LPT9"}},
		{{Name: "bad?name"}},
		{{Name: "Alpha"}, {Name: "alpha"}},
	}
	for _, configs := range portableInvalid {
		if err := ValidateDiceConfigNamesPortable(configs); err == nil {
			t.Fatalf("portable-unsafe Dice names accepted: %#v", configs)
		}
	}
}

func (*lifecycleTestOperator) Init(context.Context) error           { return nil }
func (*lifecycleTestOperator) Type() string                         { return "test" }
func (*lifecycleTestOperator) DBCheck()                             {}
func (*lifecycleTestOperator) GetDataDB(constant.DBMode) *gorm.DB   { return nil }
func (*lifecycleTestOperator) GetLogDB(constant.DBMode) *gorm.DB    { return nil }
func (*lifecycleTestOperator) GetCensorDB(constant.DBMode) *gorm.DB { return nil }
func (operator *lifecycleTestOperator) Close()                      { operator.closed.Store(true) }

func TestDiceManagerStopCancelsWaitsAndIsIdempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	operator := &lifecycleTestOperator{}
	manager := &DiceManager{Operator: operator}
	manager.SetRuntimeContext(ctx, cancel)
	taskStopped := make(chan struct{})
	manager.goRuntime(func(ctx context.Context) {
		<-ctx.Done()
		close(taskStopped)
	})

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := manager.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-taskStopped:
	default:
		t.Fatal("runtime task was not awaited")
	}
	if !operator.closed.Load() || !manager.IsStopped() {
		t.Fatal("runtime resources were not closed")
	}
	if err := manager.Stop(stopCtx); err != nil {
		t.Fatalf("second Stop() failed: %v", err)
	}
}

func TestOnebotAsyncTaskIsDrainedByRuntimeGate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &DiceManager{}
	manager.SetRuntimeContext(ctx, cancel)
	instance := &Dice{Parent: manager}
	session := &IMSession{Parent: instance}
	endpoint := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Session: session}}
	pool, err := ants.NewPool(1)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Release()
	adapter := &PlatformAdapterOnebot{EndPoint: endpoint, antPool: pool}

	started := make(chan struct{})
	release := make(chan struct{})
	if err := adapter.submitAsync(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("OneBot async task did not start")
	}

	manager.runtimeMu.Lock()
	manager.runtimeClosing = true
	manager.runtimeMu.Unlock()
	cancel()
	if err := adapter.submitAsync(func() {}); err == nil {
		t.Fatal("OneBot accepted an async task after Runtime closing began")
	}

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer waitCancel()
	if err := waitRuntimeTasks(waitCtx, &manager.runtimeWG); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitRuntimeTasks() error = %v, want deadline exceeded", err)
	}
	close(release)
	drainCtx, drainCancel := context.WithTimeout(context.Background(), time.Second)
	defer drainCancel()
	if err := waitRuntimeTasks(drainCtx, &manager.runtimeWG); err != nil {
		t.Fatalf("OneBot async task was not drained: %v", err)
	}
}
