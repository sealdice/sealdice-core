---
lang: zh-cn
title: Minecraft
---

# Minecraft

::: info 本节内容

本节将包含你在 Minecraft 服务端接入海豹核心需要了解的特定内容。

:::

## Minecraft 支持

海豹核心支持与安装了 `sealdice-minecraft` 插件的 Minecraft 服务器（如 Paper、Purpur）进行对接。

## 架设 Minecraft 服务端

下面以 Windows 平台为例，简单介绍 Minecraft 服务端的架设。

架设 Minecraft 服务器首先需要根据选定对应的 `Java` 版本。

- 服务端版本 `1.0 - 1.11.x` 可以使用 `Java 6` 和 `Java 7`，但推荐使用 `Java 8`。
- 服务端版本 `1.12(17w13a) - 1.16.5(1.17-21w18a)`，需要使用 `Java 8`。
- 服务端版本 `1.17(21w19a) - 1.17.1` 需要使用 `Java 16`。
- 服务端版本 `1.18(1.18-pre2)` 及以上版本需要使用 `Java 17` 及以上。
- 此为一般情况，部分服务端会推荐对应 Java 版本，请按需安装对应 Java 版本。

::: warning

`sealdice-minecraft` 插件需要 `Java 18` 及以上的 Java 版本，如需要在低版本运行请自行验证服务端与 Java 版本是否匹配。

:::

### 获取服务端核心

:::: tabs key:mc-server-core

== Paper

前往 [Paper MC](https://papermc.io) 下载对应版本的服务端核心文件。

== Purpur

前往 [Purpur MC](https://purpurmc.org) 下载对应版本的服务端核心文件。

== Spigot（插件不兼容）

::: danger

`sealdice-minecraft` 插件不兼容 Spigot/Bukkit 服务端。

:::

前往 [Spigot MC](https://www.spigotmc.org) 寻找最新的 BulidTools，选择要构建的版本进行编译，获得服务端核心文件。

![BuildToolsUI](./images/platform-minecraft_1.png)

在 UI 左上角的 `Select Version` 选择版本。

在 `Output Directory(Optional)` 选择编译文件输出位置。

点击右下角的 `Compile` 进行编译。

== Bukkit（插件不兼容）

::: danger

`sealdice-minecraft` 插件不兼容 Spigot/Bukkit 服务端。

:::

在 [Get Bukkit](https://www.getbukkit.org) 直接下载对应版本的服务端核心文件。

::::

### 启动服务端

新建一个空白文件夹，放入服务端核心，编写一个简单的 `bat` 启动脚本（可新建一个 `txt` 文件后，修改文件后缀）。

![Create Bat 1](./images/platform-minecraft_2.jpg)

启动脚本内容如下：

```cmd
@echo
java -Xms2G -Xmx2G -jar spigot-1.20.4.jar nogui
```

- `-Xms2G` `-Xmx2G`：此项设定服务器占用的内存，按照需求更换数字，例如 `1024m`、`6G`。
- `-jar` 与 `nogui` 之间的 `spigot-1.20.4.jar` 更改为文件夹内对应的服务端核心文件文件名。

退出并保存文件，将该文件后缀改为 `.bat`。修改后文件夹如下所示：

![Create Bat 2](./images/platform-minecraft_3.jpg)

启动 `start.bat` 文件，即可开始运行服务端。

::: tip

首次打开 `start.bat` 启动脚本，会下载 `mojang_1.20.4.jar` 等一系列的文件并创建文件夹，在下载和创建完成后会首次启动会自行关闭。

:::

打开在文件夹内新创建的 `eula.txt`，将 `eula=false` 改为 `eula=true`。

```text{4}
...
#By changing the setting below to TRUE you are indicating your agreement to our EULA (https://aka.ms/MinecraftEULA).
#Fri Feb 30 00:00:01 HKT 2024
eula=true  # 请把该项修改为 true
...
```

再次启动脚本 `start.bat`。

当出现 `Done (7.837s)! For help, type "help"` 时，服务器首次启动完毕。

在 CMD 内键入 `stop`，等待服务器保存并关闭，即可进入下一个步骤。

### 放入 `sealdice-minecraft` 插件

前往 [Sealdice Minecraft GitHub Release](https://github.com/sealdice/sealdice-minecraft/releases) 下载 `jar` 文件放进 `plugins` 文件夹。

再次运行脚本 `start.bat` 以启动服务器。

在服务器日志中出现：

```cmd{2}
[00:00:07 INFO]: [SealDicePlugin] Enabling SealDicePlugin v1.0.2*
[00:00:07 INFO]: [SealDicePlugin] ChatServer started on port: 8887
[00:00:07 INFO]: [SealDicePlugin] Server started!
```

证明 `sealdice-minecraft` 插件已经正确安装，并且插件将会开启一个端口 `8887` 供海豹使用。

使用在控制台使用 `sealport [端口]` 或在游戏中使用 `/sealport [端口]` 可以修改连接端口。

至此，Minecraft 服务端的准备工作已经完毕。

## 海豹核心与 Minecraft 服务端连接

在 `账号设置` 页，选择 `账号类型` 为 `Minecraft(Paper)`：

![SealDice UI](./images/platform-minecraft_4.jpg)

URL 的填写请根据下列情况选择：

:::: tabs

== 部署在同一台服务器

海豹核心与 Minecraft 服务端部署在同一台服务器：

![SealDice UI](./images/platform-minecraft_5.jpg)

此时输入使用 `localhost:[端口]` 即可完成连接，默认端口为 `8887`。

== 部署在不同的服务器

海豹核心与 Minecraft 服务端部署在不同的服务器：

![SealDice UI](./images/platform-minecraft_6.jpg)

此时输入 `IP:[端口]`，IP 为 Minecraft 服务端所在服务器的 IP 地址。

::: tip

请注意，大多数服务器不会默认开放所需要的端口，需要自行开放。

:::

::::

当服务器后台日志中出现：

```cmd
[00:00:10 INFO]: [SealDicePlugin] 0:0:0:0:0:0:0:1 entered the room! 
```

证明海豹核心已和 Minecraft 服务端连接完毕。

## `sealdice-minecraft` 插件的使用

### 指令

提供两个插件指令：

#### `sealdice [文本]`

使用该指令的玩家视为向海豹私聊发送了一条消息。

但是命令方块和控制台使用该指令会被视为公屏发送。

#### `sealport [端口]`

使用该指令需要 `OP` 权限。该指令用于设置与海豹连接的端口。

### 权限

提供一个权限：

#### `sealdice.admin`

该权限允许/禁止玩家设置与海豹连接的端口，默认为 `false`。
