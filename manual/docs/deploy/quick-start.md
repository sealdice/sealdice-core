---
lang: zh-cn
title: 快速开始
---

# 快速开始

::: info 本节内容

本节将会指导你如何搭建海豹核心，并在本地或远程服务器部署。

:::

## 获取海豹

可以从以下地方获取海豹的安装包：

- 官网：从 [官网下载页](https://dice.weizaima.com/download) 获取最新的正式版海豹核心安装包。
- Github：从 [Github Release](https://github.com/sealdice/sealdice-build/releases) 获取海豹核心安装包，此处提供正式版和每日构建版的发布。

::: tip 我该选择从哪里获取？
我们非常建议你使用从 **官网** 获取的正式版海豹，对于绝大多数用户来说，官网的下载最顺畅和便捷，所提供的正式版也适合绝大多数用户使用。
:::

海豹提供了多个平台的安装包，请确认你的操作系统并选择对应版本的安装包。提供的平台如下：

- Windows：普通用户首选，适用于 Windows 7 或者更高，同样可以部署在 Windows 云服务器。
  - 64位（推荐）
  - 32位
- Linux: 提供给更专业的用户使用，需要使用终端启动。适用于服务器、树莓派等设备。
  - x64
  - arm64：树莓派等 arm 设备请使用 arm64 版。
- Mac OS：提供给 Mac 用户使用，需要使用终端启动。
  - arm64：Apple Silicon 芯片（M1、M2等）请使用 arm64 版。
  - x64：Intel 芯片请使用 x64 版。
- Android：提供 Android 的 apk 安装包。

## 启动

将安装包解压到合适的目录。根据系统不同，用以下方法即可启动海豹：

::: tabs#os

@tab Windows#windows

双击运行 `sealdice-core.exe`。数秒钟后，海豹核心将会在后台运行并弹出提示。

在任务栏中找到海豹图标，点击即可打开后台（WebUI）。

@tab Linux#linux

在 `sealdice-core` 所在的目录启动终端，输入以下命令：

```bash
chmod 755 ./sealdice-core
```
给予其足够的运行权限。随后，运行 `./sealdice-core` 来启动海豹。在浏览器中输入 `localhost:3211` 来访问后台（WebUI）。

@tab MacOS#macos

在 `sealdice-core` 所在的目录启动终端，输入以下命令：

```zsh
xattr -rd com.apple.quarantine ./sealdice-core
chmod 755 ./sealdice-core
```

这两条命令移除海豹核心程序的 [隔离属性](https://zhuanlan.zhihu.com/p/611471192)，并给予其足够的运行权限。随后，运行 `./sealdice-core` 来启动海豹。

在浏览器中输入 `localhost:3211` 来访问后台（WebUI）。

@tab Android#android

Android 用户请使用手机海豹。

:::

## 添加至系统服务（可选）

如果你使用远程Linux服务器部署，那么随着SSH终端关闭，通常海豹也会一块关闭。

SealDice提供了一种自动安装为系统服务(systemd服务项)的功能，可以免去手动配置：

```shell
./sealdice-core -i
正在安装系统服务，安装完成后，SealDice将自动随系统启动
安装完成，正在启动……
```

卸载：

```
./sealdice-core --uninstall
正在卸载系统服务……
系统服务已删除
```

服务名称和启动服务使用的用户名可以自定义，输入./sealdice-core -h自行查看

安装完成后，可以使用systemctl来管理服务:

```shell
systemctl status sealdice
systemctl start sealdice
systemctl stop sealdice
journalctl -xe -u sealdice.service // 查看日志
```

更多用法参见`systemd`文档

P.S. 如果你有自己的onebot服务端，请将其设为WS主动连接模式，手动修改`data/default/serve.yaml`即可进行连接。
P.S. 如果你有自己的onebot服务端，请将其设为WS主动连接模式，手动修改`data/default/serve.yaml`即可进行连接。