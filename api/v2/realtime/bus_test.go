package realtime_test

import (
	"testing"
	"time"

	realtime "sealdice-core/api/v2/realtime"
)

func TestBusPublishesToSubscribers(t *testing.T) {
	bus := realtime.NewBus()
	ch, unsubscribe := bus.Subscribe(1)
	defer unsubscribe()

	bus.Publish(realtime.Event{
		Name:    realtime.EventSystemReady,
		Payload: map[string]any{"ok": true},
	})

	select {
	case evt := <-ch:
		if evt.Name != realtime.EventSystemReady {
			t.Fatalf("event name = %q, want %q", evt.Name, realtime.EventSystemReady)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
