package dice

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
)

type lifecycleTestOperator struct {
	closed atomic.Bool
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
