package sender_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sealdice-core/dice/sender"
)

func TestQueueSameKeyFIFO(t *testing.T) {
	q := sender.New()
	defer q.Close()

	const n = 200
	var mu sync.Mutex
	got := make([]int, 0, n)
	done := make(chan struct{})

	for i := range n {
		q.Enqueue("group:1", func() {
			mu.Lock()
			got = append(got, i)
			if len(got) == n {
				close(done)
			}
			mu.Unlock()
		})
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for jobs")
	}

	mu.Lock()
	defer mu.Unlock()
	for want, v := range got {
		if v != want {
			t.Fatalf("FIFO violated: position %d has job %d", want, v)
		}
	}
}

func TestQueueDifferentKeysDoNotBlock(t *testing.T) {
	q := sender.New()
	defer q.Close()

	aStarted := make(chan struct{})
	releaseA := make(chan struct{})
	bDone := make(chan struct{})

	q.Enqueue("A", func() {
		close(aStarted)
		<-releaseA
	})
	<-aStarted

	q.Enqueue("B", func() { close(bDone) })

	select {
	case <-bDone:
	case <-time.After(2 * time.Second):
		t.Fatal("key B was blocked by a busy key A")
	}
	close(releaseA)
}

func TestQueueCloseDrainsAndWaits(t *testing.T) {
	q := sender.New()

	saw := make(chan struct{})
	q.Enqueue("k", func() { close(saw) })
	q.Close()

	select {
	case <-saw:
	default:
		t.Fatal("Close returned before pending job ran")
	}
}

func TestQueueConcurrentEnqueueProcessesAll(t *testing.T) {
	q := sender.New()

	const goroutines = 8
	const perGoroutine = 500
	var count atomic.Int64
	var wg sync.WaitGroup

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				q.Enqueue("k", func() { count.Add(1) })
			}
		}()
	}
	wg.Wait()
	q.Close()

	if got := count.Load(); got != goroutines*perGoroutine {
		t.Fatalf("processed %d jobs, want %d", got, goroutines*perGoroutine)
	}
}
