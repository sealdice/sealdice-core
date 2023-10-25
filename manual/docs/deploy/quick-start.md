---
lang: zh-cn
title: 快速开始
---

# 快速开始

::: info 本节内容

本节将会指导你如何在你的 PC、服务器、甚至安卓手机上搭建和部署海豹核心。

:::

## 获取海豹

可以从以下两个渠道获取海豹核心安装包：

- 官网：从 [官网下载页](https://dice.weizaima.com/download) 获取最新的正式版海豹核心安装包。

- Github：从 [Github Release](https://github.com/sealdice/sealdice-build/releases) 获取海豹核心安装包。

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
  - arm64：Apple Silicon 芯片（M1、M2等）请使用 arm64 版。
  - x64：Intel 芯片请使用 x64 版。
- Android：提供 Android 的 apk 安装包。

## 启动

::: warning

**永远**不要直接运行压缩包中的海豹核心，也不要在临时目录中运行海豹核心。

:::

将安装包解压到合适的目录。考虑到后续升级，将目录名中的版本号删去或许是更好的做法。

根据系统不同，用以下方法即可启动海豹：

::: tabs#os

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

@tab Android#android

安装下载的 APK 包，给予适当的权限。为了保证海豹核心在手机上稳定运行，可以采取以下措施：

1. 允许海豹核心自启动；
2. 关闭海豹核心的省电模式、自动清理、后台禁止联网等；
3. 确保手机的省电模式保持关闭。

:::

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
systemctl status sealdice # 查看运行状态
systemctl start sealdice  # 启动海豹核心
systemctl start sealdice  # 启动或重启海豹核心
systemctl stop sealdice   # 停止海豹核心
journalctl -xe -u sealdice.service # 查看日志
```

以上命令的详细用法，请查阅你系统的 `systemd` 文档。

:::
