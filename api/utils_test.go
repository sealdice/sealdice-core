package api

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestNetworkHealthCheckerSharesInFlightResult(t *testing.T) {
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

	want := networkHealthResult{
		Total: 5,
		Ok:    []string{"seal", "github"},
		Targets: []networkHealthTarget{
			{Target: "seal", Ok: true, Duration: time.Second},
		},
		Timestamp: 123,
	}
	checker := networkHealthChecker{run: func() networkHealthResult {
		calls.Add(1)
		close(started)
		<-release
		return want
	}}

	first := make(chan networkHealthResult, 1)
	go func() {
		result, _ := checker.check(context.Background())
		first <- result
	}()
	<-started

	timer := time.AfterFunc(20*time.Millisecond, func() { close(release) })
	defer timer.Stop()
	second, err := checker.check(context.Background())
	if err != nil {
		t.Fatalf("second check returned error: %v", err)
	}
	firstResult := <-first

	if got := calls.Load(); got != 1 {
		t.Fatalf("run called %d times, want 1", got)
	}
	if !reflect.DeepEqual(firstResult, want) {
		t.Fatalf("first result = %#v, want %#v", firstResult, want)
	}
	if !reflect.DeepEqual(second, want) {
		t.Fatalf("second result = %#v, want %#v", second, want)
	}
}

func TestNetworkHealthCheckerCanceledWaiterDoesNotCancelSharedCheck(t *testing.T) {
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

	want := networkHealthResult{Total: 5, Timestamp: 456}
	checker := networkHealthChecker{run: func() networkHealthResult {
		calls.Add(1)
		close(started)
		<-release
		return want
	}}

	first := make(chan networkHealthResult, 1)
	go func() {
		result, _ := checker.check(context.Background())
		first <- result
	}()
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := checker.check(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter error = %v, want context.Canceled", err)
	}

	close(release)
	if result := <-first; !reflect.DeepEqual(result, want) {
		t.Fatalf("shared result = %#v, want %#v", result, want)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("run called %d times, want 1", got)
	}
}
