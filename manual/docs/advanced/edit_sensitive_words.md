---
lang: zh-cn
title: 编写敏感词库
---

# 编写敏感词库

::: info 本节内容

本节将介绍敏感词库的编写，请善用侧边栏和搜索，按需阅读文档。

:::

## 创建文本格式的敏感词库

你可以直接按照以下格式书写 `<words>.txt`：

```text
#notice
提醒级词汇 1
提醒级词汇 2

#caution
注意级词汇 1
注意级词汇 2

#warning
警告级词汇

#danger
危险级词汇
```

## 创建 TOML 格式的敏感词库

::: info TOML 格式

我们假定你已了解 TOML 格式。如果你对 TOML 还很陌生，可以阅读以下教程或自行在互联网搜索：

- [TOML 文档](https://toml.io/cn/v1.0.0)、[TOML 教程](https://zhuanlan.zhihu.com/p/348057345)

:::

你可以直接按照以下格式书写 `<words>.toml`：

```toml
# 元信息，用于填写一些额外的展示内容
[meta]
# 词库名称
name = '测试词库'
# 作者，和 authors 存在一个即可
author = ''
# 作者（多个），和 author 存在一个即可
authors = [ '<匿名>' ]
# 版本，建议使用语义化版本号
version = '1.0'
# 简介
desc = '一个测试词库'
# 协议
license = 'CC-BY-NC-SA 4.0'
# 创建日期，使用 RFC 3339 格式
date = 2023-10-30
# 更新日期，使用 RFC 3339 格式
updateDate = 2023-10-30

# 词表，出现相同词汇时按最高级别判断
[words]
# 忽略级词表，没有实际作用
ignore = []
# 提醒级词表
notice = [
  '提醒级词汇 1',
  '提醒级词汇 2'
]
# 注意级词表
caution = [
  '注意级词汇 1',
  '注意级词汇 2'
]
# 警告级词表
warning = [
  '警告级词汇 1',
  '警告级词汇 2'
]
# 危险级词表
danger = [
  '危险级词汇 1',
  '危险级词汇 2'
]
```
