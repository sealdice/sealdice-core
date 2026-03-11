package throttle

import (
	"sync/atomic"
	"testing"
	"time"
)

// --- Smoke tests ---

func TestDo_ExecutesOnFirstCall(t *testing.T) {
	id := "test_first_call_" + t.Name()
	var called int32
	Do(id, time.Second, func() { atomic.AddInt32(&called, 1) })
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected function to be called once on first invocation, got %d", called)
	}
}

func TestDo_ThrottledWithinInterval(t *testing.T) {
	id := "test_throttled_" + t.Name()
	var called int32

	Do(id, time.Hour, func() { atomic.AddInt32(&called, 1) })
	Do(id, time.Hour, func() { atomic.AddInt32(&called, 1) })
	Do(id, time.Hour, func() { atomic.AddInt32(&called, 1) })

	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("expected function called exactly once within throttle interval, got %d", called)
	}
}

func TestDo_AllowsCallAfterInterval(t *testing.T) {
	id := "test_after_interval_" + t.Name()
	var called int32

	Do(id, time.Millisecond, func() { atomic.AddInt32(&called, 1) })
	time.Sleep(5 * time.Millisecond)
	Do(id, time.Millisecond, func() { atomic.AddInt32(&called, 1) })

	if atomic.LoadInt32(&called) != 2 {
		t.Errorf("expected function called twice (once before and once after interval), got %d", called)
	}
}

func TestDo_DifferentIDsAreIndependent(t *testing.T) {
	idA := "test_id_a_" + t.Name()
	idB := "test_id_b_" + t.Name()
	var calledA, calledB int32

	Do(idA, time.Hour, func() { atomic.AddInt32(&calledA, 1) })
	Do(idB, time.Hour, func() { atomic.AddInt32(&calledB, 1) })

	if atomic.LoadInt32(&calledA) != 1 {
		t.Errorf("idA: expected 1 call, got %d", calledA)
	}
	if atomic.LoadInt32(&calledB) != 1 {
		t.Errorf("idB: expected 1 call, got %d", calledB)
	}
}

func TestDo_ZeroIntervalAlwaysExecutes(t *testing.T) {
	id := "test_zero_interval_" + t.Name()
	var called int32
	for range 5 {
		Do(id, 0, func() { atomic.AddInt32(&called, 1) })
	}
	if atomic.LoadInt32(&called) != 5 {
		t.Errorf("expected 5 calls with zero interval, got %d", called)
	}
}

// --- Benchmark tests ---

func BenchmarkDo_Throttled(b *testing.B) {
	id := "bench_throttled"
	// prime the throttle so subsequent calls are all no-ops
	Do(id, time.Hour, func() {})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Do(id, time.Hour, func() {})
	}
}

func BenchmarkDo_Unthrottled(b *testing.B) {
	// Each iteration uses a unique id to avoid throttling
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Do("bench_unthrottled_unique", 0, func() {})
	}
}
