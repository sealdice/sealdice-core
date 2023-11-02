---
lang: zh-cn
title: QQ
---

# QQ

::: info 本节内容

本节将包含你在 QQ 平台搭建海豹核心需要了解的特定内容。

:::

::: danger 不被 QQ 官方欢迎的机器人功能

直至目前，群聊中的 QQ 机器人普遍采用「**假用户**」方式，即通过第三方软件接入注册的另一个 QQ 。**QQ 官方并未提供正规的群聊机器人支持，并且一直在对第三方实现进行技术与非技术层面的多重打击。**

从目前的表现看来，QQ 官方会对账号行为进行检测，来区分出账号是否是正常用户（如不正常的登录方式，以不合理的速度在多地区登录等等）。我们无法得知具体的检测细节，但已证实的是，当 QQ 账号用作机器人并被检测到时，该 QQ 会视为风险账号，被官方予以警告，封禁，甚至 **永久冻结** 的惩罚。

因此，*是否在 QQ 平台搭建骰子取决于你的慎重考虑*，海豹官方无法做出任何保证。倘若出现账号被封禁等情况时，海豹官方无力解决此类问题，也不对相应后果负责。

来自官方的围追堵截让 QQ 平台的部署门槛不断拉高。目前的军备竞赛下，如此复杂的部署方式是**现状下几乎唯一的选择**。对此我们也十分无奈，只能希望并期待 QQ 官方提供正式的群聊机器人支持，让合理的需求得到合法的解。

:::

## 使用签名服务

::: danger qsing 已停止维护

原 qsign 作者已因「不可抗力」无法再维护此项目，对应原代码仓库也已删除，该方法会在未来逐渐失效，请做好预期准备。

:::

部署签名服务，即使用开源签名服务 [qsign](https://github.com/fuqiuluo/unidbg-fetch-qsign)，是目前用来绕过检测的最有效手段。

### 怎么使用签名服务？

你可以自己在本地搭一个 qsign 服务，也可以使用别人搭好的。

::: warning 自行搭建签名服务

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

![海豹的qq登录页](./images/platform-qq-qsign-1.png =65%x65%) 

点击下面的「签名服务」一栏的「简易配置」，可以看到如下配置项：

![配置签名服务](./images/platform-qq-qsign-2.png =65%x65%)

- 服务 url：你要链接的 qsign url
- 服务 key：密码
- 服务鉴权：默认为空，如果有的服务器要求特定的鉴权，就填上吧

::: note 默认的 qsign 配置

没有特殊设置的话，qsign 的 url 通常默认为 `http://localhost:13579`，key 通常默认为 `114514`。

:::

### 如何搭建签名服务

::: tip 有能力的用户可以自行搭建服务。
:::

::: tabs#os

@tab Windows#windows

可以尝试使用 [一键qsign](https://github.com/rhwong/unidbg-fetch-qsign-onekey)。

引用自说明：
> 点开以后删掉文件夹里的 `go-cqhttp.bat` 及 `go-cqhttp_windows_386.exe`，然后运行里面的 `Start_Qsign.bat` 启动qsign，按照提示依次键入 `txlib_version` 参数、设定 `host`、`port`、`key`的值。（如果不知道这些是干什么的，请直接依次按下 Enter）

@tab Linux#linux

参阅 qsign 提供的完整教程，看 [这里](https://github.com/fuqiuluo/unidbg-fetch-qsign/wiki/%E9%83%A8%E7%BD%B2%E5%9C%A8Linux)。

@tab MacOS#macos

可以尝试使用 [AutoQSignForMac](https://github.com/Verplitic/AutoQSignForMac)。

在终端运行 `start.sh` 即可配置和启动签名服务器。如果提示 `zsh: access denied`，需要先运行 `chmod -x start.sh` 来给予权限。

初次启动会选择 txlib 版本，及运行 QSign 的主机、端口和 API Key。通常情况下，可以回车跳过而使用默认配置。

:::
