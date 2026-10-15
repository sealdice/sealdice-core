package singleflight_test

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"testing/synctest"

	sharedflight "sealdice-core/utils/singleflight"
)

type testResult struct {
	Value     string
	Timestamp int64
}

func TestGroupSharesInFlightResult(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		release := make(chan struct{})
		want := testResult{Value: "shared", Timestamp: 123}
		var group sharedflight.Group[testResult]
		run := func() (testResult, error) {
			calls.Add(1)
			<-release
			return want, nil
		}

		type outcome struct {
			result testResult
			err    error
		}
		results := make(chan outcome, 2)
		for range 2 {
			go func() {
				result, err := group.Do(t.Context(), "key", run)
				results <- outcome{result: result, err: err}
			}()
		}

		// Both callers must be waiting on the in-flight call before it finishes.
		synctest.Wait()
		close(release)
		for range 2 {
			got := <-results
			if got.err != nil {
				t.Errorf("shared call returned error: %v", got.err)
			}
			if !reflect.DeepEqual(got.result, want) {
				t.Errorf("shared result = %#v, want %#v", got.result, want)
			}
		}
		if got := calls.Load(); got != 1 {
			t.Fatalf("function called %d times, want 1", got)
		}
	})
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
		result, _ := group.Do(t.Context(), "key", run)
		first <- result
	}()
	<-started

	ctx, cancel := context.WithCancel(t.Context())
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
