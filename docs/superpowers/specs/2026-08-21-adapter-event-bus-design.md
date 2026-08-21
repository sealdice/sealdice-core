# 适配器事件总线设计（AdapterEvent Bus）

> 日期：2026-08-21
> 状态：已评审，待实现

## 背景与目标

当前海豹核心的事件分发是**类型化回调**：`ExtInfo`（`dice/dice.go`）上有 `onPoke`、
`onGroupJoined`、`onGroupMemberJoined`、`onBecomeFriend`、`onGroupLeave` 等回调字段；
每个适配器（adapter）识别平台信号后各自调用 `IMSession.OnXxx(...)`，再由 `IMSession`
分发给插件。这套模型存在几个问题：

- 每个通知都要写一个 Go 结构体 + 一个 `IMSession` 方法 + 各适配器调用点，覆盖零散。
- “加好友/加群申请”等请求类事件没有统一，各适配器内部自处理，插件接触不到。
- 当前代码库不存在“被禁言”等事件。
- 不同适配器能力参差不齐（milky/gocq/onebot 丰富，多数平台很少），但没有任何可发现的
  能力清单。

本设计用**通用事件总线**统一入站事件分发，让不同适配器的能力差异变成可发现、可文档化的
清单，并提供通用 `sendRaw` 出站透传。

## 关键决策

- **分发核心**：使用第三方库 `github.com/Protocol-Lattice/GoEventBus`
  （lock-free ring buffer + fan-out + 异步 worker pool + DLQ + 中间件 + 背压）。
- **领域对象命名**：`AdapterEvent`。刻意不用 `Event`，以区别于未来可能引入的**插件自身的
  event 机制**（两者到时候不是一回事），也避免与库类型 `GoEventBus.Event` 混淆。
- **请求类事件**：包含（加好友/加群申请），但仅通知、不回复（不开放同意/拒绝）。
- **订阅方式**：插件按事件名动态订阅。
- **出站发送**：`sendRaw(平台, 动作, 参数)` 通用透传；只要该平台能力清单里有此动作即可调用，
  不另设白名单/权限。
- **兼容性**：内部完全替换为总线；对外保留旧 JS 回调名（`onPoke` 等）作为兼容映射，
  保证存量插件与配置不破坏。

## 架构总览

```
适配器(adapter)                   核心(dice)                     插件(JSVM)
   │                                 │                              │
   │ EmitEvent(AdapterEvent)         │                              │
   ├────────────► IMSession ──► Dice ─┴────────────────┐            │
   │                                GoEventBus.EventStore          │
   │                                 │  fan-out(按事件名 Projection)│
   │                                 ├──► 核心/内置订阅器(Golang)    │
   │                                 └──► 桥接到 JSVM               │
   │                                        │  onEvent(name, h)     │
   │                                        └──► 插件 handler        │
   │                                                              │
   │  sendRaw: adapter.RawAction(action, params) ◄── bus.sendRaw()  │
   └──────────────────────────────────────────────────────────────┘
```

入站（事件分发）与出站（sendRaw）是两条独立通道：GoEventBus 只管入站分发，
`sendRaw` 走适配器接口 + 能力清单。

## 1. 事件核心（入站分发）

### 1.1 领域对象 `AdapterEvent`

```go
// dice/events/adapter_event.go
type AdapterEvent struct {
    ID         string         `json:"id"`          // 唯一标识
    Name       string         `json:"name"`        // 事件名，如 "group.member_joined"
    Platform   string         `json:"platform"`    // "QQ" / "DISCORD" / ...
    EndPointID string         `json:"endpoint_id"`
    GroupID    string         `json:"group_id"`
    UserID     string         `json:"user_id"`
    SenderID   string         `json:"sender_id"`
    Raw        map[string]any `json:"raw"`         // 适配器原样透传的数据
    Time       time.Time      `json:"time"`
}
```

注意：请求类事件（加好友/加群申请）也走 `AdapterEvent`，但**只通知不回复**，不携带回复通道。

### 1.2 总线接线

- 全局/单例总线挂载在 `Dice` 上，内部持有一个 `GoEventBus.EventStore`。
- 映射：入库时 `GoEventBus.Event{Projection: ev.Name, Data: ev}`（`Projection` 复用为事件名
  字符串，`Data` 复用为 `*AdapterEvent`）。
- 提供帮助函数：

```go
func (d *Dice) EmitEvent(ctx context.Context, ev *AdapterEvent) error {
    if err := d.eventStore.Subscribe(ctx, GoEventBus.Event{
        ID:         ev.ID,
        Projection: ev.Name,
        Data:       ev,
    }); err != nil {
        return err
    }
    d.eventStore.Publish()
    return nil
}
```

- `store.Async = true` + DLQ：插件 handler 跑在 worker pool，panic/error 进 DLQ，遵循现有
  `CheckBotOnline`/恢复语义，单个订阅者异常不打断其它订阅者。
- back-pressure 策略：以 onebot 系高压场景为准，默认 `DropOldest`，必要时可对关键事件
  单独改用 `Block`/`ReturnError`。

## 2. 适配器能力清单（capability manifest）

每个适配器在运行时组装并暴露两张表：

```go
type CapabilitySet struct {
    EmitEvents map[string]EventSpec // 能发的事件名 -> 规格
    RawActions map[string]RawAction // sendRaw 支持的动作名
}

type EventSpec struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Fields      []string `json:"fields"` // AdapterEvent.Raw 中会出现的字段
    RequestOnly bool     `json:"request_only"` // 是否请求类（仅通知）
}

type RawAction struct {
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Params      map[string]string `json:"params"` // 参数名 -> 类型说明
}
```

示例：

- milky/onebot：`EmitEvents = {group.joined, group.member_joined, group.leave, poke,
  friend.request, group.request, group.muted, ...}`；
  `RawActions = {send_message, recall, set_ban, kick, get_group_member_info, ...}`
- 较弱的适配器（如某台）：`EmitEvents = {group.joined}`；`RawActions = {send_message}`。

对外提供可查询接口（`bus.getCapabilities(platform)`），官方文档据此生成能力对照表。

## 3. 发射点（adapter → bus）

- 新增统一入口 `IMSession.EmitEvent(ctx, *AdapterEvent)`，内部转发到 `Dice.EmitEvent`。
- 迁移各适配器的 `OnGroupJoined/OnPoke/OnGroupLeave/...` 调用：保留现有 `IMSession.OnXxx`
  函数名作为迁移过渡，内部改为构造 `AdapterEvent` 并 `EmitEvent`。最终每条平台信号的发射点
  收敛为一条路径。
- 平台信号的检测仍留在各适配器内（milky/onebot 各自解析 webhook/notice），总线不负责
  这些协议细节，只负责统一承载与分发。

## 4. 插件订阅 + 旧接口兼容

JS 端新增命名式订阅：

```js
bus.onEvent("group.muted", function(ctx, ev) { /* ... */ });
// 支持前缀/通配，例："group.*"
```

Go 侧：

- 新总线按 `Name` 分发到订阅器：核心/内置功能注册 Go 订阅器；JS 插件通过 JSVM 桥注册。
- **兼容层**：把旧 JS 字段名（`onPoke`、`onGroupJoined`、`onGroupMemberJoined`、
  `onBecomeFriend`、`onGroupLeave`、`onMessageReceived`…）映射到对应事件名。插件沿用旧名
  书写，底层转为 `onEvent("poke", ...)` 订阅。存量插件、存量配置完全不受影响，内部已统一。
- JSVM 里 `onEvent` 成为 `ExtInfo` 上可调用的方法（jsbind `onEvent`），旧字段保留为 setter
  兼容入口。

事件名与旧字段映射示例（迁移时确定全量清单）：

| 旧 JS 字段            | 新事件名                 |
|----------------------|--------------------------|
| `onPoke`             | `poke`                   |
| `onGroupJoined`      | `group.joined`           |
| `onGroupMemberJoined`| `group.member_joined`    |
| `onGroupLeave`       | `group.leave`            |
| `onBecomeFriend`     | `friend.joined`          |
| `onGuildJoined`      | `guild.joined`           |
| `onMessageReceived`  | `message.received`       |

## 5. sendRaw 出站

```js
bus.sendRaw("QQ", "set_ban", { groupId, userId, duration });
```

- 实现：适配器接口 `RawAction(action string, params map[string]any) (any, error)`。
- 只要适配器能力清单定义该动作即可调用（不另设权限/白名单）。
- 未在能力清单中的动作 → 返回明确错误“平台不支持该动作”，不给插件假成功。
- 事件总线内置 action 注册表，联动 `Capabilities.RawActions`。

## 6. 错误处理与治理

- 插件 handler panic/error → DLQ，可查可重放（`bus.getDeadLetters()` 供排障）。
- 事件总线提供 `Metrics()`（published/processed/errors）供排查。
- 背压：`DropOldest` 适合实时消息流；若某事件不容丢失可对该 store/通道改用 `Block`。

## 7. 迁移与测试

迁移顺序：

1. 落地 `AdapterEvent` + `Dice.EmitEvent` + GoEventBus 注入 + `onEvent` JS 接口（纯增量）。
2. 把各适配器发射点迁到统一入口。
3. 把旧 JS 字段改造成兼容映射（存量插件零破坏）。
4. sendRaw + 能力清单对外查询接口。

测试：

- 单测：事件名解析、fan-out 顺序、back-pressure 策略、DLQ 捕获 panic/error。
- 兼容映射测试：旧字段 → 新总线路由，存量插件不回归。
- 适配器级测试：milky/onebot 上报能力 = 弱平台上报能力的差异断言。
- JS 集成测试：`onEvent` 订阅、通配/前缀订阅、sendRaw 返回值与错误。

## 开放问题（迁移时确认）

- 旧字段的完整映射表（含 `onNotCommandReceived`、`onCommandReceived` 等非通知类字段是否
  纳入总线，或维持命令管线处理）。
- `store.Async` 的 worker 池大小与队列长度取值。
- 前缀订阅（`group.*`）的实现粒度（字符串前缀匹配即可满足多数场景）。
