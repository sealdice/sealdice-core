---
lang: zh-cn
title: 常见用法示例
---

# 常见用法示例

## 创建和注册扩展

::: tip 提示：扩展机制

扩展机制可以看做是海豹的 mod 管理机制，可以模块化开关海豹的任意一部分，如常用的开启 dnd 扩展，关闭 coc 扩展，关闭自动回复等。

可以查看 [.ext 命令](../use/core.md#ext-扩展管理) 来了解更多。

:::

在海豹中，**所有指令必须归属于某个扩展**，而一个扩展可以包含多条指令。

JS 脚本中创建扩展的方式如下，在创建扩展后，**还需要注册扩展**，才能让扩展起效，不要漏掉哦！

```javascript
// 如何建立一个扩展

// 首先检查是否已经存在
if (!seal.ext.find('test')) {
  // 不存在，那么建立扩展，名为 test，作者“木落”，版本 1.0.0
  const ext = seal.ext.new('test', '木落', '1.0.0');
  // 注册扩展
  seal.ext.register(ext);
}
```

::: warning 注意：JS 脚本和扩展

在实现上，海豹允许你在一个 JS 脚本文件中注册多个扩展，但我们不建议这样做。一般来说，一个 JS 脚本文件只会注册一个扩展。

:::

::: warning 注意：修改内置功能？

出于对公平性的考虑，JS 脚本不能替换内置指令和内置扩展。

:::

<!-- TODO: 添加 1.4.1 新增的 插件配置项 相关说明 -->

## 自定义指令

想要创建一条自定义指令，首先需要创建一个扩展（`seal.ExtInfo`），写好自定义指令的实现逻辑之后，再注册到扩展中。

### 创建指令

接上面的代码，假定目前已经注册好了一个名为 `test` 的扩展，现在要写一个名为 `seal` 的指令：

- 这个命令的功能为，显示「抓到一只海豹」的文案；
  - 如果命令写 `.seal ABC`，那么文案中将海豹命名为「ABC」；
  - 如果命令中没写名字，那么命名为默认值「氪豹」。

第一步，创建新的自定义指令，设置好名字和帮助信息。

```javascript
const cmdSeal = seal.ext.newCmdItemInfo();
cmdSeal.name = 'seal'; // 指令名字，可用中文
cmdSeal.help = '召唤一只海豹，可用 .seal <名字> 命名';
```

::: warning 注意：命令的帮助信息

命令的帮助信息是在使用 `.help <命令>` 时会返回的帮助内容。

帮助信息**不要**以 `.` 开头，防止查看帮助时骰子的回复触发其他骰子。

:::

第二步，编写指令的具体处理代码。

你需要编写指令对象的 `solve` 函数，在使用该指令的时候，海豹核心会调用你写的这个函数。

```javascript
cmdSeal.solve = (ctx, msg, cmdArgs) => {
    // 这里是你需要编写的内容
};
```

| 参数        | 说明                                 |
|-----------|------------------------------------|
| `ctx`     | 主要是和当前环境以及用户相关的内容，如当前发指令用户，当前群组信息等 |
| `msg`     | 原始指令内容，如指令文本，发送平台，发送时间等            |
| `cmdArgs` | 指令信息，会将用户发的信息进行分段，方便快速取用           |

这里仅说明需要用到的接口，详细可见 [插件仓库](https://github.com/sealdice/javascript/tree/main/examples_ts) `examp_ts` 下的 `seal.d.ts` 文件，里面包含了目前开放的接口的定义及其注释说明。

### 指令参数与返回值

假设用户发送过来的消息是 `.seal A B C`，那么可以用 `cmdArgs.getArgN(1)` 获取到 `A`，`cmdArgs.getArgN(2)` 获取到 `B`，`cmdArgs.getArgN(3)` 获取到 `C`。

通常会对参数值进行判断，随后作出响应。

以下代码处理的是 `.seal help` 的情形：

```javascript
cmdSeal.solve = (ctx, msg, cmdArgs) => {
  // 获取第一个参数，例如 .seal A B C
  // 这里 cmdArgs.getArgN(1) 的结果即是 A，传参为 2 的话结果是 B
  let val = cmdArgs.getArgN(1);
  switch (val) {
    case 'help': {
      // 命令为 .seal help
      // 创建一个结果对象，并将 showHelp 标记为 true，这会自动给用户发送帮助
      const ret = seal.ext.newCmdExecuteResult(true);
      ret.showHelp = true;
      return ret;
    }
    default: {
      // 没有传入参数时的代码
      return seal.ext.newCmdExecuteResult(true);
    }
  }
};
```

::: warning 注意：返回执行结果

在执行完自己的代码之后，需要返回指令结果对象，其参数是是否执行成功。

:::

### 指令核心执行流程

给消息发送者回应，需要使用 `seal.replyToSender()` 函数，前两个参数和 `solve()` 函数接收的参数一致，第三个参数是你要发送的文本。

发送的文本中，可以包含 [变量](../advanced/script.md#变量)（例如`{$t玩家}`），也可以包含 [CQ 码](https://docs.go-cqhttp.org/cqcode)，用来实现回复发送者、@发送者、发送图片、发送分享卡片等功能。

在这个例子中，我们需要获取作为海豹名字的参数，获取不到就使用默认值，随后向消息发送者发送回应。

在刚刚的位置填入核心代码，就可以完成了。

```javascript
cmdSeal.solve = (ctx, msg, cmdArgs) => {
  // 获取第一个参数，例如 .seal A B C
  // 这里 cmdArgs.getArgN(1) 的结果即是 A，传参为 2 的话结果是 B
  let val = cmdArgs.getArgN(1);
  switch (val) {
    case 'help': {
      // 命令为 .seal help
      // 创建一个结果对象，并将 showHelp 标记为 true，这会自动给用户发送帮助
      const ret = seal.ext.newCmdExecuteResult(true);
      ret.showHelp = true;
      return ret;
    }
    default: {
      // 命令为 .seal XXXX，取第一个参数为名字
      if (!val) val = '氪豹';
      // 进行回复，如果是群聊发送那么在群里回复，私聊发送则在私聊回复 (听起来是废话文学，但详细区别见暗骰例子)
      seal.replyToSender(ctx, msg, `你抓到一只海豹！取名为${val}\n它的逃跑意愿为${Math.ceil(Math.random() * 100)}`);
      return seal.ext.newCmdExecuteResult(true);
    }
  }
};
```

### 注册指令

第三步，将命令注册到扩展中。

```javascript
ext.cmdMap['seal'] = cmdSeal;
```

如果你想要给这个命令起一个别名，也就是增加一个触发词，可以这样写：

```javascript
ext.cmdMap['seal'] = cmdSeal; // 注册 .seal 指令
ext.cmdMap['海豹'] = cmdSeal; // 注册 .海豹 指令，等效于 .seal
```

完整的代码如下：

```javascript
// ==UserScript==
// @name         示例：编写一条自定义指令
// @author       木落
// @version      1.0.0
// @description  召唤一只海豹，可用.seal <名字> 命名
// @timestamp    1671368035
// 2022-12-18
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

// 编写一条自定义指令
// 先将扩展模块创建出来，如果已创建就直接使用
let ext = seal.ext.find('test');
if (!ext) {
  ext = seal.ext.new('test', '木落', '1.0.0');

  // 创建指令 .seal
  // 这个命令的功能为，显示「抓到一只海豹」的文案；
  // 如果命令写 .seal ABC，那么文案中将海豹命名为「ABC」；
  // 如果命令中没写名字，那么命名为默认值「氪豹」。
  const cmdSeal = seal.ext.newCmdItemInfo();
  cmdSeal.name = 'seal'; // 指令名字，可用中文
  cmdSeal.help = '召唤一只海豹，可用 .seal <名字> 命名';

  // 主函数，指令解析器会将指令信息解析，并储存在几个参数中
  // ctx 主要是和当前环境以及用户相关的内容，如当前发指令用户，当前群组信息等
  // msg 为原生态的指令内容，如指令文本，发送平台，发送时间等
  // cmdArgs 为指令信息，会将用户发的信息进行分段，方便快速取用
  cmdSeal.solve = (ctx, msg, cmdArgs) => {
    // 获取第一个参数，例如 .seal A B C
    // 这里 cmdArgs.getArgN(1) 的结果即是 A，传参为 2 的话结果是 B
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help': {
        // 命令为 .seal help
        // 创建一个结果对象，并将 showHelp 标记为 true，这会自动给用户发送帮助
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      default: {
        // 命令为 .seal XXXX，取第一个参数为名字
        if (!val) val = '氪豹';
        // 进行回复，如果是群聊发送那么在群里回复，私聊发送则在私聊回复 (听起来是废话文学，但详细区别见暗骰例子)
        seal.replyToSender(ctx, msg, `你抓到一只海豹！取名为${val}\n它的逃跑意愿为${Math.ceil(Math.random() * 100)}`);
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  };

  // 将命令注册到扩展中
  ext.cmdMap['seal'] = cmdSeal;

  // 无实际意义，用于消除编译报错
  export {}

  seal.ext.register(ext);
}

```

这就是最基本的模板了。

## 抽取牌堆

抽取牌堆的函数是 `seal.deck.draw(ctx, deckName, shufflePool)`

- `ctx`：`MsgContext` 类型，指令上下文，`solve()` 函数传进来的第一个参数
- `deckName`：牌堆名称，字符串类型，例如 `GRE单词`
- `shufflePool`：是否放回抽取，布尔类型，`true` 为放回抽取，`false` 为不放回抽取

返回值是一个 `map`，包含以下字段：

- `exists`：布尔类型，是否抽取成功
- `result`：字符串类型，抽取结果
- `err`：字符串类型，抽取失败的原因

### 示例代码：抽取牌堆

```javascript
// ==UserScript==
// @name         抽取牌堆示例
// @author       SzzRain
// @version      1.0.0
// @description  用于演示如何抽取牌堆
// @timestamp    1699077659
// @license      MIT
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

// 本脚本用于演示如何抽取牌堆, 共有两种实现方式
if (!seal.ext.find('draw-decks-example')) {
  const ext = seal.ext.new('draw-decks-example', 'SzzRain', '1.0.0');
  // 创建一个命令
  const cmdDrawDecks = seal.ext.newCmdItemInfo();
  cmdDrawDecks.name = 'dr';
  cmdDrawDecks.help = '使用 .dr <牌堆名> 来抽取牌堆';
  cmdDrawDecks.solve = (ctx, msg, cmdArgs) => {
    // 抽取牌堆
    // 参数1：ctx 参数2：牌堆名称 参数3：是否放回抽取
    // 返回值：{exists: true, result: '抽取结果', err: '错误原因'}
    const decks = seal.deck.draw(ctx, cmdArgs.getArgN(1), true);
    // 判断是否抽取成功
    if (decks['exists']) {
      seal.replyToSender(ctx, msg, decks['result']);
      return seal.ext.newCmdExecuteResult(true);
    } else {
      seal.replyToSender(ctx, msg, '抽取牌堆失败，原因：' + decks['err']);
      return seal.ext.newCmdExecuteResult(true);
    }
  };
  // 注册命令
  ext.cmdMap['dr'] = cmdDrawDecks;

  // 创建一个命令
  const cmdDrawDecks2 = seal.ext.newCmdItemInfo();
  cmdDrawDecks2.name = 'dr2';
  cmdDrawDecks2.help = '使用 .dr2 <牌堆名> 来抽取牌堆';
  cmdDrawDecks2.solve = (ctx, msg, cmdArgs) => {
    // 抽取牌堆的另一种写法，使用 format 函数，由于经过了 rollvm 的处理，所以代码的执行效率会更慢
    // 不过这种写法的返回值固定为字符串，省去了判断是否抽取成功的步骤
    // 参数1：ctx 参数2：海豹语表达式，其中 #{DRAW-牌堆名称} 会被替换为抽取结果
    const decks = seal.format(ctx, `#{DRAW-${cmdArgs.getArgN(1)}}`);
    seal.replyToSender(ctx, msg, decks);
  }
  // 注册命令
  ext.cmdMap['dr2'] = cmdDrawDecks2;

  // 注册扩展
  seal.ext.register(ext);
}
```

## 权限识别

海豹中的权限等级，由高到低分别是：**骰主**，**群主**，**管理员**，**邀请者**，**普通用户** 和 **黑名单用户**。
每一个身份都有一个对应的数字，可以通过 `ctx.privilegeLevel` 获取当前用户的权限等级。
每个身份所对应的数字如下表所示：

::: info

**注意：** 部分权限等级仅在群聊中有效。

从 <Badge type="tip" text="v1.4.5"/> 起，在私聊中，除了**骰主**，**白名单用户**和**黑名单用户**，其他用户被视为拥有与群管理员等同的权限，即权限值 50。

在 <Badge type="danger" text="v1.4.4"/> 或更早版本，私聊中除了**骰主**，**白名单用户**和**黑名单用户**，其他用户的权限等级为普通用户（0）。

:::

| 身份  | 权限值 |
|-----|-----|
| 骰主  | 100 |
| 白名单 | 70  |
| 群主  | 60  |
| 管理员 | 50  |
| 邀请者 | 40  |
| 普通用户 | 0   |
| 黑名单用户 | -30 |

::: tip 提示：关于白名单用户

白名单用户即通过骰主手动添加的信任名单用户，可以使用所有需要群管理权限的功能，但不具备 Master 权限。

信任名单可以通过 `.ban trust <统一ID>` 添加，通过 `.ban list trust` 查看。

:::

::: tip 提示：关于黑名单用户

通常情况下你不需要考虑黑名单用户的情况，因为黑名单用户的消息会被过滤掉，不会触发任何指令。

:::

### 示例代码：权限识别

```javascript
// ==UserScript==
// @name         权限识别样例
// @author       SzzRain
// @version      1.0.0
// @description  使用命令 .myperm 查看自己的权限
// @timestamp    1699086084
// @license      MIT
// @homepageURL  https://github.com/Szzrain
// ==/UserScript==
if (!seal.ext.find('myperm')) {
  const ext = seal.ext.new('myperm', 'SzzRain', '1.0.0');
  // 创建一个命令
  const cmdMyperm = seal.ext.newCmdItemInfo();
  cmdMyperm.name = 'myperm';
  cmdMyperm.help = '使用 .myperm 展示我的权限';
  cmdMyperm.solve = (ctx, msg, cmdArgs) => {
    let text = "普通用户";
    console.log(ctx.privilegeLevel);
    switch (ctx.privilegeLevel) {
      case 100:
        text = "master";
        break;
      case 60:
        text = "owner";
        break;
      case 50:
        text = "admin";
        break;
      case 40:
        text = "inviter";
        break;
      case -30:
        // 黑名单用户，但是由于黑名单会被过滤掉，所以实际上这里并不会执行，这里只是为了演示
        return seal.ext.newCmdExecuteResult(false);
    }
    seal.replyToSender(ctx, msg, text);
    return seal.ext.newCmdExecuteResult(true);
  }
  // 注册命令
  ext.cmdMap['myperm'] = cmdMyperm;

  // 注册扩展
  seal.ext.register(ext);
}
```

## 黑名单 / 信任名单操作 <Badge type="tip" text="v1.4.4" />

### 黑名单操作的函数

添加：`seal.ban.addBan(ctx, uid, place, reason)`

移除：`seal.ban.remove(ctx, uid)`

- `ctx`：`MsgContext` 类型，指令上下文，`solve()` 函数传进来的第一个参数
- `uid`：用户 ID，字符串类型，例如 `QQ:123456789`, `TG:123456789`
- `place`：拉黑的地方，字符串类型，随便写，一般来说在群内拉黑就写群 ID
- `reason`：拉黑原因，字符串类型，随便写

### 信任用户名单

添加：`seal.ban.addTrust(ctx, uid, place, reason)` 参数说明同上

移除：`seal.ban.remove(ctx, uid)`

::: tip 提示：相同的移除函数

黑名单和信任名单存储在同一个数据库中，因此移除时使用的是同一个函数。

你在进行移除操作时需要自己判断是否符合你的预期。

:::

### 获取黑名单 / 信任名单列表

使用 `seal.ban.getList()`

返回值为一个数组，数组中的每一项都是一个 `BanListInfoItem` 对象，包含以下字段：

- `id`：用户 ID，字符串类型
- `name`：用户昵称，字符串类型
- `score`：怒气值，整数类型
- `rank`：拉黑/信任等级 0=没事 -10=警告 -30=禁止 30=信任
- `times`：事发时间，数组类型，内部元素为整数时间戳
- `reasons`：拉黑/信任原因，数组类型，内部元素为字符串
- `places`：拉黑/信任的发生地点，数组类型，内部元素为字符串
- `banTime`：拉黑/信任的时间，整数时间戳

### 获取用户在黑名单 / 信任名单中的信息

使用 `seal.ban.getUser(uid)`

如果用户没有在黑名单 / 信任名单中，返回值为空值。

如果有则返回一个 `BanListInfoItem` 对象，字段同上。

### 示例代码：黑名单 / 信任名单操作

```javascript
// ==UserScript==
// @name         js-ban
// @author       SzzRain
// @version      1.0.0
// @description  演示 js 扩展操作黑名单
// @timestamp    1706684850
// @license      MIT
// @homepageURL  https://github.com/Szzrain
// ==/UserScript==

if (!seal.ext.find('js-ban')) {
  const ext = seal.ext.new('js-ban', 'SzzRain', '1.0.0');
  // 创建一个命令
  const cmdcban = seal.ext.newCmdItemInfo();
  cmdcban.name = 'cban';
  cmdcban.help = '使用.cban <用户id> 来拉黑目标用户，仅master可用';
  cmdcban.solve = (ctx, msg, cmdArgs) => {
      let val = cmdArgs.getArgN(1);
      switch (val) {
          case 'help': {
              const ret = seal.ext.newCmdExecuteResult(true);
              ret.showHelp = true;
              return ret;
          }
          default: {
              if (ctx.privilegeLevel === 100) {
                  seal.ban.addBan(ctx, val, "JS扩展拉黑", "JS扩展拉黑测试");
                  seal.replyToSender(ctx, msg, "已拉黑用户" + val);
              } else {
                  seal.replyToSender(ctx, msg, "你没有权限执行此命令");
              }
              return seal.ext.newCmdExecuteResult(true);
          }
      }
  }
  // 注册命令
  ext.cmdMap['cban'] = cmdcban;

  // 创建一个命令
  const cmdcunban = seal.ext.newCmdItemInfo();
  cmdcunban.name = 'cunban';
  cmdcunban.help = '使用.cunban <用户id> 来解除拉黑/移除信任目标用户，仅master可用';
  cmdcunban.solve = (ctx, msg, cmdArgs) => {
      let val = cmdArgs.getArgN(1);
      switch (val) {
          case 'help': {
              const ret = seal.ext.newCmdExecuteResult(true);
              ret.showHelp = true;
              return ret;
          }
          default: {
              if (ctx.privilegeLevel === 100) {
                  // 信任用户和拉黑用户存在同一个列表中，remove 前请先判断是否符合预期
                  seal.ban.remove(ctx, val);
                  seal.replyToSender(ctx, msg, "已解除拉黑/信任用户" + val);
              } else {
                  seal.replyToSender(ctx, msg, "你没有权限执行此命令");
              }
              return seal.ext.newCmdExecuteResult(true);
          }
      }
  }
  // 注册命令
  ext.cmdMap['cunban'] = cmdcunban;

  // 创建一个命令
  const cmdctrust = seal.ext.newCmdItemInfo();
  cmdctrust.name = 'ctrust';
  cmdctrust.help = '使用.ctrust <用户id> 来信任目标用户，仅master可用';
  cmdctrust.solve = (ctx, msg, cmdArgs) => {
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help': {
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      default: {
        if (ctx.privilegeLevel === 100) {
          seal.ban.addTrust(ctx, val, "JS扩展信任", "JS扩展信任测试");
          seal.replyToSender(ctx, msg, "已信任用户" + val);
        } else {
          seal.replyToSender(ctx, msg, "你没有权限执行此命令");
        }
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  }
  // 注册命令
  ext.cmdMap['ctrust'] = cmdctrust;

  // 创建一个命令
  const cmdcbanlist = seal.ext.newCmdItemInfo();
  cmdcbanlist.name = 'cbanlist';
  cmdcbanlist.help = '使用.cbanlist 来查看黑名单和信任列表，仅master可用';
  cmdcbanlist.solve = (ctx, msg, cmdArgs) => {
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help': {
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      default: {
        if (ctx.privilegeLevel === 100) {
          let text = "黑名单/信任列表：\n";
          seal.ban.getList().forEach((v) => {
            text += `${v.name}(${v.id}) 当前等级:${v.rank} 怒气值:${v.score}\n`;
          });
          seal.replyToSender(ctx, msg, text);
        } else {
          seal.replyToSender(ctx, msg, "你没有权限执行此命令");
        }
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  }
  // 注册命令
  ext.cmdMap['cbanlist'] = cmdcbanlist;

  // 创建一个命令
  const cmdcget = seal.ext.newCmdItemInfo();
  cmdcget.name = 'cget';
  cmdcget.help = '使用.cget <用户id> 来查看目标用户的黑名单/信任信息，仅master可用';
  cmdcget.solve = (ctx, msg, cmdArgs) => {
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help': {
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      default: {
        if (ctx.privilegeLevel === 100) {
          let info = seal.ban.getUser(val);
          if (!info) {
            seal.replyToSender(ctx, msg, "用户不存在或未被拉黑/信任");
            return seal.ext.newCmdExecuteResult(true);
          }
          let level = info.rank;
          // 不知道为什么，用 === 是 false
          if (info.rank == 30) {
            level = "信任"
          } else if (info.rank == -30) {
            level = "拉黑"
          } else if (info.rank == -10) {
            level = "警告"
          }
          let text = `用户${info.name}(${info.id}) 当前等级:${level} 怒气值:${info.score}`;
          seal.replyToSender(ctx, msg, text);
        } else {
          seal.replyToSender(ctx, msg, "你没有权限执行此命令");
        }
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  }
  // 注册命令
  ext.cmdMap['cget'] = cmdcget;

  // 注册扩展
  seal.ext.register(ext);
}
```

## 存取数据

相关的 API 是两个函数，`ExtInfo.storageSet(key, value)` 函数和 `ExtInfo.storageGet(key)`，一个存，一个取。

**关于 key：**

存储时需要指定 key，你可以设定为你的扩展的名字，也可以设定为其他的，注意不要和别的扩展的 key 重名就可以了。

就好比你在商场门口想要把随身物品存进暂存柜中，需要先找到个和别人不重复的柜子，避免放错地方或者取错东西。

**关于 value：**

存放的数据是字符串类型，且只能存一个，但如果想要存放更多的数据以及非字符串类型的数据怎么办？

答案是使用 `JSON.stringify()` 函数将存储了数据的 JS 对象转换为 [JSON](https://www.runoob.com/json/json-tutorial.html) 字符串，存储起来，需要取出的时候，再使用 `JSON.parse()` 函数将数据再转换为 JS 对象。。

### 示例代码：投喂插件

```javascript
// ==UserScript==
// @name         示例：存储数据
// @author       木落
// @version      1.0.0
// @description  投喂，格式 .投喂 <物品>
// @timestamp    1672423909
// 2022-12-31
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

// 先将扩展模块创建出来，如果已创建就直接使用
if (!seal.ext.find('test')) {
  const ext = seal.ext.new('test', '木落', '1.0.0');

  const cmdFeed = seal.ext.newCmdItemInfo();
  cmdFeed.name = '投喂';
  cmdFeed.help = '投喂，格式：.投喂 <物品>\n.投喂 记录 // 查看记录';
  cmdFeed.solve = (ctx, msg, cmdArgs) => {
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help':
      case '': {
        // .投喂 help
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      case '列表':
      case '记录':
      case 'list': {
        const data = JSON.parse(ext.storageGet('feedInfo') || '{}');
        const lst = [];
        for (let [k, v] of Object.entries(data)) {
          lst.push(`- ${k}: 数量 ${v}`);
        }
        seal.replyToSender(ctx, msg, `投喂记录:\n${lst.join('\n')}`);
        return seal.ext.newCmdExecuteResult(true);
      }
      default: {
        const data = JSON.parse(ext.storageGet('feedInfo') || '{}');
        const name = val || '空气';
        if (data[name] === undefined) {
          data[name] = 0;
        }
        data[name] += 1;
        ext.storageSet('feedInfo', JSON.stringify(data));
        seal.replyToSender(ctx, msg, `你给海豹投喂了${name}，要爱护动物！`);
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  };

  // 将命令注册到扩展中
  ext.cmdMap['投喂'] = cmdFeed;
  ext.cmdMap['feed'] = cmdFeed;

  seal.ext.register(ext);
}

```

### 示例代码：群内安价收集

这是关于数据的增加、删除、查询等操作的实现示例（修改的话就是删除之后增加）

```javascript
// ==UserScript==
// @name         群内安价收集
// @author       憧憬少
// @version      1.0.0
// @description  在群内收集群友给出的安价选项，并掷骰得出结果
// @timestamp    1676386517
// 2023-02-14 22:55:17
// @license      MIT
// @homepageURL  https://github.com/ChangingSelf/sealdice-js-ext-anchor
// ==/UserScript==

(() => {
  // src/index.ts
  const HELP = `群内安价收集 (ak 是アンカー罗马字缩写)
注意 ak 后面有空格，“.ak”也可以换成“.安价”

.ak help // 查看帮助
.ak # 标题 // 新建一轮分歧并设标题
.ak + 选项 // 需要添加的选项的内容
.ak - 序号 // 需要移除的选项的序号
.ak ? // 列出目前所有选项
.ak = // 随机抽取 1 个选项并继续
.ak = n // 随机抽取 n 个选项并继续
`;
  const STORAGE_KEY = "anchor";
  const OPTION_NUM_PER_PAGE = 15; // 列出所有选项时，每页放多少个选项
  function akNew(ctx, msg, ext, title) {
    const data = {
      "title": title,
      "options": []
    };
    ext.storageSet(STORAGE_KEY, JSON.stringify(data));
    seal.replyToSender(ctx, msg, `已新建分歧:${title}`);
  }
  function akAdd(ctx, msg, ext, option) {
    const data = JSON.parse(ext.storageGet(STORAGE_KEY) || '{"title":"","options":[]}');
    data.options.push(option);
    seal.replyToSender(ctx, msg, `当前分歧:${data.title}
已添加第${data.options.length}个选项:${option}`);
    ext.storageSet(STORAGE_KEY, JSON.stringify(data));
  }
  function akDel(ctx, msg, ext, index) {
    const data = JSON.parse(ext.storageGet(STORAGE_KEY) || '{"title":"","options":[]}');
    const removed = data.options.splice(index - 1, 1)[0];
    seal.replyToSender(ctx, msg, `当前分歧:${data.title}
已移除第${index}个选项:${removed}`);
    ext.storageSet(STORAGE_KEY, JSON.stringify(data));
  }
  function akList(ctx, msg, ext) {
    const data = JSON.parse(ext.storageGet(STORAGE_KEY) || '{"title":"","options":[]}');
    if (data.options.length === 0) {
      seal.replyToSender(ctx, msg, `当前分歧:${data.title}
还没有任何选项呢`);
      return;
    }
    let optStr = "";
    let curPageRows = 0;
    data.options.forEach((value, index) => {
      optStr += `${index + 1}.${value}
`;
      ++curPageRows;
      if (curPageRows >= OPTION_NUM_PER_PAGE) {
        seal.replyToSender(ctx, msg, `当前分歧:${data.title}
${optStr}`);
        optStr = "";
        curPageRows = 0;
      }
    });
    if (curPageRows > 0) {
      seal.replyToSender(ctx, msg, `当前分歧:${data.title}
${optStr}`);
    }
  }
  function randomInt(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  }
  function akGet(ctx, msg, ext, num = 1) {
    const data = JSON.parse(ext.storageGet(STORAGE_KEY) || '{"title":"","options":[]}');
    if (data.options.length === 0) {
      seal.replyToSender(ctx, msg, `当前分歧:${data.title}
还没有任何选项呢`);
      return;
    }
    akList(ctx, msg, ext);
    let optStr = "";
    for (let i = 0; i < num; ++i) {
      const r = randomInt(1, data.options.length);
      const result = data.options.splice(r - 1, 1);
      optStr += `${i + 1}.${result}
`;
    }
    seal.replyToSender(ctx, msg, `结果是:
${optStr}`);
    ext.storageSet(STORAGE_KEY, JSON.stringify(data));
  }
  function main() {
    let ext = seal.ext.find("anchor");
    if (!ext) {
      ext = seal.ext.new("anchor", "憧憬少", "1.0.0");

      const cmdSeal = seal.ext.newCmdItemInfo();
      cmdSeal.name = "安价";
      cmdSeal.help = HELP;
      cmdSeal.solve = (ctx, msg, cmdArgs) => {
        try {
          let val = cmdArgs.getArgN(1);
          switch (val) {
            case "#": {
              const title = cmdArgs.getArgN(2);
              akNew(ctx, msg, ext, title);
              return seal.ext.newCmdExecuteResult(true);
            }
            case "+": {
              const option = cmdArgs.getArgN(2);
              akAdd(ctx, msg, ext, option);
              return seal.ext.newCmdExecuteResult(true);
            }
            case "-": {
              const index = Number(cmdArgs.getArgN(2));
              akDel(ctx, msg, ext, index);
              return seal.ext.newCmdExecuteResult(true);
            }
            case "?":
            case "？": {
              akList(ctx, msg, ext);
              return seal.ext.newCmdExecuteResult(true);
            }
            case "=": {
              let num = 1;
              if (cmdArgs.args.length >= 2) {
                num = Number(cmdArgs.getArgN(2));
              }
              akGet(ctx, msg, ext, num);
              return seal.ext.newCmdExecuteResult(true);
            }
            case "help":
            default: {
              const ret = seal.ext.newCmdExecuteResult(true);
              ret.showHelp = true;
              return ret;
            }
          }
        } catch (error) {
          seal.replyToSender(ctx, msg, error.Message);
          return seal.ext.newCmdExecuteResult(true);
        }
      };
      ext.cmdMap["安价"] = cmdSeal;
      ext.cmdMap["ak"] = cmdSeal;

      seal.ext.register(ext);
    }
  }
  main();
})();

```

## 数据处理模板

关于取出数据来修改的函数，可以参考如下代码：

```javascript
const STORAGE_KEY = "anchor"; // 将你的 key 抽出来单独作为一个常量，方便开发阶段修改（使用了之后就不要修改了）
//函数：添加选项
function akAdd(ctx, msg, ext, option) {
    //取出数据
    const data = JSON.parse(ext.storageGet(STORAGE_KEY) || '{"title":"","options":[]}');

    //处理数据
    data.options.push(option);

    //响应发送者
    seal.replyToSender(ctx, msg, `当前分歧:${data.title}\n已添加第${data.options.length}个选项:${option}`);

    //将处理完的数据写回去
    ext.storageSet(STORAGE_KEY, JSON.stringify(data));
}
```

## 读取玩家或群组数据

可以查看下文的 [API](#js-扩展-api)。

## 编写暗骰指令

如下：

```javascript
// ==UserScript==
// @name         示例：编写暗骰指令
// @author       流溪
// @version      1.0.0
// @description  暗骰，格式.hide 原因
// @timestamp    1671540835
// 2022-12-20
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==
if (!seal.ext.find('hide')){
    const ext = seal.ext.new('hide','流溪','0.0.1');

    const cmdHide = seal.ext.newCmdItemInfo;
    cmdHide.name = 'hide';
    cmdHide.help = '暗骰，使用 .hide 面数 暗骰';
    cmdHide.solve = (ctx, msg, cmdArgs) => {
      if (msg.messageType !== 'group'){
          seal.replyToSender(ctx, msg, '暗骰只能在群内触发');
          return seal.ext.newCmdExecuteResult(true);
      }
      function rd(x){
          // 这里写的时候有点不清醒了，感觉是对的，如果不对请拷打我
          return Math.round(Math.random() * (x - 1) + 1);
      }
      let x = cmdArgs.getArgN(1);
      if (x === 'help'){
          return seal.ext.newCmdExecuteResult(true).showhelp = true;
      } else if (isNaN(Number(x))){
          // 我知道这里有更好的判断是否为数字的方法但是我不会.jpg
          seal.replyToSender(ctx, msg, `骰子面数应是数字`);
          return seal.ext.newCmdExecuteResult(true);
      } else {
          // 这就是暗骰 api 哒！
          seal.replyPerson(ctx, msg, `你在群${msg.groupId}的掷骰结果为：${rd(x)}`);
          return seal.ext.newCmdExecuteResult(true);
      }
    }
  ext.cmdMap['hide'] = cmdHide;

  seal.ext.register(ext);
}
```

可以看到使用`seal.replyPerson`做到暗骰的效果。

## 编写代骰指令

```javascript
// ==UserScript==
// @name         示例：编写代骰指令
// @author       木落
// @version      1.0.0
// @description  捕捉某人，格式.catch <@名字>
// @timestamp    1671540835
// 2022-12-20
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

// 编写代骰指令
// 先将扩展模块创建出来，如果已创建就直接使用
if (!seal.ext.find('test')) {
  const ext = seal.ext.new('test', '木落', '1.0.0');

  // 创建指令 .catch
  // 这个命令的功能为，显示“试图捕捉某人”，并给出成功率
  // 如果命令写“.catch @张三”，那么就会试着捕捉张三
  const cmdCatch = seal.ext.newCmdItemInfo();
  cmdCatch.name = 'catch';
  cmdCatch.help = '捕捉某人，格式.catch <@名字>';
  // 对这个指令启用使用代骰功能，即@某人时，可获取对方的数据，以对方身份进行骰点
  cmdCatch.allowDelegate = true;
  cmdCatch.solve = (ctx, msg, cmdArgs) => {
    // 获取对方数据，之后用 mctx 替代 ctx，mctx 下读出的数据即被代骰者的个人数据
    const mctx = seal.getCtxProxyFirst(ctx, cmdArgs);
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help': {
        // 命令为 .catch help
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      default: {
        const text = `正在试图捕捉${mctx.player.name}，成功率为${Math.ceil(Math.random() * 100)}%`;
        seal.replyToSender(mctx, msg, text);
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  };
  // 将命令注册到扩展中
  ext.cmdMap['catch'] = cmdCatch;

  seal.ext.register(ext);
}
```

## 网络请求

主要使用 [Fetch API](https://developer.mozilla.org/zh-CN/docs/Web/API/Fetch_API) 进行网络请求，详细文档见链接。`fetch` 函数返回一个 Promise，传统的写法是这样：

```javascript
// 你可以使用 generator 来重写这段代码，欢迎 pr
// 访问网址
fetch('https://api-music.imsyy.top/cloudsearch?keywords=稻香').then((resp) => {
  // 在返回对象的基础上，将文本流作为 json 解析
  resp.json().then((data) => {
    // 打印解析出的数据
    console.log(JSON.stringify(data));
  });
});
```

你也可以使用异步编程（async/await）来简化代码：

```javascript
const response = await fetch('https://api-music.imsyy.top/cloudsearch?keywords=稻香');
if (!response.ok) {
    // 处理不成功的请求...
}
const data = await response.json();
console.log(JSON.stringify(data));
```

套用这个模板，你可以写出很多调用 API 的简单扩展。

比如核心代码只有一行的「[随机猫猫图片](https://github.com/sealdice/javascript/blob/main/scripts/%E5%A8%B1%E4%B9%90%E5%90%91/%E9%9A%8F%E6%9C%BA%E7%8C%AB%E7%8C%AB%E5%9B%BE%E7%89%87.js)」扩展：

```javascript
seal.replyToSender(ctx, msg, `[CQ:image,file=https://thiscatdoesnotexist.com/,cache=0]`);
```

核心代码同样只有一行的「随机二次元图片」扩展：

```javascript
seal.replyToSender(ctx, msg, `[CQ:image,file=https://api.oick.cn/random/api.php?type=${val},cache=0]`);
```

当然，也有稍微复杂的，比如「[AI 骰娘](https://github.com/sealdice/javascript/blob/main/scripts/%E5%A8%B1%E4%B9%90%E5%90%91/dicemaid-ai.js)」扩展，但也没有太复杂，只是处理了一下发送者传过来的消息，再发送给网络 API，收到响应之后再回应发送者。

它的核心代码如下：

```javascript
const BID = ''; // 填入你的骰娘的大脑的 id
const KEY = ''; // 填入你的 key
/**
 * 给 AI 主脑发送消息并接收回复
 * @param ctx 主要是和当前环境以及用户相关的内容，如当前发指令用户，当前群组信息等
 * @param msg 为原生态的指令内容，如指令文本，发送平台，发送时间等
 * @param message 要发送给骰娘的具体消息
 */
function chatWithBot(ctx,msg,message) {
  fetch(`http://api.brainshop.ai/get?bid=${BID}&key=${KEY}&uid=${msg.sender.userId}&msg=${message}`).then(response => {
    if (!response.ok) {
      seal.replyToSender(ctx, msg, `抱歉，我连接不上主脑了。它传递过来的信息是：${response.status}`);
      return seal.ext.newCmdExecuteResult(false);
    } else {
      response.json().then(data => {
        seal.replyToSender(ctx, msg, data["cnt"]);
        return seal.ext.newCmdExecuteResult(true);
      });
      return seal.ext.newCmdExecuteResult(true);
    }
  });
}
```

## 自定义 COC 规则

```javascript
// ==UserScript==
// @name         示例：自定义 COC 规则
// @author       木落
// @version      1.0.0
// @description  自设规则，出 1 大成功，出 100 大失败。困难极难等保持原样
// @timestamp    1671886435
// 2022-12-24
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

const rule = seal.coc.newRule();
rule.index = 20; // 自定义序号必须大于等于 20，代表可用.setcoc 20 切换
rule.key = '测试'; // 代表可用 .setcoc 测试 切换
rule.name = '自设规则'; // 已切换至规则文本 name: desc
rule.desc = '出 1 大成功\n出 100 大失败';
// d100 为出目，checkValue 为技能点数
rule.check = (ctx, d100, checkValue) => {
  let successRank = 0;
  const criticalSuccessValue = 1;
  const fumbleValue = 100;
  if (d100 <= checkValue) {
    successRank = 1;
  } else {
    successRank = -1;
  }
  // 成功判定
  if (successRank == 1) {
    // 区分大成功、困难成功、极难成功等
    if (d100 <= checkValue / 2) {
      //suffix = "成功(困难)"
      successRank = 2;
    }
    if (d100 <= checkValue / 5) {
      //suffix = "成功(极难)"
      successRank = 3;
    }
    if (d100 <= criticalSuccessValue) {
      //suffix = "大成功！"
      successRank = 4;
    }
  } else {
    if (d100 >= fumbleValue) {
      //suffix = "大失败！"
      successRank = -2;
    }
  }
  let ret = seal.coc.newRuleCheckResult();
  ret.successRank = successRank;
  ret.criticalSuccessValue = criticalSuccessValue;
  return ret;
};
// 注册规则
seal.coc.registerRule(rule);

```

## 补充：使用非指令关键词

> 你是否因为自定义回复能实现的功能有限而烦恼？你是否因为自定义回复的匹配方式不全而愤怒？你是否因为自定义回复只能调用图片 API 而感到焦头烂额？
>
> 不要紧张，我的朋友，试试非指令关键词，这会非常有用。

通常情况下，我们使用 `ext.onNotCommandReceived` 作为非指令关键词的标志；这限定了只有在未收到命令且未达成自定义回复时，海豹才会触发此流程。

一个完整的非指令关键词模板如下：

```javascript
// 必要流程，注册扩展，注意即使是非指令关键词也是依附于扩展的
if (!seal.ext.find('xxx')){
  const ext = seal.ext.new('xxx','xxx','x.x.x');
  seal.ext.register(ext);
  // 这里其实是编写处理函数
  ext.onNotCommandReceived = (ctx, msg) => {
    let message = msg.message;
    // 这里请自己处理要如何达成 message 的匹配条件，js 那么多的匹配方法，足够你玩出花来。
    if(xxx){
      // 匹配到关键词了，要干什么？
      xxx;
    }
  }
}
```

## 注册插件配置项 <Badge type="tip" text="v1.4.1" />

插件若要在 UI 中注册可供用户修改的配置项，需要在插件注册后调用 `seal.ext.registerXXXConfig()` 函数注册配置项。

`XXX` 为配置项的类型，目前支持 `string`、`int`、`float`、`bool`、`template`、`option` 六种类型。注意按照小驼峰命名法大写

同样的，插件也可以使用 `seal.ext.getXXXConfig()` 函数获取配置项的值。

你也可以直接使用 `seal.ext.getConfig()` 函数获取配置项的值，这个函数会返回一个 `ConfigItem` 对象，
包含了配置项的类型、值、默认值等信息。

`ConfigItem` 对象的类型定义如下，调用时请使用 `jsbind` 中的值作为 `key`

```go
type ConfigItem struct {
    Key          string      `json:"key" jsbind:"key"`
    Type         string      `json:"type" jsbind:"type"`
    DefaultValue interface{} `json:"defaultValue" jsbind:"defaultValue"`
    Value        interface{} `json:"value,omitempty" jsbind:"value"`
    Option       interface{} `json:"option,omitempty" jsbind:"option"`
    Deprecated   bool        `json:"deprecated,omitempty" jsbind:"deprecated"`
}
```

::: tip 提示：更原始的 API

`seal.ext.registerConfig()` 函数也是可以使用的，你需要自己通过 `seal.ext.newConfigItem()` 来获取一个新的 `ConfigItem` 对象。

在对你的 `ConfigItem` 对象进行修改后，再调用 `seal.ext.registerConfig()` 函数进行注册。

:::

### 示例代码：注册配置项

```js
// ==UserScript==
// @name         js-config-example
// @author       Szzrain
// @version      1.0.0
// @description  演示 js 配置项的用法
// @timestamp    1698636875
// @license      MIT
// ==/UserScript==

if (!seal.ext.find('js-config-example')) {
  const ext = seal.ext.new('js-config-example', 'SzzRain', '1.0.0');
  // 创建一个命令
  const cmdgetConfig = seal.ext.newCmdItemInfo();
  cmdgetConfig.name = 'getconfig';
  cmdgetConfig.help = '使用.getconfig <key> 来获取配置项，仅 master 可用';
  cmdgetConfig.allowDelegate = true;
  cmdgetConfig.solve = (ctx, msg, cmdArgs) => {
    let val = cmdArgs.getArgN(1);
    switch (val) {
      case 'help': {
        const ret = seal.ext.newCmdExecuteResult(true);
        ret.showHelp = true;
        return ret;
      }
      default: {
        if (ctx.privilegeLevel !== 100) {
          seal.replyToSender(ctx, msg, "你没有权限执行此命令");
          return seal.ext.newCmdExecuteResult(true);
        }
        switch (val) {
          case "1":
            strVal = seal.ext.getStringConfig(ext, "testkey1");
            seal.replyToSender(ctx, msg, strVal);
            break;
          case "2":
            intVal = seal.ext.getIntConfig(ext, "testkey2");
            seal.replyToSender(ctx, msg, intVal);
            break;
          case "3":
            floatVal = seal.ext.getFloatConfig(ext, "testkey3");
            seal.replyToSender(ctx, msg, floatVal);
            break;
          case "4":
            boolVal = seal.ext.getBoolConfig(ext, "testkey4");
            seal.replyToSender(ctx, msg, boolVal);
            break;
          case "5":
            tmplVal = seal.ext.getTemplateConfig(ext, "testkey5");
            seal.replyToSender(ctx, msg, tmplVal);
            break;
          case "6":
            optVal = seal.ext.getOptionConfig(ext, "testkey6");
            seal.replyToSender(ctx, msg, optVal);
            break;
          default:
            let config = seal.ext.getConfig(ext, val);
            if (config) {
              seal.replyToSender(ctx, msg, config.value);
            } else {
              seal.replyToSender(ctx, msg, "配置项不存在");
            }
            break;
        }
        return seal.ext.newCmdExecuteResult(true);
      }
    }
  }
  // 注册命令
  ext.cmdMap['getconfig'] = cmdgetConfig;

  // 注册扩展
  seal.ext.register(ext);

  // 注册配置项需在 ext 注册后进行
  // 通常来说，register 函数的参数为 ext, key, defaultValue
  seal.ext.registerStringConfig(ext, "testkey1", "testvalue");
  seal.ext.registerIntConfig(ext, "testkey2", 123);
  seal.ext.registerFloatConfig(ext, "testkey3", 123.456);
  seal.ext.registerBoolConfig(ext, "testkey4", true);
  seal.ext.registerTemplateConfig(ext, "testkey5", ["1", "2", "3", "4"]);
  // 注册单选项函数的参数为 ext, key, defaultValue, options
  seal.ext.registerOptionConfig(ext, "testkey6", "1", ["1", "2", "3", "4", "5", "6", "7", "8", "9", "10"]);
}
```

注册后的配置项会在 UI 中显示，可以在 UI 中修改配置项的值

![JS 配置项](./images/js-config-example.png)

## 注册定时任务 <Badge type="tip" text="v1.4.6" />

::: tip 提示：Cron 表达式

如果你对 `cron` 或下文中提到的 Cron 表达式并不熟悉，可以参考 [Linux crontab 命令 | 菜鸟教程](https://www.runoob.com/linux/linux-comm-crontab.html) 或 [Cron 表达式 - 阿里云文档](https://help.aliyun.com/zh/ecs/user-guide/cron-expressions)。

:::

从 `v1.4.6` 版本开始，海豹核心新增了用于定时任务的 API。

### API 参数

```javascript
seal.ext.registerTask(ext, taskType, value, func, key="", description="")
```

其各个参数的含义如下：

- `taskType: string`：`registerTask` 接受两种类型的定时任务表达式——使用 Cron 表达式或使用 `hh:mm/h:mm` 格式的「每日」任务。当使用前者时，`taskType` 应填入 `"cron"`，而后者应填入 `"daily"`。
- `value: string`：
  - 当 `taskType` 填入 `"cron"` 时，`value` 应填入有效的 Cron 表达式，例如：`"*/5 * * * *"`。`registerTask` 会根据 Cron 表达式定时执行 `func`。
  - 当 `taskType` 填入 `"daily"` 时，`value` 应填入 `hh:mm` 或 `h:mm` 格式的时间，例如：`"08:00"`、`"3:00"`、`"20:35"`。`registerTask` 会根据时间，每天定时执行 `func`。
- `func: (taskCtx: JsScriptTaskCtx) => void`：定时任务的实际执行函数。其中 `taskCtx` 的数据类型为：

  ```typescript
  type JsScriptTaskCtx {
    now: number;
    key: string;
  }
  ```

  `taskCtx.now` 提供了 `func` 实际被唤起时的 Unix 时间戳；如果填写了可选参数 `key`，`taskCtx.key` 则与之相同。

  使用定时任务 API 的用户应该将实际业务逻辑放置在 `func` 内，定时任务 API 仅承担唤醒功能。
- `key: string`：可选参数。为此定时任务提供唯一索引。当填写了 `key` 时，此定时任务也会出现在 WebUI 的插件配置项中，可以通过 WebUI 修改定时任务表达式。
- `description: string`：可选参数。为此定时任务提供可读性更高的描述。当同时填写了 `key` 与 `description` 时，WebUI 的插件配置项中将会显示关于此定时任务的描述。

### 使用示例

```javascript
// 实现任意定时功能
seal.ext.registerTask(ext, "cron", "* * * * *", (taskCtx) => {
  // 检查当前时间点附近，尚未执行的所有任务
  const tasks = getTasks(taskCtx.now, ext.StorageGet("tasks"));
  for (const task of tasks) {
    // 检查当前群聊中，插件功能是否开启
    if (!checkGroupStatus(task.groupId, ext.storageGet("status"))) {
      continue;
    }
    doTask(task);
  }
});
```

```javascript
// 类似每日新闻
seal.ext.registerTask(ext, "daily", "08:30", (taskCtx) => {
  // 所有需要发送每日新闻的群聊
  const groups = getGroups(ext.StorageGet("groups"));
  for (const group of groups) {
    sendDailyNews(group);
  }
}, "daily_news", "每天触发「每日新闻」的时间");
```
