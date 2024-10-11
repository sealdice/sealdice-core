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

目前，仅有 [官方机器人服务](#官方机器人) 是被 QQ 官方认可的机器人方案。该方案可用性由 QQ 官方保证，但目前 **仅对企业用户和部分受邀个人用户开放**，同时在功能上非常受限。

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
- 如果你有 QQ 官方机器人权限，见 [官方机器人](#官方机器人)；
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

## 官方机器人 <Badge type="tip" text="v1.4.2" />

海豹从 <Badge type="tip" text="v1.4.2"/> 开始支持对接 QQ 官方的机器人服务。

::: tip 提示：QQ 机器人

QQ 官方目前已开放了机器人功能，可进入 [QQ 开放平台](https://q.qq.com/#/) 进行申请。

搭建机器人官方请参考 [QQ 机器人文档](https://bot.q.qq.com/wiki/#/)。

目前， **QQ 官方机器人已开放个体使用权限。但官方文档没有及时更新**。

同时，由于 QQ 官方对机器人能力的严格限制（包括获取 QQ 号、昵称，非 @ 时响应，私聊，群控、发送本地图片等大量功能目前不支持），目前**对接官方接口的骰子很多功能无法支持**（如跑团 Log ，暗骰，对抗等）。

QQ 官方机器人的优点，就是不用担心被风控。

:::

### 尝试一下

如果你想尝试一下这样的机器人，非常欢迎你使用海豹官方的骰子：

<img src="./images/platform-qq-bot-qrcode.jpg" alt="海豹机器人二维码" width="65%">

### 获取连接所需信息

要获取用于连接海豹的信息之前，你需要建立好一个 QQ 机器人应用。可前往 [QQ 开放平台](https://q.qq.com/#/) 进行申请，**实名**注册后，创建一个机器人应用。

创建完成后，进入机器人管理后台，切换到「开发设置」页面：

<img src="./images/platform-qq-official-1.png" alt="切换到开发设置" width="40%">

在页面中你可以看到这样的信息，其中「机器人 ID 」「机器人令牌」「机器人密钥」这三项是海豹进行连接所需要的。

![开发设置](./images/platform-qq-official-2.png)

然后在「开发设置 IP 白名单」一栏中，填写你骰子所在电脑的 **公网** IP。（使用云服务器时，请填写对应云服务商在控制台提供的 IP）

<img src="./images/platform-qq-official-4.png" alt="连接官方 Bot" width="100%">

::: warning 注意：家庭网络的 IP 变动

如果你使用的是家用网络，在本地电脑运行海豹，要注意家庭网络的 IP 通常是不固定的，运营商随时可能会更换你的 IP，遇到问题时请先检查。

:::

### 连接海豹

登录海豹并添加账号，选择「QQ(官方机器人)」。填写对应的信息点击连接。你的海豹应该可以正常连接官方机器人运作了！

<img src="./images/platform-qq-official-3.png" alt="连接官方 Bot" width="100%" />

### 指令配置

1. 进入「发布配置」页面；
2. 点击「功能配置」；
3. 点击「指令配置」；
4. 再点击右上角的「重新配置」开始编辑；
5. 点开「指令」页添加指令，「指令名」里面输入对应指令（例如 `r`、`ra`、`en`），`/` 是 QQ 官方机器人默认的指令前缀（海豹也支持使用 `/`）；
6. 然后在「指令介绍」一栏填写指令的简介；
7. 然后在「使用场景」一栏点击「QQ 频道」、「频道私信」、「QQ 群」，切记不能选中「消息列表」；
8. 确认配置完成后，扫码确认修改，就可以完成指令配置。

<img src="./images/platform-qq-official-11.png" alt="进入指令配置页" />

<img src="./images/platform-qq-official-12.png" alt="进行指令配置" />

::: details 推荐指令列表模版

<img src="./images/platform-qq-official-7.png" alt="推荐指令列表模版1" width="66%" />

<img src="./images/platform-qq-official-8.png" alt="推荐指令列表模版2" width="66%" />

:::

### 机器人上线

点开「使用范围和人员页面」，进入「编辑」页，参考下面图片中编辑使用范围与人员。

<img src="./images/platform-qq-official-6.png" alt="连接官方 Bot" />

配置完成后，点击「发布配置」页面，上传「自测报告」与「隐私协议」。

::: details 「隐私协议」与「自测报告」模板

这里提供了海豹骰的机器人「隐私协议」与「自测报告」模版。

**强烈建议你按自身情况进行修改，绝对不要原样上传，这涉及到你的机器人是否能被 QQ 官方的工作人员审核通过。**

[机器人自测报告.xlsx](https://github.com/sealdice/sealdice-manual-next/blob/main/assets/%E6%9C%BA%E5%99%A8%E4%BA%BA%E8%87%AA%E6%B5%8B%E6%8A%A5%E5%91%8A.xlsx?raw=true)

[第三方机器人隐私保护指引.docx](https://github.com/sealdice/sealdice-manual-next/blob/main/assets/%E7%AC%AC%E4%B8%89%E6%96%B9%E6%9C%BA%E5%99%A8%E4%BA%BA%E9%9A%90%E7%A7%81%E4%BF%9D%E6%8A%A4%E6%8C%87%E5%BC%95.docx?raw=true)

如果你对「指令列表」的指令进行了修改，或者新增了其他功能，请自行修改，需要在「预期输出」一栏填「指令简介」。

:::

点击「提交审核」后，等待 QQ 官方人员测试并审核（时间不定）。审核通过后，在「发布设置」页面中点击「上线机器人」。

::: tip 提示：关于 QQ 审核

目前 QQ 审核主要是测试机器人能否在所选支持的场景下，正常回应指令列表里的指令（在没有添加任何其他内容前提下），为人工审核。

在提交审核前，请善用沙盒群，测试你提交的自测报告中，所提到的指令是否都能正常工作，机器人需要正常发出回应。

如果你的指令包含一些需要填写的参数，请务必在「自测报告」表格「特殊说明」一栏里补充说明。（你可以参考模板中对 `.ra` `.sc` `.en` 指令的特殊说明。）

如果审核未通过，点击右上角的「通知」查看原因，解决后再次提交。 如果实在无法解决，可加入「QQ 机器人官方频道」，在「寻求｜｜审核和及 bug 」一栏里，发帖询问。

:::

### 使用海豹

点击「使用人员与范围」页面，查看你的机器人对应的邀请二维码，扫码添加即可。

::: warning 注意

目前官方机器人只响应包含 `@` 的消息，操作时请注意。

同时，官方机器人一次只能发一条消息，一次性发送消息太多，官方机器人会因为消息发送过于频繁而报错。

此外，官方机器人目前无法发送本地图片。

:::

### 注意事项

大部分事项 [QQ 机器人文档](https://bot.q.qq.com/wiki) 都有说明，这里补充一些文档中没有说明的其他事项：

#### 企业账号的开发者资质审核

如果你使用企业账号进行了注册，请记得在资料一栏中进行开发者资质状态审核。该审核需要将对应企业的银行卡号上传至腾讯审核，期间的等待时间可能较久。在开发者资质状态通过后，你才能将官方机器人送审。

<img src="./images/platform-qq-official-5.png" alt="开发者资质状态审核" />

#### 机器人官方频道跳转

机器人「资料卡」页面中资料卡设置一栏中，「机器人官方频道跳转」不能是骰子的「沙盒频道」。

<img src="./images/platform-qq-official-9.png" alt="机器人官方频道跳转" />

#### 功能审核未通过

有时候「功能配置」页面中，「功能配置与提审」旁显示审核通过，但实际并没有通过，这时点击「机器人上线」的会显示发送错误。

遇见这种情况请点击「通知」，看机器人哪方面没过审，修改后再次提交审核。
