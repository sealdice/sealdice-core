---
lang: zh-cn
title: 快速开始
---

# 快速开始

::: info 本节内容

本节将会指导你如何在你的 PC、服务器、甚至安卓手机上搭建和部署海豹核心。

如果你对计算机、服务器等相关知识没有任何了解，或者在阅读本节时，对内容中的一些名词完全不理解，我们非常建议你先自行学习，对基本计算机知识有所了解之后，再阅读本节进行部署。

:::

## 获取海豹

可以从以下两个渠道获取海豹核心安装包：

- 官网：从 [官网下载页](https://dice.weizaima.com/download) 获取最新的正式版海豹核心安装包。

- GitHub：从 [GitHub Release](https://github.com/sealdice/sealdice-build/releases) 获取海豹核心安装包。

  这一渠道提供两个版本：以 `版本号+发布日期` 命名的正式版，与官网一致；以 `Latest Dev Build+日期` 命名的最新构建，可能有各种 Bug，不推荐一般用户使用。

::: tip 提示：我该选择从哪里获取？

我们非常建议你使用从**官网**获取的正式版海豹。对于绝大多数用户来说，官网的下载最顺畅和便捷，所提供的正式版也适合绝大多数用户使用。

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
- Docker：提供对应 Docker 镜像，支持多种架构。
  - amd64
  - arm64
- Android：提供 Android 的 apk 安装包。

## 启动

::: warning

**永远**不要直接运行压缩包中的海豹核心，也不要在临时目录中运行海豹核心。

:::

将安装包解压到合适的目录。考虑到后续升级，将目录名中的版本号删去或许是更好的做法。

根据系统不同，用以下方法即可启动海豹：

:::: tabs key:os

== Windows

双击运行 `sealdice-core.exe`。数秒钟后，海豹核心将会在后台运行并弹出提示。

在任务栏中找到海豹图标，点击即可打开后台（WebUI）。

== Linux

在 `sealdice-core` 所在的目录启动终端，输入以下命令：

```bash
chmod +x ./sealdice-core
```

给予其足够的运行权限。随后，运行 `./sealdice-core` 来启动海豹。在浏览器中输入 `localhost:3211` 来访问后台（WebUI）。

== MacOS

在 `sealdice-core` 所在的目录启动终端，输入以下命令：

```zsh
chmod +x ./sealdice-core && xattr -rd com.apple.quarantine ./sealdice-core
chmod +x ./lagrange/Lagrange.OneBot && xattr -rd com.apple.quarantine ./lagrange/Lagrange.OneBot
```

这两条命令移除海豹核心程序和 Lagrange 的 [隔离属性](https://zhuanlan.zhihu.com/p/611471192)，并给予其足够的运行权限。

随后，运行 `./sealdice-core` 来启动海豹。在浏览器中输入 `localhost:3211` 来访问后台（WebUI）。

::: info MacOS 启动海豹失败问题排查

1. 启动时若出现 `Bad CPU type in executable`，请确认你是否下载的是正确版本的海豹。**Intel 芯片的 Mac 请下载 `darwin x64` ，Apple Silicon 芯片请下载 `darwin arm64`。**
2. 请确认 MacOS 版本高于 10.12，低版本 MacOS 不支持运行，建议尽量将 MacOS 更新至最新稳定版本。

:::

== Docker

海豹提供了官方的 Docker 镜像，支持 amd64 和 arm64 两种架构。你可以在 [此处](https://github.com/sealdice/sealdice-build/pkgs/container/sealdice) 找到各个版本的镜像。

标签为 edge 的镜像与名为 `Latest Dev Build` 的最新构建二进制发布内容一致。

标签为 beta 的镜像与名为 `Latest Beta Build` 的最新构建二进制发布内容一致。

参考以下命令运行镜像。你可能需要自行通过 `-v` 选项来指定目录挂载、修改 `-p` 调整端口暴露等：

```bash
docker run -d --name sealdice -p 3211:3211 ghcr.io/sealdice/sealdice:edge
```

在挂载目录时请注意：海豹核心在容器中的工作目录是根目录，对应的数据目录路径是：`/data` 和 `/backups`。参考 [海豹的本地文件](./about_file.md)。

如果你需要访问宿主机上监听 localhost 的服务（通常 QQ 连接服务和代理服务的默认配置皆是如此），你需要指定 `--network host` 而不是 `-p`，使容器和宿主机位于同一网络中。

::: warning 注意：容器模式下功能受限

Docker 部署的海豹功能有所限制，如无法使用内置客户端登录、无法在线更新等。

:::

== Android

请查看[安卓端登录](./android.md)。

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

## 安装为系统服务（可选）

如果你使用远程 Linux 服务器部署，或者使用终端启动海豹核心，那么，随着终端关闭，通常海豹核心也会一同关闭。

海豹核心提供了一种自动安装为系统服务（systemd 服务项）的功能，可以免去手动配置。

服务名称和服务的启动用户均可以通过提供命令行参数自定义，请运行 `./sealdice-core -h` 查阅详细说明。

默认情况下，海豹核心将尝试安装一个由 `root` 用户运行的，名为 `sealdice.service` 的服务。这要求海豹核心拥有 root 权限（通常由 sudo 运行得到）。

::: tabs key:os

== Linux

```bash
$ ./sealdice-core -i
正在安装系统服务，安装完成后，SealDice将自动随系统启动
安装完成，正在启动……
```

:::

也提供自动卸载：

::: tabs key:os

== Linux

```bash
$ ./sealdice-core --uninstall
正在卸载系统服务……
系统服务已删除
```

:::

安装完成后，可以使用 systemctl 来管理服务：

::: tabs key:os

== Linux

```shell
systemctl status sealdice   # 查看运行状态
systemctl start sealdice    # 启动海豹核心
systemctl restart sealdice  # 启动或重启海豹核心
systemctl stop sealdice     # 停止海豹核心
journalctl -xe -u sealdice.service # 查看日志
```

以上命令的详细用法，请查阅你系统的 `systemd` 文档。

:::

## 更新海豹

当有新版本海豹核心时，你可以从 WebUI 或 `.bot` 回复语中看见新版本提示。

目前海豹有三种更新方法，用户可以自行选择自己喜欢的方式，对于安卓端用户请看[安卓豹更新](###安卓豹更新)。

请骰主进行更新时确保自己可以接触海豹后台，以免更新失败。同时更新前请做好备份，以免数据丢失。

### 自动更新

若有新版本，后台（WebUI）的「主页」会显示一个较为显眼的更新按钮，可以直接点击按钮更新。

你还可以使用命令更新：执行 `.master checkupdate`，此指令需要 Master 权限，具体请看 [Master 管理](../config/basic.md#master-管理)。

如果你采用了任何自动拉起进程的手段，包括但不限于 Linux 系统的 systemd 等，**切勿**使用自动更新。请稳妥地停止进程后进行手动替换更新。

### 手动更新

从海豹官网下载全新的安装包，解压后请勿运行，直接覆盖替换旧版本海豹，然后启动海豹即可。

### 上传固件 <Badge type="tip" text="v1.4.0"/>

从 <Badge type="tip" text="v1.4.0"/> 起，海豹支持后台固件升级功能，你可以在「综合设置」-「基本设置」-「海豹」找到这项功能。

你可以使用指定的压缩包对当前海豹进行覆盖，上传完成后会自动重启海豹。

### 安卓豹更新

安卓端海豹核心无法使用以上方法进行更新，你可以直接下载新版本海豹进行安装，会自动替换旧版本海豹核心。

更新前请「导出数据」以备份数据。
