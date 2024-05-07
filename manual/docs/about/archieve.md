---
lang: zh-cn
title: 归档
---

# 归档

::: info 本节内容

本节将记录海豹曾经使用但新版本已弃用的功能。

:::

## Go-cqhttp / Mirai

::: warning 注意：海豹已于 <Badge type="warning" text="v1.4.5"/> 版本弃用此方案，以下内容归档于 2024 年 5 月 7 日。

:::

::: danger 危险：此方案已经接近不可用

由于 QQ 官方的检测，使用 Go-cqhttp 方案成功连接的成功率已经越来越低。即使成功连接，也可能面临高达每月 2 次的频繁冻结等情况。

我们不推荐任何用户再使用此方案连接 QQ 平台。

:::

::: danger 危险：Go-cqhttp 已停止维护

Go-cqhttp 的开发者已无力维护项目（见 [go-cqhttp/issue#2471](https://github.com/Mrs4s/go-cqhttp/issues/2471)）。在未来 qsign 签名服务彻底被官方封死之后，Go-cqhttp 将无法继续使用。

:::

### 使用签名服务

::: danger 危险：qsign 已停止维护

原 qsign 作者已因「不可抗力」无法再维护此项目，对应原代码仓库也已删除，该方法会在未来逐渐失效，请做好预期准备。

:::

部署签名服务，即使用开源签名服务 [qsign](https://github.com/fuqiuluo/unidbg-fetch-qsign)，是目前用来绕过检测的最有效手段。

#### 怎么使用签名服务？

你可以自己在本地搭一个 qsign 服务，也可以使用别人搭好的。

::: warning 注意：自行搭建签名服务

如果你的动手能力足够强或者有足够的电脑知识，**强烈推荐** 自己搭建本地签名服务器。

使用他人的签名服务可能会泄漏以下信息 *（截取自 qsign 的说明）*：

> - 登录账号
> - 登录时间
> - 登录后发送的消息内容
> - 登录后发送消息的群号/好友 ID

不会泄露的信息：

> - 账号密码
> - 账号 `session`
> - 群列表/好友列表
> - 接收的消息
> - 除发送消息外的任何历史记录

使用共享签名服务可能会提高账号被封禁的概率。

:::

在登录账号的时候会看到这样一个界面：

<img src="./images/platform-qq-qsign-1.png" alt="海豹的 qq 登录页" width="65%">

点击下面的「签名服务」一栏的「简易配置」，可以看到如下配置项：

<img src="./images/platform-qq-qsign-2.png" alt="配置签名服务" width="65%">

- 服务 url：你要链接的 qsign url
- 服务 key：密码
- 服务鉴权：默认为空，如果有的服务器要求特定的鉴权，就填上吧

::: tip 提示：默认的 qsign 配置

没有特殊设置的话，qsign 的 url 通常默认为 `http://localhost:13579`，key 通常默认为 `114514`。

:::

#### 如何搭建签名服务

::: tip 提示：有能力的用户可以自行搭建服务。
:::

::: warning

由于项目的特殊性，下面的方法随时可能失效，我们不对信息的及时性做任何保证。可以加入海豹官方群寻求帮助。

:::

::: tabs key:os

== Windows

可以尝试使用 [一键 qsign](https://github.com/rhwong/unidbg-fetch-qsign-onekey)。

引用自说明：
> 点开以后删掉文件夹里的 `go-cqhttp.bat` 及 `go-cqhttp_windows_386.exe`，然后运行里面的 `Start_Qsign.bat` 启动 qsign，按照提示依次键入 `txlib_version` 参数、设定 `host`、`port`、`key`的值。（如果不知道这些是干什么的，请直接依次按下 Enter）

== Linux

参阅 qsign 提供的完整教程，看 [这里](https://github.com/fuqiuluo/unidbg-fetch-qsign/wiki/%E9%83%A8%E7%BD%B2%E5%9C%A8Linux)。

== MacOS

可以尝试使用 [AutoQSignForMac](https://github.com/Verplitic/AutoQSignForMac)。

在终端运行 `start.sh` 即可配置和启动签名服务器。如果提示 `zsh: access denied`，需要先运行 `chmod -x start.sh` 来给予权限。

初次启动会选择 txlib 版本，及运行 QSign 的主机、端口和 API Key。通常情况下，可以回车跳过而使用默认配置。

:::

### 手动抓取 ticket

目前，Gocq 的过验证码网站已经停止服务，你需要自行抓取 ticket 进行登录。

步骤如下：

#### 前置工作

启动海豹，打开海豹的管理 ui，账号设置，添加账号；

账号类型选 QQ 账号，设备选「Android Pad-可共存」（此协议登录手机可同时在线，qsign 仅 Android 协议和 Android Pad 协议可用）：

![添加账号](./images/select-account.png)

版本选择 8.9.70（如果你的 qsign 是其它版本，请选择对应版本）：

![选择 qsign 版本](./images/qsign-version.png)

填写 QQ 账号密码：

![填写 QQ 账号密码](./images/qq-account-and-pass.png)

选择简易配置：

![选择简易配置](./images/qsign-select.png)

服务 url 填你的 qsign 服务地址。

服务 key 填你的 qsign 服务密码，没有可以不填。

服务鉴权不填写。

![服务鉴权](./images/qsign-config.png)

接着点击登录，然后退出海豹（结束进程）。

#### 分离部署登录流程

1. 将 `go-cqhttp\go-cqhttp.exe` 文件复制到 `海豹目录/data/default/extra/gocqQQ号(你登录骰娘的qq号)` 这个文件夹下。

   ![文件夹结构](./images/gocq-folder.png)

2. 双击运行 `go-cqhttp.exe`，两次确认后出现 `go-cqhttp.bat` 文件。

   ![运行警告 1](./images/gocq-warn1.png) ![运行警告 2](./images/gocq-warn2.png)

3. 双击运行 `go-cqhttp.bat`，出现以下消息后输入 `2`，回车，复制链接到浏览器（终端选中后右键即可复制粘贴，没有选项）。

   ![运行](./images/gocq-step1.png)

   ::: tip 提示：出现 `open image cache db failed`

   出现该报错的原因很可能是因为 gocq 的缓存数据库损坏，可以尝试删除 gocq 的 data 目录后重新运行 gocq。（注意是 gocq 的 data 而不是海豹的！）

   :::

4. 按照 [手动抓取 ticket 教程 - 哔哩哔哩](https://b23.tv/GRGg4GR) 视频教程操作，成功滑条后（需要抓 ticket，不只是滑条）复制 ticket 到终端后回车。

   ![滑块 ticket 输入](./images/gocq-step2.png)

5. 如果登录成功，你应当能看到一条类似于 `2022-05-06 20:00:00 [INFO] 登录成功，欢迎使用：XXX` 的日志。

   ![登录成功](./images/gocq-success.png)

   同时你应当在下方看到一条类似于 `2022-05-06 20:00:00 [INFO] CQ Websocket 服务器已启动：[::]:8080` 的日志。

   结尾的 `8080` 即为 gocq 的 ws 端口。你的端口号可能不同，总之请记住这个端口号。

6. 打开海豹，删除之前添加的账号，然后重新添加账号，选择 QQ(OneBot11 分离部署)

   ![添加账号](./images/onebot11.png)

   在连接地址中填写 `ws://localhost:8080`（请把`8080`替换为你的 gocq 端口号）。填写完成后点击下一步。
7. 你的账号应当已经成功连接。

   ![连接成功](./images/onebot11-success.png)

### GoCqhttp FAQ

#### 1. 出现 Code 1

1. 账号密码错误，输入正确的账号密码。

#### 2. QSign 闪退

1. 确认没有启动多个 qsign（多个 qsign 需要端口不重复）；
2. 确认端口没有重复。若重复，请重新配置 qsign，修改 port；
3. 将 qsign 文件夹放到硬盘根目录试试（如 D 盘、C 盘）。

#### 3. 出现 Code 45

1. 没连上 qsign，重启 qsign，重新登录；
2. QSign 协议版本和 gocq 协议版本没对应，切换对应协议后重新登录；
3. 该版本 qsign 已失效，升级版本。

#### 4. 出现 Code 235

1. Ticket 复制错误（多复制了 `""` 或少复制了内容），重新登录；
2. 更换网络进行滑条，如电脑连手机热点，复制链接发给手机滑条，换台电脑等；
3. 登录频繁，过段时间重新尝试登录（时间不确定）。

#### 5. 出现 Code 237

1. 登录过于频繁，请等待一段时间继续；
2. 内置的 ticket 抓取工具失效，需要手动抓取；
3. 让他人帮助你滑条。

#### 5. 如何启动多个 qsign？（仅当需要备用签名或不同协议版本的时候有此需求）

解压一个新的 qsign 文件，重新配置，端口需要输入不同于前面的端口。

#### 6. 什么是 go-cqhttp？（通常简称 gocq）

登录 QQ 的程序，现各大骰系都用此程序，此外还有 mirai 等其他程序。

#### 7. 什么是签名？

- 手机 QQ 有内置的签名程序，可以不太准确的简单理解为会生成一些密码发给腾讯，让它识别是不是人；
- qsign 把手机 QQ 的签名程序偷了出来，提供了让我们自己生成密码的功能；
- gocq 在配置后可以对接 qsign，那样骰子也可以证明自己是正常人了。

## Chronocat Red 协议

::: warning 注意：海豹已于 <Badge type="warning" text="v1.4.5"/> 版本弃用此方案，以下内容归档于 2024 年 5 月 7 日。

:::

::: danger 危险：支持 Chronocat 为 0.0.54

海豹 Red 协议所适配的目标 Chronocat 版本为 0.0.54，低于该版本（主要为 0.0.52）缺少必要接口，使用时会出现报错，不建议使用。

:::

在账号添加中，选择「QQ Red 协议」，填写相应信息进行连接。
