package dice

import (
	"testing"
	"time"
)

func TestAdapterEventFields(t *testing.T) {
	ev := &AdapterEvent{
		Name:     EventNameGroupMuted,
		Platform: "QQ",
		GroupID:  "QQ:123",
		UserID:   "QQ:456",
		Raw:      map[string]any{"duration": float64(600)},
		Time:     time.Now(),
	}
	if ev.Name != "group.muted" {
		t.Fatalf("事件名常量错误: %s", ev.Name)
	}
	if ev.Raw["duration"] != float64(600) {
		t.Fatalf("Raw 字段错误: %v", ev.Raw["duration"])
	}
}
