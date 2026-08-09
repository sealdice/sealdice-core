package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/api"
	"sealdice-core/dice"
	"sealdice-core/utils/constant"
)

type supervisorTestOperator struct {
	closed atomic.Bool
	dbType string
}

func TestDiceServeEndpointCancelsDelayedQQStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := &dice.Dice{Logger: zap.NewNop().Sugar()}
	endpoint := &dice.EndPointInfo{EndPointInfoBase: dice.EndPointInfoBase{
		Platform: "QQ", ProtocolType: "unknown", Enable: true,
	}}
	done := make(chan struct{})
	go func() {
		diceServeEndpoint(ctx, d, endpoint)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("delayed QQ startup ignored Runtime cancellation")
	}
}

func (*supervisorTestOperator) Init(context.Context) error { return nil }
func (operator *supervisorTestOperator) Type() string {
	if operator.dbType == "" {
		return constant.SQLITE
	}
	return operator.dbType
}
func (*supervisorTestOperator) DBCheck()                             {}
func (*supervisorTestOperator) GetDataDB(constant.DBMode) *gorm.DB   { return nil }
func (*supervisorTestOperator) GetLogDB(constant.DBMode) *gorm.DB    { return nil }
func (*supervisorTestOperator) GetCensorDB(constant.DBMode) *gorm.DB { return nil }
func (operator *supervisorTestOperator) Close()                      { operator.closed.Store(true) }

func newSupervisorTestRuntime() *applicationRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &dice.DiceManager{
		Operator: &supervisorTestOperator{},
		IsReady:  true,
	}
	primary := &dice.Dice{Parent: manager, Logger: zap.NewNop().Sugar(), BaseConfig: dice.BaseConfig{Name: "default"}}
	primary.ImSession = &dice.IMSession{Parent: primary}
	manager.Dice = []*dice.Dice{primary}
	manager.SetRuntimeContext(ctx, cancel)
	return &applicationRuntime{manager: manager, ctx: ctx, cancel: cancel}
}

func assertPublishedRuntime(t *testing.T, expected *dice.DiceManager) {
	t.Helper()
	type runtimeSnapshot struct {
		manager *dice.DiceManager
		ok      bool
	}
	result := make(chan runtimeSnapshot, 1)
	go func() {
		var published *dice.DiceManager
		ok := api.WithCurrentRuntime(func(manager *dice.DiceManager) {
			published = manager
		})
		result <- runtimeSnapshot{manager: published, ok: ok}
	}()
	select {
	case snapshot := <-result:
		if !snapshot.ok || snapshot.manager != expected {
			t.Fatalf("published Runtime = %p, available=%v, want %p", snapshot.manager, snapshot.ok, expected)
		}
	case <-time.After(time.Second):
		t.Fatal("API Runtime gate remained locked")
	}
}

func assertRuntimeUnavailable(t *testing.T) {
	t.Helper()
	maintenanceCtx, maintenanceCancel := context.WithTimeout(context.Background(), time.Second)
	defer maintenanceCancel()
	if err := api.BeginRuntimeMaintenanceContext(maintenanceCtx); err != nil {
		t.Fatalf("API Runtime gate remained locked: %v", err)
	}
	api.EndRuntimeMaintenance()

	result := make(chan bool, 1)
	go func() {
		result <- api.WithCurrentRuntime(func(*dice.DiceManager) {})
	}()
	select {
	case available := <-result:
		if available {
			t.Fatal("API Runtime unexpectedly remained available")
		}
	case <-time.After(time.Second):
		t.Fatal("API Runtime gate remained locked")
	}
}

func TestRuntimeSupervisorStopDrainsAPIReadersBeforeClosingDatabase(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	runtimeInstance := newSupervisorTestRuntime()
	runtimeInstance.manager.IsReady = false
	operator := runtimeInstance.manager.Operator.(*supervisorTestOperator)
	supervisor.current = runtimeInstance
	api.ReplaceRuntime(runtimeInstance.manager)

	readerEntered := make(chan struct{})
	releaseReader := make(chan struct{})
	readerDone := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(releaseReader) })
	go func() {
		api.WithCurrentRuntime(func(*dice.DiceManager) {
			close(readerEntered)
			<-releaseReader
		})
		close(readerDone)
	}()
	select {
	case <-readerEntered:
	case <-time.After(time.Second):
		t.Fatal("API reader did not acquire the Runtime gate")
	}

	stopDone := make(chan error, 1)
	go func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
		defer stopCancel()
		stopDone <- supervisor.stop(stopCtx)
	}()
	select {
	case err := <-stopDone:
		t.Fatalf("stop returned before the API reader drained: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if operator.closed.Load() {
		t.Fatal("database closed while an API reader still held the Runtime gate")
	}

	releaseOnce.Do(func() { close(releaseReader) })
	select {
	case <-readerDone:
	case <-time.After(time.Second):
		t.Fatal("API reader did not finish")
	}
	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("stop did not finish after the API reader drained")
	}
	if !operator.closed.Load() {
		t.Fatal("database was not closed after API readers drained")
	}
	assertRuntimeUnavailable(t)
}

func TestRuntimeSupervisorStopTimesOutBeforeClosingDatabase(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	runtimeInstance := newSupervisorTestRuntime()
	runtimeInstance.manager.IsReady = false
	operator := runtimeInstance.manager.Operator.(*supervisorTestOperator)
	supervisor.current = runtimeInstance
	supervisor.state.Store("running")
	api.ReplaceRuntime(runtimeInstance.manager)

	readerEntered := make(chan struct{})
	releaseReader := make(chan struct{})
	readerDone := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releaseReader) })
		_ = supervisor.stop(context.Background())
	})
	go func() {
		api.WithCurrentRuntime(func(*dice.DiceManager) {
			close(readerEntered)
			<-releaseReader
		})
		close(readerDone)
	}()
	select {
	case <-readerEntered:
	case <-time.After(time.Second):
		t.Fatal("API reader did not acquire the Runtime gate")
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	startedAt := time.Now()
	err := supervisor.stop(stopCtx)
	stopCancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stop error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 500*time.Millisecond {
		t.Fatalf("stop ignored its context while draining API readers: %v", elapsed)
	}
	if operator.closed.Load() {
		t.Fatal("database closed after Runtime gate acquisition timed out")
	}
	if supervisor.current != runtimeInstance || supervisor.state.Load() != "running" {
		t.Fatal("Runtime changed after gate acquisition timed out")
	}
	assertPublishedRuntime(t, runtimeInstance.manager)

	releaseOnce.Do(func() { close(releaseReader) })
	select {
	case <-readerDone:
	case <-time.After(time.Second):
		t.Fatal("API reader did not finish")
	}
	retryCtx, retryCancel := context.WithTimeout(context.Background(), time.Second)
	defer retryCancel()
	if err = supervisor.stop(retryCtx); err != nil {
		t.Fatalf("stop retry failed after the API reader drained: %v", err)
	}
	if !operator.closed.Load() {
		t.Fatal("database remained open after the successful stop retry")
	}
	assertRuntimeUnavailable(t)
}

func TestRuntimeSupervisorRollsBackPreCommitPanics(t *testing.T) {
	for _, panicPhase := range []string{"prepare", "apply", "build"} {
		t.Run(panicPhase, func(t *testing.T) {
			supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
			configureRunnableRestore(supervisor, "pending")
			supervisor.committedFn = func() (bool, error) { return false, nil }
			oldRuntime := newSupervisorTestRuntime()
			fallback := newSupervisorTestRuntime()
			supervisor.current = oldRuntime
			api.ReplaceRuntime(oldRuntime.manager)
			supervisor.prepareFn = func(*dice.DiceManager) error {
				if panicPhase == "prepare" {
					panic("injected prepare panic")
				}
				return nil
			}
			supervisor.applyFn = func() error {
				if panicPhase == "apply" {
					panic("injected apply panic")
				}
				return nil
			}
			buildCalls := 0
			supervisor.buildFn = func() (*applicationRuntime, error) {
				buildCalls++
				if panicPhase == "build" && buildCalls == 1 {
					panic("injected build panic")
				}
				return fallback, nil
			}
			supervisor.commitFn = func() error { return nil }
			supervisor.markSucceededFn = func() error { return nil }
			rolledBack := false
			supervisor.rollbackFn = func(string) error {
				rolledBack = true
				return nil
			}
			supervisor.statusFn = func(string, string) error { return nil }
			t.Cleanup(func() { _ = supervisor.stop(context.Background()) })

			if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
				t.Fatal("restore unexpectedly succeeded after injected panic")
			}
			if !rolledBack {
				t.Fatal("pre-commit panic did not trigger rollback")
			}
			if supervisor.current != fallback || supervisor.state.Load() != "running" {
				t.Fatal("fallback Runtime was not published after pre-commit panic")
			}
			assertPublishedRuntime(t, fallback.manager)
		})
	}
}

func TestRuntimeSupervisorFailsClosedAfterCommitPanics(t *testing.T) {
	for _, panicPhase := range []string{"commit", "mark"} {
		t.Run(panicPhase, func(t *testing.T) {
			supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
			configureRunnableRestore(supervisor, "pending")
			supervisor.committedFn = func() (bool, error) { return false, nil }
			oldRuntime := newSupervisorTestRuntime()
			candidate := newSupervisorTestRuntime()
			supervisor.current = oldRuntime
			api.ReplaceRuntime(oldRuntime.manager)
			supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
			supervisor.applyFn = func() error { return nil }
			supervisor.buildFn = func() (*applicationRuntime, error) { return candidate, nil }
			supervisor.commitFn = func() error {
				if panicPhase == "commit" {
					panic("injected commit panic")
				}
				return nil
			}
			supervisor.markSucceededFn = func() error {
				panic("injected mark panic")
			}
			rolledBack := false
			supervisor.rollbackFn = func(string) error {
				rolledBack = true
				return nil
			}
			supervisor.statusFn = func(string, string) error { return nil }
			t.Cleanup(func() { _ = supervisor.stop(context.Background()) })

			if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
				t.Fatal("restore unexpectedly succeeded after injected panic")
			}
			if rolledBack {
				t.Fatal("restore rolled back after the commit boundary")
			}
			if supervisor.current != nil || supervisor.state.Load() != "degraded" {
				t.Fatal("post-commit panic did not leave Runtime degraded and unpublished")
			}
			if !candidate.manager.IsStopped() {
				t.Fatal("candidate Runtime was not stopped after post-commit panic")
			}
			assertRuntimeUnavailable(t)
		})
	}
}

func TestRuntimeSupervisorWorkerPanicFailsClosed(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	runtimeInstance := newSupervisorTestRuntime()
	supervisor.current = runtimeInstance
	api.ReplaceRuntime(runtimeInstance.manager)
	supervisor.statusFn = func(string, string) error { return nil }
	supervisor.restoreRunFn = func(context.Context, string) error {
		panic("injected outer worker panic")
	}
	t.Cleanup(func() { _ = supervisor.stop(context.Background()) })

	if err := supervisor.runRestoreQueueItem(context.Background(), "test-operation"); err == nil {
		t.Fatal("worker panic was not reported")
	}
	if supervisor.state.Load() != "degraded" {
		t.Fatal("worker panic did not mark Runtime degraded")
	}
	assertRuntimeUnavailable(t)
}

func TestRuntimeSupervisorMarksCommittedRestoreAfterPublish(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	candidate := newSupervisorTestRuntime()
	supervisor.buildFn = func() (*applicationRuntime, error) { return candidate, nil }
	supervisor.committedFn = func() (bool, error) { return true, nil }
	marked := false
	supervisor.markSucceededFn = func() error {
		if supervisor.current != candidate {
			t.Fatal("committed restore was marked before supervisor publication")
		}
		published := api.WithCurrentRuntime(func(manager *dice.DiceManager) {
			if manager != candidate.manager {
				t.Fatal("API published a different Runtime before marking restore success")
			}
		})
		if !published {
			t.Fatal("committed restore was marked before API publication")
		}
		marked = true
		return nil
	}
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	manager, err := supervisor.start()
	if err != nil {
		t.Fatal(err)
	}
	if manager != candidate.manager || !marked {
		t.Fatal("committed restore was not marked after publishing its Runtime")
	}
}

func TestRuntimeSupervisorRejectsExternalDatabaseBeforeQuiesce(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	runtimeInstance := newSupervisorTestRuntime()
	operator := runtimeInstance.manager.Operator.(*supervisorTestOperator)
	operator.dbType = constant.MYSQL
	supervisor.current = runtimeInstance
	prepareCalled := false
	supervisor.prepareFn = func(*dice.DiceManager) error {
		prepareCalled = true
		return nil
	}
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
		t.Fatal("restore() unexpectedly accepted an external database Runtime")
	}
	if prepareCalled {
		t.Fatal("restore preparation ran before rejecting the external database")
	}
	if !runtimeInstance.manager.IsReady {
		t.Fatal("external database rejection quiesced the running Runtime")
	}
	if operator.closed.Load() {
		t.Fatal("external database rejection closed the running database")
	}
	if supervisor.current != runtimeInstance {
		t.Fatal("external database rejection replaced the running Runtime")
	}
}

func TestRuntimeSupervisorRejectsEmptyCandidateBeforeCommit(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	supervisor.current = newSupervisorTestRuntime()
	invalid := newSupervisorTestRuntime()
	invalid.manager.Dice = nil
	supervisor.buildFn = func() (*applicationRuntime, error) { return invalid, nil }
	supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
	supervisor.applyFn = func() error { return nil }
	committed := false
	supervisor.commitFn = func() error {
		committed = true
		return nil
	}
	supervisor.markSucceededFn = func() error { return nil }
	rolledBack := false
	supervisor.rollbackFn = func(string) error {
		rolledBack = true
		return nil
	}
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
		t.Fatal("restore() unexpectedly accepted an empty candidate")
	}
	if committed {
		t.Fatal("empty candidate reached the commit boundary")
	}
	if !rolledBack {
		t.Fatal("empty candidate failure was not rolled back")
	}
}

func TestRuntimeSupervisorRejectsUnsafeDiceNameBeforeCommit(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	supervisor.current = newSupervisorTestRuntime()
	invalid := newSupervisorTestRuntime()
	invalid.manager.Dice[0].BaseConfig.Name = "../escape"
	supervisor.buildFn = func() (*applicationRuntime, error) { return invalid, nil }
	supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
	supervisor.applyFn = func() error { return nil }
	committed := false
	supervisor.commitFn = func() error {
		committed = true
		return nil
	}
	supervisor.markSucceededFn = func() error { return nil }
	supervisor.rollbackFn = func(string) error { return nil }
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
		t.Fatal("restore() unexpectedly accepted an unsafe Dice name")
	}
	if committed {
		t.Fatal("unsafe Dice name reached the commit boundary")
	}
}

func configureRunnableRestore(supervisor *runtimeSupervisor, state string) {
	supervisor.runnableIDFn = func() (string, error) { return "test-operation", nil }
	supervisor.getStatusFn = func() dice.RestoreStatus {
		return dice.RestoreStatus{State: state, OperationID: "test-operation", SourceName: "source.zip"}
	}
}

func TestRuntimeSupervisorRollsBackAndRebuildsAfterBuildFailure(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	supervisor.current = newSupervisorTestRuntime()
	fallback := newSupervisorTestRuntime()
	buildCalls := 0
	supervisor.buildFn = func() (*applicationRuntime, error) {
		buildCalls++
		if buildCalls == 1 {
			return nil, errors.New("injected start failure")
		}
		return fallback, nil
	}
	supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
	supervisor.applyFn = func() error { return nil }
	supervisor.commitFn = func() error { return nil }
	supervisor.markSucceededFn = func() error { return nil }
	rolledBack := false
	supervisor.rollbackFn = func(string) error {
		rolledBack = true
		return nil
	}
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	err := supervisor.restore(context.Background(), "test-operation")
	if err == nil {
		t.Fatal("restore() unexpectedly succeeded")
	}
	if !rolledBack || buildCalls != 2 {
		t.Fatalf("rollback=%v buildCalls=%d, want true/2", rolledBack, buildCalls)
	}
	if supervisor.current != fallback || supervisor.state.Load() != "running" {
		t.Fatal("fallback Runtime was not installed")
	}
}

func TestRuntimeSupervisorStaysDegradedWhenRollbackFails(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	supervisor.current = newSupervisorTestRuntime()
	buildCalls := 0
	supervisor.buildFn = func() (*applicationRuntime, error) {
		buildCalls++
		return nil, errors.New("injected start failure")
	}
	supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
	supervisor.applyFn = func() error { return nil }
	supervisor.commitFn = func() error { return nil }
	supervisor.markSucceededFn = func() error { return nil }
	supervisor.rollbackFn = func(string) error { return errors.New("injected rollback failure") }
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() { api.ReplaceRuntime(nil) })

	if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
		t.Fatal("restore() unexpectedly succeeded")
	}
	if buildCalls != 1 {
		t.Fatalf("build calls = %d, want 1", buildCalls)
	}
	if supervisor.current != nil || supervisor.state.Load() != "degraded" {
		t.Fatal("Runtime was rebuilt after rollback failure")
	}
}

func TestRuntimeSupervisorDoesNotRollbackAfterCommit(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	supervisor.current = newSupervisorTestRuntime()
	candidate := newSupervisorTestRuntime()
	supervisor.buildFn = func() (*applicationRuntime, error) { return candidate, nil }
	supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
	supervisor.applyFn = func() error { return nil }
	supervisor.commitFn = func() error { return nil }
	supervisor.markSucceededFn = func() error { return errors.New("injected mark failure") }
	rolledBack := false
	supervisor.rollbackFn = func(string) error {
		rolledBack = true
		return nil
	}
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
		t.Fatal("restore() unexpectedly succeeded")
	}
	if rolledBack {
		t.Fatal("restore rolled back after the commit boundary")
	}
	if supervisor.current != candidate || supervisor.state.Load() != "degraded" {
		t.Fatal("committed Runtime was not retained in degraded state")
	}
}

func TestRuntimeSupervisorDoesNotRollbackWhenCommitResultIsAmbiguous(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	supervisor.current = newSupervisorTestRuntime()
	candidate := newSupervisorTestRuntime()
	supervisor.buildFn = func() (*applicationRuntime, error) { return candidate, nil }
	supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
	supervisor.applyFn = func() error { return nil }
	supervisor.commitFn = func() error { return errors.New("injected commit fsync failure") }
	supervisor.markSucceededFn = func() error { return nil }
	rolledBack := false
	supervisor.rollbackFn = func(string) error {
		rolledBack = true
		return nil
	}
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() { api.ReplaceRuntime(nil) })

	if err := supervisor.restore(context.Background(), "test-operation"); err == nil {
		t.Fatal("restore() unexpectedly succeeded")
	}
	if rolledBack {
		t.Fatal("restore rolled back after an ambiguous commit result")
	}
	if supervisor.current != nil || supervisor.state.Load() != "degraded" {
		t.Fatal("ambiguous commit result did not leave Runtime unavailable and degraded")
	}
	if !candidate.manager.IsStopped() {
		t.Fatal("candidate Runtime was not discarded after commit uncertainty")
	}
}

func TestRuntimeSupervisorPreparesBeforeFinalize(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "pending")
	oldRuntime := newSupervisorTestRuntime()
	oldOperator := oldRuntime.manager.Operator.(*supervisorTestOperator)
	supervisor.current = oldRuntime
	candidate := newSupervisorTestRuntime()
	var order []string
	supervisor.prepareFn = func(*dice.DiceManager) error {
		if oldOperator.closed.Load() {
			t.Fatal("database closed before PrepareScheduledRestore")
		}
		order = append(order, "prepare")
		return nil
	}
	supervisor.applyFn = func() error {
		if !oldOperator.closed.Load() {
			t.Fatal("database still open during ApplyScheduledRestore")
		}
		order = append(order, "apply")
		return nil
	}
	supervisor.buildFn = func() (*applicationRuntime, error) {
		order = append(order, "build")
		return candidate, nil
	}
	supervisor.commitFn = func() error {
		order = append(order, "commit")
		return nil
	}
	supervisor.markSucceededFn = func() error {
		order = append(order, "mark")
		return nil
	}
	supervisor.rollbackFn = func(string) error { return nil }
	supervisor.statusFn = func(string, string) error { return nil }
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	if err := supervisor.restore(context.Background(), "test-operation"); err != nil {
		t.Fatal(err)
	}
	want := []string{"prepare", "apply", "build", "commit", "mark"}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Fatalf("restore order = %v, want %v", order, want)
	}
}

func TestRuntimeSupervisorResumesPendingBeforeStatusWasWritten(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	configureRunnableRestore(supervisor, "idle")
	supervisor.current = newSupervisorTestRuntime()
	candidate := newSupervisorTestRuntime()
	supervisor.prepareFn = func(*dice.DiceManager) error { return nil }
	supervisor.applyFn = func() error { return nil }
	supervisor.buildFn = func() (*applicationRuntime, error) { return candidate, nil }
	supervisor.commitFn = func() error { return nil }
	supervisor.markSucceededFn = func() error { return nil }
	supervisor.rollbackFn = func(string) error { return nil }
	supervisor.statusFn = func(string, string) error { return nil }
	completed := make(chan error, 1)
	restoreRun := supervisor.restoreRunFn
	supervisor.restoreRunFn = func(ctx context.Context, operationID string) error {
		err := restoreRun(ctx, operationID)
		completed <- err
		return err
	}
	t.Cleanup(func() {
		api.ReplaceRuntime(nil)
		_ = supervisor.stop(context.Background())
	})

	if err := supervisor.enqueueRunnableScheduledRestore(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-completed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the startup restore queue")
	}
	supervisor.mu.Lock()
	current := supervisor.current
	supervisor.mu.Unlock()
	if current != candidate || supervisor.state.Load() != "running" {
		t.Fatal("runnable pending restore was not resumed from an idle external status")
	}
}

func TestRuntimeSupervisorRestoreQueueCoalescesActiveOperation(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	started := make(chan struct{})
	release := make(chan struct{})
	completed := make(chan struct{})
	var calls atomic.Int32
	var startedOnce sync.Once
	var completedOnce sync.Once
	supervisor.restoreRunFn = func(ctx context.Context, operationID string) error {
		if operationID != "same-operation" {
			t.Errorf("operationID = %q", operationID)
		}
		calls.Add(1)
		startedOnce.Do(func() { close(started) })
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
			completedOnce.Do(func() { close(completed) })
			return nil
		}
	}
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		_ = supervisor.stop(context.Background())
	})

	if !supervisor.enqueueRestore("same-operation") {
		t.Fatal("initial enqueue was rejected")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("restore worker did not start")
	}
	for range 100 {
		if !supervisor.enqueueRestore("same-operation") {
			t.Fatal("coalesced enqueue was rejected")
		}
	}
	close(release)
	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("restore worker did not complete")
	}
	time.Sleep(10 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("restore calls = %d, want 1", got)
	}
}

func TestRuntimeSupervisorDropsStaleQueuedOperation(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
	supervisor.current = newSupervisorTestRuntime()
	supervisor.getStatusFn = func() dice.RestoreStatus {
		return dice.RestoreStatus{State: "pending", OperationID: "new-operation"}
	}
	supervisor.runnableIDFn = func() (string, error) { return "new-operation", nil }
	prepared := false
	supervisor.prepareFn = func(*dice.DiceManager) error {
		prepared = true
		return nil
	}
	supervisor.statusFn = func(string, string) error { return nil }

	if err := supervisor.restore(context.Background(), "stale-operation"); err != nil {
		t.Fatal(err)
	}
	if prepared {
		t.Fatal("stale queued operation executed the current pending restore")
	}
}
