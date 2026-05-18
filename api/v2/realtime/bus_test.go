package realtime

import (
	"testing"
	"time"
)

func TestBusPublishesToSubscribers(t *testing.T) {
	bus := NewBus()
	ch, unsubscribe := bus.Subscribe(1)
	defer unsubscribe()

	bus.Publish(Event{
		Name:    EventSystemReady,
		Payload: map[string]any{"ok": true},
	})

	select {
	case evt := <-ch:
		if evt.Name != EventSystemReady {
			t.Fatalf("event name = %q, want %q", evt.Name, EventSystemReady)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
