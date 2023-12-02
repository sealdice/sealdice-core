---
lang: zh-cn
title: 数据库检查和修复
---

# 数据库检查和修复

::: info 本节内容

本节内容描述了如何对海豹的数据库进行完整性检查，同时在遇到问题时如何进行修复。

数据库损坏有时会在突发断电，或者硬盘受到物理损坏时发生。

:::

## 确定问题

如果你发现录卡会在重启海豹后丢失，数据库文件变得很大(通常在200MB以上)，建议对数据库进行完整性检查。

首先，通过托盘右键退出的方式，停止海豹的运行，然后稍作等待，以确保数据被写入硬盘。

随后，使用命令行程序进入海豹的目录，执行（如果对使用命令行感到困难，后面有一个简化的替代方案）：

```
sealdice-core /db-check
```

输出结果如下：
```
>sealdice-core /db-check
ok
ok
ok
数据库检查结果：
data.db: true
data-logs.db: true
data-censor.db: true
```

这代表数据库是正常的，海豹的三个数据库文件：

data.db - 人物卡和群内临时卡
data-logs.db - 跑团日志
data-censor.db - 敏感词库

如果你对使用命令行感到困难，可以这样做：

打开记事本，将以下内容复制进去：
```
sealdice-core /db-check
pause
```

然后保存为“检查.cmd”，或者“check.cmd”，保存在海豹的主程序所在目录，或者保存完复制过去。

然后双击“检查.cmd”执行他，之后同上。


## 修复数据库 - 通过回滚备份

一般来说，首选的修复方式是通过备份，因为这样可以完美修复，同时损失的数据时间范围是确定的，发现得早可能就是一两天（默认会12小时自动备份一次）。

当你做了完整性检查之后，现在你已经知道是哪个数据文件坏掉了。

现在找出最近的几个备份，替换 data/defualt/ 目录下对应的 db 文件回之前的版本。

因为不能确定什么时候开始出问题，在完成替换后建议再做一次完整性检查，如果还有问题就替换更早的版本。


## 修复数据库 - 通过数据库修复指令

这种办法有一定的操作难度，酌情进行使用。这里我们以windows系统为例，其他操作系统大同小异

首先需要下载一个 sqlite.exe，可以从其[官网](https://www.sqlite.org/download.html)，找到 Precompiled Binaries for Windows 进行下载

下载完成后，找出sqlite.exe放到空目录备用，注意sqlite必须是3.40以上版本，不然没有.recover指令。

接下来，将损坏的数据文件（如data.db）从海豹的data/defualt/目录中复制出来，放在和sqlite.exe同一个目录。

再之后，使用命令行工具打开这个目录，在此目录下，执行下面的指令：

导出数据：
```
sqlite3.exe data.db
.output 1.sql
.recover
.exit
```

恢复数据到a.db
```
sqlite3.exe a.db
.read 1.sql
delete from attrs_group where id is null;
delete from attrs_user where id is null;
delete from group_info where id is null;
delete from attrs_group_user where id is null;
delete from ban_info where id is null;
delete from group_player_info where id is null;
```

接下来这个a.db就是修好的数据库了，将其复制回海豹的原路径，并改名回data.db即可。
