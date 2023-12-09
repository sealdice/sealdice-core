---
lang: zh-cn
title: Slack
---

# Slack

::: info 本节内容

本节将包含你在 Slack 平台接入海豹核心需要了解的特定内容。

:::

## Slack 支持 <Badge type="tip" text="v1.4.2" vertical="middle" />

海豹从 `v1.4.2` 开始支持对接 Slack。

## 获取 Token

要获取用于连接海豹的 Token 之前，你需要建立好一个应用。登录 [Slack Api 平台](https://api.slack.com/apps)，点击「Create New App」，再点击「From Scratch」。按照要求填写应用名字，并选择你想要海豹被部署到的工作区后点击「Create App」。

::: warning 私骰模式

Slack 平台的机制使没有公开发布的应用无法加入被指定的单个工作区之外的地方。如有需求，可查看 Slack 提供的[发布指引](https://api.slack.com/authentication/oauth-v2)。

:::

### APP Token

在侧边栏点击「Socket Mode」，打开「Enable Socket Mode」的开关。此处会弹出一个窗口，这将会是你的 `APP Token`。按需填写名字，并复制保存。
确认后点击下方「Enable Events」，打开开关后在「Subscribe to bot events」下添加如下事件：

1. `message.channels`
1. `message.groups`
1. `message.im`
1. `message.mpim`

::: warning 事件注意

如果这不是你期望的情况，请不要添加 `app_mention`。这个权限会让海豹只接收被 @ 到的指令和消息，导致 log 等功能无法正常工作。

:::

### Bot Token
在侧边栏点击「OAuth & Permissions」，下滑在「Bot Token Scopes」下，添加海豹运作需要的权限:

1. `channels:history`
1. `channels:read`
1. `chat:write`
1. `chat:write.customize`
1. `files:write`
1. `groups:history`
1. `groups:read`
1. `im:history`
1. `im:read`
1. `im:write`
1. `mpim:history`
1. `mpim:read`
1. `links.embed:write`
1. `links:write`

点击侧边栏前往「Basic Information」页面，在「Install Your App」下点击「Install to Workplace」。完成后回到「OAuth & Permissions」页面，在「OAuth Tokens for Your Workspace」下找到「Bot User OAuth Token」。这是你的 `Bot Token`。复制并保存。

## 连接海豹

登录海豹并添加账号，选择「Slack」。在对应的区域填入 `APP Token` 和 `Bot Token`，点击连接。你的海豹应该可以正常在 Slack 平台运作了！
