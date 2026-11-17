package realtime_test

import (
	"sync"
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

func TestSlowStateSubscriberClosesWithoutBlockingOthers(t *testing.T) {
	bus := realtime.NewBus()
	slow, unsubscribeSlow := bus.Subscribe(1)
	defer unsubscribeSlow()
	fast, unsubscribeFast := bus.Subscribe(4)
	defer unsubscribeFast()
	first := realtime.Event{Name: realtime.EventSystemReady}
	bus.Publish(first)
	done := make(chan struct{})
	go func() {
		bus.Publish(realtime.Event{Name: realtime.EventIMConnectionList})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow subscriber blocked publication")
	}
	<-slow // The buffered event is still readable before the stream closes.
	select {
	case _, ok := <-slow:
		if ok {
			t.Fatal("slow stream must reconnect for a fresh snapshot")
		}
	default:
		t.Fatal("slow stream remained open after overflow")
	}
	if (<-fast).Name != first.Name || (<-fast).Name != realtime.EventIMConnectionList {
		t.Fatal("healthy subscriber lost state")
	}
	unsubscribeSlow() // Safe after overflow closed the subscription.
}

func TestLogBackpressureDoesNotCloseSubscription(t *testing.T) {
	bus := realtime.NewBus()
	ch, unsubscribe := bus.Subscribe(1)
	defer unsubscribe()
	bus.Publish(realtime.Event{Name: realtime.EventLogsAppend})
	bus.Publish(realtime.Event{Name: realtime.EventLogsAppend})
	<-ch
	bus.Publish(realtime.Event{Name: realtime.EventSystemReady})
	if evt, ok := <-ch; !ok || evt.Name != realtime.EventSystemReady {
		t.Fatal("log overflow closed or blocked a healthy stream")
	}
}

func TestPublishAndUnsubscribeCanRace(t *testing.T) {
	bus := realtime.NewBus()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 1000 {
			bus.Publish(realtime.Event{Name: realtime.EventLogsAppend})
		}
	}()
	go func() {
		defer wg.Done()
		for range 1000 {
			ch, unsubscribe := bus.Subscribe(1)
			unsubscribe()
			unsubscribe()
			for range ch {
			}
		}
	}()
	wg.Wait()
}
