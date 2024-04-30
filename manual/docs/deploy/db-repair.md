---
lang: zh-cn
title: 数据库检查和修复
---

# 数据库检查和修复

::: info 本节内容

本节包括两项内容：如何判断海豹的数据库是否损坏，在损坏时如何修复。

数据库损坏发生的原因很多，包括但不限于突发断电、硬盘受到物理损坏、或硬盘空间占满。

:::

## 确定问题

如果你发现角色卡或 Log 会在重启海豹后丢失，或数据库文件变得很大（通常在 200MB 以上），建议对数据库进行完整性检查。

首先，停止海豹运行。在 Windows 系统上，你可以右键点击托盘图标并选择退出；在其他系统上，通常是在执行海豹核心的终端使用 ctrl+c 组合键；如果你注册了系统服务，通常是使用 `systemctl stop sealdice` 命令。

稍作等待，以确保数据被写入硬盘。

随后，使用命令行程序进入海豹的目录，执行以下命令。如果对使用命令行感到困难，后面有一个简化的替代方案。

::: tabs key:shell

== Windows-命令提示符（cmd）

```shell
sealdice-core /db-check
```

== 其他 Shell

```shell
./sealdice-core --db-check
```

:::

你将看到类似的输出

```text
数据库检查结果：
data.db: true
data-logs.db: true
data-censor.db: true
```

这代表数据库是正常的。列出的是海豹核心使用的 3 个数据库文件：

- data.db - 人物卡和群内临时卡
- data-logs.db - 跑团日志
- data-censor.db - 敏感词库

如果有某个数据库文件后输出了 `false`，说明该文件内容损坏。

### 无法使用命令行

如果你对使用命令行感到困难，可以这样做：

打开记事本，将以下内容复制进去：

```shell
sealdice-core /db-check
pause
```

将它命名为 `检查.cmd`，保存在海豹的主程序所在目录，或者保存完复制过去。

双击 `检查.cmd` 执行，之后同上。

## 修复数据库 - 通过回滚备份

对于大多数情况，我们推荐直接回滚到备份文件的状态。这种方法简便、容易成功。代价是损失从备份时间点到当前时间的数据。

海豹核心默认每 12 小时进行一次备份，你可在 `backups/` 目录下找到所有的备份文件。备份的时间可以直接查看文件创建时间，也可以从文件名中确定。

将你损坏的数据库文件另外保存一份以防万一，并且确保你的硬盘有适当的空闲空间。

在备份文件中找到最新的一份（如果你能确定导致你数据库出问题的事件，也可以找到该时间点前的最后一份），从中解压出数据库文件，替换掉你发现损坏的数据库。

替换完成后，再进行一次完整性检查。如果仍然提示损坏，则使用更早的一份备份重新替换，直到数据库文件正常。

## 修复数据库 - 通过数据库修复指令

如果你熟悉 Sqlite 3，或者没有可用的备份文件，尝试以下方案。

这种办法有一定的操作难度，酌情进行使用。这里我们以 Windows 系统为例。

首先，你需要安装或下载一个 Sqlite 3 程序。

你可以从其[官网下载页](https://www.sqlite.org/download.html)，找到 Precompiled Binaries for Windows，下载其中的 sqlite-tools。确保你下载的是 3.40 以上版本，通常来说，直接下载最新版即可。

下载完成后，找出 sqlite.exe 放到空目录备用。

将损坏的数据文件（如 data.db）从海豹的 data/default/ 目录中复制出来，放在和 sqlite.exe 同一个目录。

使用命令行工具打开这个目录，在此目录下，执行下面的指令：

导出数据：

```shell
sqlite3.exe data.db
.output 1.sql
.recover
.exit
```

恢复数据到 a.db，并删除无效数据。

```shell
sqlite3.exe a.db
.read 1.sql
delete from attrs_group where id is null;
delete from attrs_user where id is null;
delete from group_info where id is null;
delete from attrs_group_user where id is null;
delete from ban_info where id is null;
delete from group_player_info where id is null;
.exit
```

接下来这个 a.db 就是修好的数据库了，将其复制回海豹的原路径，并改名回 data.db。
