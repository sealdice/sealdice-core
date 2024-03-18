---
lang: zh-cn
title: 快速开始
---

# 快速开始

::: info 本节内容

本节将会指导你如何在你的 PC、服务器、甚至安卓手机上搭建和部署海豹核心。

如果你对计算机、服务器等相关知识没有任何了解，或者在阅读本节时，对内容中的一些名词完全不理解，我们非常建议你先行阅读 [计算机相关](./about_pc.md) 与 [开源程序相关](./about_opensource.md) 的科普内容后，再返回这一节。

:::

## 获取海豹

可以从以下两个渠道获取海豹核心安装包：

- 官网：从 [官网下载页](https://dice.weizaima.com/download) 获取最新的正式版海豹核心安装包。

- GitHub：从 [GitHub Release](https://github.com/sealdice/sealdice-build/releases) 获取海豹核心安装包。

  这一渠道提供两个版本：以 `版本号+发布日期` 命名的正式版，与官网一致；以 `Latest Dev Build+日期` 命名的最新构建，可能有各种 Bug，不推荐一般用户使用。

::: tip 我该选择从哪里获取？

我们非常建议你使用从**官网**获取的正式版海豹。对于绝大多数用户来说，官网的下载最顺畅和便捷，所提供的正式版也适合绝大多数用户使用。

**但是**，如果你使用的是**中国移动**运营的网络，由于运营商的限制政策，你可能无法从官网下载。可以尝试使用其他运营商的网络。

:::

海豹提供了多个平台的安装包，请确认你的操作系统并选择对应版本的安装包。提供的平台如下：

- Windows：普通用户首选，适用于 Windows 7 或者更高，同样可以部署在 Windows 云服务器。
  - 64 位：适合绝大多数用户使用。
  - 32 位：只在你明确知道为何要使用 32 位版本的情况下使用 32 位版本。
- Linux: 提供给更专业的用户使用，需要使用终端启动。适用于服务器、树莓派等设备。
  - x64：绝大多数使用 Intel 或 AMD CPU 的服务器都应使用此版本。
  - arm64：树莓派等 arm 设备请使用 arm64 版。
- MacOS：提供给 Mac 用户使用，需要使用终端启动。
  - arm64：Apple Silicon 芯片（M1、M2 等）请使用 arm64 版。
  - x64：Intel 芯片请使用 x64 版。
- Android：提供 Android 的 apk 安装包。

## 启动

::: warning

**永远**不要直接运行压缩包中的海豹核心，也不要在临时目录中运行海豹核心。

:::

将安装包解压到合适的目录。考虑到后续升级，将目录名中的版本号删去或许是更好的做法。

根据系统不同，用以下方法即可启动海豹：

:::: tabs#os

@tab Windows#windows

双击运行 `sealdice-core.exe`。数秒钟后，海豹核心将会在后台运行并弹出提示。

在任务栏中找到海豹图标，点击即可打开后台（WebUI）。

@tab Linux#linux

在 `sealdice-core` 所在的目录启动终端，输入以下命令：

```bash
chmod +x ./sealdice-core
```

给予其足够的运行权限。随后，运行 `./sealdice-core` 来启动海豹。在浏览器中输入 `localhost:3211` 来访问后台（WebUI）。

@tab MacOS#macos

在 `sealdice-core` 所在的目录启动终端，输入以下命令：

```zsh
xattr -rd com.apple.quarantine ./sealdice-core
chmod +x ./sealdice-core
```

这两条命令移除海豹核心程序的 [隔离属性](https://zhuanlan.zhihu.com/p/611471192)，并给予其足够的运行权限。

随后，运行 `./sealdice-core` 来启动海豹。在浏览器中输入 `localhost:3211` 来访问后台（WebUI）。

::: info MacOS 启动海豹失败问题排查

1. 启动时若出现 `Bad CPU type in executable`，请确认你是否下载的是正确版本的海豹。**Intel 芯片的 Mac 请下载 `darwin x64` ，Apple Silicon 芯片请下载 `darwin arm64`。**
2. 请确认 MacOS 版本高于 10.12，低版本 MacOS 不支持运行，建议尽量将 MacOS 更新至最新稳定版本。

:::

@tab Android#android

安装下载的 APK 包，给予适当的权限。为了保证海豹核心在手机上稳定运行，可以采取以下措施：

1. 允许海豹核心自启动；
2. 关闭海豹核心的省电模式、自动清理、后台禁止联网等；
3. 确保手机的省电模式保持关闭。

::::

## 连接平台

在完成上面的步骤后，你已经成功启动了海豹核心。接下来请根据你所需要对接平台的接入手册，来连接海豹和对应平台。

见「连接平台」一章，包括：

- [QQ](./platform-qq.md)
- [KOOK](./platform-kook.md)
- [DoDo](./platform-dodo.md)
- [Discord](./platform-discord.md)
- [Telegram](./platform-telegram.md)
- [Slack](./platform-slack.md)
- [钉钉](./platform-dingtalk.md)
- ……

## 安装为系统服务（可选）

如果你使用远程 Linux 服务器部署，或者使用终端启动海豹核心，那么，随着终端关闭，通常海豹核心也会一同关闭。

海豹核心提供了一种自动安装为系统服务（systemd 服务项）的功能，可以免去手动配置。

服务名称和服务的启动用户均可以通过提供命令行参数自定义，请运行 `./sealdice-core -h` 查阅详细说明。

默认情况下，海豹核心将尝试安装一个由 `root` 用户运行的，名为 `sealdice.service` 的服务。这要求海豹核心拥有 root 权限（通常由 sudo 运行得到）。

::: tabs#os

@tab Linux#linux

```bash
$ ./sealdice-core -i
正在安装系统服务，安装完成后，SealDice将自动随系统启动
安装完成，正在启动……
```

:::

也提供自动卸载：

::: tabs#os

@tab Linux#linux

```bash
$ ./sealdice-core --uninstall
正在卸载系统服务……
系统服务已删除
```

:::

安装完成后，可以使用 systemctl 来管理服务：

::: tabs#os

@tab Linux#linux

```shell
systemctl status sealdice   # 查看运行状态
systemctl start sealdice    # 启动海豹核心
systemctl restart sealdice  # 启动或重启海豹核心
systemctl stop sealdice     # 停止海豹核心
journalctl -xe -u sealdice.service # 查看日志
```

以上命令的详细用法，请查阅你系统的 `systemd` 文档。

:::
