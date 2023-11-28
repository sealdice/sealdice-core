---
lang: zh-cn
title: Discord
---

# Discord
::: info 本节内容

本节将包含你在 Discord 平台接入海豹核心需要了解的特定内容。

:::

## 获取 Token

要获取用于连接海豹的 Token 之前，你需要建立好一个应用。登陆 [Discord 开发者平台](https://discord.com/developers/applications/1178793642148769905/bot)，点击「New application」，按照要求填写应用名字并确认创建。完成后，点击侧边导航栏「bot」，打开「Privileged Gateway Intents」目录下全部三个开关：`Presence Intent`，`Server Members Intent`，`Message Content Intent`。完成后点击「Save Changes」。

建立好应用后上滑，点击 `Reset Token`，确认后点击 `Copy` 复制。

## 连接海豹

::: tip 代理模式

如果你海豹所处的位置直接访问 Discord 服务有困难，我们提供了通过 HTTP 代理访问的途径。

:::

登录海豹并添加账号，选择「Discord账号」。在 `Token` 处粘贴你得到的 Token，点击连接。你的海豹应该可以正常在 Discord 运作了！

## 邀请海豹

要申请用于邀请海豹至 Discord 服务器的邀请链接，前往侧边导航栏「OAuth2」下方的子目录「URL Generator」，在 `Scope` 中选择 `Bot` 后在下方 `Bot Permissions` 中选择你希望海豹拥有的权限。完成后复制下方生成的 URL，复制到浏览器打开。

::: warning 权限不足

如果给予海豹的权限不充分，可能会导致无法发送消息或图片。若你不确定具体应该添加哪些权限，可直接添加 `Administrator`（管理员）权限。

:::
