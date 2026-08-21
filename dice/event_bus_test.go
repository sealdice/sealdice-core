//nolint:testpackage
package dice

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func newTestBus(t *testing.T) *EventBus {
	b := NewEventBus(nil)
	t.Cleanup(func() { _ = b.Close(t.Context()) })
	return b
}

func TestEventBusExactMatch(t *testing.T) {
	b := newTestBus(t)
	var got []string
	_ = b.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		got = append(got, ev.Name)
		return nil
	})
	_ = b.Emit(&AdapterEvent{Name: EventNamePoke})
	_ = b.Emit(&AdapterEvent{Name: EventNameGroupJoined})
	if len(got) != 1 || got[0] != EventNamePoke {
		t.Fatalf("精确匹配失败: %v", got)
	}
}

func TestEventBusPrefixAndWildcard(t *testing.T) {
	b := newTestBus(t)
	var mu sync.Mutex
	groupHits, allHits := 0, 0
	_ = b.OnEvent("group.*", func(ctx context.Context, ev *AdapterEvent) error {
		mu.Lock()
		groupHits++
		mu.Unlock()
		return nil
	})
	_ = b.OnEvent("*", func(ctx context.Context, ev *AdapterEvent) error {
		mu.Lock()
		allHits++
		mu.Unlock()
		return nil
	})
	_ = b.Emit(&AdapterEvent{Name: EventNameGroupJoined})
	_ = b.Emit(&AdapterEvent{Name: EventNameGroupMuted})
	_ = b.Emit(&AdapterEvent{Name: EventNamePoke})
	mu.Lock()
	defer mu.Unlock()
	if groupHits != 2 {
		t.Fatalf("前缀订阅命中数错误: %d", groupHits)
	}
	if allHits != 3 {
		t.Fatalf("全匹配订阅命中数错误: %d", allHits)
	}
}

func TestEventBusOrderAndErrors(t *testing.T) {
	b := newTestBus(t)
	var order []int
	_ = b.OnEvent("x.y", func(ctx context.Context, ev *AdapterEvent) error {
		order = append(order, 1)
		return errors.New("第一个订阅者失败")
	})
	_ = b.OnEvent("x.y", func(ctx context.Context, ev *AdapterEvent) error {
		order = append(order, 2)
		return nil
	})
	_ = b.Emit(&AdapterEvent{Name: "x.y"})
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("单个订阅者失败不应中断后续订阅者: %v", order)
	}
	published, _, errs := b.Metrics()
	if published != 1 || errs == 0 {
		t.Fatalf("指标错误: published=%d errors=%d", published, errs)
	}
	dl := b.DeadLetters()
	if len(dl) != 1 {
		t.Fatalf("DLQ 应记录 1 条失败事件: %d", len(dl))
	}
	if b.DrainDeadLetters(); len(b.DeadLetters()) != 0 {
		t.Fatal("清空 DLQ 失败")
	}
}

func TestEventBusPanicRecovery(t *testing.T) {
	b := newTestBus(t)
	var called bool
	_ = b.OnEvent("p.y", func(ctx context.Context, ev *AdapterEvent) error {
		panic("boom")
	})
	_ = b.OnEvent("p.y", func(ctx context.Context, ev *AdapterEvent) error {
		called = true
		return nil
	})
	_ = b.Emit(&AdapterEvent{Name: "p.y"}) // 不应 panic 外泄
	if !called {
		t.Fatal("panic 订阅者不应阻断后续订阅者")
	}
	if len(b.DeadLetters()) != 1 {
		t.Fatalf("panic 应进 DLQ: %d", len(b.DeadLetters()))
	}
}

func TestEventBusInvalidPattern(t *testing.T) {
	b := newTestBus(t)
	if err := b.OnEvent("", func(ctx context.Context, ev *AdapterEvent) error { return nil }); err == nil {
		t.Fatal("空事件名应报错")
	}
	if err := b.OnEvent("a.*b", func(ctx context.Context, ev *AdapterEvent) error { return nil }); err == nil {
		t.Fatal("非法通配位置应报错")
	}
}

func TestEventBusSubscribeDuringDispatch(t *testing.T) {
	// 订阅者在分发过程中再订阅不应死锁
	b := newTestBus(t)
	done := make(chan struct{})
	_ = b.OnEvent("d.y", func(ctx context.Context, ev *AdapterEvent) error {
		_ = b.OnEvent("d.z", func(ctx context.Context, ev *AdapterEvent) error { return nil })
		close(done)
		return nil
	})
	_ = b.Emit(&AdapterEvent{Name: "d.y"})
	<-done
}
