package dice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	goeventbus "github.com/Protocol-Lattice/GoEventBus"
	"github.com/dop251/goja"
)

// adapterEventProjection 所有适配器事件共用的单一 projection。
// 事件名的精确/前缀匹配在 EventBus 自层完成，以支持 "group.*" 订阅。
const adapterEventProjection = "sealdice.adapter_event"

// EventBus 适配器事件总线，以 GoEventBus 为分发底座（ring buffer + fan-out + DLQ）。
// 默认同步分发：与旧 IMSession.OnXxx 行为一致（发布者协程内联执行、顺序确定）。
type EventBus struct {
	dice       *Dice
	dispatcher *goeventbus.Dispatcher
	store      *goeventbus.EventStore

	mu   sync.RWMutex
	subs []*eventBusSubscription
	next int64
}

type eventBusSubscription struct {
	id      int64
	pattern string // 精确事件名 / "xxx.*" 前缀 / "*"
	goFn    func(ctx context.Context, ev *AdapterEvent) error
	// JS 订阅：
	jsFn   func(ctx *MsgContext, ev *AdapterEvent)
	jsLoop int64
	fromJS bool
}

// NewEventBus 创建事件总线。dice 可为 nil（单测场景，JS 分发不可用）。
func NewEventBus(d *Dice) *EventBus {
	dispatcher := &goeventbus.Dispatcher{}
	b := &EventBus{dice: d, dispatcher: dispatcher}
	dispatcher.Register(adapterEventProjection, func(ctx context.Context, ev goeventbus.Event) (goeventbus.Result, error) {
		aev, ok := ev.Data.(*AdapterEvent)
		if !ok {
			return goeventbus.Result{}, fmt.Errorf("事件总线收到非 AdapterEvent 载荷: %T", ev.Data)
		}
		return goeventbus.Result{}, b.route(aev)
	})
	store := goeventbus.NewEventStore(dispatcher, 1<<16, goeventbus.DropOldest)
	store.DLQ = goeventbus.NewDeadLetterQueue()
	b.store = store
	return b
}

// Emit 发射事件。补齐 ID/Time 后入队并同步分发。
func (b *EventBus) Emit(ev *AdapterEvent) error {
	if ev == nil {
		return errors.New("事件为空")
	}
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	if ev.ID == "" {
		b.mu.Lock()
		b.next++
		seq := b.next
		b.mu.Unlock()
		ev.ID = fmt.Sprintf("aev-%d-%d", ev.Time.UnixNano(), seq)
	}
	if err := b.store.Subscribe(context.Background(), goeventbus.Event{
		ID:         ev.ID,
		Projection: adapterEventProjection,
		Data:       ev,
	}); err != nil {
		return err
	}
	b.store.Publish()
	return nil
}

// OnEvent 注册 Go 侧订阅。pattern 为精确事件名、前缀（"group.*"）或 "*"。
func (b *EventBus) OnEvent(pattern string, fn func(ctx context.Context, ev *AdapterEvent) error) error {
	if fn == nil {
		return errors.New("处理器为空")
	}
	if !validEventPattern(pattern) {
		return fmt.Errorf("无效的事件订阅模式: %q", pattern)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.next++
	b.subs = append(b.subs, &eventBusSubscription{id: b.next, pattern: pattern, goFn: fn})
	return nil
}

// OnEventJS 注册 JS 侧订阅。jsLoop 为订阅时的事件循环版本号，过期（插件重载）后静默丢弃。
func (b *EventBus) OnEventJS(pattern string, jsLoop int64, fn func(ctx *MsgContext, ev *AdapterEvent)) error {
	if fn == nil {
		return errors.New("处理器为空")
	}
	if !validEventPattern(pattern) {
		return fmt.Errorf("无效的事件订阅模式: %q", pattern)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.next++
	b.subs = append(b.subs, &eventBusSubscription{id: b.next, pattern: pattern, jsFn: fn, jsLoop: jsLoop, fromJS: true})
	return nil
}

// route 分发事件。先快照匹配的订阅者再逐个调用，避免 handler 内再订阅造成死锁。
func (b *EventBus) route(ev *AdapterEvent) error {
	b.mu.RLock()
	matched := make([]*eventBusSubscription, 0, 4)
	for _, sub := range b.subs {
		if matchEventPattern(sub.pattern, ev.Name) {
			matched = append(matched, sub)
		}
	}
	b.mu.RUnlock()

	var errs []error
	for _, sub := range matched {
		if sub.fromJS {
			if err := b.invokeJs(sub, ev); err != nil {
				errs = append(errs, fmt.Errorf("订阅#%d(%s): %w", sub.id, sub.pattern, err))
			}
			continue
		}
		if sub.goFn == nil {
			continue
		}
		if err := safeInvokeGo(sub, ev); err != nil {
			errs = append(errs, fmt.Errorf("订阅#%d(%s): %w", sub.id, sub.pattern, err))
		}
	}
	return errors.Join(errs...)
}

func safeInvokeGo(sub *eventBusSubscription, ev *AdapterEvent) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("处理器 panic: %v", r)
		}
	}()
	return sub.goFn(context.Background(), ev)
}

// invokeJs 把 JS 处理器调度回其所属事件循环执行，参照 callWithJsCheck 的既有模式。
func (b *EventBus) invokeJs(sub *eventBusSubscription, ev *AdapterEvent) error {
	if b.dice == nil || b.dice.ExtLoopManager == nil {
		return errors.New("JS 环境不可用")
	}
	loop, err := b.dice.ExtLoopManager.GetLoop(sub.jsLoop)
	if err != nil {
		// 循环已过期（插件重载），静默丢弃而非报错
		//nolint:nilerr // 过期订阅有意静默丢弃
		return nil
	}
	waitRun := make(chan int, 1)
	var handlerErr error
	loop.RunOnLoop(func(vm *goja.Runtime) {
		defer func() {
			if r := recover(); r != nil {
				handlerErr = fmt.Errorf("JS 处理器异常: %v", r)
			}
			waitRun <- 1
		}()
		if sub.jsFn == nil {
			return
		}
		fn := sub.jsFn
		fn(ev.Ctx, ev)
	})
	<-waitRun
	return handlerErr
}

func validEventPattern(pattern string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == "" {
		return false
	}
	if !strings.HasSuffix(pattern, ".*") {
		return validEventName(pattern)
	}
	return validEventName(strings.TrimSuffix(pattern, ".*"))
}

func validEventName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_':
		default:
			return false
		}
	}
	return true
}

func matchEventPattern(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}
	return pattern == name
}

// Metrics 返回 published/processed/errors 计数。
func (b *EventBus) Metrics() (published, processed, errs uint64) {
	return b.store.Metrics()
}

// DeadLetters 返回死信快照。
func (b *EventBus) DeadLetters() []goeventbus.DeadLetter {
	return b.store.DLQ.Entries()
}

// DrainDeadLetters 取出并清空死信。
func (b *EventBus) DrainDeadLetters() []goeventbus.DeadLetter {
	return b.store.DLQ.Drain()
}

// Close 关闭总线底座（停掉 GoEventBus 的 worker 池）。进程退出场景可不调用，测试中必须调用。
func (b *EventBus) Close(ctx context.Context) error {
	return b.store.Drain(ctx)
}
