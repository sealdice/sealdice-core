---
lang: zh-cn
title: QQ
---

# QQ

::: info 本节内容

本节将包含你在 QQ 平台接入海豹核心需要了解的特定内容。

:::

## 前言

### 有关 QQ 平台机器人的说明

直至目前，绝大部分群聊中的 QQ 机器人采用「**假用户**」方式，即通过第三方软件接入注册的另一个 QQ。**QQ 官方一直在对第三方实现进行技术与非技术层面的多重打击。**

从目前的表现看来，QQ 官方会对账号行为进行检测，来区分出账号是否是正常用户（如不正常的登录方式，以不合理的速度在多地区登录等等）。我们无法得知具体的检测细节，但已证实的是，当 QQ 账号用作机器人并被检测到时，该 QQ 会视为风险账号，被官方予以警告，封禁，临时甚至 **永久冻结** 的惩罚。

尽管不同方案之间的差异很大（比如基于 Android QQ 协议的 [Go-Cqhttp](/about/archieve.md#Go-cqhttp/Mirai) 已经**基本不可用**，而 [Lagrange](#lagrange) 和 [LLOneBot](#llonebot) 等基于 NTQQ 的方案目前比较稳定），但需要明白的是，这些方案都由社区第三方软件提供，实质上以 QQ 官方角度等同于「**外挂软件**」，并不受到官方支持（甚至是被打击的目标）。

因此，*是否在 QQ 平台搭建这样的非官方机器人取决于你的慎重考虑*。同时，第三方方案的可用性也可能会随时间推移而存在变化，海豹官方无法做出任何保证。

目前，仅有 [官方机器人服务](#官方机器人) 是被 QQ 官方认可的机器人方案。该方案可用性由 QQ 官方保证，但目前 **仅对企业用户和部分受邀个人用户开放**，同时在功能上非常受限。

如果有可能，建议迁移到其它平台，在 QQ 平台选择何种方式取决于你自己的选择。

::: danger

倘若出现账号被封禁等情况，海豹官方无力解决此类问题，也不对相应后果负责。

:::

### 对接引导

所有支持的途径参见目录，本节提供了多种对接途径的引导。

从 <Badge type="tip" text="v1.4.5" /> 开始，我们推荐使用 [内置客户端](#内置客户端) 进行连接，这是面向一般用户提供的简单对接方式。

对于需要使用更加灵活的方案的用户，我们推荐如下：

- 需要比较简单的部署流程，希望资源占用低的，见 [Lagrange](#lagrange) 或 [NapCat](#napcatqq)；
- 需要比较简单的部署流程，不是特别在意资源占用的，见 [LLOneBot](#llonebot)；
- 通过 docker 部署海豹的，见 [QQ - Docker 中的海豹](./platform-qq-docker)；
- 如果你有 QQ 官方机器人权限，见 [官方机器人](#官方机器人)；
- Go-cqhttp 与 QSign 方案因可用性原因已被弃用。**我们不建议任何用户再使用此方式部署 QQ 接入，同时强烈建议正在使用该方案的用户迁移**。[之前的资料](/about/archieve.md#Go-cqhttp/Mirai)保留备查。

不同的对接方式适应不同的情况，可能会存在途径特有的功能缺失和其它问题，请根据自己的情况选择适合的方式。

::: warning 注意：对接内置客户端、Lagrange、LLOneBot 和 Napcat 等 PC 端协议时最好关闭 `戳一戳` 功能

由于内置客户端、Lagrange、LLOneBot 和 Napcat **并未完全实现**戳一戳功能。请使用上述客户端连接 QQ 的用户关闭海豹核心后台位于 `综合设置` - `基本设置` 的 `启用戳一戳` 开关，以免产生不必要的报错和麻烦。

<img src="./images/platform-qq-turnoff.png" alt="关闭戳一戳开关" width="80%">

:::

::: warning 注意

内置客户端、Lagrange、LLOneBot 和 Napcat 都占用 PC 端协议。在使用这些连接方式时，不可同时登录 PC 端 QQ，否则将导致挤占下线。

由于官方 QQ 设定，PC 端协议（即以上四种登录方式）每隔 30 天需要重新登录。

:::

## 内置客户端 <Badge type="tip" text="v1.4.5" />

海豹从 <Badge type="tip" text="v1.4.5"/> 开始提供了内置客户端的连接方式。

::: warning

需要知道的是，该方案也是前言中提到的非官方机器人，并不受到 QQ 官方认可。

:::

::: danger 危险：部分过时系统不支持

内置客户端暂不支持 Windows 7，Windows Server 2008，32 位 Windows 也不可用。

Windows Server 2012 可能会缺少部分运行库，需要自行下载安装。

:::

进入海豹 Web UI 的「账号设置」新增连接，选择账号类型「QQ(内置客户端)」，这也是默认选项，填写 QQ 号：

<img src="./images/platform-qq-builtin-1.png" alt="内置客户端" width="80%">

随后使用登录了目标账号的手机 QQ 尽快扫码登录：

<img src="./images/platform-qq-builtin-2.png" alt="内置客户端扫码登录" width="40%">

在手机上确认登录以后，等待状态变为「已连接」即可。

::: info 内置客户端 BUG

由于内置客户端的实现不完全，会有莫名其妙的 bug，所以我们推荐有能力的骰主使用手册中的其他方案。

但如果你仍然决定使用「QQ(内置客户端)」，当遇到无法使用时可以尝试以下解决方案：  

- PC 端：删除 `data/default/extra/lagrange-QQ号` 文件夹，重启海豹，删除账号重新添加。
- 安卓端：停止海豹核心，在右上角设置中将「文件同步模式」打开，返回主界面，点击「导出数据」，到显示的目录删除 `data/default/extra/lagrange-QQ号` 目录，然后点击「导入数据」，删除账号重新添加。

:::

## Lagrange <Badge type="tip" text="v1.4.2" />

海豹从 <Badge type="tip" text="v1.4.2"/> 开始适配了 Lagrange（拉格兰）的连接。

::: info Lagrange

[Lagrange](https://github.com/KonataDev/Lagrange.Core)（拉格兰）是一个 NTQQ 协议相关的开源项目。其包括目前实现了 Linux NTQQ 协议的 Lagrange.Core，和提供 OneBot-V11 Api 的 Lagrange.Onebot 两部分。

与 GoCqhttp 类似，Lagrange 可以很方便的在多个平台（Windows、Linux、Mac）部署，海豹核心可以对接其提供的 OneBot-V11 Api 来提供 QQ 骰子服务。

:::

### 准备 Lagrange

可以在 [Lagrange Github Release](https://github.com/KonataDev/Lagrange.Core/releases) 中获取到 Nightly 版程序，根据你的系统选择相应版本下载，例如：

- Windows 通常选择 `win-x64` 版本；
- Mac（Intel 芯片）选择 `osx-x64` 的版本；
- Mac（Arm 芯片，如 M1、M2、M3 等）选择 `osx-arm64` 的版本；
- ……

![Lagrange Nightly Release](./images/platform-qq-lagrange-release.png)

::: details 补充：使用 Lagrange Action 版本

你还可以选择使用 Lagrange 在 Action 中自动构建的版本，这些版本是 **最新** 的构建。在使用这些版本时，你需要安装对应版本的 .Net SDK。

**除特殊情况外，我们始终建议你选择前面提到的 Nightly 版本。**

可以在 [Lagrange Github 仓库](https://github.com/KonataDev/Lagrange.Core) 中的 Action 页面，进入位于列表最前一条的最新制品页面，根据你的系统选择相应版本。

<img src="./images/platform-qq-lagrange-1.png" alt="Lagrange Action" width="80%">

点击进入页面后拉到最下方，选择相应版本下载。

<img src="./images/platform-qq-lagrange-2.png" alt="Lagrange Action Artifacts" width="40%">

:::

### 运行 Lagrange

解压下载的 Nightly 版的 Lagrange 压缩文件，你可以看见名如 `Lagrange.OneBot.exe` 的应用程序，双击启动即可。启动时有可能会先弹出如下警告，按步骤允许即可：

<img src="./images/platform-qq-lagrange-3.png" alt="Lagrange 运行警告 1" width="40%">

<img src="./images/platform-qq-lagrange-4.png" alt="Lagrange 运行警告 2" width="40%">

成功启动后可以发现打开了如下的命令行窗口，其中提示已创建了一个配置文件：

<img src="./images/platform-qq-lagrange-5.png" alt="Lagrange 启动后提示" width="80%">

可以发现，在程序所在的文件夹中多出了一个 `appsettings.json`，这是 [Lagrange 的配置文件](#lagrange-配置文件)，**你需要打开并修改其中的一些项**。也可以在启动前直接手动新建 `appsettings.json` 并写入内容。

修改后内容大致如下：

`appsettings.json`：

```json{11-12,20-28}
{
  "Logging": {
    "LogLevel": {
      "Default": "Information",
      "Microsoft": "Warning",
      "Microsoft.Hosting.Lifetime": "Information"
    }
  },
  "SignServerUrl": "https://sign.lagrangecore.org/api/sign",
  "Account": {
    "Uin": 0,
    "Password": "",
    "Protocol": "Linux",
    "AutoReconnect": true,
    "GetOptimumServer": true
  },
  "Message": {
    "IgnoreSelf": true
  },
  "Implementations": [
    {
      "Type": "ForwardWebSocket",
      "Host": "127.0.0.1",
      "Port": 8081,
      "HeartBeatInterval": 5000,
      "AccessToken": ""
    }
  ]
}
```

其中有几个重要的设置项需要填写和注意：

- `Password` 为空时为扫码，这里请留空。
- `SignServerUrl`：NTQQ 的签名服务地址，**注意此处的签名服务需要是 Linux NTQQ 签名服务，不可以使用 QSign、Shamrock 等提供的 Android QQ 签名服务**。
- `Implementations`：这是 Lagrange 的连接配置，海豹将使用 `ForwardWebSocket`，即正向 WebSocket 方式连接 Lagrange，该方式下的 `Host` 和 `Port` 是 Lagrange 将提供的 **OneBot-V11 正向 WS 服务地址**，记下以供后续使用。如果对应端口已占用请自行调整。

::: info Linux NTQQ 的签名服务

拉格兰项目提供公共签名服务，运行程序后默认生成的配置文件中已经包含了签名地址。

可访问[拉格兰项目的 GitHub 仓库，在其 README 中](https://github.com/KonataDev/Lagrange.Core?tab=readme-ov-file#signserver)验证其是否正确有效。

:::

::: warning 注意：保证连接模式匹配

Lagrange 默认生成的配置文件生成的是 `ReverseWebSocket`（即反向 WebSocket），如果你使用该种连接方式，下文海豹对接时应该选择「OneBot 11 反向 WS」模式。

海豹推荐使用正向连接，如果你选择正向连接方式，需要像上述示例中的配置文件一样调整为 `ForwardWebSocket`，下文海豹对接时按引导执行即可。

具体的连接细节还可以参见 Lagrange 文档的 [配置文件](https://lagrangedev.github.io/Lagrange.Doc/Lagrange.OneBot/Config/#%E9%85%8D%E7%BD%AE%E6%96%87%E4%BB%B6) 一节。

:::

修改配置完成后的文件夹如下：

<img src="./images/platform-qq-lagrange-6.png" alt="正式运行前的 Lagrange 文件夹" width="40%">

在配置文件按需要正确修改后，在命令行中按任意键，Lagrange 将正式运行。在同一文件夹下会出现一张登录二维码图片 `qr-0.png`，在二维码过期前尽快使用手机 QQ 扫码连接。

::: details 补充：Action 版 Lagrange 与 .Net SDK

Action 中获取的 Lagrange 依赖 .Net SDK，如果你在运行 Lagrange 时出现报错，需要去下载 [.Net SDK](https://dotnet.microsoft.com/zh-cn/download) 并安装。

在下载 Lagrange 时，后缀中的数字说明了其对 .Net 版本的需求，请根据说明下载对应版本（例如后面是 8.0，则需安装 SDK 的版本为 8.0）。

此外，与 Nightly 下载后解压的单文件版本的 Lagrange 不同，在解压 Action 制品压缩文件后，你可以看见 `Lagrange.OneBot.exe` 等多个文件，这些文件都是运行必不可少的。

包括生成的配置文件在内，一个正确的 Action 版 Lagrange 文件夹如下：

<img src="./images/platform-qq-lagrange-7.png" alt="Action 版的 Lagrange 文件夹" width="40%">

:::

### 海豹连接 Lagrange

进入海豹 Web UI 的「账号设置」新增链接，选择账号类型「QQ(onebot11正向WS)」。

账号填写骰子的 QQ 号，连接地址使用上面记下的 WS 正向服务地址 `ws://{Host}:{Port}`，如 `ws://127.0.0.1:8081`。

<img src="./images/platform-qq-lagrange-8.png" alt="海豹连接 Lagrange" width="100%">

成功连接后即可使用。

### Lagrange 配置文件

与可执行文件在同级目录下的 `appsettings.json` 文件，是 Lagrange 的配置文件。

最新的 Lagrange 会在没有该文件时自动创建默认配置，如果没有生成该文件，你可以按照 [官方仓库的最新说明](https://github.com/KonataDev/Lagrange.Core) 手动创建这一文件。

::: warning 注意：使用最新的 Lagrange

**我们始终建议你升级到程序的最新版本，而不是为了沿用旧配置而保持旧版本。**

:::

::: warning 注意：Lagrange 配置文件格式变更

Lagrange 项目对其配置文件的格式进行过更改。如果你是在 2024 年 2 月 18 日或之前下载的 Lagrange 程序，请你参考下面的版本。

目前最新的 Lagrange 可以识别两个版本的配置文件，但依然建议修改为最新格式。

:::

::: details 补充：旧版 Lagrange 配置

如果你使用的是 2024 年 2 月 18 日或之前下载的 Lagrange 程序，或使用前面提到的配置出现问题，请尝试替换为以下配置：

```json
{
  "Logging": {
    "LogLevel": {
      "Default": "Trace",
      "Microsoft": "Warning",
      "Microsoft.Hosting.Lifetime": "Information"
    }
  },
  "SignServerUrl": "",
  "Account": {
    "Uin": 0,
    "Password": "",
    "Protocol": "Linux",
    "AutoReconnect": true,
    "GetOptimumServer": true
  },
  "Message": {
    "IgnoreSelf": true
  },
  "Implementation": {
    "ForwardWebSocket": {
      "Host": "127.0.0.1",
      "Port": 8081,
      "HeartBeatInterval": 5000,
      "AccessToken": ""
    }
  }
}
```

配置项的含义与之前的说明相同，可以做相同处理。

:::

## LLOneBot <Badge type="tip" text="v1.4.2" />

海豹从 <Badge type="tip" text="v1.4.2"/> 版本开始支持通过 OneBot 协议连接 LLOneBot。

::: info LLOneBot

[LiteLoaderQQNT](https://github.com/LiteLoaderQQNT/LiteLoaderQQNT)（LiteLoader）是 NTQQ 的插件加载器，允许通过插件注入 QQ 实现某些特定的功能。

[LLOneBot](https://github.com/LLOneBot/LLOneBot) 则是 Liteloader 的插件之一，可以实现劫持客户端对外开放 API，可以理解为装在 PC 上的 Shamrock。

:::

### 安装 LLOneBot

请参考 [官方文档](https://llonebot.github.io/zh-CN/) 中的说明。

### 配置海豹

安装完成后重新登录 QQ，进入 LLOneBot 的设置页：

![LLOneBot 设置页](./images/platform-qq-llonebot-2.png)

支持两种方式与海豹对接：

- 正向连接：默认开放的正向 ws 端口为 3001，在海豹的新添账号选择「QQ(onebot11正向WS)」，账号处随便填写，连接地址填 `ws://localhost:3001`。
- 反向连接：关闭正向连接开关，打开反向连接，点击「添加」，输入 `ws://127.0.0.1:4001/ws`，在海豹的新添账号选择「QQ(onebot11反向WS)」，输入账号。

::: tip

- 如若想修改端口请在 LLOneBot 的设置 UI 自行修改。
- 请注意设置中的正向连接和反向连接请 **不要同时打开**，否则会发不出消息。
- **如果你是在服务器上部署，可能需要使用 [Mem Reduct](https://memreduct.org/mem-reduct-download/) 之类的工具定时清理过高的内存占用。**

:::

## NapCatQQ

::: info NapCatQQ

[NapCatQQ](https://github.com/NapNeko/NapCatQQ) 是在后台低占用运行的无头（没有界面）的 NTQQ，具体占用会因人而异，QQ 群、好友越多占用越高。

[NapCat 官方文档](https://napneko.github.io/zh-CN/)

**注意同个账号不能同时登录原版 QQ 和 NapCatQQ**。

:::

NapCat 是基于官方 NTQQ 实现的 Bot 框架，因此在开始前，你需要根据 [NapCatQQ](https://napneko.github.io/zh-CN/guide/getting-started#%E5%AE%89%E8%A3%85-qq) 的手册安装官方 QQ，若 QQ 版本过低会导致程序无法正常启动。

### 下载 NapCatQQ

前往 [NapCatQQ Release](https://github.com/NapNeko/NapCatQQ/releases) 页面下载最新版本。

### 启动 NapCatQQ

在启动前，你需要修改 `config/onebot11.json` 内容，并重名为 `onebot11_<你的QQ号>.json` ，如 `onebot11_1234567.json` 。

json 配置内容参数解释：

```json{18-25}
{
  "http": {
    // 是否启用http服务, true为启动，false为禁用
    "enable": false,
    // HTTP服务监听的 ip 地址，为空则监听所有地址
    "host": "",
    // http服务端口
    "port": 3000,
    // http上报密钥，可为空
    "secret": "",
    // 是否启用http心跳
    "enableHeart": false,
    // 是否启用http上报服务
    "enablePost": false,
    // http上报地址, 如["http://127.0.0.1:8080/onebot/v11/http"]
    "postUrls": []
  },
  "ws": {
    // 是否启用正向websocket服务
    "enable": true,
    // 正向websocket服务监听的 ip 地址，为空则监听所有地址
    "host": "",
    // 正向websocket服务端口
    "port": 3001
  },
  "reverseWs": {
    // 是否启用反向websocket服务
    "enable": false,
    // 反向websocket对接的地址, 如["ws://127.0.0.1:8080/onebot/v11/ws"]
    "urls": []
  },
  "GroupLocalTime": {
    "Record": false,//是否开启本地群聊时间记录
    "RecordList": []//开启全部群 ["-1"]  单个群配置 ["11111"] 多个群 ["1","2","3"]
  },
  // 是否开启调试模式，开启后上报消息会携带一个raw字段，为原始消息内容
  "debug": false,
  // ws心跳间隔，单位毫秒
  "heartInterval": 30000,
  // 消息上报格式，array为消息组，string为cq码字符串
  "messagePostFormat": "array",
  // 是否将本地文件转换为URL，如果获取不到url则使用base64字段返回文件内容
  "enableLocalFile2Url": true,
  // 音乐签名URL，用于处理音乐相关请求
  "musicSignUrl": "",
  // 是否上报自己发送的消息
  "reportSelfMessage": false,
  // access_token，可以为空
  "token": ""
}
```

其中有几个重要的设置项需要填写和注意：

- `ws:enable`：这是 NapCat 的 ws 正向连接配置，你需要将其修改为 `true`，即启用正向 WebSocket 方式连接 NapCatQQ。
- `ws:port`：这是正向连接端口，请记下以便后续使用。

也可以使用 WebUI 进行配置，具体请看 [NapCat 手册](https://napneko.github.io/zh-CN/guide/config)。

修改完文件后请根据 [NapCatQQ](https://napneko.github.io/zh-CN/guide/getting-started#%E5%90%AF%E5%8A%A8) 的教程启动程序，扫码登录即可。

### 海豹连接

进入海豹 Web UI 的「账号设置」新增链接，选择账号类型「QQ(onebot11正向WS)」。

账号填写骰子的 QQ 号，连接地址使用上面记下的 ws 正向服务地址 `ws://127.0.0.1:{wsPort}`，如 `ws://127.0.0.1:3001`。

## Chronocat <Badge type="tip" text="v1.4.2" />

海豹从 <Badge type="tip" text="v1.4.2"/> 开始适配了 Chronocat（超时空猫猫）中 Red 协议的连接，从 <Badge type="tip" text="v1.4.5"/> 开始适配了与 Chronocat 的 Satori 协议连接。

::: warning 注意：Chronocat 的新旧版本

[Chronocat](https://chronocat.vercel.app/) 主要分为 0.0.x 的旧版本和 0.2.x 及以上的新版本。

在 0.0.x 的旧版本中，海豹主要对接其 Red 协议，海豹从 <Badge type="tip" text="v1.4.3"/> 开始弃用此协议。  

在目前的新版本中，Chronocat 移除了 Red 协议，提供 Satori 协议的连接支持，使用该版本的见 [Chronocat Satori 协议](#chronocat-satori-协议)。

:::

### Chronocat Satori 协议 <Badge type="tip" text="v1.4.5" />

在账号添加中，选择「[WIP]Satori」，填写相应信息进行连接。

## 官方机器人 <Badge type="tip" text="v1.4.2" />

海豹从 <Badge type="tip" text="v1.4.2"/> 开始支持对接 QQ 官方的机器人服务。

::: tip 提示：QQ 机器人

QQ 官方目前已开放了机器人功能，可进入 [QQ 开放平台](https://q.qq.com/#/) 进行申请。

但截止到目前，**QQ 官方机器人的群聊权限并未对所有人开放**。我们也希望在未来，每个人都能轻松地对接上官方提供的机器人服务。

同时，由于 QQ 官方对机器人能力的严格限制（包括获取 QQ 号、昵称，非 @ 时响应，私聊，群控等大量功能目前不支持），目前**对接官方接口的骰子很多功能无法支持**（如跑团 Log，暗骰，对抗等）。

:::

### 尝试一下

如果你想尝试一下这样的机器人，非常欢迎你使用海豹官方的骰子：

<img src="./images/platform-qq-bot-qrcode.jpg" alt="海豹机器人二维码" width="65%">

### 获取连接所需信息

要获取用于连接海豹的信息之前，你需要建立好一个 QQ 机器人应用。可前往 [QQ 开放平台](https://q.qq.com/#/) 进行申请，进行**实名**注册后，创建一个机器人应用。

创建完成后，进入机器人管理后台，切换到「开发设置」页面：

<img src="./images/platform-qq-official-1.png" alt="切换到开发设置" width="40%">

在页面中你可以看到这样的信息，其中「机器人 ID」「机器人令牌」「机器人密钥」这三项是海豹进行连接所需要的。

![开发设置](./images/platform-qq-official-2.png)

### 连接海豹

登录海豹并添加账号，选择「QQ(官方机器人)」。填写对应的信息点击连接。你的海豹应该可以正常连接官方机器人运作了！

<img src="./images/platform-qq-official-3.png" alt="连接官方 Bot" width="100%">

### 使用海豹

::: warning

目前官方机器人只响应包含 `@` 的消息，操作时请注意。

:::
