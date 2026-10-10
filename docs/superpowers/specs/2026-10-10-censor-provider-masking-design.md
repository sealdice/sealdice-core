# 敏感词检测 Provider 化与脱敏设计

日期：2026-10-10
分支：`sensitive-words`
基线：`seal/master` @ `4c2d1603`

## 背景

现有敏感词功能（`dice/censor/*`、`dice/dice_censor.go`）存在若干问题：

- **匹配算法**：手写 trie 从每个位置重扫、无 fail 链接，复杂度约 `O(n·L)`；且 `trie.go:68-71` 遇 `cur.end` 即 `break`，只取最短前缀，长词/高级别词漏报。
- **大小写**：`addWord` 一律小写词库键（`censor.go:230`），但 `Check` 从不对正文小写（`censor.go:275`），`CaseSensitive=false` 实际失效。
- **热路径**：`CensorManager.Check` 每条被检查消息无条件执行 5 次 `COUNT`（`dice_censor.go:125` + `service/censor_log.go:44`），命中再同步写库。
- **脱敏缺失**：命中「对外发送」时整条替换为模板（`im_helpers.go:437`、`config.go:975` 注释已预告「未来可加入部分拦截」）。
- **不可扩展**：检测逻辑写死在内置词库，无法接入语义模型或外部 API。

本设计在**边缘设备（1~2 核）**约束下：内存、启动时间、磁盘 I/O 为硬约束，吞吐非瓶颈。目标是用纯 Go、零 CGO 的方式修复上述问题，并提供可插拔的检测 Provider 与「脱敏」处置。

## 目标

1. 修复大小写失效与最长匹配漏报。
2. 用纯 Go Aho-Corasick 替换手写 trie，检测结果返回**命中 span（原始文本 rune 偏移）**。
3. 抽象 **Provider** 检测层，v1 交付 `localAC`（内置，离线必跑）与 `httpProvider`（外部 API）。
4. 新增**脱敏**处置：按 span 将命中片段替换为占位符，作用于对外发送路径。
5. 保持现有配置与行为向后兼容。

## 非目标（本次不做）

- **log 脱敏**：跑团日志（`dice/storylog/*`、`api/story_log.go`）维持现状存原文，不做读取/上传脱敏。
- **Provider 对外开放**：配置结构保留 `censorProviders` 字段，但 v1 不提供 API/UI 编辑入口。
- **第三方插件加载**：不实现运行时 `.so`/JS 插件加载；同接口后续增量接入。
- **语义级脱敏**：不引入序列标注模型；`VerdictOnly` 的 Provider 不做 span 脱敏。
- **DB 热路径治理**：本设计不改动计数/写库路径（另立任务）。

## 术语

- **Hit**：一次命中，含 `Start/End`（原始文本 rune 偏移）、`Word`、`Level`。
- **Capability**：Provider 能力，`VerdictOnly`（只给结论）或 `SpanCapable`（结论 + 位置）。
- **Disposition**：对内容的处置，`Pass | Mask | Block | Warn`。
- **归一化偏移映射**：归一化后文本 rune 下标 → 原始文本 rune 下标的映射，用于把命中 span 翻译回原文。

## 架构总览

```
文本 ──► 归一化(NFKC+小写+去零宽, 生成偏移映射) ──► 匹配文本 + norm→orig 映射
                                                      │
                                                      ▼
                              Engine ──► [localAC(必跑), httpProvider(可选)] ──► 合并 Result
                                                      │
                                                      ▼
                              Policy: (Level × Capability) ──► Disposition
                                                      │
                          ┌───────────────────────────┼──────────────────────────┐
                          ▼                           ▼                          ▼
                    Mask: Masker 按 span 替换    Block: 整条模板替换        Warn: 警告/通知
                          │
                          ▼
                       真正发送
```

原则：**检测与处置分离**。Provider 只产出命中；Policy 决定处置；文本变换与侧效应分开。

## 组件设计

### 1. censor 核心改造（`dice/censor`）

- **匹配器**：引入纯 Go Aho-Corasick，候选 `github.com/BobuSumisu/aho-corasick`（double-array，MIT）。实现时须验证其返回 `Start/End` 偏移与许可证；若不满足，退回 `github.com/cloudflare/ahocorasick` 或自实现基于 double-array 的 AC。匹配器构建后**不可变**。
- **返回值**：`Check` 由 `map[string]Level` 改为返回 `[]Hit`，携带原始文本 rune 偏移。
  ```go
  type Hit struct {
      Start, End int   // 原始文本 rune 偏移，半开区间 [Start, End)
      Word       string
      Level      Level
  }
  ```
- **大小写修复**：归一化阶段对正文与词库键同时做小写转换，消除 `CaseSensitive=false` 失效问题。`CaseSensitive=true` 时跳过小写步骤。
- **最长匹配**：由 AC 保证（同起点取最长，重叠命中全部返回）。
- **拼音**：保留现能力——将词库词的拼音串作为附加 pattern 插入，命中同样带 span。
- **过滤正则**：保留 `FilterRegexStr`。正则替换与归一化共同构建**匹配文本**，并一同纳入 `norm→orig` 偏移映射（见第 4 节）。任何改变长度的步骤都必须产出映射，否则该 hit 降级。
- **等级**：保留 `Ignore/Notice/Caution/Warning/Danger` 与「同词取最高级」。

### 2. Provider 抽象与 Engine 管线（`dice/censor` 或新包 `dice/censor/provider`）

```go
type Capability int

const (
    VerdictOnly Capability = iota // 只给结论
    SpanCapable                   // 结论 + span
)

type Span struct {
    Start, End int // 原始文本 rune 偏移
}

type Result struct {
    Level  Level    // 风险等级
    Reason string   // 命中原因/标签/说明，必填
    Spans  []Span   // 仅当 Provider.Capability() == SpanCapable 时有效
}

type Provider interface {
    Name() string
    Capability() Capability
    Check(ctx context.Context, text string) (*Result, error) // 返回非 nil Result 视为命中
    Reload() error
}
```

- **localAC**：包装 AC 匹配器，`SpanCapable`，离线必跑，支持脱敏（主路径）。`Reason` 为命中词。
- **httpProvider**：按配置调用外部 API。请求体包含匹配文本；响应 JSON 契约：
  ```json
  { "level": 3, "reason": "语义风险:涉政", "spans": [[5, 8]] }
  ```
  `spans` 缺省或能力声明为 `VerdictOnly` 时忽略。配置项：`{name, url, token, timeoutMs, failMode, capability}`。`failMode ∈ {open, closed}`。
- **Engine**：持有 `[]Provider`。`Check` 时 `localAC` 同步内联执行，`httpProvider` 执行并受 `timeoutMs` 约束；合并规则：
  - 取所有命中 `Level` 最高值作为结论；
  - `Reason` 汇总各 Provider 的原因；
  - `Spans` 取所有 `SpanCapable` 命中的并集。
- **重载**：构建新的 Provider 集合与 `localAC` 匹配器，**原子替换指针**；匹配路径只读不可变快照。

### 3. Policy 处置（`dice/dice_censor.go`）

`Disposition` 由 `(Level × Capability)` 决定：

| 命中来源 | 处置 |
|---|---|
| `SpanCapable`（localAC、span 型 httpProvider） | **Mask**：按 span 脱敏 |
| `VerdictOnly`（分类型 httpProvider） | **Block 或 Warn**：不脱敏；`Reason` 带进警告/通知/日志 |

- 保留现有**阈值 + handler**（`SendWarning/SendNotice/BanUser/BanGroup/BanInviter/AddScore`）作为侧效应，与内容处置解耦。
- 侧效应触发逻辑（`dice_censor.go:246-347`）保持：按等级从高到低、超阈值执行、`mctx.Censored` 去重。
- `VerdictOnly` 命中的处置默认沿用「整条模板替换 + 警告」。

### 4. Masker（`dice/dice_censor.go`）

- 占位符来源：模板 `核心:拦截_敏感词过滤_替换占位符`，默认值 `■`（U+25A0 BLACK SQUARE）。经 `DiceFormatTmpl` 解析，取固定值（不随机）。
- 替换规则：对每个 span，将 `[Start, End)` 的 runes 逐个替换为占位符串，长度概念保留。例：`黑夜总会来临` 命中 `夜总会` → `黑■■■来临`。若占位符为空串则删除命中片段。
- **偏移映射**：匹配在归一化文本上进行，`span` 先经 `norm→orig` 映射翻译回原始文本再替换。映射按 rune 逐个建立；若某归一化步骤产生**长度变化**（如连字折叠）导致无法一一映射，则该 hit 降级为 **Block/Warn**，不强行打码。
- 对外发送接入点（`dice/im_helpers.go`）：
  - `TryReplyToSenderMergedForward:133`
  - `ReplyGroupRaw:419`
  - `ReplyPersonRaw:491`
  在现有「剥海豹码/CQ码 → 检查」流程后，由 `Block` 改为按 disposition 执行 Mask（替换 span）或 Block（模板替换）。

### 5. 配置与模式

- **模式（已定）**：脱敏**跟随现有 `censorMode`**——仅 `censorMode=OnlyOutputReply` 时对外发送路径检查并脱敏；输入拦截仍按 mode。不改变 `AllInput`/`OnlyInputCommand` 的语义。
- 新增配置字段（结构保留、v1 不开放编辑）：
  - `censorProviders`：Provider 列表（name/url/token/timeoutMs/failMode/capability）。运行时若为空则仅 `localAC`。
- 新增默认模板：
  - `核心:拦截_敏感词过滤_替换占位符`，默认 `■`，SubType `拦截`（注册于 `config.go` 模板表与 SubType 表）。
- 复用现有：`enableCensor`、`censorMode`、`censorCaseSensitive`、`censorMatchPinyin`、`censorFilterRegexStr`、`censorThresholds`、`censorHandlers`、`censorScores`。

## 数据流

1. 对外发送前，若 `enableCensor && censorMode==OnlyOutputReply`：剥海豹码/CQ码 → 归一化（生成偏移映射）。
2. `Engine.Check` 运行 Provider，合并为 `Result`。
3. `Policy` 由 `(Level × Capability)` 得出 `Disposition`。
4. `Mask`：`Masker` 用偏移映射把 span 翻回原文并用占位符替换；`Block`：整条替换模板；`Warn`：发送警告。
5. 侧效应 handler 按阈值执行（不变）。
6. 发送最终文本。

## 错误处理与降级

- **远端超时/报错**：按 `failMode` 处理；`open`（默认）跳过并告警（对齐现有 `dice_censor.go:210` 行为），`closed` 视为命中拦截。
- **归一化无法映射**：该 hit 降级为 Block/Warn。
- **无有效 span**：不脱敏，按能力回退到 Block/Warn。
- **Provider 未提供 `Reason`**：视为契约错误，记录日志并按 `VerdictOnly` 处理。
- **重载期间**：匹配读取旧快照，新快照构建完成后原子切换，无数据竞争。

## 兼容性

- 现有配置文件零迁移：新字段缺省即仅启用 `localAC`。
- 现有模板与阈值/handler/scores 语义不变。
- 行为差异仅在：修 Bug 后命中结果更准；`OnlyOutputReply` 模式由「整条替换」变为「按 span 脱敏」。后者为本设计的目标行为。

## 测试计划

- **censor 单元测试**：AC span 正确性（重叠、最长匹配）、大小写（含 `CaseSensitive` 两态）、拼音 span、过滤正则、同词取最高级。
- **偏移映射测试**：NFKC/小写/去零宽下的 `norm→orig` 映射；长度变化时的降级判定。
- **Policy 真值表**：`(Level × Capability) → Disposition` 全覆盖。
- **Masker 测试**：多 span、相邻 span、空占位符、边界（首/尾/整条）、模板解析。
- **httpProvider 测试**：mock server 正常/错误/超时；`failMode` open/closed；spans 解析。
- **集成测试**：`OnlyOutputReply` 模式下对外回复被脱敏；`VerdictOnly` 命中走警告；`AllInput`/`OnlyInputCommand` 输入路径行为不变。
- **并发测试**：`Reload` 与 `Check` 并发下的竞态（`-race`）。

## 后续（非本次范围）

- 开放 `censorProviders` 的 API/UI 配置入口。
- Provider 插件加载（JS 钩子或其它）。
- 语义级脱敏（序列标注 / toxic span 模型）。
- DB 热路径治理（内存计数 + 异步落库）。
- log 脱敏（如后续需要）。
