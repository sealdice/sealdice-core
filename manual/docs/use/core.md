---
lang: zh-cn
title: 核心指令
---

# 核心指令

::: info 本节内容

本节将介绍海豹核心的核心指令。

核心指令是无法被关闭的指令。与之相对的概念是扩展指令，扩展指令可以被关闭。

:::

## `.master` 骰主指令

此指令通常只能由具有 Master 权限的用户使用。

`.master add me` 为指令发送者添加 Master 权限（仅没有添加过 Master 时可用）。

`.master add @A` 为 A 添加 Master 权限。

`.master del @A` 移除 A 的 Master 权限。

`.master list` 查看当前 Master 权限列表。

`.master relogin` 30 秒后重新登录，有可能使骰子失联。

`.master reboot` 重新启动海豹核心（需要二次确认）。

`.master checkupdate` 检查并进行远程更新（需要二次确认）。

`.master unlock <解锁码>` 清空 Master 权限列表，并给自己重新添加 Master 权限。

`.master backup` 做一次备份。

`.master reload (deck|js|helpdoc)` 重新加载牌堆/js/帮助文档。

`.master quitgroup <群组 ID> [<理由>]`  从指定群组中退出，必须在同一平台使用。

::: warning 注意：保护好你的骰子

我们认为，拥有海豹核心的最终判定方式是可以接触到 WebUI。因此，该解锁码仅能通过 WebUI 的「综合设置 - 基本设置」获取。

你可以将 Master 权限授予若干位维护人员。但是，确保只有你完全信任的人能够接触到你骰子的 WebUI 与本地文件。

如果你的 WebUI 可以通过公开互联网访问，确保你设置了合适的密码。

:::

## `.ban` 黑白名单指令

::: warning 注意：统一 ID

在海豹核心的指令和后台设置中，你会经常用到**统一 ID**。这是海豹核心用于标识不同平台的用户和群组的通用格式。对于每个用户，其形式为 `平台:序列号`；对于群组，其形式为 `平台-Group:序列号`。

使用 `.userid` 指令查看自己和所在群聊的统一 ID。

在 QQ 平台，序列号即为 QQ 号与群号。

:::

此指令只能由具有 Master 权限的用户使用。

`.ban help` 查看帮助。

`.ban add <统一 ID> [<原因>]` 使用统一 ID，将指定用户拉入黑名单，理由可不填。

`.ban trust <统一 ID>` 使用统一 ID，将指定用户添加至信任列表。

`.ban rm <统一 ID>` 使用统一 ID，将指定用户移出黑名单/信任列表。

`.ban list` 展示列表。

`.ban list (ban|warn|trust)` 只显示被禁用/被警告/信任用户。

`.ban query <统一 ID>` 查看指定用户黑白名单情况。

## `.bot` 骰子管理

如果你启用了 `忽略 .bot 裸指令`，**你必须 AT 骰子账号，才能使用 bot 命令**。为了简单，在以下示例中略去 AT 的部分。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.bot on', send: true},
{content: '<海豹bot> 已启用 SealDice <版本号>'},
{content: '.bot off', send: true},
{content: '<海豹bot> 停止服务'},
{content: '.bot bye', send: true},
{content: '收到指令，5s后将退出当前群组'},
{content: '.bot', send: true},
{content: 'SealDice <版本号>'},
]" />
<!-- autocorrect-enable -->

:::

## `.ext` 扩展管理

除了本节所述的「核心指令」之外，海豹的其他功能都作为「扩展」提供。每一个扩展提供若干指令和其他功能，并可以单独开关。你可在每个群聊中启用不同的扩展。

`.ext list` 查看扩展列表和开启情况。

`.ext <扩展名>` 查询指定扩展的信息。

`.ext <扩展名> (on|off)` 开启、关闭指定扩展。

目前，海豹提供 7 个内置扩展，它们的详细信息在本章的后续内容中逐一介绍。同时，海豹核心也支持通过装载 JavaScript 脚本添加第三方扩展。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.ext', send: true},
{content: '检测到以下扩展：\n1. [开]coc7 - 版本:1.0.0 作者:木落\n2. [开]log - 版本:1.0.0 作者:木落\n3. [开]fun - 版本:1.0.0 作者:木落\n4. [开]deck - 版本:1.0.0 作者:木落\n5. [关]reply - 版本:1.0.0 作者:木落\n6. [开]dnd5e - 版本:1.0.0 作者:木落\n7. [开]story - 版本:1.0.0 作者:木落\n使用命令: .ext <扩展名> on/off 可以在当前群开启或关闭某扩展。\n命令: .ext <扩展名> 可以查看扩展介绍及帮助'},
{content: '.ext coc7 on', send: true},
{content: '打开扩展 coc7'},
{content: '.ext reply', send: true},
{content: '> [reply] 版本1.2.0 作者木落\n> 自定义回复模块，支持各种文本匹配和简易脚本'},
{content: '.ext reply off', send: true},
{content: '关闭扩展 reply'},
]" />
<!-- autocorrect-enable -->

:::

可以在 UI 界面中「综合设置 - 基本设置」的最底下，设置各个扩展及其指令的默认开启状态。

## `.set` 骰子面数管理

海豹默认骰子面数为100，你可以通过以下指令在 coc/dnd 规则中切换，更多规则请看[其它规则支持](./other_rules.md)。

`.set info`  查看当前面数设置。

`.set <面数>`  设置群内骰子面数。

`.set dnd`  设置群内骰子面数为20，并自动开启对应扩展。

`.set (coc|coc7)`  设置群内骰子面数为100，并自动开启对应扩展。

`.set clr`  清除群内骰子面数设置。

## `.r` 骰点

`.r <表达式> [<原因>]`

别名：`.roll`

### 常用示例

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r', send: true},
{content: '<木落>掷出了 D100=69'},
{content: '.r d50', send: true},
{content: '<木落>掷出了 d50=[1d50=48]=48'},
{content: '.r d50 天气不错', send: true},
{content: '由于天气不错，<木落>掷出了 d50=[1d50=4]=4'},
{content: '.r 5d24 骰5个24面骰', send: true},
{content: '由于骰5个24面骰，<木落>掷出了 5d24=[5d24=60, 7+20+15+1+17]=60'},
{content: '.r 4d6k3 骰4个6面骰，选3个最大的', send: true},
{content: '由于骰4个6面骰，选3个最大的，<木落>掷出了 4d6k3=[{6 5 3 | 1 }]=14'},
{content: '.r 100 + 3 * 2', send: true},
{content: '<木落>掷出了 100 + 3 * 2=100 + 6=106'},
]" />
<!-- autocorrect-enable -->

:::

或许你已注意到，`.r` 指令的表达式在不包含骰子算符时，相当于计算器。海豹的计算只支持整数，出现的小数被立即舍弃。

### 多轮骰点

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 2#d10', send: true},
{content: '<木落>掷骰2次:\nd10=[1d10=7]=7\nd10=[1d10=8]=8'},
]" />
<!-- autocorrect-enable -->

:::

### 在骰点中使用属性值

你可在表达式中包含属性值或其他变量。

::: info 示例

此时木落的侦查技能点是 53

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 侦查+10', send: true},
{content: '<木落>掷出了 侦查+10=53[侦查=53] + 10=63'},
]" />
<!-- autocorrect-enable -->

:::

### 奖励骰与惩罚骰

CoC 规则中，对于百分骰的一种补偿骰法，通过额外骰一定数量的十位骰，选择组成的最好结果或最坏结果。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r b', send: true},
{content: '<木落>掷出了 b=[D100=72, 奖励 4]=42'},
{content: '.r b3', send: true},
{content: '<木落>掷出了 b3=[D100=96, 奖励 4 6 3]=36'},
{content: '.r p4 惩罚骰', send: true},
{content: '由于惩罚骰，<木落>掷出了 p4=[D100=27, 惩罚 5 6 8 7]=87'},
]" />
<!-- autocorrect-enable -->

:::

### 优势骰与劣势骰

D&D 规则中对 20 面骰的一种补偿骰法。额外骰一次，取较高或较低结果。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.set 20', send: true},
{content: '设定默认骰子面数为 20'},
{content: '.r d20优势', send: true},
{content: '<木落>掷出了 d20优势=[{7 | 6 }]=7'},
{content: '.r d劣势', send: true},
{content: '<木落>掷出了 d劣势=[{16 | 18 }]=16'},
]" />
<!-- autocorrect-enable -->

:::

优势骰与劣势骰也可使用通用的表达式表达

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 2d20k1 等于优势骰', send: true},
{content: '由于等于优势骰，<木落>掷出了 2d20k1=[{19 | 17 }]=19'},
{content: '.r 2d20q1 等于劣势骰', send: true},
{content: '由于等于劣势骰，<木落>掷出了 2d20k1=[{19 | 17 }]=17'},
]" />
<!-- autocorrect-enable -->

:::

### fvtt 骰点兼容

:::: info 示例

::: tabs

== 优势骰

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r d20kh', send: true},
{content: '<木落>掷出了 d20kh=[{10 | 3 }]=10'},
]" />
<!-- autocorrect-enable -->

== 劣势骰

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r d20kl', send: true},
{content: '<木落>掷出了 d20kl=[{6 | 15 }]=6'},
]" />
<!-- autocorrect-enable -->

== 排除低值

骰 4 个排除 1 个最低值：

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 4d6dl1', send: true},
{content: '<木落>掷出了 4d6dl1=[{5 3 2 | 1 }]=10'},
]" />
<!-- autocorrect-enable -->

== 排除高值

骰 4 个排除 1 个最高值：

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 4d6dh1', send: true},
{content: '<木落>掷出了 4d6dh1=[{3 3 5 | 6 }]=11'},
]" />
<!-- autocorrect-enable -->

:::

::::

### fate 命运骰

一种特殊的六面骰，六个面分别为 -、-、0、0、+、+，分别代表 -1、0、1。

骰点时投掷 4 次，加在一起为结果。

:::: info 示例

::: tabs

== 一般使用

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r f', send: true},
{content: '<木落>掷出了 f=[---+]=-2'},
]" />
<!-- autocorrect-enable -->

== 带补正的情况

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r f+1', send: true},
{content: '<木落>掷出了 f+1=0[+0-0] + 1=1'},
]" />
<!-- autocorrect-enable -->

:::

::::

### WOD 骰点

WOD 骰点规则是一个多轮骰点规则，国内多见于无限团。

进行 WOD 骰点需要设定以下参数：**骰池数量 X、加骰线 Y、骰子面数 Z、成功线 N**，其中 X Y 是必须的，成功线默认为 8，骰子面数默认为 10。

骰 X 个 Z 面骰，每有一个大于等于成功线 N 的骰，成功数加 1，每有一个大于等于加骰线 Y 的骰，加骰数加 1，进入下一轮。

在第二轮中，骰上一轮中**加骰数**个 Z 面骰，重复进行计算。以此类推。

最后计算总计成功数。

表达式形如 `<X>a<Y>[m<Z>][k<N>]`。其中的大写字母用相应参数替换。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 5a6', send: true},
{content: '<木落>掷出了 5a6=[成功2/8 轮数:3 {4,<10*>,<10*>,5,1},{5,<6>},{1}]=2'},
{content: '.r 10a6k4m9', send: true},
{content: '<木落>掷出了 10a6k4m9=[成功11/16 轮数:3 {1,<6*>,5*,3,<8*>,5*,<8*>,<6*>,2,<8*>},{5*,2,<9*>,1,4*},{5*}]=11'},
]" />
<!-- autocorrect-enable -->

:::

在计算过程中，每一轮骰点被包含在一对花括号 `{}` 中；达到加骰线 Y 的骰点用 `<>` 标记；达到成功线 N 的骰点用 `*` 标记。

你可指定 Y = 0，这时不进行加骰而只骰一轮。

你可将 `k<N>` 替换成 `q<M>`，这时，最终计算的是**小于等于 M**的骰子总数。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 5a6q4', send: true},
{content: '<木落>掷出了 5a6q4=[成功4/9 轮数:3 {<9>,5,<9>,3*,<9>},{<10>,3*,2*},{2*}]=4'},
]" />
<!-- autocorrect-enable -->

:::

### 双十字骰点

双十字规则是一种多轮的骰点规则。

进行双十字骰点需要设定以下参数：**骰池数量 X、暴击线 Y、骰子面数 Z**，其中 X Y 是必须的，骰子面数 Z 默认为 10。

骰 X 个 Z 面骰，出目大于等于暴击线的骰子称为此骰子「暴击」。只要存在暴击的骰子，就称本轮暴击，进入下一轮；否则计算最终骰点。

第二轮中，骰 上一轮中暴击的骰子数 个 Z 面骰，统计暴击数，判断进入下一轮或结束。以此类推。

最终的骰点结果为：暴击轮数 * 10 + 最后一轮中最大点数。

表达式形如 `<X>c<Y>[m<Z>]`。其中的大写字母用相应参数替换。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 4c3m7', send: true},
{content: '<木落>掷出了 4c3m7=[出目32/9 轮数:4 {<4>,2,<4>,<5>},{<7>,1,2},{<7>},{2}]=32'},
]" />
<!-- autocorrect-enable -->

:::

在计算过程中，每一轮骰点被包含在一对花括号 `{}` 中；达到暴击线 Y 的骰点用 `<>` 标记。

### 混合运算

以上所有骰法，加、减、乘、除、乘方等 5 个数学运算，以及括号 `()` 可以被组合使用，以进行更复杂的运算。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 30 + (-1d20) + 49', send: true},
{content: '<木落>掷出了 30 + (-1d20) + 49=30 + -1[1d20=1] + 49=78'},
{content: '.r d50 * 3 + 2', send: true},
{content: '<木落>掷出了 d50 * 3 + 2=21[1d50=21] * 3 + 2=65'},
{content: '.r d50 * 3 + (2 - p2) 多项式', send: true},
{content: '由于多项式，<木落>掷出了 d50 * 3 + (2 - p2)=25[1d50=25] * 3 + -64[D100=6, 惩罚 6 5]=11'},
]" />
<!-- autocorrect-enable -->

:::

特别地，上文所述的「骰法」`d` `b` `p` `f` `a` `c` 均可作为运算符使用。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.r 1d1d1d1d1d1d1d1d1d1d1d1d1d1d(20+1d3*4)', send: true},
{content: '<木落>掷出了 1d1d1d1d1d1d1d1d1d1d1d1d1d1d(20+1d3*4)=13'},
{content: '.r 1d10+(1+32)d(4*6)d20', send: true},
{content: '<木落>掷出了 1d10+(1+32)d(4*6)d20=1[1d10=1] + 3941[33d24=384,384d20=3941]=3942'},
]" />
<!-- autocorrect-enable -->

:::

## `.rh` 暗骰

这个指令的格式与普通骰点 `.r` 完全相同，区别在于发送骰点结果的方式。

在发送指令的群聊中，海豹核心会进行提示，但提示不包含骰点结果。

骰点结果将由海豹核心私聊给指令发送者。

::: info 收不到结果？

在 QQ 平台上，如果你不是海豹账号的好友，将无法进行私聊。也就无法收到暗骰结果。

:::

:::: info 示例

::: tabs

== 群聊

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.rh d50', send: true},
{content: '命运正在低语！'},
]" />
<!-- autocorrect-enable -->

== 收到的私聊

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '来自群<群名>(群号)的暗骰:\n<木落>掷出了 d10=[1d10=3]=3'},
]" />
<!-- autocorrect-enable -->

:::

::::

## `.rx` / `.rxh` 特殊骰点

这个指令的格式与普通骰点 `.r` 完全相同，区别在于允许额外 AT 其他人，以使用对方的属性。

这种操作称为「代骰」，你会在许多其他指令中看到代骰用法。

::: info 示例

此时木落的侦查是 75，Szz 的侦查是 80

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.rx 侦查+1d20**2', send: true},
{content: '<木落>掷出了 侦查+1d20**2=75[侦查=75] + 324[1d20=18]=399'},
{content: '.rx 侦查+1d20**2 @Szz', send: true},
{content: '由<木落>代骰:\n<Szz>掷出了 侦查+1d20**2=80[侦查=80] + 144[1d20=12]=224'},
]" />
<!-- autocorrect-enable -->

:::

## `.nn` 角色名设定

`.nn` 查看当前角色名。

`.nn <角色名>` 修改角色名，角色名中不能带有空格。

`.nn clr` 重置角色名，即，将角色名设置为 IM 平台的昵称。

角色名被用于在进行各种操作和记录 Log 时显示。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.nn 简·拉基·茨德', send: true},
{content: '<木落>(IM 账号)的昵称被设定为<简·拉基·茨德>'},
{content: '.r', send: true},
{content: '<简·拉基·茨德>掷出了 D100=16'},
{content: '.nn', send: true},
{content: '玩家的当前昵称为: <简·拉基·茨德>'},
{content: '.nn clr', send: true},
{content: '<简·拉基·茨德>(IM 账号)的昵称已重置为<木落>'},
]" />
<!-- autocorrect-enable -->

:::

## `.pc` 角色卡管理

::: info

为了使用这个指令，需要先介绍海豹核心管理你角色卡的方式。

在每个群聊（对于这里，私聊也被认为是一个特殊的群聊）中，你都自动拥有一张独立的角色卡。这些角色卡互相无任何影响。

你还可以拥有若干与特定群聊无关的角色卡，这些角色卡可以被绑定到任意数量的群聊。这时，绑定的群聊中角色数据**互相同步**，在一处的修改就会影响其他各处。

`pc` 指令的作用是管理上述第二种群聊无关角色卡。

:::

`.pc new <角色名>` 新建一张角色卡，并绑定到当前群聊。

`.pc save [<角色名>]` 将你当前群聊中的独立卡数据保存为个人角色卡。你可指定保存的角色名，如不指定，将使用当前群聊中你的角色名。

`.pc update` 从 <Badge type="tip" text="NextVersion"/> 起，这个子指令将你当前群聊中的独立卡数据更新到个人角色卡。两个角色卡的角色名必须一致。

`.pc list` 列出你所保存的所有角色。

`.pc tag (<角色名>|<角色序号>)` 将指定角色卡绑定到当前群聊。

`.pc tag` 不带有角色名参数，将本群的绑定关系解除。你在本群的角色将会恢复为独立卡的数据。

`.pc untagAll (<角色名>|<角色序号>)` 将指定角色卡从其绑定的所有群解绑。

`.pc load (<角色名>|<角色序号>)` 使用指定角色卡的数据覆盖当前群聊的独立卡。这不会将角色卡绑定到当前群聊。

`.pc del/rm (<角色名>|<角色序号>)` 删除指定角色卡。

### 使用现存角色卡：绑定与不绑定

从 <Badge type="tip" text="NextVersion"/> 起，由于 `.pc update` 指令的加入，海豹支持了两种使用现存角色卡的模式。

一种是绑定的模式，好处是用完不用更新，坏处是实时修改无法撤销。操作方式是：

::: info 示例

假设你已经有了一张名为「木落落」的个人角色卡。

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.pc tag 木落落', send: true},
{content: '切换角色<木落落>，绑定成功'},
{content: '.st hp=0', send: true},
{content: '<木落落>的coc7属性录入完成，本次录入了1条数据'},
]" />
<!-- autocorrect-enable -->

这时，用户的「木落落」角色卡数据已自动更新。

:::

另一种是不绑定的模式，好处是可以控制更新数据的时机，坏处是需要一步额外操作。操作方式是：

::: info 示例

假设你已经有了一张名为「木落落」的个人角色卡。

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.pc load 木落落', send: true},
{content: '角色<木落落>加载成功，欢迎回来'},
{content: '.st hp=0', send: true},
{content: '<木落落>的coc7属性录入完成，本次录入了1条数据'},
{content: '.pc update', send: true},
{content: '角色<木落落>储存成功'},
]" />
<!-- autocorrect-enable -->

在游戏进程中修改角色卡时，这些修改都是针对群内角色卡的。只在执行 `.pc update` 指令时才更新到个人角色卡。

:::

## `.find` 词条查询指令

海豹核心默认搭载了 CoC 的《怪物之锤》、《魔法大典》和 D&D 规则的一系列资料。这些资料在海豹的 `data/helpdpc` 目录下的不同文件中被整理成词条，并支持使用此指令进行查询。

`.find [#<分组名>] <关键字> [--num=<页大小>] [--page=<页码>]` 用关键字进行词条查询；如果提供了分组名，则只在指定分组中进行查询；可指定每页大小和页码，默认为每页 4 条，显示第 1 页。

「分组名」是指海豹 `data/helpdoc` 目录下的子目录名。对于内置的帮助文档，分组名分别是「COC」和「DND」。

`.find <数字ID>` 显示该 ID 的词条。

`.find --rand` 显示随机词条。

别名：`.查询`

查询功能在不同设备上的实现是不同的。在 x86 设备上，海豹核心使用稍微更多的内存使用全文搜索，这赋予了查询指令强大的获取能力。在其他平台上，由于搜索库的限制，海豹核心使用词条标题模糊搜索。

从 <Badge type="tip" text="v1.4.2" /> 版本起，你可以在 WebUI 的「扩展功能 - 帮助文档」中设置分组的别名。

### 指定默认查询分组 <Badge type="tip" text="v1.4.2" />

从 <Badge type="tip" text="v1.4.2"/> 版本起，海豹支持在每个群组中分别设置默认的查询分组。

`.find config --group` 查看当前群组的默认查询分组。

`.find config --group=<分组名>` 设置当前群组的默认查询分组。

`.find config --groupclr` 清除当前群组的默认查询分组。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.find config --group=COC', send: true},
{content: '指定默认搜索分组为COC'},
{content: '.find 火球术', send: true},
{content: '未找到搜索结果'},
{content: '.find config --groupclr', send: true},
{content: '已清空默认搜索分组，原分组为COC'},
{content: '.find 火球术', send: true},
{content: '[全文搜索]最优先结果:\n词条: PHB法术:火球术'},
]" />
<!-- autocorrect-enable -->

:::

### 全文搜索

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.find 30尺 法术', send: true},
{content: '全部结果:\n[序号3066]【术士:超魔法:远程法术】 匹配度 0.16\n[序号3015]【游侠:驯兽师:法术共享】 匹配度 0.13\n[序号2396]【法术详述:迷踪步】 匹配度 0.12\n[序号1319]【法术详述:阿迦纳萨喷火术】 匹配度 0.12\n[序号507]【法术详述:智能堡垒/智力堡垒/智慧堡垒/智能壁垒/心智堡垒/心智壁垒】 匹配度 0.12\n[序号2514]【法术详述:水下呼吸/水中呼吸】 匹配度 0.11\n[序号2212]【法术详述:原力法阵】 匹配度 0.11\n[序号1403]【法术详述:众星冠冕/星辰冠冕/星之冠冕】 匹配度 0.11\n[序号2243]【法术详述:造水/枯水术/造水术/枯水术】 匹配度 0.11\n[序号2176]【法术详述:秘法眼】 匹配度 0.11\n\n(本次搜索由全文搜索完成)'},
]" />
<!-- autocorrect-enable -->

:::

因为多个文本匹配度相近，因此没有列出最佳匹配条目的正文内容。用这条指令可以查看：

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.find 2212', send: true},
{content: '词条: 法术详述:原力法阵\n原力法阵 Circle of Power\n圣武士\n5环 防护\n施法时间：1动作\n施法距离：自身（30尺半径）\n法术成分：V\n持续时间：专注，至多10分钟\n你身上发出神圣能量并以扭曲散溢的魔法能量构成一个半径30尺的球状力场。法术持续时间内力场将以你为中心随你移动。力场范围内的友方生物（包括你自己）为对抗法术或其他魔法效应而进行的豁免检定具有优势。此外，受本法术效应影响的生物在对抗豁免成功则伤害减半的法术或魔法效应时，若成功则不受伤害。'},
]" />
<!-- autocorrect-enable -->

:::

**这么好用，那代价是什么呢？**

更多的内存占用和变慢的启动速度。

大致来说，**每 1 MB 帮助文本会产生约 15 MB 额外内存占用**。

### 快速文档查找

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.find 测试词条', send: true},
{content: '最优先结果:\n词条: 测试:测试词条\n他在命运的沉浮中随波逐流, 扮演着受害与加害者的双重角色\n\n全部结果:\n[序号2]【测试:测试词条】 匹配度 67.00\n\n(本次搜索由快速文档查找完成)'},
]" />
<!-- autocorrect-enable -->

:::

### 致谢

CoC《怪物之锤》的整理者为：**蜜瓜包**、**October**；

CoC 魔法大典的整理者为：**魔骨**、**NULL**、**Dr.Amber**；

D&D 系列资料的整理者主要为 DicePP 项目组成员，包括**Farevell**、**梨子**、**花作噫**、**邪恶**、**惠惠**、**赵小安**等。

这些资料的原始出处和译者很多已经不可考，此处无法一一列出，甚为遗憾。

也在此感谢一代又一代无名作者和译者做出的工作。

## `.help` 帮助指令

`.help [<词条名>]` 显示指定词条的帮助文档。

`.help reload` 重新装载帮助文档。仅 Master 可用。

## `.set` 设定默认骰子面数 / 设定游戏系统

`.set info` 查看当前默认骰子面数。如果从未设置过，将显示为「0」。

`.set dnd` 设置群内骰子面数为 20，并自动开启 D&D 扩展。

`.set (coc|coc7)` 设置群内骰子面数为 100，并自动开启 CoC 扩展。

`.set <面数>` 设定群内默认骰子面数。

`.set <面数> --my` 设定个人专属默认骰子面数。

`.set clr` 清除群内骰子面数设置。

`.set clr --my` 清除个人骰子面数设置。

如果通过「规则模板」机制添加了自设规则，并有相关配置，可以使用 `.set <规则名>` 切换为对应规则的默认骰面数。你可以通过 `.set help` 来查看当前可用的关键字。

::: info 示例

<!-- autocorrect-disable -->
<ChatBox :messages="[
{content: '.set 20', send: true},
{content: '设定默认骰子面数为 20'},
{content: '.set coc', send: true},
{content: '设定群组默认骰子面数为 100\n提示:已切换至100面骰，并自动开启coc7扩展'},
{content: '.set dnd', send: true},
{content: '设定群组默认骰子面数为 20\n提示:已切换至20面骰，并自动开启dnd5e扩展。'},
{content: '.set info', send: true},
{content: '个人骰子面数: 0\n群组骰子面数: 20\n当前骰子面数: 20'},
]" />
<!-- autocorrect-enable -->

:::

## `.botlist` 机器人列表

这个指令用于标记同一群聊内的其他机器人。

当一个账号被标记后，对于与 TA 相关的消息，海豹核心会按以下规则进行忽略：

1. 如果 TA 被 AT，忽略；
2. 如果是 TA 发出的消息，忽略。

这可避免机器人互相响应造成的危险的循环。

`.botlist add @A @B @C` 标记 A、B、C 为机器人。

`.botlist add @A @B --s` 同上，不过骰子不会做出回复。

`.botlist del @A @B @C` 去除 A、B、C 的标记。

`.botlist list` 查看当前标记列表。
