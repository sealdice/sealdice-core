package logger

import (
	"testing"
	"time"
)

func TestUIWriterBroadcastsParsedLogItemsToSubscribers(t *testing.T) {
	writer := NewUIWriter()
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

func TestUIWriterSnapshotIsDetachedFromInternalItems(t *testing.T) {
	writer := NewUIWriter()
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
