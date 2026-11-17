package logger_test

import (
	"testing"
	"time"

	"sealdice-core/logger"
)

func TestUIWriterBroadcastsParsedLogItemsToSubscribers(t *testing.T) {
	writer := logger.NewUIWriter()
	ch, unsubscribe := writer.Subscribe()
	defer unsubscribe()

	raw := []byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"hello"}`)
	if _, err := writer.Write(raw); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	select {
	case item := <-ch:
		if item.Level != "info" {
			t.Fatalf("Level = %q, want info", item.Level)
		}
		if item.Module != "core" {
			t.Fatalf("Module = %q, want core", item.Module)
		}
		if item.Msg != "hello" {
			t.Fatalf("Msg = %q, want hello", item.Msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast log item")
	}
}

func TestUIWriterSnapshotAfterUsesMonotonicSequence(t *testing.T) {
	writer := logger.NewUIWriter()
	writer.LogLimit = 2
	for _, msg := range []string{"first", "second", "third"} {
		_, _ = writer.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"` + msg + `"}`))
	}

	items, latest, epoch, reset := writer.SnapshotAfter(0, "")
	if !reset || latest != 3 || epoch == "" || len(items) != 2 {
		t.Fatalf("SnapshotAfter(0) = %+v, latest=%d, epoch=%q, reset=%v", items, latest, epoch, reset)
	}
	if items[0].Seq != 2 || items[1].Seq != 3 {
		t.Fatalf("retained sequence = %d, %d; want 2, 3", items[0].Seq, items[1].Seq)
	}

	items, latest, epoch, reset = writer.SnapshotAfter(1, epoch)
	if reset || latest != 3 || len(items) != 2 || items[0].Seq != 2 || items[1].Seq != 3 {
		t.Fatalf("SnapshotAfter(1) = %+v, latest=%d, reset=%v", items, latest, reset)
	}

	items, latest, epoch, reset = writer.SnapshotAfter(2, epoch)
	if reset || latest != 3 || len(items) != 1 || items[0].Seq != 3 {
		t.Fatalf("SnapshotAfter(2) = %+v, latest=%d, reset=%v", items, latest, reset)
	}

	items, latest, epoch, reset = writer.SnapshotAfter(3, epoch)
	if reset || latest != 3 || len(items) != 0 {
		t.Fatalf("SnapshotAfter(3) = %+v, latest=%d, reset=%v", items, latest, reset)
	}

	items, latest, _, reset = writer.SnapshotAfter(99, epoch)
	if !reset || latest != 3 || len(items) != 2 {
		t.Fatalf("SnapshotAfter(99) = %+v, latest=%d, reset=%v; want full retained window", items, latest, reset)
	}

	items, latest, _, reset = writer.SnapshotAfter(1, "old-process")
	if !reset || latest != 3 || len(items) != 2 {
		t.Fatalf("SnapshotAfter(old process) = %+v, latest=%d, reset=%v; want full retained window", items, latest, reset)
	}
}

func TestUIWriterSnapshotIsDetachedFromInternalItems(t *testing.T) {
	writer := logger.NewUIWriter()
	_, _ = writer.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"first"}`))

	snapshot := writer.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("len(snapshot) = %d, want 1", len(snapshot))
	}
	snapshot[0].Msg = "mutated"

	again := writer.Snapshot()
	if again[0].Msg != "first" {
		t.Fatalf("internal item mutated through snapshot: %q", again[0].Msg)
	}
}
