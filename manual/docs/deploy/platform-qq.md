---
lang: zh-cn
title: QQ
---

# QQ

::: info 本节内容

本节将包含你在 QQ 平台接入海豹核心需要了解的特定内容。

:::

## 官方机器人服务 <Badge type="tip" text="v1.4.2" vertical="middle" />

::: warning QQ 机器人

QQ 官方目前已开放了机器人功能，可进入 [QQ 开放平台](https://q.qq.com/#/) 进行申请。

但截止到目前，**QQ 官方机器人的群聊权限并未对所有人开放**。我们也希望在未来，每个人都能轻松地对接上官方提供的机器人服务。

同时，由于 QQ 官方对机器人能力的严格限制（包括获取 QQ 号、昵称，非 @ 时响应，私聊，群控等大量功能目前不支持），目前**对接官方接口的骰子很多功能无法支持**（如跑团 Log，暗骰，对抗等）。

:::

海豹从 `v1.4.2` 开始支持对接 QQ 官方的机器人服务。

### 尝试一下

如果你想尝试一下这样的机器人，非常欢迎你使用海豹官方的骰子：

![海豹机器人二维码](./images/platform-qq-bot-qrcode.jpg =65%x65%)

### 获取连接所需信息

要获取用于连接海豹的信息之前，你需要建立好一个 QQ 机器人应用。可前往 [QQ 开放平台](https://q.qq.com/#/) 进行申请，进行**实名**注册后，创建一个机器人应用。

创建完成后，进入机器人管理后台，切换到「开发设置」页面：

![切换到开发设置](./images/platform-qq-official-1.png =40%x40%)

在页面中你可以看到这样的信息，其中「机器人ID」「机器人令牌」「机器人密钥」这三项是海豹进行连接所需要的。

![开发设置](./images/platform-qq-official-2.png)

### 连接海豹

登录海豹并添加账号，选择「QQ(官方bot)」。填写对应的信息点击连接。你的海豹应该可以正常连接官方机器人运作了！

![连接官方Bot](./images/platform-qq-official-3.png =80%x80%)

### 使用海豹

::: warning 注意

目前官方机器人只响应包含 `@` 的消息，操作时请注意。

:::

## Go-cqhttp / Mirai

::: danger 不被 QQ 官方欢迎的第三方机器人

直至目前，绝大部分群聊中的 QQ 机器人采用「**假用户**」方式，即通过第三方软件接入注册的另一个 QQ 。**QQ 官方一直在对第三方实现进行技术与非技术层面的多重打击。**

从目前的表现看来，QQ 官方会对账号行为进行检测，来区分出账号是否是正常用户（如不正常的登录方式，以不合理的速度在多地区登录等等）。我们无法得知具体的检测细节，但已证实的是，当 QQ 账号用作机器人并被检测到时，该 QQ 会视为风险账号，被官方予以警告，封禁，甚至 **永久冻结** 的惩罚。

因此，*是否在 QQ 平台搭建这样的非官方机器人取决于你的慎重考虑*，复杂的部署方式是**现状下几乎唯一的选择**。海豹官方无法做出任何保证。倘若出现账号被封禁等情况，海豹官方无力解决此类问题，也不对相应后果负责。

如果有可能，建议迁移到其它平台，或者使用 [QQ 官方提供的机器人服务](#官方机器人服务)。

:::

### 使用签名服务

::: danger qsign 已停止维护

原 qsign 作者已因「不可抗力」无法再维护此项目，对应原代码仓库也已删除，该方法会在未来逐渐失效，请做好预期准备。

:::

部署签名服务，即使用开源签名服务 [qsign](https://github.com/fuqiuluo/unidbg-fetch-qsign)，是目前用来绕过检测的最有效手段。

#### 怎么使用签名服务？

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

#### 如何搭建签名服务

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

## Sharmrock <Badge type="tip" text="v1.0.7" vertical="middle" />

### 前情提要
:::info
Shamrock 是一个基于 LSPosed/Xposed 实现劫持 QQ 的软件。你可以在手机/模拟器中代替 gocq，只要系统不杀进程那么你的骰娘服务就可以一直进行，遗憾的是由于 root 权限获取变得越来越困难，模拟器实现 shamrock 的效果也不是很可观，因此这种解决方案仅仅适用于供自己使用的骰娘，或者供朋友使用。当然如果你的电脑可以一直运行那么 shamrock 也是一个很不错的选择。
:::
本节内容主要包括使用 shamrock 代替 ***gocq*** 。由于 gocq 已经停止开发，而 shamrock 具有良好生态，因此舍弃 gocq 是个很好的选择。
::: warning
请注意，如果你想使用 shamrock 代替 gocq，请确保你有良好的计算机使用能力。由于模拟器对性能要求较高，包括 ***轻量级服务器***、旧电脑、小主机等配置较低的设备可能无法支持使用 Shamrock。
:::

::: info
本节主要讲解如何使用模拟器实现 shamrock，如果你有一台已经 root 的手机，也可以参考本节内容，本教程不涉及说明如何 root 手机。
:::

### 准备模拟器
本节主要使用[夜神模拟器](https://www.yeshen.com/)教程。
::: warning
确保你的安卓版本在安卓 8 以上，而在安卓 11 以下，最好使用安卓 9。
:::

### 准备面具模块
:::warning
在使用之前，请在模拟器设置中打开 root 选项，也叫**超级用户**，软件中获取的一切权限都给予**通过**，包括 **root 权限**。
:::
- 这里有一个[工具]( https://cowtransfer.com/s/9794ead1113d47)
把它安装到模拟器上。
- 然后启动软件，输入 m 回车,再输入 y 回车，会索取超级用户权限，给予，然后输入 1 回车，再输入 a 回车，输入 1 回车，然后面具就安装到你的模拟器上了。
- 打开面具模块，此时面具会索取超级用户权限，给予，此时你会发现你的超级用户权限那里是灰的，***关闭你的超级用户权限***重新启动你的模拟器。
- ***然后你就会发现你的超级用户模块已经激活。*** 在面具的设置里启动 zygisk 模块，随后你需要重新启动模拟器，使得 zygisk 模块生效。
![zygisk](./images/platform-qq-shamrock-1.png)
***此时你的面具模块安装完成！可以开始下一步任务了。***
### 安装 lsposed 模块
请在这里[下载](https://github.com/LSPosed/LSPosed/releases)。
:::warning
请选择以 zygisk 结尾的下载。
:::
下载完成后，把文件上传到模拟器中。一般情况下，直接把文件拖动到模拟器就可以传文件了，且文件一般在 picture 文件夹中，如果没有请参照你使用的模拟器说明。

- 在你传完文件之后，在最右侧切换到「模块」页后，你可以看到从本地安装的选项。单击你刚刚传到模拟器里的文件，等待安装完成即可，随后你可以在右下角看到重启的恩扭，单机等待重启。
- 安装完成后应该这样。
![lsposed](./images/platform-qq-shamrock-2.png)
**完成！**
### 安装 shamrock 模块
请在[这里](https://github.com/whitechi73/OpenShamrock/actions/workflows/build-apk.yml)下载。直接将 apk 文件托动到模拟器即可下载。***此时你应该做的是将 qq 安装到你的模拟器中，可以访问 im.qq.com 下载***。
- 首先你应该先启动 shamrock，好让你 qq 启动时能够注入 shamrock 库。
- 在通知上面你可以打开 lspose 的主页，在**模块一栏中开启 shamrock 模块**。
![shamrock](./images/platform-qq-shamrock-3.png)
- 选中 shamrock，选中 qq，长嗯 qq 并选择 ***强行停止***。
![shamrock](./images/platform-qq-shamrock-4.png)
- 随后打开 qq，你便能看到 ***加载 shamrock 库成功***的字样，就代表你成功了。
- 打开 shamrock 软件，启用 ws 服务。
![shamrock](./images/platform-qq-shamrock-5.png)
:::warning
此时就代表你的 gocq 搭建完成了，你不需要修改 shamrock 的任何内容，当然如果你懂的话可以去修改。
:::
### 准备开放端口供海豹对接
- 首先请下载 [adb](https://developer.android.google.cn/studio/releases/platform-tools?hl=zh-cn) 解压到电脑中任何可用的位置。
- 随后你需要去找模拟器供 adb 连接的端口，路径示例如下。![shamrock](./images/platform-qq-shamrock-6.png)
  - nox 是模拟器根路径。
  - nox_4 是模拟器的编号，你可以在多开助手中看到你的编号。
  - 选中的文件就是要找的文件，在 vsc 中（或者任何一个文本编辑器）中打开。
![shamrock](./images/platform-qq-shamrock-7.png)
  - guestport 对应5555的 hostport 即为所需 port，对于我的就是 ***62028***，记住这个数字。
- 在你解压的 platform-tools 里打开终端，或者你不熟练的话可以把 platform-tools 加入环境变量在启用终端，也可以在 platform-tools 里新建一个文件，把下面的命令写到文件里面，然后把扩展名改为.bat。
- 在打开的终端中输入命令。
```bash
.\adb connect 127.0.0.1:端口
```
对于我的来说就是。
```bash
.\adb connect 127.0.0.1:62028
```
- 随后。
``` bash
.\adb forward tcp:5800 tcp:5800
```

***大功告成！***

![shamrock](./images/platform-qq-shamrock-8.png)
### 对接海豹程序 <Badge type="tip" text="v1.4.2" vertical="middle" />
:::warning
请使用 1.4.1 以上版本的海豹进行适配，低版本的海豹未提供 ***shamrock 协议适配***，你可以选择到 qq 群里下载 dev 版海豹，也可以选择等待更新。
:::
- 在账号添加中，选择 ***qq 分离部署***按照下面的格式进行填写。
![shamrock](./images/platform-qq-shamrock-9.png)
 ***完成！*** 你可以享受几乎所有的 gocq 功能，现在，你的骰娘可以正常使用啦！
