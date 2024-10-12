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

尽管不同方案之间的差异很大（比如基于 Android QQ 协议的 Go-Cqhttp 已经**基本不可用**，而 [Lagrange](#lagrange) 和 [LLOneBot](#llonebot) 等基于 NTQQ 的方案目前比较稳定），但需要明白的是，这些方案都由社区第三方软件提供，实质上以 QQ 官方角度等同于「**外挂软件**」，并不受到官方支持（甚至是被打击的目标）。

因此，*是否在 QQ 平台搭建这样的非官方机器人取决于你的慎重考虑*。同时，第三方方案的可用性也可能会随时间推移而存在变化，海豹官方无法做出任何保证。

目前，仅有 [官方机器人服务](./platform-qq-official.md) 是被 QQ 官方认可的机器人方案。该方案可用性由 QQ 官方保证，但目前 **仅对企业用户和部分受邀个人用户开放**，同时在功能上非常受限。

如果有可能，建议迁移到其它平台，在 QQ 平台选择何种方式取决于你自己的选择。

::: danger

倘若出现账号被封禁等情况，海豹官方无力解决此类问题，也不对相应后果负责。

:::

### 对接引导

所有支持的途径参见目录，本节提供了多种对接途径的引导。

从 <Badge type="tip" text="v1.4.5" /> 开始，我们推荐使用 [内置客户端](#内置客户端) 进行连接，这是面向一般用户提供的简单对接方式，对于*有能力*的骰主，我们*更推荐*其他的分离部署方案。

对于需要使用更加灵活的方案的用户，我们推荐如下：

- 需要比较简单的部署流程，希望资源占用低的，见 [Lagrange](#lagrange)；
- 需要比较简单的部署流程，不是特别在意资源占用的，见 [LLOneBot](#llonebot)；
- 通过 docker 部署海豹的，见 [QQ - Docker 中的海豹](./platform-qq-docker)；
- 如果你有 QQ 官方机器人权限，见 [官方机器人](./platform-qq-official.md)；
- Go-cqhttp 与 QSign 方案因可用性原因已被弃用。**我们不建议任何用户再使用此方式部署 QQ 接入，同时强烈建议正在使用该方案的用户迁移**。

不同的对接方式适应不同的情况，可能会存在途径特有的功能缺失和其它问题，请根据自己的情况选择适合的方式。

::: warning 注意：对接基于 NTQQ PC 端协议的 QQ 方案时，注意对方是否支持 `戳一戳` 功能

内置客户端/Lagrange、LLOneBot 和 Napcat 等基于 NTQQ PC 的 QQ 方案，在旧版本中由于 NTQQ 旧协议本身不支持的原因，缺失该功能。

请使用：

- 海豹版本 <Badge type="tip" text="v1.4.6" /> 之前的内置客户端
- <Badge type="tip" text="6e350b0" /> 之前的 Lagrange
- <Badge type="tip" text="v3.27.0" /> 之前的 LLOneBot
- <Badge type="tip" text="v1.6.7" /> 之前的 Napcat
- ……

等方案的用户及时更新或**关闭**位于 `综合设置` - `基本设置` 的 `启用戳一戳` 开关，以免产生不必要的报错信息。

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

进入海豹 Web UI 的「账号设置」新增连接，选择账号类型「QQ(内置客户端)」，这也是默认选项，填写 QQ 号，其余内容无需修改：

<img src="./images/platform-qq-builtin-1.png" alt="内置客户端" width="80%">

随后使用登录了目标账号的手机 QQ 尽快扫码登录（二维码会在十秒左右出现，请耐心等待）：

<img src="./images/platform-qq-builtin-2.png" alt="内置客户端扫码登录" width="40%">

在手机上确认登录以后，等待状态变为「已连接」即可。

登录的账号由扫码的账号决定，请不要询问 `为什么登录的是我自己的账号` 之类的问题。

::: warning 安卓端海豹扫码

由于 QQ 的安全策略并不支持图片识别或长按扫描二维码登录，你需要两个手机（一个运行海豹，一个扫码）或下载 TIM 软件扫码登录。

:::

::: warning 内置客户端版本

使用此方案应当尽快更新到 <Badge type="tip" text="v1.4.6"/> 及以上版本的海豹，当遇到登录失败、无法回复等情况请先 `尝试删除账号重新添加`、`在「账号设置」界面切换签名服务` 等方法。

对于 <Badge type="tip" text="v1.4.6"/> 及以上版本的海豹，修改签名时*请勿随意修改签名版本*，除非你知道自己在干什么。

:::

## 分离部署

::: info 分离部署

不同于内置客户端，分离部署为海豹核心和 QQ 登录框架分别启动，然后按照各个框架的连接协议将海豹核心和 QQ 登录框架连接起来。

相比之下分离部署有更强的稳定性，但操作难度也有一定程度的增加。

:::

使用此方法你可能需要对「QQ(onebot11正向WS)」、「QQ(onebot11正向WS)」、「QQ(onebot11正向WS)」、「[WIP]Satori」的区别有一定了解。

「QQ(onebot11正向WS)」遵循 onebot11 标准，由海豹核心主动连接 QQ 登录框架。在 UI 界面添加「连接地址」格式应当为 `ws://{Host}:{Port}`。

「QQ(onebot11反向WS)」遵循 onebot11 标准，由 QQ 登录框架主动连接海豹核心。在 UI 界面添加「连接地址」格式应当为 `{Host}:{Port}`。

「[WIP]Satori」遵循 Satori 标准，由海豹核心主动连接 QQ 登录框架。

### Lagrange <Badge type="tip" text="v1.4.2" />

海豹从 <Badge type="tip" text="v1.4.2"/> 开始适配了 Lagrange（拉格兰）的连接。

::: info Lagrange

[Lagrange](https://github.com/KonataDev/Lagrange.Core)（拉格兰）是一个 NTQQ 协议相关的开源项目。其包括目前实现了 Linux NTQQ 协议的 Lagrange.Core，和提供 OneBot-V11 API 的 Lagrange.Onebot 两部分。

与 GoCqhttp 类似，Lagrange 可以很方便的在多个平台（Windows、Linux、Mac）部署，海豹核心可以对接其提供的 OneBot-V11 API 来提供 QQ 骰子服务。

:::

#### 登录 Lagrange

请按照 [Lagrange 手册](https://lagrangedev.github.io/Lagrange.Doc/Lagrange.OneBot/Config/)自行部署 Lagrange，并按照手册和自己的需求填写配置文件。

#### 海豹连接 Lagrange

进入海豹 Web UI 的「账号设置」新增链接，按照自己的 Lagrange 配置文件选择 onebot11 账号类型，填写 QQ 号和「连接地址」。

成功连接后即可使用。

### LLOneBot <Badge type="tip" text="v1.4.2" />

海豹从 <Badge type="tip" text="v1.4.2"/> 版本开始支持通过 OneBot 协议连接 LLOneBot。

::: info LLOneBot

[LiteLoaderQQNT](https://github.com/LiteLoaderQQNT/LiteLoaderQQNT)（LiteLoader）是 NTQQ 的插件加载器，允许通过插件注入 QQ 实现某些特定的功能。

[LLOneBot](https://github.com/LLOneBot/LLOneBot) 则是 Liteloader 的插件之一，可以实现劫持客户端对外开放 API，可以理解为装在 PC 上的 Shamrock。

:::

::: warning 使用此方案的用户请注意不要随意*更新* QQ 客户端。

由于 QQ 客户端检测机制的变化，更新 QQ 客户端后可能导致方案不可用，并且更新后需要重新安装登录框架，所以不建议用户随意更新 QQ 客户端。

:::

#### 安装 LLOneBot

请参考 [官方文档](https://llonebot.github.io/zh-CN/) 中的说明。

#### 配置 LLOneBot

安装完成后重新登录 QQ，在 QQ 设置中 LLOneBot 的设置页：

![LLOneBot 设置页](./images/platform-qq-llonebot-2.png)

支持两种方式与海豹对接：

- 正向连接：默认开放的正向 ws 端口为 3001，在海豹的新添账号选择「QQ(onebot11正向WS)」，账号处随便填写，连接地址填 `ws://localhost:3001`。
- 反向连接：关闭 LLOneBot 的正向连接开关，打开 LLOneBot 的反向连接开关，在「反向WebSocket监听地址」里点击「添加」，输入 `ws://127.0.0.1:4001/ws`，然后在海豹的新添账号选择「QQ(onebot11反向WS)」，输入账号。

::: tip

- 如若想修改端口请在 LLOneBot 的设置 UI 自行修改。
- 请注意设置中的正向连接和反向连接请 **不要同时打开**，否则会发不出消息。
- **如果你是在服务器上部署，可能需要使用 [Mem Reduct](https://memreduct.org/mem-reduct-download/) 之类的工具定时清理过高的内存占用。**

:::

### NapCatQQ

::: info NapCatQQ

[NapCatQQ](https://github.com/NapNeko/NapCatQQ) 是在后台低占用运行的无头（没有界面）的 NTQQ，具体占用会因人而异，QQ 群、好友越多占用越高。

[NapCat 官方文档](https://napneko.github.io/zh-CN/)

:::

::: warning 使用此方案的用户请注意不要随意*更新* QQ 客户端。

由于 QQ 客户端检测机制的变化，更新 QQ 客户端后可能导致方案不可用，并且更新后需要重新安装登录框架，所以不建议用户随意更新 QQ 客户端。

:::

NapCat 是基于官方 NTQQ 实现的 Bot 框架，因此在开始前，你需要根据 [NapCatQQ](https://napneko.github.io/zh-CN/guide/getting-started#%E5%AE%89%E8%A3%85-qq) 的手册安装官方 QQ，若 QQ 版本过低会导致程序无法正常启动。

#### 下载 NapCatQQ

请按照 [NapCat 官方手册](https://napneko.github.io/zh-CN/guide/getting-started)下载安装，如果你不确定自己可以完全理解 NapCat 官方手册并操作，请不要安装 9.9.12 版本 QQ。

然后按照[基础配置](https://napneko.github.io/zh-CN/guide/config/basic)和自己的需求修改配置文件。

#### 海豹连接

进入海豹 Web UI 的「账号设置」新增链接，按照自己的配置文件选择 onebot11 账号类型，填写 QQ 号和「连接地址」。

成功连接后即可使用。

### Chronocat <Badge type="tip" text="v1.4.2" />

从 <Badge type="tip" text="v1.4.5"/> 开始适配了与 Chronocat 的 Satori 协议连接。

#### 安装 Chronocat

请按照 [官方手册](https://chronocat.vercel.app/guide/install/shell) 安装 Chronocat。

#### Chronocat Satori 协议 <Badge type="tip" text="v1.4.5" />

在账号添加中，选择「[WIP]Satori」，填写相应信息进行连接。
