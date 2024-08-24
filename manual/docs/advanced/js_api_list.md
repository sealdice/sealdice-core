---
lang: zh-cn
title: API 列表
---

# API 列表

::: info 本节内容

本节为海豹提供的 JS API 列表。目前的内容是从上古文档中直接迁移过来的，难免存在错误和缺失，参考本节时请注意识别。

更好的方式是参考海豹提供的 [seal.d.ts](https://raw.githubusercontent.com/sealdice/sealdice-js-ext-template/master/types/seal.d.ts) 文件。（但同样存在缺失）

如果你需要最准确的内容，当前只能查阅海豹源码。主要查看 [dice_jsvm.go](https://github.com/sealdice/sealdice-core/blob/master/dice/dice_jsvm.go)，还有一些 API 以结构体方法的形式存在。

:::

> 其中 ctx 为信息的 MsgContext，msg 为信息的 Message，一般会在定义指令函数时就声明，如：

```javascript
cmd.solve = (ctx, msg, cmdArgs) => {
    someFunction;
}
```

下面是 api 的说明（完全了吧......应该？）：

```javascript
//被注释掉的 api 是可以提供的，但是在源码中被注释。
//seal.setVarInt(ctx, `$XXX`, valueToSet) //`$XXX`即 rollvm（初阶豹语）中的变量，其会将$XXX 的值设定为 int 类型的 valueToSet。
//seal.setVarStr(ctx, `$XXX`, valueToSet) //同上，区别是设定的为 str 类型的 valueToSet。
seal.replyGroup(ctx, msg, something) //向收到指令的群中发送 something。
seal.replyPerson(ctx, msg, something) //顾名思义，类似暗骰，向指令触发者（若为好友）私信 something。
seal.replyToSender(ctx, msg, something) //同上，区别是群内收到就群内发送，私聊收到就私聊发送。
seal.memberBan(ctx, groupID, userID, dur) //将指定群的指定用户封禁指定时间 (似乎只实现了 walleq 协议？)
seal.memberKick(ctx, groupID, userID)  //将指定群的指定用户踢出 (似乎也只实现了 walleq 协议？)
seal.format(ctx, something) //将 something 经过一层 rollvm 转译并返回，注意需要配合 replyToSender 才能发送给触发者！
seal.formatTmpl(ctx, something) //调用自定义文案 something
seal.getCtxProxyFirst(ctx, cmdArgs)  //获取被 at 的第一个人，等价于 getCtxProxyAtPos(ctx, cmdArgs, 0)
seal.vars.intGet(ctx, `$XXX`) //返回一个数组，其为 `[int 类型的触发者的该变量的值，bool]` 当 strGet 一个 int 或 intGet 一个 str 时 bool 为 false，若一切正常则为 true。（之所以会有这么奇怪的说法是因为 rollvm 的「个人变量」机制）。
seal.vars.intSet(ctx, `$XXX`, valueToSet) //`$XXX` 即 rollvm（初阶豹语）中的变量，其会将 $XXX 的值设定为 int 类型的 valueToSet。
seal.vars.strGet(ctx, `$XXX`) //返回一个数组，其为 `[str 类型的触发者的该变量的值，bool]`（之所以会有这么奇怪的说法是因为 rollvm 的「个人变量」机制），当 strGet 一个 int 或 intGet 一个 str 时 bool 为 false，如果一切正常则为 true。
seal.vars.strSet(ctx, `$XXX`, valueToSet) //`$XXX` 即 rollvm（初阶豹语）中的变量，其会将 $XXX 的值设定为 str 类型的 valueToSet。
//seal.vars.varSet(ctx, `$XXX`, valueToSet) //可能是根据数据类型自动推断 int 或 str？
//seal.vars.varGet(ctx, `$XXX`) //同上
seal.ext.newCmdItemInfo() //用来定义新的指令；没有参数，个人觉得可以视其为类（class）。
seal.ext.newCmdExecuteResult(bool) //用于判断指令执行结果，true 为成功，false 为失败。
seal.ext.new(extName, extAuthor, Version) //用于建立一个名为 extName，作者为 extAuthor，版本为 Version 的扩展。注意，extName，extAuthor 和 Version 均为字符串。
seal.ext.find(extName) //用于查找名为 extname 的扩展，若存在则返回 true，否则返回 false。
seal.ext.register(newExt) //将扩展 newExt 注册到系统中。注意 newExt 是 seal.ext.new 的返回值，将 register 视为 seal.ext.new() 是错误的。
seal.coc.newRule() //用来创建自定义 coc 规则，github.com/sealdice/javascript/examples 中已有详细例子，不多赘述。
seal.coc.newRuleCheckResult() //同上，不多赘述。
seal.coc.registerRule(rule) //同上，不多赘述。
seal.deck.draw(ctx, deckname, isShuffle) //他会返回一个抽取牌堆的结果。这里有些复杂：deckname 为需要抽取的牌堆名，而 isShuffle 则是一个布尔值，它决定是否放回抽取；false 为放回，true 为不放回。
seal.deck.reload() //重新加载牌堆。
//下面是 1.2 新增 api
seal.newMessage() //返回一个空白的 Message 对象，结构与收到消息的 msg 相同
seal.createTempCtx(endpoint, msg) // 制作一个 ctx, 需要 msg.MessageType 和 msg.Sender.UserId
seal.applyPlayerGroupCardByTemplate(ctx, tmpl) // 设定当前 ctx 玩家的自动名片格式
seal.gameSystem.newTemplate(string) //从 json 解析新的游戏规则。
seal.gameSystem.newTemplateByYaml(string) //从 yaml 解析新的游戏规则。
seal.getCtxProxyAtPos(ctx, cmdArgs, pos) //获取第 pos 个被 at 的人，pos 从 0 开始计数
atob(base64String) //返回被解码的 base64 编码
btoa(string) //将 string 编码为 base64 并返回

//下面是 1.4.1 新增 api
seal.ext.newConfigItem() //用于创建一个新的配置项，返回一个 ConfigItem 对象
seal.ext.registerConfig(configItem) //用于注册一个配置项，参数为 ConfigItem 对象
seal.ext.getConfig(ext, "key") //用于获取一个配置项的值，参数为扩展对象和配置项的 key
seal.ext.registerStringConfig(ext, "key", "defaultValue") //用于注册一个 string 类型的配置项，参数为扩展对象、配置项的 key 和默认值
seal.ext.registerIntConfig(ext, "key", 123) //用于注册一个 int 类型的配置项，参数为扩展对象、配置项的 key 和默认值
seal.ext.registerFloatConfig(ext, "key", 123.456) //用于注册一个 float 类型的配置项，参数为扩展对象、配置项的 key 和默认值
seal.ext.registerBoolConfig(ext, "key", true) //用于注册一个 bool 类型的配置项，参数为扩展对象、配置项的 key 和默认值
seal.ext.registerTemplateConfig(ext, "key", ["1", "2", "3", "4"]) //用于注册一个 template 类型的配置项，参数为扩展对象、配置项的 key 和默认值
seal.ext.registerOptionConfig(ext, "key", "1", ["1", "2", "3", "4", "5", "6", "7", "8", "9", "10"]) //用于注册一个 option 类型的配置项，参数为扩展对象、配置项的 key、默认值和可选项
seal.ext.getStringConfig(ext, "key") //用于获取一个 string 类型配置项的值，参数为扩展对象和配置项的 key
seal.ext.getIntConfig(ext, "key") //用于获取一个 int 类型配置项的值，参数为扩展对象和配置项的 key
seal.ext.getFloatConfig(ext, "key") //用于获取一个 float 类型配置项的值，参数为扩展对象和配置项的 key
seal.ext.getBoolConfig(ext, "key") //用于获取一个 bool 类型配置项的值，参数为扩展对象和配置项的 key
seal.ext.getTemplateConfig(ext, "key") //用于获取一个 template 类型配置项的值，参数为扩展对象和配置项的 key
seal.ext.getOptionConfig(ext, "key") //用于获取一个 option 类型配置项的值，参数为扩展对象和配置项的 key

//下面是 1.4.4 新增 api
seal.setPlayerGroupCard(ctx, tmpl) //设置当前 ctx 玩家的名片
seal.ban.addBan(ctx, id, place, reason)
seal.ban.addTrust(ctx, id, place, reason)
seal.ban.remove(ctx, id)
seal.ban.getList()
seal.ban.getUser(id)
```
<!-- TODO: 添加 1.4.1 中新增的插件配置项 -->

以下为部分 api 使用示例。

::: tip

声明和注册扩展的代码部分已省略。

:::

## `replyGroup` `replyPerson` `replyToSender`

```javascript
//在私聊触发 replyGroup 不会回复
seal.replyGroup(ctx, msg, 'something'); //触发者会收到"something"的回复
seal.replyPerson(ctx, msg, 'something'); //触发者会收到"something"的私聊回复
seal.replyToSender(ctx, msg, 'something'); //触发者会收到"something"的回复
```

## `memberBan` `memberKick`

> 是否保留待议

```javascript
//注意这些似乎只能在 WQ 协议上实现;
seal.memberBan(ctx, groupID, userID, dur) //将群为 groupID，userid 为 userID 的人封禁 dur（单位未知）
seal.memberKick(ctx, groupID, userID) //将群为 groupID，userid 为 userID 的人踢出那个群
```

## `format` `formatTmpl`

```javascript
//注意 format 不会自动 reply，而是 return，所以请套一层 reply
seal.replyToSender(ctx, msg, seal.format(`{$t玩家}的人品为：{$t人品}`))
//{$t人品} 是一个 rollvm 变量，其值等于 .jrrp 出的数值
//回复：
//群主的人品为：87
seal.replyToSender(ctx, msg, seal.formatTmpl(unknown))
//这里等大佬来了再研究
```

## `getCtxProxyFirst` `getCtxProxyAtPos`

```javascript
cmd.solve = (ctx, msg, cmdArgs) => {
    let ctxFirst = seal.getCtxProxyFirst(ctx, cmdArgs)
    seal.replyToSender(ctx, msg, ctxFirst.player.name)
}
ext.cmdMap['test'] = cmd
//输入：.test @A @B
//返回：A 的名称。这里其实获取的是 A 玩家的 ctx，具体见 ctx 数据结构。
cmd.solve = (ctx, msg, cmdArgs) => {
    let ctx2 = seal.getCtxProxyAtPos(ctx, 2)
    seal.replyToSender(ctx, msg, ctx3.player.name)
}
ext.cmdMap['test'] = cmd
//输入：.test @A @B @C
//返回：C（第三个被@的人，从 0 开始计算）的名称。这里其实获取的是 C 玩家的 ctx，具体见 ctx 数据结构。
```

## `vars`

```javascript
// 要看懂这里你可能需要学习一下初阶豹语
seal.vars.intSet(ctx, `$m今日打卡次数`, 8) //将触发者的该个人变量设置为 8
seal.vars.intGet(ctx, `$m今日打卡次数`) //返回 [8,true]
seal.vars.strSet(ctx, `$g群友经典语录`, `我要 Git Blame 一下看看是谁写的`) //将群内的该群组变量设置为“我要 Git Blame 一下看看是谁写的”
seal.vars.strGet(ctx, `$g群友经典语录`) //返回 ["我要 Git Blame 一下看看是谁写的",true]
```

## `ext`

```javascript
//用于注册扩展和定义指令的 api，已有详细示例，不多赘述
```

## `coc`

```javascript
//用于创建 coc 村规的 api，已有详细示例，不多赘述
```

## `deck`

```javascript
seal.deck.draw(ctx, `煤气灯`, false) //返回 放回抽取牌堆“煤气灯”的结果
seal.deck.draw(ctx, `煤气灯`, true) //返回 不放回抽取牌堆“煤气灯”的结果
seal.deck.reload() //重新加载牌堆
```

## 自定义 TRPG 规则相关

```javascript
//这里实在不知道如何举例了
seal.gameSystem.newTemplate(string) //从 json 解析新的游戏规则。
seal.gameSystem.newTemplateByYaml(string) //从 yaml 解析新的游戏规则。
seal.applyPlayerGroupCardByTemplate(ctx, tmpl) // 设定当前 ctx 玩家的自动名片格式
seal.setPlayerGroupCard(ctx, tmpl) // 立刻设定当前 ctx 玩家的名片格式
```

## 其他

```javascript
seal.newMessage() //返回一个空白的 Message 对象，结构与收到消息的 msg 相同
seal.createTempCtx(endpoint, msg) // 制作一个 ctx, 需要 msg.MessageType 和 msg.Sender.UserId
seal.getEndPoints() //返回骰子（应该？）的 EndPoints
seal.getVersion() //返回一个 map，键值为 version 和 versionCode
```

## `ctx` 的内容

```javascript
//在 github.com/sealdice/javascript/examples_ts/seal.d.ts 中有完整内容
// 成员
ctx.group // 当前群信息 (对象)
ctx.player // 当前玩家数据 (对象)
ctx.endPoint // 接入点数据 (对象)
// 以上三个对象内容暂略
ctx.isCurGroupBotOn // bool
ctx.isPrivate // bool 是否私聊
ctx.privilegeLevel // int 权限等级：40 邀请者、50 管理、60 群主、70 信任、100 master
ctx.delegateText // string 代骰附加文本
// 方法 (太长，懒.)
chBindCur
chBindCurGet
chBindGet
chBindGetList
chExists
chGet
chLoad
chNew
chUnbind
chUnbindCur
chVarsClear
chVarsGet
chVarsNumGet
chVarsUpdateTime
loadGroupVars
loadPlayerGlobalVars
loadPlayerGroupVars,notice
```

## `ctx.group` 的内容

```javascript
// 成员
active
groupId
guildId
groupName
cocRuleIndex
logCurName
logOn
recentDiceSendTime
showGroupWelcome
groupWelcomeMessage
enteredTime
inviteUserId
// 方法
extActive
extClear
extGetActive
extInactive
extInactiveByName
getCharTemplate
isActive
playerGet
```

## `ctx.player` 的内容

```javascript
// 成员
name
userId
lastCommandTime
autoSetNameTemplate
// 方法
getValueNameByAlias
```

## `ctx.endPoint` 的内容

```javascript
// 成员
baseInfo
id
nickname
state
userId
groupNum
cmdExecutedNum
cmdExecutedLastTime
onlineTotalTime
platform
enable
// 方法
adapterSetup
refreshGroupNum
setEnable
unmarshalYAML
```

## `msg` 的内容

```javascript
// 成员
msg.time // int64 发送时间
msg.messageType // string group 群聊 private 私聊
msg.groupId // string 如果是群聊，群号
msg.guildId // string 服务器群组号，会在 discord,kook,dodo 等平台见到
msg.sender // 发送者信息 (对象)
    sender.nickname
    sender.userId
msg.message
msg.rawId // 原始信息 ID, 用于撤回等
msg.platform // 平台
// 方法
// (似乎目前没有？)
```

## `cmdArgs` 的内容

```javascript
// 成员
.command // string
.args // []string
.kwargs // []Kwarg
.at // []AtInfo
.rawArgs // string
.amIBeMentioned // bool (为何要加一个 Be?)
.cleanArgs // string 一种格式化后的参数，也就是中间所有分隔符都用一个空格替代
.specialExecuteTimes // 特殊的执行次数，对应 3# 这种
// 方法
.isArgEqual(n, ss...) // 返回 bool, 检查第 n 个参数是否在 ss 中
.eatPrefixWith(ss...) // 似乎是从 cleanArgs 中去除 ss 中第一个匹配的前缀
.chopPrefixToArgsWith(ss...) // 不懂
.getArgN(n) // -> string
.getKwarg(str) // -> Kwarg 如果有名为 str 的 flag，返回对象，否则返回 null/undefined(不确定)
.getRestArgsFrom(n) // -> string 获取从第 n 个参数之后的所有参数，用空格拼接成一个字符串
```
