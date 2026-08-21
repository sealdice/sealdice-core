# 适配器事件总线（AdapterEvent Bus）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 以 GoEventBus 为分发底座实现 `AdapterEvent` 事件总线：统一入站事件分发、按事件名动态订阅（含前缀）、旧 JS 回调兼容、适配器能力清单与 `sendRaw` 出站透传。

**Architecture:** 所有适配器事件走单一 GoEventBus projection，由 `EventBus` 自层做事件名精确/前缀匹配（支持 `group.*`）。旧 `IMSession.OnXxx` 方法保留签名，内部改为构造 `AdapterEvent` 并发射；旧逻辑（含旧回调字段调用）迁移为"兼容订阅器"（注册时机最早，保证先于插件回调执行）。`sendRaw` 独立于总线，经 `RawActionAdapter` 可选接口 + 能力清单校验后透传到适配器。

**Tech Stack:** Go 1.25、github.com/Protocol-Lattice/GoEventBus、goja（既有 JSVM）。

**设计文档:** `docs/superpowers/specs/2026-08-21-adapter-event-bus-design.md`

**工作区:** `.worktrees/refactor-cmd`（分支 `refactor/cmd`）

## 对设计文档的两处落地调整（已论证）

1. **`AdapterEvent` 放 `dice` 包而非 `dice/events` 包**：事件需携带 `Ctx *MsgContext`，而 `dice/events` 被 `dice` 引用，反向引用会循环导入。
2. **总线默认同步分发（`Async=false`）**：GoEventBus 异步模式不保证同一事件内 handler 顺序，且高压下 `DropOldest` 会丢事件（入群致辞等旧逻辑不能丢）。同步模式与现状行为完全一致（发布者协程内联执行），panic/error 仍进 DLQ。`Async` 开关保留在代码中，后续可按实测放开。

## 关键既有事实（执行前速读）

- 旧事件回调字段在 `ExtInfo`（`dice/dice.go:92-108`）：`OnPoke`、`OnGroupJoined`、`OnGroupMemberJoined`、`OnBecomeFriend`、`OnGuildJoined`、`OnGroupLeave` 等，jsbind 标签映射到 JS。
- `IMSession.OnPoke/OnGroupLeave/OnGroupJoined/OnGroupMemberJoined`（`dice/im_session.go:1289,1361,1995,2027`）是 milky/onebot_util/official_qq 的调用入口；gocq/telegram/dingtalk/kook/walleq 直接在适配器内联遍历扩展。
- JS 回调必须经 `callWithJsCheck`（`dice/ext.go:324`）模式：按 `JSLoopVersion` 取 loop，`loop.RunOnLoop` 内调用。
- `PlatformAdapterOnebot.sendEmitter`（`dice/platform_adapter_onebot.go:46`）实现 `Raw(ctx, action, params)` 全透传（`dice/imsdk/onebot/emitter.go:47`）。
- `PlatformAdapterMilky.IntentSession`（`*milky.Session`）有 `SendGroupMessage/SendPrivateMessage/SendGroupNudge`；`pa.GetGroupMemberInfo` 已有。
- `IMSession.EndPoints`（`dice/im_session.go:658`）；`EndPointInfoBase` 有 `Platform`、`ProtocolType`、`Enable`、`State`（`dice/im_session.go:413-430`）。
- `JsLoopManager.SetLoop` 返回版本号（`dice/dice.go:159`）。
- GoEventBus：`Dispatcher.Register(projection, handler)`、`NewEventStore(&disp, size, policy)`（size 必须 2 的幂）、`store.Subscribe+Publish`、`store.DLQ`、`store.Metrics()`。handler 签名 `func(context.Context, Event) (Result, error)`。

---

### Task 1: 添加 GoEventBus 依赖

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 拉取依赖**

```bash
go get github.com/Protocol-Lattice/GoEventBus
```

- [ ] **Step 2: 验证可编译**

```bash
go build ./...
```

预期：无报错。

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "build: 引入 GoEventBus 作为适配器事件总线分发底座"
```

---

### Task 2: AdapterEvent 类型与事件名常量

**Files:**
- Create: `dice/adapter_event.go`
- Test: `dice/adapter_event_test.go`

- [ ] **Step 1: 写失败测试**

```go
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
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run TestAdapterEventFields -v
```

预期：编译失败，`undefined: AdapterEvent`。

- [ ] **Step 3: 实现 `dice/adapter_event.go`**

```go
package dice

import (
	"time"
)

// 适配器事件名常量。命名规约：领域.动作，全小写，点分。
// 新增事件名时同步更新 docs/superpowers/specs/2026-08-21-adapter-event-bus-design.md 的映射表。
const (
	EventNamePoke              = "poke"                // 戳一戳
	EventNameGroupJoined       = "group.joined"        // 骰子自身加入群
	EventNameGroupMemberJoined = "group.member_joined" // 其他群成员加入群
	EventNameGroupLeave        = "group.leave"         // 群成员离开/被踢
	EventNameGroupMuted        = "group.muted"         // 群禁言（预留，暂无适配器发射）
	EventNameFriendJoined      = "friend.joined"       // 成为好友
	EventNameGuildJoined       = "guild.joined"        // 加入频道（KOOK 等）
	EventNameFriendRequest     = "friend.request"      // 好友申请（仅通知）
	EventNameGroupRequest      = "group.request"       // 加群申请/邀请（仅通知）
)

// AdapterEvent 适配器事件的统一封装。
// 命名刻意区别于未来可能的"插件事件机制"，也区别于库类型 goeventbus.Event。
// 请求类事件（friend.request / group.request）仅通知，不携带回复通道。
type AdapterEvent struct {
	ID         string         `jsbind:"id"          json:"id"`
	Name       string         `jsbind:"name"        json:"name"`       // 事件名，见 EventNameXxx 常量
	Platform   string         `jsbind:"platform"    json:"platform"`   // 如 "QQ"、"DISCORD"
	EndPointID string         `jsbind:"endPointId"  json:"endpoint_id"`
	GroupID    string         `jsbind:"groupId"     json:"group_id"` // UNI-ID 格式，如 QQ:123456
	UserID     string         `jsbind:"userId"      json:"user_id"`  // 事件主体（如被禁言者）
	SenderID   string         `jsbind:"senderId"    json:"sender_id"` // 操作发起者
	Raw        map[string]any `jsbind:"raw"         json:"raw"` // 适配器原样透传的附加数据
	Time       time.Time      `jsbind:"time"        json:"time"`

	// Ctx 携带构造事件时的消息上下文，供 JS/Go 订阅者回复等操作使用。不序列化。
	Ctx *MsgContext `json:"-" yaml:"-"`
	// Detail 携带旧类型化载荷（*events.PokeEvent 等），仅供兼容订阅器还原旧逻辑。不序列化，不供 JS 使用。
	Detail any `json:"-" yaml:"-"`
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./dice/ -run TestAdapterEventFields -v
```

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add dice/adapter_event.go dice/adapter_event_test.go
git commit -m "feat(bus): 新增 AdapterEvent 统一事件封装与事件名常量"
```

---

### Task 3: 适配器能力清单注册表

**Files:**
- Create: `dice/adapter_capabilities.go`
- Test: `dice/adapter_capabilities_test.go`

- [ ] **Step 1: 写失败测试**

```go
package dice

import "testing"

func TestAdapterCapabilitiesRegistry(t *testing.T) {
	set := AdapterCapabilitySet{
		ProtocolType: "milky",
		Platform:     "QQ",
		EmitEvents: map[string]AdapterEventSpec{
			EventNamePoke: {Name: EventNamePoke, Description: "戳一戳"},
		},
		RawActions: map[string]AdapterRawActionSpec{
			"get_group_member_info": {Name: "get_group_member_info", Description: "获取群成员信息"},
		},
	}
	RegisterAdapterCapabilities(set)

	got := GetAdapterCapabilities("milky")
	if got == nil {
		t.Fatal("按协议类型查询能力失败")
	}
	if _, ok := got.EmitEvents[EventNamePoke]; !ok {
		t.Fatal("EmitEvents 缺少 poke")
	}
	if _, ok := got.RawActions["get_group_member_info"]; !ok {
		t.Fatal("RawActions 缺少 get_group_member_info")
	}

	merged := GetAdapterCapabilitiesByPlatform("QQ")
	if len(merged) == 0 {
		t.Fatal("按平台聚合查询能力失败")
	}

	if _, ok := GetAdapterCapabilities("not-exist"); ok {
		t.Fatal("不存在的协议类型应返回 nil")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run TestAdapterCapabilitiesRegistry -v
```

预期：编译失败，类型未定义。

- [ ] **Step 3: 实现 `dice/adapter_capabilities.go`**

```go
package dice

import (
	"sort"
	"sync"
)

// AdapterEventSpec 描述适配器可发射的一个事件。
type AdapterEventSpec struct {
	Name        string `jsbind:"name"        json:"name"`
	Description string `jsbind:"description" json:"description"`
	RequestOnly bool   `jsbind:"requestOnly" json:"request_only"` // 请求类（仅通知）
}

// AdapterRawActionSpec 描述适配器支持的一个 sendRaw 动作。
type AdapterRawActionSpec struct {
	Name        string            `jsbind:"name"        json:"name"`
	Description string            `jsbind:"description" json:"description"`
	Params      map[string]string `jsbind:"params"      json:"params,omitempty"` // 参数名 -> 类型说明
}

// AdapterCapabilitySet 一个适配器（按 ProtocolType 区分，如 milky/onebot/gocq）的能力清单。
type AdapterCapabilitySet struct {
	ProtocolType string                          `json:"protocol_type"` // 如 "milky"、"onebot"、"gocq"
	Platform     string                          `json:"platform"`      // 如 "QQ"、"DISCORD"
	EmitEvents   map[string]AdapterEventSpec     `json:"emit_events"`
	RawActions   map[string]AdapterRawActionSpec `json:"raw_actions"`
}

var (
	adapterCapabilityMu  sync.RWMutex
	adapterCapabilities  = map[string]AdapterCapabilitySet{} // key: ProtocolType
)

// RegisterAdapterCapabilities 注册适配器能力清单。适配器在各自文件的 init() 中调用。
// 重复注册同一 ProtocolType 时以后注册者为准（测试中允许重复注册）。
func RegisterAdapterCapabilities(set AdapterCapabilitySet) {
	adapterCapabilityMu.Lock()
	defer adapterCapabilityMu.Unlock()
	adapterCapabilities[set.ProtocolType] = set
}

// GetAdapterCapabilities 按协议类型查询能力清单。
func GetAdapterCapabilities(protocolType string) (AdapterCapabilitySet, bool) {
	adapterCapabilityMu.RLock()
	defer adapterCapabilityMu.RUnlock()
	set, ok := adapterCapabilities[protocolType]
	return set, ok
}

// GetAdapterCapabilitiesByPlatform 按平台聚合该平台下所有协议的能力清单（能力查询 UI/JS 用）。
func GetAdapterCapabilitiesByPlatform(platform string) []AdapterCapabilitySet {
	adapterCapabilityMu.RLock()
	defer adapterCapabilityMu.RUnlock()
	var out []AdapterCapabilitySet
	for _, set := range adapterCapabilities {
		if set.Platform == platform {
			out = append(out, set)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProtocolType < out[j].ProtocolType })
	return out
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./dice/ -run TestAdapterCapabilitiesRegistry -v
```

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add dice/adapter_capabilities.go dice/adapter_capabilities_test.go
git commit -m "feat(bus): 适配器能力清单注册表"
```

---

### Task 4: EventBus 核心（GoEventBus 封装 + 事件名路由）

**Files:**
- Create: `dice/event_bus.go`
- Test: `dice/event_bus_test.go`

- [ ] **Step 1: 写失败测试**

```go
package dice

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func newTestBus() *EventBus {
	return NewEventBus(nil)
}

func TestEventBusExactMatch(t *testing.T) {
	b := newTestBus()
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
	b := newTestBus()
	var groupMu sync.Mutex
	groupHits, allHits := 0, 0
	_ = b.OnEvent("group.*", func(ctx context.Context, ev *AdapterEvent) error {
		groupMu.Lock()
		groupHits++
		groupMu.Unlock()
		return nil
	})
	_ = b.OnEvent("*", func(ctx context.Context, ev *AdapterEvent) error {
		groupMu.Lock()
		allHits++
		groupMu.Unlock()
		return nil
	})
	_ = b.Emit(&AdapterEvent{Name: EventNameGroupJoined})
	_ = b.Emit(&AdapterEvent{Name: EventNameGroupMuted})
	_ = b.Emit(&AdapterEvent{Name: EventNamePoke})
	groupMu.Lock()
	defer groupMu.Unlock()
	if groupHits != 2 {
		t.Fatalf("前缀订阅命中数错误: %d", groupHits)
	}
	if allHits != 3 {
		t.Fatalf("全匹配订阅命中数错误: %d", allHits)
	}
}

func TestEventBusOrderAndErrors(t *testing.T) {
	b := newTestBus()
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
	b := newTestBus()
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
	b := newTestBus()
	if err := b.OnEvent("", func(ctx context.Context, ev *AdapterEvent) error { return nil }); err == nil {
		t.Fatal("空事件名应报错")
	}
	if err := b.OnEvent("a.*b", func(ctx context.Context, ev *AdapterEvent) error { return nil }); err == nil {
		t.Fatal("非法通配位置应报错")
	}
}

func TestEventBusSubscribeDuringDispatch(t *testing.T) {
	// 订阅者在分发过程中再订阅不应死锁
	b := newTestBus()
	done := make(chan struct{})
	_ = b.OnEvent("d.y", func(ctx context.Context, ev *AdapterEvent) error {
		_ = b.OnEvent("d.z", func(ctx context.Context, ev *AdapterEvent) error { return nil })
		close(done)
		return nil
	})
	_ = b.Emit(&AdapterEvent{Name: "d.y"})
	<-done
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run 'TestEventBus' -v
```

预期：编译失败，`undefined: NewEventBus`。

- [ ] **Step 3: 实现 `dice/event_bus.go`**

```go
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
	if pattern == "" || pattern == "*" {
		return pattern == "*"
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
```

注意：`validEventPattern` 中 `if pattern == "" || pattern == "*" { return pattern == "*" }` 的写法等价于：空串非法、`*` 合法。

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./dice/ -run 'TestEventBus' -v
```

预期：全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add dice/event_bus.go dice/event_bus_test.go
git commit -m "feat(bus): EventBus 核心——GoEventBus 封装与事件名精确/前缀路由"
```

---

### Task 5: Dice/IMSession 集成

**Files:**
- Modify: `dice/dice.go`（Dice 结构体 + Init）
- Modify: `dice/im_session.go`（新增 EmitEvent）
- Test: `dice/event_bus_integration_test.go`

- [ ] **Step 1: 写失败测试**

```go
package dice

import (
	"context"
	"testing"
)

func TestDiceEventBusWiring(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)

	var hits int
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		hits++
		return nil
	})

	d.ImSession.EmitEvent(&AdapterEvent{
		Name:     EventNamePoke,
		Platform: "QQ",
		Ctx:      &MsgContext{Dice: d, Session: d.ImSession, EndPoint: &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ"}}},
	})
	if hits != 1 {
		t.Fatalf("EmitEvent 应触发订阅: %d", hits)
	}
}

func TestIMSessionEmitEventNilSafe(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d}
	d.EventBus = NewEventBus(d)
	// 无 Ctx、无平台字段也不应 panic，且补齐 Platform/EndPointID
	var gotPlatform string
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		gotPlatform = ev.Platform
		return nil
	})
	d.ImSession.EmitEvent(&AdapterEvent{Name: EventNamePoke, Ctx: &MsgContext{Dice: d, Session: d.ImSession, EndPoint: &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "TG"}}}})
	if gotPlatform != "TG" {
		t.Fatalf("应从 Ctx.EndPoint 补齐 Platform: %q", gotPlatform)
	}
	d.ImSession.EmitEvent(nil) // 不应 panic
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run 'TestDiceEventBusWiring|TestIMSessionEmitEventNilSafe' -v
```

预期：编译失败，`d.EventBus` 未定义 / `EmitEvent` 未定义。

- [ ] **Step 3: 修改 `dice/dice.go`**

在 Dice 结构体 `DirtyGroups` 字段（`dice/dice.go:285` 附近）后追加：

```go
	/* 适配器事件总线 */
	EventBus *EventBus `json:"-" yaml:"-"`
```

在 `Init` 中 `d.DirtyGroups = new(SyncMap[string, int64])`（`dice/dice.go:340`）之后追加：

```go
	// 适配器事件总线：先装配总线与兼容订阅器，再进入后续 JS/端点加载，
	// 保证旧逻辑订阅先于任何插件订阅注册（分发顺序与旧行为一致）。
	d.EventBus = NewEventBus(d)
	d.registerAdapterEventCompat()
```

（`registerAdapterEventCompat` 在 Task 6 实现。为使本任务可独立编译，本任务先在 `dice/event_bus_compat.go` 中放置空实现，Task 6 替换为真实逻辑。）

- [ ] **Step 4: 新建 `dice/event_bus_compat.go` 占位（本任务临时）**

```go
package dice

// registerAdapterEventCompat 将旧 IMSession.OnXxx 通知逻辑注册为总线兼容订阅器。
// Task 6 会替换为真实实现；此占位保证装配顺序先于 JS 加载。
func (d *Dice) registerAdapterEventCompat() {}
```

- [ ] **Step 5: 在 `dice/im_session.go` 末尾（`OnMessageEdit` 等方法之后、文件尾部）新增**

```go
// EmitEvent 适配器事件统一入口。补齐事件的平台/端点信息后经总线分发。
func (s *IMSession) EmitEvent(ev *AdapterEvent) {
	if ev == nil {
		return
	}
	d := s.Parent
	if d == nil || d.EventBus == nil {
		return
	}
	if ev.Ctx == nil {
		ev.Ctx = &MsgContext{Session: s, Dice: d}
	}
	if ev.Ctx.Session == nil {
		ev.Ctx.Session = s
	}
	if ev.Ctx.Dice == nil {
		ev.Ctx.Dice = d
	}
	if ev.Platform == "" && ev.Ctx.EndPoint != nil {
		ev.Platform = ev.Ctx.EndPoint.Platform
	}
	if ev.EndPointID == "" && ev.Ctx.EndPoint != nil {
		ev.EndPointID = ev.Ctx.EndPoint.ID
	}
	_ = d.EventBus.Emit(ev)
}
```

- [ ] **Step 6: 跑测试与整体构建**

```bash
go test ./dice/ -run 'TestDiceEventBusWiring|TestIMSessionEmitEventNilSafe' -v
go build ./...
```

预期：PASS、构建通过。

- [ ] **Step 7: 提交**

```bash
git add dice/dice.go dice/im_session.go dice/event_bus_compat.go dice/event_bus_integration_test.go
git commit -m "feat(bus): Dice 装配 EventBus 与 IMSession.EmitEvent 统一入口"
```

---

### Task 6: 兼容层——旧 OnXxx 改造为总线发射

**Files:**
- Modify: `dice/im_session.go`（OnPoke/OnGroupLeave/OnGroupJoined/OnGroupMemberJoined 四处）
- Modify: `dice/event_bus_compat.go`（真实兼容订阅器）
- Test: `dice/event_bus_compat_test.go`

**改造原则**：`IMSession.OnXxx` 签名不变（适配器调用点零改动），内部构造 `AdapterEvent`（`Detail` 携带旧类型化载荷）并 `EmitEvent`；旧方法体原样搬到 `legacyOnXxx`，由兼容订阅器调用。旧回调字段的调用语义（群激活过滤、`callWithJsCheck`、遍历范围）逐字保留。

- [ ] **Step 1: 写失败测试**

```go
package dice

import (
	"testing"

	"sealdice-core/dice/events"
)

// OnPoke 改造后：事件经总线分发，插件订阅者可收到（含 Raw/Detail 载荷）。
// 兼容订阅器的 legacy 行为由既有回归测试（go test ./dice/...）兜底。
func TestOnPokeEmitsAdapterEvent(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)
	d.registerAdapterEventCompat()

	var gotName, gotGroup string
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		gotName = ev.Name
		gotGroup = ev.GroupID
		return nil
	})

	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ"}}
	d.ImSession.EndPoints = append(d.ImSession.EndPoints, ep)
	ctx := &MsgContext{Dice: d, Session: d.ImSession, EndPoint: ep}

	// 兼容订阅器内 legacyOnPoke 会访问群激活链路，群不存在时直接返回，不应 panic
	d.ImSession.OnPoke(ctx, &events.PokeEvent{GroupID: "", SenderID: "QQ:1", TargetID: "QQ:2"})

	if gotName != EventNamePoke {
		t.Fatalf("插件订阅者未收到 poke 事件: %q", gotName)
	}
	d.ImSession.OnPoke(ctx, &events.PokeEvent{GroupID: "QQ:10001", SenderID: "QQ:1", TargetID: "QQ:2"})
	if gotGroup != "QQ:10001" {
		t.Fatalf("事件载荷 GroupID 不符: %q", gotGroup)
	}
}
```

（import 需含 `"context"`。）

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run TestCompatSubscriberRunsLegacyPath -v
```

预期：FAIL——当前 OnPoke 直接走旧逻辑，`newAPICalled` 为 false。

- [ ] **Step 3: 改造 `dice/im_session.go` 四个方法**

对每个方法应用同一模式。以 `OnPoke`（`dice/im_session.go:1995`）为例——原体整体改名为 `legacyOnPoke`（接收者与参数不变，函数体逐字保留），新的 `OnPoke` 为：

```go
// OnPoke 戳一戳事件入口。构造 AdapterEvent 经事件总线分发；
// 旧逻辑（群激活检查 + 旧回调字段调用）在兼容订阅器中执行（dice/event_bus_compat.go）。
func (s *IMSession) OnPoke(ctx *MsgContext, event *events.PokeEvent) {
	if ctx == nil || event == nil {
		return
	}
	s.EmitEvent(&AdapterEvent{
		Name:       EventNamePoke,
		GroupID:    event.GroupID,
		UserID:     event.TargetID,
		SenderID:   event.SenderID,
		Raw:        map[string]any{"isPrivate": event.IsPrivate},
		Ctx:        ctx,
		Detail:     event,
	})
}
```

`OnGroupLeave`（`dice/im_session.go:2027`）同理：

```go
// OnGroupLeave 群成员离开/被踢事件入口。旧逻辑在兼容订阅器中执行。
func (s *IMSession) OnGroupLeave(ctx *MsgContext, event *events.GroupLeaveEvent) {
	if ctx == nil || event == nil {
		return
	}
	s.EmitEvent(&AdapterEvent{
		Name:       EventNameGroupLeave,
		GroupID:    event.GroupID,
		UserID:     event.UserID,
		SenderID:   event.OperatorID,
		Ctx:        ctx,
		Detail:     event,
	})
}
```

`OnGroupJoined`（`dice/im_session.go:1290`）与 `OnGroupMemberJoined`（`dice/im_session.go:1361`）同理，事件字段：

```go
// OnGroupJoined 骰子自身加入群事件入口。旧逻辑（自动激活、入群致辞、旧回调）在兼容订阅器中执行。
func (s *IMSession) OnGroupJoined(ctx *MsgContext, msg *Message) {
	if ctx == nil || msg == nil {
		return
	}
	s.EmitEvent(&AdapterEvent{
		Name:     EventNameGroupJoined,
		GroupID:  msg.GroupID,
		SenderID: msg.Sender.UserID, // 邀请人，Adapter 已保证放入 Sender
		Raw:      map[string]any{"message": msg.Message},
		Ctx:      ctx,
		Detail:   msg,
	})
}

// OnGroupMemberJoined 其他群成员加入群事件入口。旧逻辑（迎新致辞）在兼容订阅器中执行。
func (s *IMSession) OnGroupMemberJoined(ctx *MsgContext, msg *Message) {
	if ctx == nil || msg == nil {
		return
	}
	s.EmitEvent(&AdapterEvent{
		Name:     EventNameGroupMemberJoined,
		GroupID:  msg.GroupID,
		UserID:   msg.Sender.UserID,
		Raw:      map[string]any{"message": msg.Message},
		Ctx:      ctx,
		Detail:   msg,
	})
}
```

- [ ] **Step 4: 实现真实兼容订阅器（替换 `dice/event_bus_compat.go`）**

```go
package dice

import (
	"context"

	"sealdice-core/dice/events"
)

// registerAdapterEventCompat 将旧 IMSession.OnXxx 通知逻辑注册为总线订阅器。
// 在 Dice.Init 中最先注册（早于任何 JS 插件订阅），因此旧逻辑先于插件回调执行，
// 与旧实现的顺序语义一致。
func (d *Dice) registerAdapterEventCompat() {
	if d.EventBus == nil {
		return
	}
	_ = d.EventBus.OnEvent(EventNamePoke, func(ctx context.Context, ev *AdapterEvent) error {
		poke, _ := ev.Detail.(*events.PokeEvent)
		if poke == nil {
			return nil
		}
		d.ImSession.legacyOnPoke(ev.Ctx, poke)
		return nil
	})
	_ = d.EventBus.OnEvent(EventNameGroupLeave, func(ctx context.Context, ev *AdapterEvent) error {
		leave, _ := ev.Detail.(*events.GroupLeaveEvent)
		if leave == nil {
			return nil
		}
		d.ImSession.legacyOnGroupLeave(ev.Ctx, leave)
		return nil
	})
	_ = d.EventBus.OnEvent(EventNameGroupJoined, func(ctx context.Context, ev *AdapterEvent) error {
		msg, _ := ev.Detail.(*Message)
		if msg == nil {
			return nil
		}
		d.ImSession.legacyOnGroupJoined(ev.Ctx, msg)
		return nil
	})
	_ = d.EventBus.OnEvent(EventNameGroupMemberJoined, func(ctx context.Context, ev *AdapterEvent) error {
		msg, _ := ev.Detail.(*Message)
		if msg == nil {
			return nil
		}
		d.ImSession.legacyOnGroupMemberJoined(ev.Ctx, msg)
		return nil
	})
}
```

同时把 `im_session.go` 中原四个方法体重命名为 `legacyOnPoke`、`legacyOnGroupLeave`、`legacyOnGroupJoined`、`legacyOnGroupMemberJoined`（函数体逐字不动，仅改函数名与注释首行）。

- [ ] **Step 5: 跑测试与既有回归**

```bash
go test ./dice/ -run 'TestCompatSubscriberRunsLegacyPath|TestEventBus' -v
go build ./...
```

预期：PASS、构建通过。

- [ ] **Step 6: 提交**

```bash
git add dice/im_session.go dice/event_bus_compat.go dice/event_bus_compat_test.go
git commit -m "refactor(bus): OnPoke/OnGroupLeave/OnGroupJoined/OnGroupMemberJoined 改造为总线发射，旧逻辑迁移至兼容订阅器"
```

---

### Task 7: 新增事件发射（friend/guild/request 类）

**Files:**
- Modify: `dice/platform_adapter_milky.go`
- Modify: `dice/platform_adapter_onebot_util.go`
- Modify: `dice/platform_adapter_gocq.go`
- Modify: `dice/platform_adapter_kook.go`
- Test: `dice/adapter_event_emit_test.go`

**原则**：这些事件旧分发是适配器内联循环（`OnBecomeFriend`/`OnGuildJoined`）或根本没有（request 类）。本任务只**并行发射**事件，不注册兼容订阅器、不动旧循环——旧插件路径零变化，无双触发。

- [ ] **Step 1: 写失败测试**

```go
package dice

import (
	"context"
	"testing"
)

func TestNotifyOnlyEventEmission(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)

	var names []string
	_ = d.EventBus.OnEvent("*", func(ctx context.Context, ev *AdapterEvent) error {
		names = append(names, ev.Name)
		return nil
	})

	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "milky"}}
	d.ImSession.EndPoints = append(d.ImSession.EndPoints, ep)
	ctx := &MsgContext{Dice: d, Session: d.ImSession, EndPoint: ep}

	EmitFriendRequest(ctx, "QQ:10086", "你好")
	EmitGroupRequest(ctx, "QQ:10001", "QQ:10086", "invite")
	EmitFriendJoined(ctx, &Message{Platform: "QQ", GroupID: "", Sender: SenderBase{UserID: "QQ:10086"}})
	EmitGuildJoined(ctx, &Message{Platform: "KOOK", GroupID: "KOOK:1", Sender: SenderBase{UserID: "KOOK:2"}})

	want := []string{EventNameFriendRequest, EventNameGroupRequest, EventNameFriendJoined, EventNameGuildJoined}
	if len(names) != len(want) {
		t.Fatalf("发射数量不符: %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("第 %d 个事件应为 %s，实为 %s", i, want[i], names[i])
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run TestNotifyOnlyEventEmission -v
```

预期：编译失败，`EmitFriendRequest` 等未定义。

- [ ] **Step 3: 新建 `dice/adapter_event_emit.go`（发射帮助函数）**

```go
package dice

// 通知类事件的标准化发射帮助函数。
// 这些事件旧分发路径是适配器内联扩展循环（或不存在），因此只发射、不设兼容订阅器。

// EmitFriendRequest 好友申请（仅通知）。
func EmitFriendRequest(ctx *MsgContext, userID string, comment string) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:     EventNameFriendRequest,
		UserID:   userID,
		Raw:      map[string]any{"comment": comment},
		Ctx:      ctx,
	})
}

// EmitGroupRequest 加群申请/邀请（仅通知）。subType 为 "add"/"invite"。
func EmitGroupRequest(ctx *MsgContext, groupID string, userID string, subType string) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:     EventNameGroupRequest,
		GroupID:  groupID,
		UserID:   userID,
		Raw:      map[string]any{"subType": subType},
		Ctx:      ctx,
	})
}

// EmitFriendJoined 成为好友。
func EmitFriendJoined(ctx *MsgContext, msg *Message) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:     EventNameFriendJoined,
		UserID:   msg.Sender.UserID,
		Raw:      map[string]any{"message": msg.Message},
		Ctx:      ctx,
		Detail:   msg,
	})
}

// EmitGuildJoined 加入频道。
func EmitGuildJoined(ctx *MsgContext, msg *Message) {
	if ctx == nil || ctx.Session == nil {
		return
	}
	ctx.Session.EmitEvent(&AdapterEvent{
		Name:     EventNameGuildJoined,
		GroupID:  msg.GroupID,
		SenderID: msg.Sender.UserID,
		Raw:      map[string]any{"message": msg.Message},
		Ctx:      ctx,
		Detail:   msg,
	})
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./dice/ -run TestNotifyOnlyEventEmission -v
```

预期：PASS。

- [ ] **Step 5: 在适配器中接线（每处一行发射，置于旧循环/旧处理之前）**

1. milky 好友申请：`dice/platform_adapter_milky.go` `handelFriendRequest`（约 474 行）内，`txt := fmt.Sprintf("收到QQ好友邀请...` 之前加：

```go
	EmitFriendRequest(ctx, uid, comment)
```

2. onebot 好友申请：`dice/platform_adapter_onebot_util.go` `handleReqFriendAction`（约 371 行）内，`txt := fmt.Sprintf("收到QQ好友邀请...` 之前加：

```go
	EmitFriendRequest(ctx, canonicalOnebotUserID(req.Get("user_id").String()), comment)
```

3. gocq 好友申请：`dice/platform_adapter_gocq.go` 定位（`grep -n '收到QQ好友邀请' dice/platform_adapter_gocq.go`，约 754 行附近 IgnoreFriendRequest 判断处），在通知日志前加同款 `EmitFriendRequest(ctx, <邀请人UID>, <comment>)`，UID 使用该处已有的格式化变量。

4. gocq 群申请：`dice/platform_adapter_gocq.go:635` 附近（`request_type":"group"` 注释处），处理逻辑入口加：

```go
	EmitGroupRequest(ctx, msgQQ.GroupID, msgQQ.UserID, msgQQ.SubType)
```

（`msgQQ` 为该处已有变量；若字段名不同——以该处结构体 `SubType string`（gocq.go:143）为准，GroupID/UserID 用实际字段名。）

5. milky 成为好友：`dice/platform_adapter_milky.go:572` 附近 `ext.OnBecomeFriend` 循环之前加：

```go
	EmitFriendJoined(ctx, msg)
```

6. onebot_util 成为好友：`dice/platform_adapter_onebot_util.go:253` 附近 `ext.OnBecomeFriend` 循环之前加同款。

7. gocq 成为好友：`dice/platform_adapter_gocq.go:828` 附近 `ext.OnBecomeFriend` 循环之前加同款。

8. kook 加入频道：`dice/platform_adapter_kook.go:315` 附近 `TriggerExtHook` 之前加：

```go
	EmitGuildJoined(mctx, msg)
```

（telegram/dingtalk/walleq/official_qq 等平台的事件发射留待后续批次，不在本计划范围。）

- [ ] **Step 6: 构建 + 全量 dice 回归**

```bash
go build ./...
go test ./dice/ -run 'TestNotifyOnlyEventEmission|TestCompat|TestEventBus' -v
```

预期：通过。

- [ ] **Step 7: 提交**

```bash
git add dice/adapter_event_emit.go dice/adapter_event_emit_test.go dice/platform_adapter_milky.go dice/platform_adapter_onebot_util.go dice/platform_adapter_gocq.go dice/platform_adapter_kook.go
git commit -m "feat(bus): 发射 friend/guild/request 类通知事件（并行发射，不改旧分发路径）"
```

---

### Task 8: seal.bus JS API（onEvent / getCapabilities）

**Files:**
- Create: `dice/event_bus_js.go`
- Modify: `dice/dice_jsvm.go`（seal 对象区，约 171 行 `ext := vm.NewObject()` 之前）
- Test: `dice/event_bus_js_test.go`

- [ ] **Step 1: 写失败测试（真实 goja 运行时）**

```go
package dice

import (
	"testing"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
)

func TestSealBusOnEventJS(t *testing.T) {
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{}}
	d.EventBus = NewEventBus(d)
	d.ExtLoopManager = NewJsLoopManager()

	loop := eventloop.NewEventLoop()
	version := d.ExtLoopManager.SetLoop(loop)

	got := make(chan string, 1)
	loop.RunOnLoop(func(vm *goja.Runtime) {
		// JS 回调向测试回传结果的辅助函数
		_ = vm.Set("__received", func(s string) { got <- s })
		bus := vm.NewObject()
		_ = registerBusObject(vm, bus, d, version)
		_ = vm.Set("bus", bus)
		_, err := vm.RunString(`
			bus.onEvent("group.muted", function(ctx, ev) {
				__received(ev.name + "|" + ev.groupId + "|" + ev.raw.duration);
			});
		`)
		if err != nil {
			t.Errorf("脚本执行失败: %v", err)
		}
	})
	loop.Start()
	defer loop.StopNoWait()

	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Platform: "QQ"}}
	ctx := &MsgContext{Dice: d, Session: d.ImSession, EndPoint: ep}
	d.ImSession.EmitEvent(&AdapterEvent{
		Name:    EventNameGroupMuted,
		GroupID: "QQ:10001",
		Raw:     map[string]any{"duration": 600},
		Ctx:     ctx,
	})

	select {
	case r := <-got:
		if r != "group.muted|QQ:10001|600" {
			t.Fatalf("JS 收到的事件数据不符: %s", r)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("JS 订阅者未被触发")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run TestSealBusOnEventJS -v
```

预期：编译失败，`registerBusObject` 未定义。

- [ ] **Step 3: 实现 `dice/event_bus_js.go`**

```go
package dice

import (
	"github.com/dop251/goja"
)

// registerBusObject 组装 seal.bus 对象。
// versionID 为当前 JS 事件循环版本号（调用方取自 dice_jsvm.go 中既有 versionID 变量）。
func registerBusObject(vm *goja.Runtime, bus *goja.Object, d *Dice, versionID int64) error {
	if err := bus.Set("onEvent", func(name string, handler func(ctx *MsgContext, ev *AdapterEvent)) error {
		return d.EventBus.OnEventJS(name, versionID, handler)
	}); err != nil {
		return err
	}
	if err := bus.Set("getCapabilities", func(platform string) []AdapterCapabilitySet {
		return GetAdapterCapabilitiesByPlatform(platform)
	}); err != nil {
		return err
	}
	return nil
}
```

注：`sendRaw` 在 Task 10 注册（依赖 Dice.SendRaw）。goja 会把 JS function 转换为 `func(ctx *MsgContext, ev *AdapterEvent)`；该转换闭包与当前 VM 绑定，故必须经 `EventBus.invokeJs` 的 `RunOnLoop` 回同版本循环调用（Task 4 已实现）。

- [ ] **Step 4: 接线 `dice/dice_jsvm.go`**

在 `ext := vm.NewObject()`（`dice/dice_jsvm.go:171`）之前插入：

```go
		bus := vm.NewObject()
		_ = seal.Set("bus", bus)
		if err := registerBusObject(vm, bus, d, versionID); err != nil {
			d.Logger.Errorf("注册 seal.bus 失败: %v", err)
		}
```

（`versionID` 为该作用域既有变量，`dice_jsvm.go:174` 处已在用。）

- [ ] **Step 5: 跑测试确认通过**

```bash
go test ./dice/ -run 'TestSealBus|TestEventBus' -v
```

预期：PASS。

- [ ] **Step 6: 提交**

```bash
git add dice/event_bus_js.go dice/event_bus_js_test.go dice/dice_jsvm.go
git commit -m "feat(bus): seal.bus JS API——onEvent 动态订阅与能力查询"
```

---

### Task 9: RawActionAdapter 接口与 Dice.SendRaw

**Files:**
- Create: `dice/raw_action.go`
- Test: `dice/raw_action_test.go`

- [ ] **Step 1: 写失败测试**

```go
package dice

import (
	"context"
	"testing"
)

type fakeRawAdapter struct {
	PlatformAdapter // 嵌入接口零值即可满足类型断言
}

func (f *fakeRawAdapter) RawAction(ctx context.Context, action string, params map[string]any) (any, error) {
	return map[string]any{"echo": params["user_id"]}, nil
}

func TestDiceSendRaw(t *testing.T) {
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "fakeproto",
		Platform:     "QQ",
		RawActions: map[string]AdapterRawActionSpec{
			"get_group_member_info": {Name: "get_group_member_info"},
		},
	})

	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{
		{
			EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "fakeproto", Enable: true, State: StateConnected},
			Adapter:          &fakeRawAdapter{},
		},
	}}

	ret, err := d.SendRaw("QQ", "get_group_member_info", map[string]any{"user_id": 123})
	if err != nil {
		t.Fatalf("SendRaw 失败: %v", err)
	}
	m, _ := ret.(map[string]any)
	if m["echo"] != float64(123) && m["echo"] != 123 {
		t.Fatalf("返回值不符: %v", ret)
	}

	// 不在能力清单中的动作必须报错
	if _, err := d.SendRaw("QQ", "no_such_action", nil); err == nil {
		t.Fatal("未声明的动作应报错")
	}
	// 无匹配端点
	if _, err := d.SendRaw("TG", "get_group_member_info", nil); err == nil {
		t.Fatal("无匹配端点应报错")
	}
	// 端点未实现 RawActionAdapter
	d2 := &Dice{}
	d2.ImSession = &IMSession{Parent: d2, EndPoints: []*EndPointInfo{
		{
			EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "fakeproto", Enable: true, State: StateConnected},
			Adapter:          fakeNoRawAdapter{},
		},
	}}
	if _, err := d2.SendRaw("QQ", "get_group_member_info", nil); err == nil {
		t.Fatal("未实现 RawAction 的适配器应报错")
	}
}

type fakeNoRawAdapter struct {
	PlatformAdapter
}
```

（`fakeNoRawAdapter` 与 `fakeRawAdapter` 的区别：不实现 `RawAction`。）

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run TestDiceSendRaw -v
```

预期：编译失败，`SendRaw` 未定义。

- [ ] **Step 3: 实现 `dice/raw_action.go`**

```go
package dice

import (
	"context"
	"fmt"
)

// RawActionAdapter 可选接口：支持 sendRaw 动作透传的适配器实现之。
// 未实现的适配器在 SendRaw 分发时被类型断言自然跳过。
type RawActionAdapter interface {
	RawAction(ctx context.Context, action string, params map[string]any) (any, error)
}

// SendRaw 出站动作透传：按平台定位在线端点，校验能力清单后调用适配器。
// 只要该协议能力清单声明了此动作即可调用，不另设权限。
func (d *Dice) SendRaw(platform, action string, params map[string]any) (any, error) {
	if d.ImSession == nil {
		return nil, fmt.Errorf("会话未初始化")
	}
	for _, ep := range d.ImSession.EndPoints {
		if ep == nil || ep.Adapter == nil {
			continue
		}
		if ep.Platform != platform || !ep.Enable {
			continue
		}
		caps, ok := GetAdapterCapabilities(ep.ProtocolType)
		if !ok {
			continue
		}
		if _, declared := caps.RawActions[action]; !declared {
			continue // 该协议未声明此动作，尝试下一个同平台端点
		}
		ra, ok := ep.Adapter.(RawActionAdapter)
		if !ok {
			continue
		}
		if params == nil {
			params = map[string]any{}
		}
		return ra.RawAction(context.Background(), action, params)
	}
	// 无任何端点声明该动作：给出可定位的错误
	protos := ""
	for _, set := range GetAdapterCapabilitiesByPlatform(platform) {
		if _, declared := set.RawActions[action]; declared {
			protos += set.ProtocolType + ","
		}
	}
	if protos != "" {
		return nil, fmt.Errorf("平台 %s 声明了动作 %s（协议 %s），但没有可用端点", platform, action, protos)
	}
	return nil, fmt.Errorf("平台 %s 不支持动作 %s", platform, action)
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./dice/ -run TestDiceSendRaw -v
```

预期：PASS。

- [ ] **Step 5: 提交**

```bash
git add dice/raw_action.go dice/raw_action_test.go
git commit -m "feat(bus): RawActionAdapter 可选接口与 Dice.SendRaw 能力校验分发"
```

---

### Task 10: onebot/milky 的 RawAction 实现与能力声明

**Files:**
- Modify: `dice/platform_adapter_onebot.go`（RawAction 实现 + init 能力声明）
- Modify: `dice/platform_adapter_milky.go`（RawAction 实现 + init 能力声明）
- Test: `dice/raw_action_adapters_test.go`

- [ ] **Step 1: 写失败测试**

```go
package dice

import (
	"context"
	"testing"
)

func TestAdapterCapabilityDeclarations(t *testing.T) {
	ob, ok := GetAdapterCapabilities("onebot")
	if !ok {
		t.Fatal("onebot 能力未注册")
	}
	for _, name := range []string{"send_group_msg", "send_private_msg", "delete_msg", "set_group_ban", "set_group_kick", "get_group_member_info"} {
		if _, ok := ob.RawActions[name]; !ok {
			t.Fatalf("onebot RawActions 缺少 %s", name)
		}
	}
	for _, name := range []string{EventNamePoke, EventNameGroupJoined, EventNameGroupMemberJoined, EventNameGroupLeave, EventNameFriendJoined, EventNameFriendRequest, EventNameGroupRequest} {
		if _, ok := ob.EmitEvents[name]; !ok {
			t.Fatalf("onebot EmitEvents 缺少 %s", name)
		}
	}

	mk, ok := GetAdapterCapabilities("milky")
	if !ok {
		t.Fatal("milky 能力未注册")
	}
	for _, name := range []string{"get_group_member_info", "send_group_nudge"} {
		if _, ok := mk.RawActions[name]; !ok {
			t.Fatalf("milky RawActions 缺少 %s", name)
		}
	}
}

func TestMilkyRawActionParamValidation(t *testing.T) {
	pa := &PlatformAdapterMilky{}
	// 未连接（IntentSession 为 nil）时应返回错误而非 panic
	_, err := pa.RawAction(context.Background(), "get_group_member_info", map[string]any{"group_id": 1, "user_id": 2})
	if err == nil {
		t.Fatal("未连接时应报错")
	}
	// 参数类型错误应报错而非 panic
	_, err = pa.RawAction(context.Background(), "get_group_member_info", map[string]any{"group_id": "abc", "user_id": 2})
	if err == nil {
		t.Fatal("非法参数应报错")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./dice/ -run 'TestAdapterCapabilityDeclarations|TestMilkyRawActionParamValidation' -v
```

预期：编译失败，`RawAction` 方法未定义。

- [ ] **Step 3: onebot 实现（`dice/platform_adapter_onebot.go` 文件尾部追加）**

```go
// onebot 能力声明：onebot 协议动作天然全透传（emitter.Raw），
// 能力清单声明常用动作，作为插件可发现、可校验的动作面。
func init() {
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "onebot",
		Platform:     "QQ",
		EmitEvents: map[string]AdapterEventSpec{
			EventNamePoke:              {Name: EventNamePoke, Description: "戳一戳"},
			EventNameGroupJoined:       {Name: EventNameGroupJoined, Description: "骰子加入群"},
			EventNameGroupMemberJoined: {Name: EventNameGroupMemberJoined, Description: "群成员加入"},
			EventNameGroupLeave:        {Name: EventNameGroupLeave, Description: "群成员离开/被踢"},
			EventNameFriendJoined:      {Name: EventNameFriendJoined, Description: "成为好友"},
			EventNameFriendRequest:     {Name: EventNameFriendRequest, Description: "好友申请（仅通知）", RequestOnly: true},
			EventNameGroupRequest:      {Name: EventNameGroupRequest, Description: "加群申请/邀请（仅通知）", RequestOnly: true},
		},
		RawActions: map[string]AdapterRawActionSpec{
			"send_group_msg":       {Name: "send_group_msg", Description: "发送群消息", Params: map[string]string{"group_id": "int64", "message": "onebot消息段数组"}},
			"send_private_msg":     {Name: "send_private_msg", Description: "发送私聊消息", Params: map[string]string{"user_id": "int64", "message": "onebot消息段数组"}},
			"delete_msg":           {Name: "delete_msg", Description: "撤回消息", Params: map[string]string{"message_id": "int32"}},
			"set_group_ban":        {Name: "set_group_ban", Description: "群禁言", Params: map[string]string{"group_id": "int64", "user_id": "int64", "duration": "int64秒，0解禁"}},
			"set_group_kick":       {Name: "set_group_kick", Description: "移出群聊", Params: map[string]string{"group_id": "int64", "user_id": "int64", "reject_add_request": "bool"}},
			"get_group_member_info": {Name: "get_group_member_info", Description: "获取群成员信息", Params: map[string]string{"group_id": "int64", "user_id": "int64", "no_cache": "bool"}},
		},
	})
}

// RawAction onebot 动作全透传：直接转发 OneBot 协议 action。
func (p *PlatformAdapterOnebot) RawAction(ctx context.Context, action string, params map[string]any) (any, error) {
	if p.sendEmitter == nil {
		return nil, fmt.Errorf("onebot 端点未连接")
	}
	raw, err := p.sendEmitter.Raw(ctx, emitter.Action(action), params)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		// 无法结构化时返回原始报文
		return map[string]any{"raw": string(raw)}, nil
	}
	return out, nil
}
```

import 需补 `"context"`、`"encoding/json"`、`"fmt"`（`emitter` 已有）。

- [ ] **Step 4: milky 实现（`dice/platform_adapter_milky.go` 文件尾部追加）**

```go
// milky 能力声明。RawAction 首批仅暴露参数为标量的动作（避免消息段编解码进入 v1）。
func init() {
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "milky",
		Platform:     "QQ",
		EmitEvents: map[string]AdapterEventSpec{
			EventNamePoke:              {Name: EventNamePoke, Description: "戳一戳"},
			EventNameGroupJoined:       {Name: EventNameGroupJoined, Description: "骰子加入群"},
			EventNameGroupMemberJoined: {Name: EventNameGroupMemberJoined, Description: "群成员加入"},
			EventNameGroupLeave:        {Name: EventNameGroupLeave, Description: "群成员离开/被踢"},
			EventNameFriendJoined:      {Name: EventNameFriendJoined, Description: "成为好友"},
			EventNameFriendRequest:     {Name: EventNameFriendRequest, Description: "好友申请（仅通知）", RequestOnly: true},
		},
		RawActions: map[string]AdapterRawActionSpec{
			"get_group_member_info": {Name: "get_group_member_info", Description: "获取群成员信息", Params: map[string]string{"group_id": "int64", "user_id": "int64"}},
			"send_group_nudge":      {Name: "send_group_nudge", Description: "群内戳一戳", Params: map[string]string{"group_id": "int64", "user_id": "int64"}},
		},
	})
}

// rawActionParam 从 map 参数取 int64，缺失或类型不符时报错。
func rawActionParam(params map[string]any, key string) (int64, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("缺少参数 %s", key)
	}
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case float64:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("参数 %s 类型不符: %T", key, v)
	}
}

// RawAction milky 动作透传（首批：标量参数动作）。
func (pa *PlatformAdapterMilky) RawAction(ctx context.Context, action string, params map[string]any) (any, error) {
	if pa.IntentSession == nil {
		return nil, fmt.Errorf("milky 端点未连接")
	}
	switch action {
	case "get_group_member_info":
		groupID, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		userID, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return pa.getGroupMemberInfo(strconv.FormatInt(groupID, 10), strconv.FormatInt(userID, 10), false)
	case "send_group_nudge":
		groupID, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		userID, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		if err := pa.IntentSession.SendGroupNudge(groupID, userID); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	default:
		return nil, fmt.Errorf("milky 不支持动作 %s", action)
	}
}
```

import 需补 `"context"`、`"fmt"`（`strconv` 已有）。若 `getGroupMemberInfo` 的返回类型不是 `(any, error)` 兼容形式，包一层 `map[string]any` 序列化返回。

- [ ] **Step 5: 跑测试确认通过**

```bash
go test ./dice/ -run 'TestAdapterCapabilityDeclarations|TestMilkyRawActionParamValidation|TestDiceSendRaw' -v
go build ./...
```

预期：PASS、构建通过。

- [ ] **Step 6: 提交**

```bash
git add dice/platform_adapter_onebot.go dice/platform_adapter_milky.go dice/raw_action_adapters_test.go
git commit -m "feat(bus): onebot/milky RawAction 实现与能力清单声明"
```

---

### Task 11: seal.bus.sendRaw 与全量验证

**Files:**
- Modify: `dice/event_bus_js.go`（注册 sendRaw）
- Test: `dice/event_bus_js_test.go`（追加）

- [ ] **Step 1: 在 `registerBusObject`（`dice/event_bus_js.go`）中追加 sendRaw**

```go
	if err := bus.Set("sendRaw", func(platform string, action string, params map[string]any) (any, error) {
		return d.SendRaw(platform, action, params)
	}); err != nil {
		return err
	}
```

（注意：sendRaw 在 JS 循环线程上同步执行网络 IO（onebot echo 超时 10s），v1 接受此限制，文档中注明。）

- [ ] **Step 2: 追加 JS 级测试**

```go
func TestSealBusSendRawJS(t *testing.T) {
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "fakeproto2",
		Platform:     "QQ",
		RawActions: map[string]AdapterRawActionSpec{
			"echo_action": {Name: "echo_action"},
		},
	})
	d := &Dice{}
	d.ImSession = &IMSession{Parent: d, EndPoints: []*EndPointInfo{
		{
			EndPointInfoBase: EndPointInfoBase{Platform: "QQ", ProtocolType: "fakeproto2", Enable: true, State: StateConnected},
			Adapter:          &fakeRawAdapter{},
		},
	}}
	d.EventBus = NewEventBus(d)
	d.ExtLoopManager = NewJsLoopManager()

	loop := eventloop.NewEventLoop()
	version := d.ExtLoopManager.SetLoop(loop)
	errCh := make(chan error, 1)
	loop.RunOnLoop(func(vm *goja.Runtime) {
		bus := vm.NewObject()
		_ = registerBusObject(vm, bus, d, version)
		_ = vm.Set("bus", bus)
		_, err := vm.RunString(`
			var r = bus.sendRaw("QQ", "echo_action", {user_id: 42});
			if (!r || r.echo !== 42) { throw new Error("sendRaw 返回不符: " + JSON.stringify(r)); }
		`)
		errCh <- err
	})
	loop.Start()
	defer loop.StopNoWait()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("sendRaw JS 调用失败: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("sendRaw JS 调用超时")
	}
}
```

（`fakeRawAdapter` 复用 Task 9 定义；需 `import "time"`。）

- [ ] **Step 3: 全量验证**

```bash
go build ./...
go test ./dice/... ./message/...
gofmt -l dice/ | grep -v '^$' && echo "存在未格式化文件" || echo "gofmt clean"
go vet ./dice/...
```

预期：构建通过、全部测试 PASS、gofmt 无输出、vet 无告警。

- [ ] **Step 4: 提交**

```bash
git add dice/event_bus_js.go dice/event_bus_js_test.go
git commit -m "feat(bus): seal.bus.sendRaw 出站透传 JS API"
```

---

## 明确不在本计划范围（后续批次）

- telegram/dingtalk/walleq/officialqq/dodo/discord/minecraft/satori/http 等平台的事件发射与能力声明
- `group.muted` 的实际发射（onebot `group_ban` notice，gocq.go:1014 附近有原始信号，接线路径同 Task 7）
- `message.received` 等消息管线事件（spec 已界定不入总线）
- 请求类事件的同意/拒绝能力（spec 已界定仅通知）
- 异步 worker 池开关与配置化（代码留有 `store.Async` 位）
- 旧字段回调的完整映射表（poke/group.leave/group.joined/group.member_joined 已迁移；friend.joined/guild.joined 保持适配器内联调用旧字段）

## 风险与回退

- 行为风险集中在 Task 6（四个 OnXxx 改造）。回退方式：`git revert` 对应提交，适配器调用点未动，回退无残留。
- Task 7 均为并行发射（新增代码路径），出问题可单独 revert。
- GoEventBus 依赖仅被 `dice/event_bus.go` 引用，边界清晰，可整体替换为简单 fan-out 而不影响其余代码。
