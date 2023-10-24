---
lang: zh-cn
title: 编写 JS 插件
---

# 编写 JS 插件

::: info 本节内容

本节将介绍 JS 脚本的编写，请善用侧边栏和搜索，按需阅读文档。

:::

编写扩展需要了解JavaScript，它非常好学，你如果已经学了[自定义回复](#自定义回复)里使用的[内置脚本语言](#内置脚本语言)，你会发现很多内容只是换了个写法而已。

由于网上优秀的JavaScript教程非常多（见下方），在这里就不赘述，而是重点讲一讲海豹提供的接口。

- [JavaScript-菜鸟教程](https://www.runoob.com/js/js-tutorial.html)
- [JavaScript教程-廖雪峰](https://www.liaoxuefeng.com/wiki/1022910821149312)

- 扩展编写示例：https://github.com/sealdice/javascript/tree/main/examples
- 已有扩展下载：https://github.com/sealdice/javascript/tree/main/scripts

## 0.如何优雅地偷懒

放弃系统自带的记事本软件，拥抱[vscode](https://code.visualstudio.com/)吧，正如@于言诺 大佬在《海豹骰点核心SealDice 个人自定义回复编写经验》中所言：

> 请所有阅读本文档的海豹自定义回复编写者注意，只要你的设备支持，你都不应该使用windows自带的txt或任何非代码编写器的软件编写自定义回复或牌堆。
>
> 有那么多好用的代码编写器，如windows的vs,vscode,sublime text,安卓的MT管理器，请不要折磨自己用txt，至少这些软件还会告诉你世界上有个很好用的东西叫**报错**。

vscode相比于记事本的优势：

- 写错的地方会给你标出来，不用等到安装完才发现
- 强大的编辑器功能，例如语法高亮、显示行号、灵活的多行光标、自动补全、悬停提示、批量重命名、正则查找替换、自动保存、版本控制等
- 丰富的扩展
- 更多这里写不下的功能

如果嫌官网下载得太慢，还可以在群文件的“工具＆内置文件”中找到vscode的安装包。

如果觉得英文界面用起来太困难，可以在vscode扩展商店中安装中文扩展。

（顺便一提，和海豹合作的跑团replay视频制作软件[回声工坊](https://github.com/DanDDXuanX/TRPG-Replay-Generator)也主要使用vscode作为log编辑工具，拥有vscode扩展支持。使用vscode说不定以后也会有相应的辅助扩展来帮忙在写JS扩展时偷懒）

（已经有了，名为[Sealdice Snippets](https://marketplace.visualstudio.com/items?itemName=yxChangingSelf.sealdice-snippets)，在vscode的扩展商店就能找到并直接安装，提供了一些常见代码片段，可以快速生成模板代码）

## 1.创建一个JS扩展

你可以选择使用JavaScript或者Typescript来作为你的编写语言。

Typescript是JavaScript的超集，如果你不太懂这是啥，可以理解为是强化版的JavaScript（虽然这么说可能不太准确）。

Typescript文件可以编译为JavaScript文件，安装的时候也是安装编译好的JS文件，但Typescript更优秀的特性可以帮助你在编写扩展的阶段更方便。

**更推荐使用Typescript，不过你可以先从JavaScript开始。**

如果你打算使用JavaScript，那么新建一个文本文件，将文件扩展名改为`.js`即可。

如果你打算使用Typescript，可以使用[github上的扩展模板](https://github.com/sealdice/sealdice-js-ext-template)，注册扩展和指令的代码已经写好，可以直接编译出一个可直接装载的JS扩展文件。

JS扩展的示例：

```javascript
// ==UserScript==
// @name         示例:如何开始
// @author       木落
// @version      1.0.0
// @description  这是一个演示脚本，并没有任何实际作用。
// @timestamp    1671368035
// 2022-12-18
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

/*
这里是海豹支持的js脚本范例
海豹使用的js脚本引擎为goja.
在几次更新后，goja支持了ES6的基本上全部特性，包括async/await，promise和generator.

特别注意一点是js引擎的整型为32位，请小心溢出问题。

推荐使用的语法风格为airbnb风格，内容较多这里不赘述，其有代表性的一些特征为：
使用两空格缩进，{不换行，必须写分号，只用let不写var等。

if (true) {
  let a = 123;
  console.log(a);
}

推荐有经验的用户使用typescript，但注意要编译打包后才能使用，target选es6应当可以工作。

还有一个小提示：
console打印出来的东西不光会在控制台中出现，在日志中也会显示。
涉及网络请求或延迟执行的内容，有时候不会在控制台调试面板上显示出来，而在日志中能看到。

以及重要提醒：
不要灌铅！不要灌铅！不要灌铅！
*/

console.log('这是测试控制台');
console.log('可以这样来查看变量详情：');
console.log(Object.keys(seal));
console.log('更多内容正在制作中...');
console.log('注意: 测试版！API仍然可能发生重大变化！');
```

## 2.留下作者信息

每个JS扩展需要在开头以注释的形式留下如下信息以便大家使用：

```js
// ==UserScript==
// @name         脚本的名字
// @author       木落
// @version      1.0.0
// @description  这是一个演示脚本，并没有任何实际作用。
// @timestamp    1672066028
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==
```

| 属性         | 含义                                                         |
| ------------ | ------------------------------------------------------------ |
| @name        | JS扩展的名称，会展示在插件列表页面                           |
| @author      | 作者名                                                       |
| @version     | 版本号，可以自己定义，但建议遵循[语义版本控制（Semantic Versioning）](https://semver.org/lang/zh-CN/) |
| @description | 对扩展的功能的描述                                           |
| @timestamp   | 最后更新时间，以秒为单位的unix时间戳，可以搜索一些[时间戳在线转换工具](https://developer.aliyun.com/skills/timestamp.html)来获取当前时间戳 |
| @license     | 开源协议，示例中的Apache-2是一个比较自由的协议，允许任意使用和分发(包括商用) |
| @homepageURL | 你的扩展的主页链接，有github仓库可以填仓库链接，没有则可以填海豹官方插件仓库链接 |

## 3.自定义扩展

扩展机制可以看做是海豹的mod管理器，可以模块化开关海豹的任意一部分，如常用的开启dnd扩展，关闭coc扩展，关闭自动回复等等。

可以通过[.ext命令](#.ext 扩展管理)来进行具体操作，**所有指令必须归属于某个扩展**，而一个扩展可以包含多条指令。

例如内置扩展`coc7`包含了如下指令：coc、en、li、rc/ra、rcv/rav、sc、setcoc、st、ti

可以通过Web图形界面中【综合设置】下的【基本设置】的最底下，设置各个扩展及其指令的默认开启状态。

出于对公平性的考虑，js脚本不能替换内置指令和内置扩展。

创建扩展之后，要注意**还需要注册扩展**，才能让扩展起效，不要漏掉哦！

```js
// 如何建立一个扩展

// 首先检查是否已经存在
if (!seal.ext.find('test')) {
  // 不存在，那么建立扩展，名为test，作者“木落”，版本1.0.0
  const ext = seal.ext.new('test', '木落', '1.0.0');
  // 注册扩展
  seal.ext.register(ext);
}
```

## 4.自定义指令

想要创建一条自定义指令，首先需要创建一个扩展（`seal.ExtInfo`），写好自定义指令的实现逻辑之后，再注册到扩展中。

接上一个标题下的代码，假设目前已经注册好了一个名为`test`的扩展，现在要写一个名为`seal`的指令。

- 这个命令的功能为，显示“抓到一只海豹的文案”；
- 如果命令写“.seal ABC”，那么文案中将海豹命名为“ABC”；
- 如果命令中没写名字，那么命名为默认值“氪豹”。

第一步，创建新的自定义指令，设置好名字和帮助信息。

```js
const cmdSeal = seal.ext.newCmdItemInfo();
cmdSeal.name = 'seal'; // 指令名字，可用中文
cmdSeal.help = '召唤一只海豹，可用.seal <名字> 命名';
```

第二步，编写指令的具体处理代码。

你需要编写指令对象的`solve`函数，而在使用该指令的时候，海豹核心会调用你写的这个函数。

```js
cmdSeal.solve = (ctx, msg, cmdArgs) => {
    //这里是你需要编写的内容
};
```

| 参数    | 说明                                                         |
| ------- | ------------------------------------------------------------ |
| ctx     | 主要是和当前环境以及用户相关的内容，如当前发指令用户，当前群组信息等 |
| msg     | 原始指令内容，如指令文本，发送平台，发送时间等               |
| cmdArgs | 指令信息，会将用户发的信息进行分段，方便快速取用             |

这里仅说明需要用到的接口，详细可见[插件仓库](https://github.com/sealdice/javascript/tree/main/examples_ts)的`examp_ts`目录下的`seal.d.ts`文件，里面包含了目前开放的接口的定义及其注释说明。

**参数与返回值**

假设用户发送过来的消息是`.seal A B C`，那么可以用`cmdArgs.getArgN(1)`获取到`A`，`cmdArgs.getArgN(2)`获取到`B`，`cmdArgs.getArgN(3)`获取到`C`。

通常会对参数值进行判断，随后作出响应。

以下代码处理的是`.seal help`的情形：

```js
cmdSeal.solve = (ctx, msg, cmdArgs) => {
  // 获取第一个参数，例如 .seal A B C
  // 这里 cmdArgs.getArgN(1) 的结果即是A，传参为2的话结果是B
  let val = cmdArgs.getArgN(1);
  switch (val) {
    case 'help': {
      // 命令为 .seal help
      // 创建一个结果对象，并将showHelp标记为true，这会自动给用户发送帮助
      const ret = seal.ext.newCmdExecuteResult(true);
      ret.showHelp = true;
      return ret;
    }
    default: {
      //没有传入参数时的代码
      return seal.ext.newCmdExecuteResult(true);
    }
  }
};
```

注意，在执行完自己的代码之后，需要返回指令结果对象，其参数是是否执行成功。

**核心代码**

给消息发送者回应，需要使用`seal.replyToSender()`函数，前两个参数和`solve()`函数接收的参数一致，第三个参数是你要发送的文本。

发送的文本中，可以包含[自定义回复](#自定义回复)中的变量（例如`{$t玩家}`），也可以包含[CQ码](https://docs.go-cqhttp.org/cqcode)，用来实现回复发送者、@发送者、发送图片、发送分享卡片等功能。

在这个例子中，我们需要获取作为海豹名字的参数，获取不到就使用默认值，随后向消息发送者发送回应。

在刚刚的位置填入核心代码，就可以完成了。

```js
cmdSeal.solve = (ctx, msg, cmdArgs) => {
  // 获取第一个参数，例如 .seal A B C
  // 这里 cmdArgs.getArgN(1) 的结果即是A，传参为2的话结果是B
  let val = cmdArgs.getArgN(1);
  switch (val) {
    case 'help': {
      // 命令为 .seal help
      // 创建一个结果对象，并将showHelp标记为true，这会自动给用户发送帮助
      const ret = seal.ext.newCmdExecuteResult(true);
      ret.showHelp = true;
      return ret;
    }
    default: {
      // 命令为 .seal XXXX，取第一个参数为名字
      if (!val) val = '氪豹';
      // 进行回复，如果是群聊发送那么在群里回复，私聊发送则在私聊回复(听起来是废话文学，但详细区别见暗骰例子)
      seal.replyToSender(ctx, msg, `你抓到一只海豹！取名为${val}\n它的逃跑意愿为${Math.ceil(Math.random() * 100)}`);
      return seal.ext.newCmdExecuteResult(true);
    }
  }
};
```

第三步，将命令注册到扩展中。

```js
ext.cmdMap['seal'] = cmdSeal;
```

如果你想要给这个命令起一个别称，也就是增加一个触发词，可以这样写：

```js
ext.cmdMap['seal'] = cmdSeal;//注册.seal指令
ext.cmdMap['海豹'] = cmdSeal;//注册.海豹指令，等效于.seal
```

完整的代码如下：

```js
// ==UserScript==
// @name         示例:编写一条自定义指令
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
  seal.ext.register(ext);
}

// 创建指令 .seal
// 这个命令的功能为，显示“抓到一只海豹的文案”
// 如果命令写“.seal ABC”，那么文案中将海豹命名为“ABC”
// 如果命令中没写名字，那么命名为默认值“氪豹”
const cmdSeal = seal.ext.newCmdItemInfo();
cmdSeal.name = 'seal'; // 指令名字，可用中文
cmdSeal.help = '召唤一只海豹，可用.seal <名字> 命名';

// 主函数，指令解析器会将指令信息解析，并储存在几个参数中
// ctx 主要是和当前环境以及用户相关的内容，如当前发指令用户，当前群组信息等
// msg 为原生态的指令内容，如指令文本，发送平台，发送时间等
// cmdArgs 为指令信息，会将用户发的信息进行分段，方便快速取用
cmdSeal.solve = (ctx, msg, cmdArgs) => {
  // 获取第一个参数，例如 .seal A B C
  // 这里 cmdArgs.getArgN(1) 的结果即是A，传参为2的话结果是B
  let val = cmdArgs.getArgN(1);
  switch (val) {
    case 'help': {
      // 命令为 .seal help
      // 创建一个结果对象，并将showHelp标记为true，这会自动给用户发送帮助
      const ret = seal.ext.newCmdExecuteResult(true);
      ret.showHelp = true;
      return ret;
    }
    default: {
      // 命令为 .seal XXXX，取第一个参数为名字
      if (!val) val = '氪豹';
      // 进行回复，如果是群聊发送那么在群里回复，私聊发送则在私聊回复(听起来是废话文学，但详细区别见暗骰例子)
      seal.replyToSender(ctx, msg, `你抓到一只海豹！取名为${val}\n它的逃跑意愿为${Math.ceil(Math.random() * 100)}`);
      return seal.ext.newCmdExecuteResult(true);
    }
  }
}

// 将命令注册到扩展中
ext.cmdMap['seal'] = cmdSeal;

// 无实际意义，用于消除编译报错
export { }
```

这就是最基本的模板了。

## 5.进行随机

由于木落没有在示例中写这部分，我也不太清楚是不是这里有什么接口可以调用，如果后面有的话那就再补上。

这里就先摆一个随机整数的生成函数吧：

```js
/**
 * 生成随机整数
 * @param min 最小值
 * @param max 最大值
 * @returns 位于[min,max]区间的随机整数
 */
function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}
```

## 6.抽取牌堆

原来还可以抽取牌堆吗？但目前开放的API里面找不到这个API，等待木落烤饼吧。

## 7.存取数据

相关的API是两个函数，`ExtInfo.storageSet(key,value)`函数和`ExtInfo.storageGet(key)`，一个存，一个取。

**关于key：**

存储时需要指定key，你可以设定为你的扩展的名字，也可以设定为其他的，注意不要和别的扩展的key重名就可以了。

就好比你在商场门口想要把随身物品存进暂存柜中，需要先找到个和别人不重复的柜子，避免放错地方或者取错东西。

**关于value：**

存放的数据是字符串类型，且只能存一个，但如果想要存放更多的数据以及非字符串类型的数据怎么办？

答案是使用`JSON.stringify()`函数将存储了数据的JS对象转换为[JSON](https://www.runoob.com/json/json-tutorial.html)字符串，存储起来，需要取出的时候，再使用`JSON.parse()`函数将数据再转换为JS对象。

### 示例代码：投喂插件

```js
// ==UserScript==
// @name         示例:存储数据
// @author       木落
// @version      1.0.0
// @description  投喂，格式 .投喂 <物品>
// @timestamp    1672423909
// 2022-12-31
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

// 先将扩展模块创建出来，如果已创建就直接使用
let ext = seal.ext.find('test');
if (!ext) {
  ext = seal.ext.new('test', '木落', '1.0.0');
  seal.ext.register(ext);
}
const cmdFeed = seal.ext.newCmdItemInfo();
cmdFeed.name = '投喂';
cmdFeed.help = '投喂，格式: .投喂 <物品>\n.投喂 记录 // 查看记录';
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

```

### 示例代码：群内安价收集

这是关于数据的增加、删除、查询等操作的实现示例（修改的话就是删除之后增加）

```js
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
  const HELP = `群内安价收集(ak是アンカー罗马字缩写)
注意ak后面有空格，“.ak”也可以换成“.安价”

.ak help //查看帮助
.ak # 标题 //新建一轮分歧并设标题
.ak + 选项 //需要添加的选项的内容
.ak - 序号 //需要移除的选项的序号
.ak ? //列出目前所有选项
.ak = //随机抽取1个选项并继续
.ak = n //随机抽取n个选项并继续
`;
  const STORAGE_KEY = "anchor";
  const OPTION_NUM_PER_PAGE = 15;//列出所有选项时，每页放多少个选项
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
      seal.ext.register(ext);
    }
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
  }
  main();
})();

```

### 模板

关于取出数据来修改的函数，可以参考如下代码：

```js
const STORAGE_KEY = "anchor";//将你的key抽出来单独作为一个常量，方便开发阶段修改（使用了之后就不要修改了）
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

## 8.读取玩家或群组数据

相关API比较多，还是等木落佬写示例代码吧。或者查看下文的[API](#JS扩展API)。

## 9.编写暗骰指令

如下：  

```js
// ==UserScript==
// @name         示例:编写暗骰指令
// @author       流溪
// @version      1.0.0
// @description  暗骰，格式.hide 原因
// @timestamp    1671540835
// 2022-12-20
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==
ext = seal.ext.find('hide');
if (!ext){
    ext = seal.ext.new('hide','流溪','0.0.1');
    seal.ext.register(ext);
}
const cmdHide = seal.ext.newCmdItemInfo;
cmdHide.name = 'hide';
cmdHide.help = '暗骰，使用 .hide 面数 暗骰';
cmdHide.solve = (ctx, msg, cmdArgs) => {
    if (msg.messageType !== 'group'){
        seal.replyToSender(ctx, msg, '暗骰只能在群内触发');
        return seal.ext.newCmdExecuteResult(true);
    }
    function rd(x){
        //这里写的时候有点不清醒了，感觉是对的，如果不对请拷打我
        return Math.round(Math.random() * (x - 1) + 1);
    }
    let x = cmdArgs.getArgN(1);
    if (x === 'help'){
        return seal.ext.newCmdExecuteResult(true).showhelp = true;
    } else if (isNaN(Number(x))){
        //我知道这里有更好的判断是否为数字的方法但是我不会.jpg
        seal.replyToSender(ctx, msg, `骰子面数应是数字`);
        return seal.ext.newCmdExecuteResult(true);
    } else {
        //这就是暗骰api哒！
        seal.replyPerson(ctx, msg, `你在群${msg.groupId}的掷骰结果为：${rd(x)}`);
        return seal.ext.newCmdExecuteResult(true);
    }
}
ext.cmdMap['hide'] = cmdHide;
```

可以看到使用`seal.replyPerson`做到暗骰的效果。

## 10.编写代骰指令

```js
// ==UserScript==
// @name         示例:编写代骰指令
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
let ext = seal.ext.find('test');
if (!ext) {
  ext = seal.ext.new('test', '木落', '1.0.0');
  seal.ext.register(ext);
}
// 创建指令 .catch
// 这个命令的功能为，显示“试图捕捉某人”，并给出成功率
// 如果命令写“.catch @张三”，那么就会试着捕捉张三
const cmdCatch = seal.ext.newCmdItemInfo();
cmdCatch.name = 'catch';
cmdCatch.help = '捕捉某人，格式.catch <@名字>';
// 对这个指令启用使用代骰功能，即@某人时，可获取对方的数据，以对方身份进行骰点
cmdCatch.allowDelegate = true;
cmdCatch.solve = (ctx, msg, cmdArgs) => {
  // 获取对方数据，之后用mctx替代ctx，mctx下读出的数据即被代骰者的个人数据
  const mctx = seal.getCtxProxyFirst(ctx, msg);
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

```

## 11.网络请求

主要使用[Fetch API](https://developer.mozilla.org/zh-CN/docs/Web/API/Fetch_API)进行网络请求，详细文档见链接。

```js
// 你可以使用async/await和generator来重写这段代码，欢迎pr
// 访问网址
fetch('https://api-music.imsyy.top/cloudsearch?keywords=稻香').then((resp) => {
  // 在返回对象的基础上，将文本流作为json解析
  resp.json().then((data) => {
    // 打印解析出的数据
    console.log(JSON.stringify(data));
  });
});
```

套用这个模板，你可以写出很多调用API的简单扩展。

比如核心代码只有一行的[“随机猫猫图片”](https://github.com/sealdice/javascript/blob/main/scripts/%E5%A8%B1%E4%B9%90%E5%90%91/%E9%9A%8F%E6%9C%BA%E7%8C%AB%E7%8C%AB%E5%9B%BE%E7%89%87.js)扩展：

```js
seal.replyToSender(ctx, msg, `[CQ:image,file=https://thiscatdoesnotexist.com/,cache=0]`);
```

核心代码同样只有一行的“随机二次元图片”扩展：

```js
seal.replyToSender(ctx, msg, `[CQ:image,file=https://api.oick.cn/random/api.php?type=${val},cache=0]`);
```

当然，也有稍微复杂的，比如“[AI骰娘](https://github.com/sealdice/javascript/blob/main/scripts/%E5%A8%B1%E4%B9%90%E5%90%91/dicemaid-ai.js)”扩展，但也没有太复杂，只是处理了一下发送者传过来的消息，再发送给网络API，收到响应之后再回应发送者。

它的核心代码如下：

```js
const BID = ''; //填入你的骰娘的大脑的id
const KEY = ''; //填入你的key
/**
 * 给AI主脑发送消息并接收回复
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

## 12.自定义COC规则

```js
// ==UserScript==
// @name         示例:自定义COC规则
// @author       木落
// @version      1.0.0
// @description  自设规则，出1大成功，出100大失败。困难极难等保持原样
// @timestamp    1671886435
// 2022-12-24
// @license      Apache-2
// @homepageURL  https://github.com/sealdice/javascript
// ==/UserScript==

const rule = seal.coc.newRule();
rule.index = 20; // 自定义序号必须大于等于20，代表可用.setcoc 20切换
rule.key = '测试'; // 代表可用 .setcoc 测试 切换
rule.name = '自设规则'; // 已切换至规则文本 name: desc
rule.desc = '出1大成功\n出100大失败';
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

## 使用TS模板

### clone或下载项目

推荐的流程：

1. 在[模板项目仓库](https://github.com/sealdice/sealdice-js-ext-template)点击“Use this template”按钮，使用该模板在自己的github上创建一个扩展的仓库，并设置为自己的扩展的名字
2. `git clone`到本地，进行开发

如果你没有github账号，也不会用git：

1. 在[模板项目仓库](https://github.com/sealdice/sealdice-js-ext-template)点击“Code”按钮，在出现的浮窗中选择“Download ZIP”，这样就会下载一个压缩包
2. 解压后进行开发

### 编译项目

NPM是一种javascript的包管理工具，可以管理项目使用的依赖包。

一开始，需要先将所需依赖包安装好，在命令行使用如下命令：

```bash
npm install
```

当你写好了代码，需要将ts文件转换为js文件以便上传到海豹骰时，在命令行使用如下命令：

```bash
npm run build
```

编译成功的js文件在dist目录下，默认的名字是`sealdce-js-ext.js`。

### 填写个人信息

当插件开发完成后(或者开始开发时)，你需要修改几处地方：

- header.txt 这个文件是你插件的描述信息
- tools/build-config.js 最开头一行"var filename = 'sealdce-js-ext.js';"，改成你中意的名字，注意不要与现有的重名

### 目录结构

只列出其中主要的一些文件

- src
  - `index.ts`：你的扩展的代码就写在这个文件里
- tools
  - `build-config`：一些编译的配置，影响`index.ts`编译成js文件的方式
  - `build.js`：在命令`npm run build`执行时所运行的脚本，用于读取`build-config`并按照配置进行编译
- types
  - `seal.d.ts`：类型文件，海豹核心提供的扩展API
- `header.txt`：扩展头信息，会在编译时自动加到目标文件头部
- `package.json`：命令`npm install`时就在安装这个文件里面所指示的依赖包
- `tsconfig.json`：typescript的配置

## 补充：使用非指令关键词  

你是否因为自定义回复能实现的功能有限而烦恼？你是否因为自定义回复的匹配方式不全而愤怒？你是否因为自定义回复只能调用图片api而感到焦头烂额？不要紧张，我的朋友，试试非指令关键词，这会非常有用。  

通常情况下，我们使用`ext.onNotCommandReceived`作为非指令关键词的标志；这限定了只有在未收到命令且未达成自定义回复时，海豹才会触发此流程。  

一个完整的非指令关键词模板如下：  

```javascript
//必要流程，注册扩展，注意即使是非指令关键词也是依附于扩展的  
if (!seal.ext.find('xxx')){    
    ext = seal.ext.new('xxx','xxx','x.x.x');    
    seal.ext.register(ext); 
    //这里其实是编写处理函数     
    ext.onNotCommandReceived = (ctx, msg) => {    
        let message = msg.message;  
        //这里请自己处理要如何达成message的匹配条件，js那么多的匹配方法，足够你玩出花来。  
        if(xxx){
          //匹配到关键词了，要干什么？  
          xxx;
        }
    }
}
```

## JS扩展API

> 这里只是粗略的整理，具体请看[jsvm源码](https://github.com/sealdice/sealdice-core/blob/master/dice/dice_jsvm.go)。

按类别整理。

> 其中ctx为信息的MsgContext，msg为信息的Message，一般会在定义指令函数时就声明，如:

```javascript
cmd.solve = (ctx, msg, cmdArgs) => {
    someFunction;
} 
```

下面是api的说明（完全了吧......应该？）：

```javascript
//被注释掉的api是可以提供的，但是在源码中被注释。  
//seal.setVarInt(ctx, `$XXX`, valueToSet) //`$XXX`即rollvm（初阶豹语）中的变量，其会将$XXX的值设定为int类型的valueToSet。  
//seal.setVarStr(ctx, `$XXX`, valueToSet) //同上，区别是设定的为str类型的valueToSet。  
seal.replyGroup(ctx, msg, something) //向收到指令的群中发送something。  
seal.replyPerson(ctx, msg, something) //顾名思义，类似暗骰，向指令触发者（若为好友）私信something。  
seal.replyToSender(ctx, msg, something) //同上，区别是群内收到就群内发送，私聊收到就私聊发送。  
seal.memberBan(ctx, groupID, userID, dur) //将指定群的指定用户封禁指定时间(似乎只实现了walleq协议?)
seal.memberKick(ctx, groupID, userID)  //将指定群的指定用户踢出(似乎也只实现了walleq协议?)
seal.format(ctx, something) //将something经过一层rollvm转译并返回，注意需要配合replyToSender才能发送给触发者！  
seal.formatTmpl(ctx, something) //调用自定义文案something  
seal.getCtxProxyFirst(ctx, cmdArgs)  //获取被at的第一个人, 等价于getCtxProxyAtPos(ctx, 0)  
seal.vars.intGet(ctx, `$XXX`) //返回一个数组，其为[int类型的触发者的该变量的值,bool]当strGet一个int或intGet一个str时bool为false，若一切正常则为true。（之所以会有这么奇怪的说法是因为rollvm的“个人变量”机制）。  
seal.vars.intSet(ctx, `$XXX`, valueToSet) //`$XXX`即rollvm（初阶豹语）中的变量，其会将$XXX的值设定为int类型的valueToSet。  
seal.vars.strGet(ctx, `$XXX`) //返回一个数组，其为[str类型的触发者的该变量的值,bool]（之所以会有这么奇怪的说法是因为rollvm的“个人变量”机制），当strGet一个int或intGet一个str时bool为false，如果一切正常则为true。  
seal.vars.strSet(ctx, `$XXX`, valueToSet) //`$XXX`即rollvm（初阶豹语）中的变量，其会将$XXX的值设定为str类型的valueToSet。  
//seal.vars.varSet(ctx, `$XXX`, valueToSet) //可能是根据数据类型自动推断int或str？
//seal.vars.varGet(ctx, `$XXX`) //同上
seal.ext.newCmdItemInfo() //用来定义新的指令；没有参数，个人觉得可以视其为类（class）。  
seal.ext.newCmdExecuteResult(bool) //用于判断指令执行结果，true为成功，false为失败。  
seal.ext.new(extName, extAuthor, Version) //用于建立一个名为extName，作者为extAuthor，版本为Version的扩展。注意，extName， extAuthor和Version均为字符串。  
seal.ext.find(extName) //用于查找名为extname的扩展，若存在则返回true，否则返回false。  
seal.ext.register(newExt) //将扩展newExt注册到系统中。注意newExt是seal.ext.new的返回值，将register视为seal.ext.new()是错误的。  
seal.coc.newRule() //用来创建自定义coc规则，github.com/sealdice/javascript/examples中已有详细例子，不多赘述。  
seal.coc.newRuleCheckResult() //同上，不多赘述。  
seal.coc.registerRule(rule) //同上，不多赘述。  
seal.deck.draw(ctx, deckname, isShuffle) //他会返回一个抽取牌堆的结果。这里有些复杂：deckname为需要抽取的牌堆名，而isShuffle则是一个布尔值，它决定是否放回抽取；false为放回，true为不放回。  
seal.deck.reload() //重新加载牌堆。  
//下面是1.2新增api  
seal.newMessage() //返回一个空白的Message对象, 结构与收到消息的msg相同
seal.createTempCtx(endpoint, msg) // 制作一个ctx, 需要msg.MessageType和msg.Sender.UserId
seal.applyPlayerGroupCardByTemplate(ctx, tmpl) // 设定当前ctx玩家的自动名片格式
seal.gameSystem.newTemplate(string) //从json解析新的游戏规则。  
seal.gameSystem.newTemplateByYaml(string) //从yaml解析新的游戏规则。 
seal.getCtxProxyAtPos(ctx, pos) //获取第pos个被at的人, pos从0开始计数 
seal.atob(base64String) //返回被解码的base64编码  
seal.btoa(string) //将string编码为base64并返回
```

### 部分api使用示例

> 声明和注册扩展的代码部分已省略。

#### 1: replyGroup, replyPerson, replyToSender:

```js
//在私聊触发replyGroup不会回复
seal.replyGroup(ctx, msg, 'something'); //触发者会收到"something"的回复
seal.replyPerson(ctx, msg, 'something'); //触发者会收到"something"的私聊回复
seal.replyToSender(ctx, msg, 'something'); //触发者会收到"something"的回复
```

#### 2: memberBan, memberKick

> 是否保留待议

```js
//注意这些似乎只能在WQ协议上实现;
seal.memberBan(ctx, groupID, userID, dur) //将群为groupID，userid为userID的人封禁dur（单位未知）
seal.memberKick(ctx, groupID, userID) ////将群为groupID，userid为userID的人踢出那个群
```

#### 3: format, formatTmpl

```js
//注意format不会自动reply，而是return，所以请套一层reply
seal.replyToSender(ctx, msg, seal.format(`{$t玩家}的人品为：{$t人品}`))
//{$t人品}是一个rollvm变量，其值等于.jrrp出的数值
//回复：
//群主的人品为：87
seal.replyToSender(ctx, msg, seal.formatTmpl(unknown))
//这里等大佬来了再研究
```

#### 4: getCtxProxyFirst, getCtxProxyAtPos

```js
cmd.solve = (ctx, msg, cmdArgs) => {
    let ctxFirst = seal.getCtxProxyFirst(ctx, cmdArgs)
    seal.replyToSender(ctx, msg, ctxFirst.player,name)
}
ext.cmdMap['test'] = cmd
//输入：.test @A @B
//返回：A的名称。这里其实获取的是A玩家的ctx，具体见文末的ctx数据结构。
cmd.solve = (ctx, msg, cmdArgs) => {
    let ctx3 = seal.getCtxProxyAtPos(ctx, 3)
    seal.replyToSender(ctx, msg, ctx3.player,name)
}
ext.cmdMap['test'] = cmd
//输入：.test @A @B @C
//返回：C（第三个被@的人）的名称。这里其实获取的是C玩家的ctx，具体见文末的ctx数据结构。
```

#### 5: vars

```js
//要看懂这里你可能需要学习一下初阶豹语
seal.vars.intSet(ctx, `$m今日打胶次数`， 8) //将触发者的该个人变量设置为8
seal.vars.intGet(ctx, `$m今日打胶次数`) //返回 [8,true]
seal.vars.strSet(ctx, `$g群友发癫语录`, `一条也没有，快来发癫吧`) //将群内的该群组变量设置为“一条也没有，快来发癫吧！”
seal.vars.strGet(ctx, `$g群友发癫语录`) //返回 ["一条也没有，快来发癫吧",true]
```

#### 6: ext

```js
//用于注册扩展和定义指令的api，已有详细示例，不多赘述
```

#### 7: coc

```js
//用于创建coc村规的api，已有详细示例，不多赘述
```

#### 8: deck

```js
seal.deck.draw(ctx, `煤气灯`, false) //返回 放回抽取牌堆“煤气灯”的结果
seal.deck.draw(ctx, `煤气灯`, true) //返回 不放回抽取牌堆“煤气灯”的结果
seal.deck.reload() //重新加载牌堆
```

#### 9: 自定义trpg规则相关

```js
//这里实在不知道如何举例了
seal.gameSystem.newTemplate(string) //从json解析新的游戏规则。  
seal.gameSystem.newTemplateByYaml(string) //从yaml解析新的游戏规则。
seal.applyPlayerGroupCardByTemplate(ctx, tmpl) // 设定当前ctx玩家的自动名片格式
```

#### 10: 其他

```js
seal.newMessage() //返回一个空白的Message对象, 结构与收到消息的msg相同
seal.createTempCtx(endpoint, msg) // 制作一个ctx, 需要msg.MessageType和msg.Sender.UserId
seal.atob(base64String) //返回被解码的base64编码  
seal.btoa(string) //将string编码为base64并返回
seal.getEndPoints() //返回骰子（应该？）的EndPoints
seal.getVersion() //返回一个map，键值为version和versionCode
```

### `ctx` 的内容

```javascript
//在github.com/sealdice/javascript/examples_ts/seal.d.ts中有完整内容
// 成员
ctx.group // 当前群信息(对象)
ctx.player // 当前玩家数据(对象)
ctx.endPoint // 接入点数据(对象)
// 以上三个对象内容暂略
ctx.isCurGroupBotOn // bool 
ctx.isPrivate // bool 是否私聊
ctx.privilegeLevel // int 权限等级 40邀请者 50管理 60群主 70信任 100master
ctx.delegateText // string 代骰附加文本
// 方法 (太长,懒.)
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

#### `ctx.group` 的内容

```js
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

#### `ctx.player` 的内容

```js
// 成员
name
userId
lastCommandTime
autoSetNameTemplate
// 方法
getValueNameByAlias
```

#### `ctx.endPoint` 的内容

```js
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

### `msg` 的内容

```js
// 成员
msg.time // int64 发送时间
msg.messageType // string group群聊 private私聊
msg.groupId // string 如果是群聊, 群号
msg.guildId // string 服务器群组号，会在discord,kook,dodo等平台见到
msg.sender // 发送者信息(对象)
    sender.nickname
    sender.userId
msg.message
msg.rawId // 原始信息ID, 用于撤回等
msg.platform // 平台
// 方法
// (似乎目前没有?)
```

### `cmdArgs` 的内容

```js
// 成员
.command // string
.args // []string
.kwargs // []Kwarg
.at // []AtInfo
.rawArgs // string
.amIBeMentioned // bool (为何要加一个Be?)
.cleanArgs // string 一种格式化后的参数，也就是中间所有分隔符都用一个空格替代
.specialExecuteTimes // 特殊的执行次数，对应 3# 这种
// 方法
.isArgEqual(n, ss...) // 返回bool, 检查第n个参数是否在ss中
.eatPrefixWith(ss...) // 似乎是从cleanArgs中去除ss中第一个匹配的前缀
.chopPrefixToArgsWith(ss...) // 不懂
.getArgN(n) // -> string
.getKwarg(str) // -> Kwarg 如果有名为str的flag,返回对象,否则返回null/undefined(不确定)
.getRestArgsFrom(n) // -> string 获取从第n个参数之后的所有参数, 用空格拼接成一个字符串
```

### 