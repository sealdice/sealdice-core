---
lang: zh-cn
title: 海豹的本地文件
---

# 海豹的本地文件

::: info 本节内容

本节将介绍海豹核心主要的本地文件和它们的作用，以及一些与文件相关的问题。

:::

## SealDice 文件目录树

``` tree
├─backups                    // 备份文件目录，里面是备份的压缩包
├─data                       // 数据目录
│  ├─censor                  // 敏感词库文件
|  ├─decks                   // 牌堆文件
|  ├─default
│  │  ├─configs              // 自定义骰点回执
|  |  ├─extantions           // 各个模块的特化文件放在这里，也是插件的数据目录
|  |  |  ├─coc7
|  |  |  ├─dnd5e
|  |  |  └─reply             // 比如你的自定义回复文件
|  |  ├─extra                // 放置你使用的QQ客户端配置文件
|  |  |  └─lagrange-qq123456 // 此处 123456 代指你骰子的QQ，是内置客户端的配置文件夹。
|  |  ├─log-exports          // log end 后导出的 log 文件
|  |  └─scripts              // 插件脚本
|  ├─helpdoc                 // 查询文档放置位置
|  ├─images                  // 放置图片资源的文件夹，海豹为了安全不允许调用非海豹文件夹内的图片
|  └─names                   // 姓名文件 . name 指令无法使用可以看看
├─_help_cache                // 全文搜索索引缓存
└─lagrange                   // 内置客户端的二进制文件
```

## 常用文件

- `./data/dice.yaml` 配置核心文件。

- `./data/main.log` 核心日志，可以查看报错消息。

- `./data/default/serve.yaml` 账号协议配置文件。

- `./data/default/record.log` 运行日志文件，可以查看报错信息。

- `./data/default/data.db` 数据库文件，存有人物卡、群组状态等信息。

- `./data/default/data-logs.db` log 数据库文件，存有 log 日志。  

- `./data/default/data-censor.db` 敏感词数据库文件，存有拦截日志等信息。

- `./data/default/configs/text-temple.yaml` 自定义文案的本体。

- `./data/default/extra/lagrange-qq[骰子QQ]` 内置客户端的配置文件夹。

## 安卓端文件路径

请查看[安卓端文件路径](android.md#%E6%B5%B7%E8%B1%B9%E6%96%87%E4%BB%B6%E8%B7%AF%E5%BE%84)一节。

## 数据库检查和修复

请查看[数据库检查和修复](./db-repair)一节。

## 数据迁移

请看[迁移](./transfer)一节。
