# API 参考

> 下文中属性默认值为 undefined 或其他内容代表该属性可选

## 注意事项

### TagColor

TagColor 是一个几乎所有组件都拥有的参数，用于群头衔。

其类型为 `QTagColors | keyof typeof QTagColors | QTagCustomize`

其中 QTagColor 是一个 enum，用于消息中名字后面的`管理员`、`群头像`等 tag
中，原型位于 `lib/QTagColors.d.ts` 中，你可以使用
`import { QTagColors } from 'fake-qq-ui'` 来导入，其包含以下几种颜色：

- `sage_green`：鼠尾草绿
- `red`：红色
- `orange`：橙色，用于群主
- `purple`：紫色，用于自定义头衔
- `blue`：蓝色，用于管理员
- `grey`：灰色，用于群等级

而 QTagCustomize 则是一个 interface，使用时传入一个对象。

用法如下：

```vue
<script setup lang="ts">
import { QTagColors } from 'fake-qq-ui'
</script>

<template>
  <!-- 传入字符串 -->
  <q-text name="你" tag="群主" tag-color="orange">Something...</q-text>
  <!-- 传入 QTagColors -->
  <q-text name="我" tag="管理员" :tag-color="QTagColors.blue">Something...</q-text>
  <!-- 传入满足 QTagCustomize 对象 -->
  <q-text name="他" tag="猜猜我是谁" :tag-color="{ backgroundColor: '#222', color: '#fff' }">
    Something...
  </q-text>
</template>
```

## QHeader

一个仿 macOS 的标题栏。

### QHeader 插槽

| 插槽名  | 说明 |
| ------- | ---- |
| default | 标题 |

## QMain

一个带有基础样式的容器，用于容纳消息组件（非必须）。

### QMain 插槽

| 插槽名  | 说明             |
| ------- | ---------------- |
| default | 内部包裹消息组件 |

## QText

纯文本消息、图文消息。

### QText 插槽

| 插槽名  | 说明     |
| ------- | -------- |
| default | 图文内容 |

- 换行请使用 `<br />`
- 图片请使用 `<img />`
- 链接请使用 `<a>link<a/>`
- @ 请使用 `<a at>@user</a>`

### QText 属性

| 属性         | 说明                                         | 类型      | 默认值          |
| ------------ | -------------------------------------------- | --------- | --------------- |
| self         | 是否是浏览者视角发送（即消息是否在右边）     | `boolean` | false           |
| isBot        | 是否为机器人消息                             | `boolean` | false           |
| name         | 用户名                                       | `string`  | -               |
| avatar       | 头像来源地址                                 | `string`  | undefined       |
| tag          | 群头衔内容，留空则表示没有头衔               | `string`  | undefined       |
| tagColor     | 群头像颜色，留空即为灰色                     | `enum`    | QTagColors.grey |
| maxImgWidth  | 图文消息中图片的最大宽度（防止过大影响观感） | `string`  | 230px           |
| maxImgHeight | 图文消息中图片的最大高度（防止过大影响观感） | `string`  | 250px           |

## QImage

图片消息、图片文件。

### QImage 属性

| 属性        | 说明                                                           | 类型      | 默认值          |
| ----------- | -------------------------------------------------------------- | --------- | --------------- |
| self        | 是否是浏览者视角发送（即消息是否在右边）                       | `boolean` | false           |
| isBot       | 是否为机器人消息                                               | `boolean` | false           |
| name        | 用户名                                                         | `string`  | -               |
| avatar      | 头像来源地址                                                   | `string`  | undefined       |
| tag         | 群头衔内容，留空则表示没有头衔                                 | `string`  | undefined       |
| tagColor    | 群头像颜色，留空即为灰色                                       | `enum`    | QTagColors.grey |
| src         | 图片来源地址                                                   | `string`  | -               |
| isFile      | 是否是文件形式的图片                                           | `boolean` | false           |
| fileName    | 图片文件名（当且仅当 `isFile` 为 true 时起作用）               | string    | undefined       |
| fileSize    | 图片文件大小（当且仅当 `isFile` 为 true 时起作用）             | string    | undefined       |
| canDownload | 用户点击时是否下载该图片（当且仅当 `isFile` 为 true 时起作用） | `boolean` | true            |
| maxWidth    | 图文图片的最大宽度（防止过大影响观感）                         | `string`  | 230px           |
| maxHeight   | 图文图片的最大高度（防止过大影响观感）                         | `string`  | 250px           |

## QFile

文件消息。

### QFile 属性

| 属性        | 说明                                     | 类型      | 默认值          |
| ----------- | ---------------------------------------- | --------- | --------------- |
| self        | 是否是浏览者视角发送（即消息是否在右边） | `boolean` | false           |
| isBot       | 是否为机器人消息                         | `boolean` | false           |
| name        | 用户名                                   | `string`  | -               |
| avatar      | 头像来源地址                             | `string`  | undefined       |
| tag         | 群头衔内容，留空则表示没有头衔           | `string`  | undefined       |
| tagColor    | 群头像颜色，留空即为灰色                 | `enum`    | QTagColors.grey |
| fileName    | 文件名                                   | string    | undefined       |
| fileSize    | 文件大小                                 | string    | undefined       |
| fileSrc     | 文件链接                                 | `string`  | -               |
| iconSrc     | 文件图标来源地址                         | `string`  | -               |
| canDownload | 用户点击时是否下载该文件                 | `boolean` | true            |

## QTip

提示文本。  
（如戳一戳、撤回提示、时间等）

### QTip 插槽

| 插槽名  | 说明     |
| ------- | -------- |
| default | 文字内容 |

- 链接、@ 请使用 `<a>content</a>`

### QTip 属性

| 属性   | 说明                                                | 类型      | 默认值 |
| ------ | --------------------------------------------------- | --------- | ------ |
| isTime | 暂时内容是否是时间（当设为 true 时 CSS 有略微区别） | `boolean` | false  |

## QReply

引用（回复）消息。

### QReply 插槽

| 插槽名  | 说明     |
| ------- | -------- |
| default | 图文内容 |

> 与 QText 组件用法一致

### QReply 属性

| 属性          | 说明                                       | 类型      | 默认值          |
| ------------- | ------------------------------------------ | --------- | --------------- | --- |
| self          | 是否是浏览者视角发送（即消息是否在右边）   | `boolean` | false           |
| isBot         | 是否为机器人消息                           | `boolean` | false           |
| name          | 用户名                                     | `string`  | -               |
| avatar        | 头像来源地址                               | `string`  | undefined       |
| tag           | 群头衔内容，留空则表示没有头衔             | `string`  | undefined       |
| tagColor      | 群头像颜色，留空即为灰色                   | `enum`    | QTagColors.grey |
| target        | 被回复用户的昵称                           | `string`  | -               |
| replyText     | 被回复的文字                               | `string`  | ''              |
| replyImageUrl | 被回复的图片来源地址（有图片时不显示文字） | `string`  | undefined       |     |
| replyImageAlt | 被回复的图片描述                           | `string`  | undefined       |
| maxImgWidth   | 被回复的图片的最大宽度                     | `string`  | '200px'         |
| maxImgHeight  | 被回复的图片的最大高度                     | `string`  | '220px'         |

## QVoice / QVoiceLegacy

语音消息。

> [!IMPORTANT]
>
> 1. QVoice 组件使用现代的 AudioContext 实现，加载时会先使用 [Fetch API](https://developer.mozilla.org/zh-CN/docs/Web/API/Fetch_API)
>    下载音频后分析得到音频强度分布，因此 src 参数中的地址不能出现跨域访问（除非对方站点允许你跨域）
> 2. 由于 QVoice 组件使用 AudioContext 实现，因此首次加载时会在浏览器控制台输出警告信息，为正常现象
> 3. QVoiceLegacy 组件为使用 `<audio>` 标签实现的旧组件，该实现方式不支持在展示语音强度分布，
>    所展示效果为每次加载时随机生成

### QVoice / QVoiceLegacy 属性

| 属性     | 说明                                     | 类型      | 默认值                 |
| -------- | ---------------------------------------- | --------- | ---------------------- |
| self     | 是否是浏览者视角发送（即消息是否在右边） | `boolean` | false                  |
| isBot    | 是否为机器人消息                         | `boolean` | false                  |
| name     | 用户名                                   | `string`  | -                      |
| avatar   | 头像来源地址                             | `string`  | undefined              |
| tag      | 群头衔内容，留空则表示没有头衔           | `string`  | undefined              |
| tagColor | 群头像颜色，留空即为灰色                 | `enum`    | QTagColors.grey        |
| src      | 语音文件 URL                             | `string`  | -                      |
| text     | 语音转文字结果                           | `string`  | `[呃，什么都没有听到]` |
| volume   | 音量                                     | `number`  | `1.0`                  |

## QForward

合并转发消息。

### QForward 属性

| 属性     | 说明                                     | 类型       | 默认值           |
| -------- | ---------------------------------------- | ---------- | ---------------- |
| self     | 是否是浏览者视角发送（即消息是否在右边） | `boolean`  | false            |
| isBot    | 是否为机器人消息                         | `boolean`  | false            |
| name     | 用户名                                   | `string`   | -                |
| avatar   | 头像来源地址                             | `string`   | undefined        |
| tag      | 群头衔内容，留空则表示没有头衔           | `string`   | undefined        |
| tagColor | 群头像颜色，留空即为灰色                 | `enum`     | QTagColors.grey  |
| title    | 合并转发标题                             | `string`   | `群聊的聊天记录` |
| contents | 合并转发内容                             | `string[]` | -                |
