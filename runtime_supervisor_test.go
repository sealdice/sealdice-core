package main

import (
	"context"
	"errors"
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

func (*supervisorTestOperator) Init(context.Context) error           { return nil }
func (*supervisorTestOperator) Type() string                         { return "test" }
func (*supervisorTestOperator) DBCheck()                             {}
func (*supervisorTestOperator) GetDataDB(constant.DBMode) *gorm.DB   { return nil }
func (*supervisorTestOperator) GetLogDB(constant.DBMode) *gorm.DB    { return nil }
func (*supervisorTestOperator) GetCensorDB(constant.DBMode) *gorm.DB { return nil }
func (operator *supervisorTestOperator) Close()                      { operator.closed.Store(true) }

func newSupervisorTestRuntime() *applicationRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &dice.DiceManager{
		Operator: &supervisorTestOperator{},
		Dice:     []*dice.Dice{{}},
	}
	manager.SetRuntimeContext(ctx, cancel)
	return &applicationRuntime{manager: manager, ctx: ctx, cancel: cancel}
}

func TestRuntimeSupervisorRollsBackAndRebuildsAfterStartFailure(t *testing.T) {
	supervisor := newRuntimeSupervisor(runtimeBuildOptions{})
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
	supervisor.applyFn = func() error { return nil }
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

	err := supervisor.restore(context.Background())
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
