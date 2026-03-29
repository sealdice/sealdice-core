# Message IR / UniMessage 设计说明

## 目标

海豹仍然继续支持用户侧输入：

- 普通文本
- CQCode
- 海豹码

但它们不再应当作为海豹内部消息流转时的“主协议”。

当前目标是把海豹内部消息统一到一套中间表示，也就是 Message IR / UniMessage：

- 外层使用 `MessageEnvelope` 承载元信息
- 内容使用 `Segments []message.IMessageElement`

在当前实现中，`Segments` 就是实际落地的 UniMessage。

## 核心模型

当前消息外层结构位于：

- `dice/im_session.go`

核心规则：

- `Message.Segment` 是唯一真源
- `Message.Message` 是兼容文本视图

也就是说：

1. 业务逻辑应优先以 `Segment` 为准
2. `Message` 只用于兼容旧逻辑、旧扩展、旧日志文本消费

这套主从关系由：

- `(*Message).normalizeForPipeline()`

负责收口。

它的职责是：

1. 当消息只有旧文本时，把旧文本桥接成 `Segment`
2. 当消息只有 `Segment` 时，派生兼容 `Message`

## 元素集合

当前 Message IR 元素定义位于：

- `message/message.go`

当前最小元素集合包括：

- `TextElement`
- `AtElement`
- `ReplyElement`
- `ImageElement`
- `RecordElement`
- `FileElement`
- `FaceElement`
- `PokeElement`
- `TTSElement`
- `DefaultElement`

其中 `DefaultElement` 必须保留，用于承接未知平台消息段或暂未支持的消息能力，避免静默丢失信息。

## 解析层

用户侧旧语法的统一入口目前仍然是：

- `message.ConvertStringMessage()`

它负责把：

- 普通文本
- CQCode
- 海豹码

统一转换为 `Segments`。

为了避免在“仅仅做桥接”时触发文件读取或网络访问，当前还支持：

- `message.WithResolveResource(false)`

这个选项用于：

- 把旧文本标准化为 `Segment`
- 但不去真的解析图片/文件资源

这样可以保证入口统一时不会引入额外 IO 副作用。

## 文本视图

当前已经补出的文本投影能力位于：

- `message/message_segment_text.go`

目前已有三类基础能力：

- `SegmentsToText()`
  - 兼容文本视图
- `SegmentsToLegacyCQText()`
  - 旧 CQ 文本兼容视图
- `ToSegmentText()` / `ParseSegmentText()`
  - 带占位符映射的文本视图基础能力

当前建议使用方式：

- 兼容旧命令逻辑：`SegmentsToLegacyCQText()`
- 兼容普通文本视图：`SegmentsToText()`

后续目标：

- 给命令层补真正的 `CommandTextView`
- 逐步减少“segment 再拼回 CQ”的桥接依赖

## 执行入口

当前执行入口状态如下：

- `IMSession.ExecuteNew()` 是 segment-first 的主执行入口
- `IMSession.Execute()` 也已经会先做消息标准化

这意味着：

- 海豹现在已经具备统一入站标准化能力
- 即使有些旧逻辑还在读 `Message.Message`，消息在进入主链路前也已经尽量统一成了 `Segment`

## 发送链路

当前发送方向的目标是：

- 业务层产出 `Segments`
- adapter 负责把 `Segments` 编码成平台 payload

兼容规则是：

- `SendToGroup(text)` / `SendToPerson(text)` 继续保留
- 但它们应该只做：
  - `text -> ConvertStringMessage(text) -> SendSegment*`

目前这条规则已经在第一批 adapter 上落地。

## 当前已对齐的 adapter

目前已完成第一轮 segment-first 发送或 segment-preserving 入站改造的 adapter 包括：

- `platform_adapter_gocq.go`
- `platform_adapter_walleq.go`
- `platform_adapter_milky.go`
- `platform_adapter_discord.go`
- `platform_adapter_kook.go`
- `platform_adapter_dodo.go`
- `platform_adapter_satori.go`

说明：

- 这些 adapter 并不都已经做到“完全原生结构化入站”
- 但至少已经进入同一套 `Segments` 主链路
- `SealChat` 本轮未改，因为需要考虑项目对接影响
- `Red` 本轮不再考虑

## 目前已经统一到什么程度

当前已经做到的部分：

1. 海豹已经有实际上的内部 Message IR
   - 即 `Message.Segment`

2. 旧文本输入已经能够统一桥接到 IR
   - 通过 `normalizeForPipeline()`

3. 第一批 adapter 已能以 `SendSegment*` 作为主发送链路

4. smoke tests 已更新为覆盖：
   - 旧文本桥接到 segment
   - `Segment` 为真源
   - `Message` 为兼容视图

## 目前还没有完全统一的部分

当前仍处于过渡状态的部分：

1. 命令解析仍然在部分地方依赖 Legacy CQ 文本桥
   - `dice/cmd_parse.go`

2. 文本切分仍然是按旧的文本/CQ 模型工作
   - `utils/split_text.go`

3. 日志、敏感词、扩展等仍大量消费 `Message.Message`
   - 而不是专门的 IR 视图

4. 一些 adapter 的入站只是“最小 segment 化”
   - 不是完整的原生结构化解析

## 当前阶段判断

当前状态可以这样定义：

- 是，海豹现在已经有了内部 UniMessage / Message IR
- 是，入口已经基本统一到 segment-first 语义
- 是，第一批 adapter 已经开始围绕它工作
- 否，整个项目还没有完全进入“只认 UniMessage”的最终状态

换句话说：

- 这不是还在概念设计阶段
- 也不是已经彻底完成
- 而是已经进入“IR 已建立、入口已统一、部分 adapter 已切换、业务消费层仍需继续收口”的中间阶段

## 下一步建议

下一步最值得继续推进的工作是：

1. 给命令层补 `CommandTextView`
   - 逐步替代 `SegmentsToLegacyCQText()`

2. 让消息切分变成 segment-aware
   - 不再只依赖 CQ 文本保护式切分

3. 继续安全地推进其他 adapter
   - 例如 `official_qq`
   - 以及不涉及外部协议对接风险的平台

4. 逐步让业务消费方从 `Message.Message` 转向 IR 视图
   - 命令
   - 日志
   - 敏感词
   - 扩展
