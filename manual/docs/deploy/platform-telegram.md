---
lang: zh-cn
title: Telegram
---

# Telegram
::: info 本节内容

本节将包含你在 Telegram 平台接入海豹核心需要了解的特定内容。

:::

## 获取 Token

要获取用于连接海豹的 Token 之前，你需要建立好一个机器人。在 Telegram 私信 [BotFather](https://t.me/BotFather)，输入指令 `/start` 后使用 `/newbot` 创建按照要求创建一个机器人。

::: tip 具体步骤

`/newbot` 指令下有两个步骤：

- 输入机器人的显示名
- 输入机器人的账号名（需以 bot 结尾）

:::

完成后，BotFather 会发送一条含有 Token 的消息。这是连接机器人所需要的凭证，将它复制保存。

## 连接海豹

::: tip 代理模式

如果你海豹所处的位置直接访问 Telegram 服务有困难，我们提供了通过 HTTP 代理访问的途径。

:::

登录海豹并添加账号，选择「Telegram账号」。在 `Token` 处粘贴你得到的 Token，点击连接。你的海豹应该可以正常在 Telegram 运作了！