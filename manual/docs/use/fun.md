---
lang: zh-cn
title: 功能
---

# 功能

::: info 本节内容

本节将展示海豹的「功能」扩展提供的指令，主要为快捷指令、ping、welcome 等额外指令，同时也包括今日人品、智能鸽子等娱乐相关指令。

此外，小众规则指令暂时也放在本扩展中。请善用侧边栏和搜索，按需阅读文档。

:::

## `.jrrp` 今日人品

`.jrrp`

今日人品是一个每个用户独立、每天刷新的 D100 随机数。你可通过 [自定义文案](../config/custom_text.md) 编写对它的解读。

## `.gugu` 人工智能鸽子

`.gugu` 或 `.咕咕`

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.gugu', send: true},
{content: '🕊️:今天发版本，领导说发不完不让走'},
]" />
<!-- autocorrect-enable -->

:::

`.gugu 来源` 查看鸽子背后的故事。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.gugu 来源', send: true},
{content: '🕊️: 前往了一个以前捕鲸的小岛度假~这里人很亲切！但是吃了这里的鱼肉料理之后有点晕晕的诶...想到前几天<木落>的短信，还是别追究他为什么不在了。\n\t——鹊鹊结合实际经历创作'},
]" />
<!-- autocorrect-enable -->

:::

## `.jsr` 不重复骰点

`.jsr <次数># <面数> [<原因>]` 投掷指定面数的骰子指定次数，使每一次的结果都不相同。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.jsr 5# 100', send: true},
{content: '<木落>掷骰5次:\nD100=6\nD100=71\nD100=53\nD100=31\nD100=2'},
]" />
<!-- autocorrect-enable -->

:::

以这种方式骰出的点数会自动成为一个骰池，供 [drl](#drl-骰池抽取) 指令使用。

## `.drl` 骰池抽取

骰池（Draw Lot）是一组来自相同面数骰子、出目互不相同的骰点，支持每次抽取一个。骰池是每个群组共用的。

`.drl new <个数># <面数>` 创建一个指定数量、指定面数的骰池。

`.drl` / `.drlh` 从当前骰池抽取一个数值，后者将结果私聊发送。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.drl', send: true},
{content: '当前群组无骰池，请使用.drl new创建一个。'},
{content: '.drl new 10# 1000', send: true},
{content: '创建骰池成功，骰子面数1000，可抽取10次。'},
{content: '.drl', send: true},
{content: '<木落>掷出了 D1000=568'},
{content: '.drl # 第 10 次', send: true},
{content: '<木落>掷出了 D1000=539\n骰池已经抽空，现在关闭。'},
]" />
<!-- autocorrect-enable -->

:::

## `.text` 文本模板测试

`.text <文本模板>`

骰子会将模板内容解析后返回，其中含有的表达式和变量都将被求值。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.text 看看手气 {1d16}', send: true},
{content: '看看手气 2'},
]" />
<!-- autocorrect-enable -->

:::

## `.send` 向骰主发送消息 / 回复消息

从 <Badge type="tip" text="v1.4.6"/> 起，海豹将此指令迁移至「功能」扩展。

`.send <消息内容>`

拥有 Master 权限的用户将看到消息内容和发送者的 IM 账号，如果是来自群组，也能看到群号。

:::: info 示例

::: tabs

== 群聊

<!-- autocorrect-disable -->
<ChatBox :messages="[
{username: 'Szz', avatar: 'user2', content: '.send 骰主你好！'},
]" />
<!-- autocorrect-enable -->

== Master 收到的消息

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '一条来自群组<群名>(群号)，作者<Szz>(用户 IM 账号)的留言:\n骰主你好！'},
]" />
<!-- autocorrect-enable -->

:::

::::

`.send to <对方ID> <消息内容>`

Master 可以通过这个指令进行回复。目标 ID 可以是群号，也可以是个人的 IM 账号。将收到的消息中的对应 ID 复制到此处即可。

:::: info 示例

::: tabs

== Master 回复

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '一条来自群组<群名>(群号)，作者<Szz>(用户 IM 账号)的留言:\n骰主你好！'},
{content: '.send to <群号> 我收到了！', send: true},
]" />
<!-- autocorrect-enable -->

== 群聊

<!-- autocorrect-disable -->
<ChatBox :messages="[
{username: 'Szz', avatar: 'user2', content: '.send 骰主你好！'},
{content: '本消息由骰主<木落>通过指令发送:\n我收到了！'},
]" />
<!-- autocorrect-enable -->

:::

::::

## `.welcome` 新人入群欢迎

从 <Badge type="tip" text="v1.4.6"/> 起，海豹将此指令迁移至「功能」扩展。

`.welcome (on|off)` 开启、关闭功能

`.welcome show` 查看当前欢迎语

`.welcome set <欢迎语>` 设定欢迎语

## `.ping` 指令响应测试 <Badge type="tip" text="v1.4.2" />

从 <Badge type="tip" text="v1.4.2"/> 版本起，海豹支持 `.ping` 指令，用于指示海豹回复你一条消息。

从 <Badge type="tip" text="v1.4.6"/> 起，海豹将此指令迁移至「功能」扩展。

`.ping` 海豹回复你一条消息。

::::: info 为什么要有这个指令？

对于绝大多数情况，这个指令似乎都没有实际作用。事实上，这个指令的存在是为了解决 **QQ 官方 Bot 在频道私聊中**的以下问题：

如果你向机器人连续发送 3 条频道私聊消息而没有收到回复，在机器人回复你之前，你将无法继续发送频道私聊消息。
而机器人并不会主动向你发送消息，这就造成了死锁。

此时，你可以在**频道**中向海豹核心发送 `.ping` 指令，海豹核心会在**频道私聊**中回复你，以打破死锁。

:::: tabs

== 频道私聊

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '我发出第一条消息。', send: true},
{content: '我发出第二条消息。', send: true},
{content: '我发出第三条消息，机器人怎么还不理我？', send: true},
]" />
<!-- autocorrect-enable -->

::: tip

如果出现类似场景，可以发现 QQ 会提示你已经禁止再向骰子发送私聊消息，要求在骰子回应后才能再次发送。

但用户已经无法再通过私聊发送正确的指令，触发骰子的回应了。

此时，用户可以去频道公屏发送一个 `.ping` 指令。

:::

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: 'pong！这里是海豹核心！'},
{content: '好耶，我又可以发私信给骰子了！', send: true},
]" />
<!-- autocorrect-enable -->

== 频道公屏

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '@海豹核心 .ping', send: true},
]"/>
<!-- autocorrect-enable -->

::::

:::::

## `.alias` 定义快捷指令 & `.&` 使用快捷指令 <Badge type="tip" text="v1.4.3"/>

从 <Badge type="tip" text="v1.4.3"/> 版本起，海豹支持使用`.alias` 定义快捷指令。同时使用 `.&/.a <快捷指令>` 触发快捷指令。

从 <Badge type="tip" text="v1.4.6"/> 起，海豹将此指令迁移至「功能」扩展。

`.alias <别名> <指令>` 将 `.&<别名>` 定义为指定指令的快捷触发方式。在群聊中默认定义群快捷指令。

`.alias --my <别名> <指令>` 将 `.&<别名>` 定义为个人快捷指令。

`.alias del/rm <别名>` 删除快捷指令。在群聊中默认删除群快捷指令。

`.alias del/rm --my <别名>` 删除个人快捷指令。

`.alias show/list` 显示目前可用的快捷指令。

使用快捷指令的方式如下，支持携带参数：

`.& <别名> [可能的参数]` 或 `.a <别名> [可能的参数]`

海豹支持 **个人快捷指令** 和 **群快捷指令** 两种模式：

- 个人快捷指令：与用户关联，定义后用户可以在私聊、骰子所在的所有群进行使用。
- 群快捷指令：与群关联，定义后该群内所有人都可以使用。

:::: info 快捷指令示例

::: tabs

== 私聊

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.alias 终极答案 .r 11+45-14 计算生命、宇宙以及一切的问题的答案', send: true},
{content: '已成功定义指令「.r 11+45-14 计算生命、宇宙以及一切的问题的答案」的个人快捷方式「终极答案」，触发方式：\n.&终极答案 或\n.a 终极答案'},
{content: '.&终极答案', send: true},
{content: '※ 个人快捷指令 .r 11+45-14 计算生命、宇宙以及一切的问题的答案\n由于计算生命、宇宙以及一切的问题的答案，<木落>掷出了 11+45-14=11+45-14=42'},
]"/>
<!-- autocorrect-enable -->

:::

::::

## 额外 TRPG 规则支持

### `.ww` WOD 规则骰点

`.ww <表达式>` 骰点，表达式用法请参考 [核心指令](./core.md#wod-骰点) 节。

特别地，使用 `ww` 指令时允许省略加骰线 `aY`，这时将使用默认值进行骰点。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.ww 5', send: true},
{content: '<木落>掷出了 5a10=[成功0/5 {6,2,2,7,4}]=0'},
]" />
<!-- autocorrect-enable -->

:::

`.ww set <默认值>` 设定默认值，默认值可以由 加骰线 `aY`、骰子面数 `mZ`、成功线 `kN` 中的部分任意组合。默认值的生效范围是当前群组。

`.ww set clr` 重置默认值为：加骰线 10、面数 10、成功线 8，即 `a10m10k8`。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.ww set a6m10k9', send: true},
{content: '成功线k: 已修改为9\n骰子面数m: 已修改为10\n加骰线a: 已修改为6'},
{content: '.ww 5', send: true},
{content: '<木落>掷出了 5a6=[成功1/11 轮数:4 {<9*>,5,<6>,3,3},{<6>,<6>},{<6>,<7>},{5,4}]=1'},
{content: '.ww set clr', send: true},
{content: '骰池设定已恢复默认'},
]" />
<!-- autocorrect-enable -->

:::

### `.dx` 双重十字规则骰点

`.dx <表达式>` 骰点，表达式用法请参考 [核心指令](./core.md#双十字骰点) 节。

### `.ek` 共鸣性怪异规则骰点

`.ek <技能名>(+<奖励骰>) (<判定线>)`

<!-- TODO 由熟悉规则的朋友结合规则说明用法 -->

### `.ekgen` 共鸣性怪异规则属性生成

`.ekgen [<数量>]` 生成指定数量的共鸣性怪异角色属性。

### `.rsr` 暗影狂奔规则骰点

`.rsr <骰数>` 骰点。

骰指定数量的 D6。每有一个骰子骰出 5 或 6 称为 1 成功度。

如果超过半数的骰子骰出 1，称为这次骰点「失误」；失误的同时成功度为 0，称为「严重失误」。
