package singleflight_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	sharedflight "sealdice-core/utils/singleflight"
)

type testResult struct {
	Value     string
	Timestamp int64
}

func TestGroupSharesInFlightResult(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	want := testResult{Value: "shared", Timestamp: 123}
	var group sharedflight.Group[testResult]
	run := func() (testResult, error) {
		calls.Add(1)
		close(started)
		<-release
		return want, nil
	}

	first := make(chan testResult, 1)
	go func() {
		result, _ := group.Do(context.Background(), "key", run)
		first <- result
	}()
	<-started

	timer := time.AfterFunc(20*time.Millisecond, func() { close(release) })
	defer timer.Stop()
	second, err := group.Do(context.Background(), "key", run)
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	firstResult := <-first

	if got := calls.Load(); got != 1 {
		t.Fatalf("function called %d times, want 1", got)
	}
	if !reflect.DeepEqual(firstResult, want) {
		t.Fatalf("first result = %#v, want %#v", firstResult, want)
	}
	if !reflect.DeepEqual(second, want) {
		t.Fatalf("second result = %#v, want %#v", second, want)
	}
}

func TestGroupCanceledWaiterDoesNotCancelSharedCall(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	want := testResult{Value: "completed", Timestamp: 456}
	var group sharedflight.Group[testResult]
	run := func() (testResult, error) {
		calls.Add(1)
		close(started)
		<-release
		return want, nil
	}

	first := make(chan testResult, 1)
	go func() {
		result, _ := group.Do(context.Background(), "key", run)
		first <- result
	}()
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := group.Do(ctx, "key", run); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter error = %v, want context.Canceled", err)
	}

	close(release)
	if result := <-first; !reflect.DeepEqual(result, want) {
		t.Fatalf("shared result = %#v, want %#v", result, want)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("function called %d times, want 1", got)
	}
}
