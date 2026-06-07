# Fake QQ UI

一个全方位模仿 QQ NT 的聊天气泡样式与功能的 VUE 组件库

> [!NOTE]  
> 全方位模仿：大部分样式来自与 QQ NT for Windows，部分样式经修改

[![npm](https://img.shields.io/npm/v/fake-qq-ui)](https://www.npmjs.com/package/fake-qq-ui)

## 已实现

- 纯文本消息
- 图文消息
- 图片消息
- 图片文件消息
- 文件消息
- 引用（回复）消息
- 语音
  - 点击播放
  - 进度条显示音频音量大小
  - 显示音频时长
  - 动态音频气泡长度
  - 播放动画
  - 播放时点击暂停
  - 右键显示“语音转文本”内容
- 合并转发消息

### TODO

- 合并转发消息
- 文字头像

## 注意事项

- 本项目大量 HTML 和 CSS 来自 QQ
- 内置的 **QHeader** 与 **QMain** 组件不一定适合所有人，你可以使用自己的聊天窗口样式包裹消息组件
- 气泡颜色默认为白色，因此背景建议使用很淡的灰色

## 使用方法

1. 前往 [Release](https://github.com/Redlnn/FakeQQ-UI/releases/latest)
   页面，寻找最新的版本的 tar 包下载地址并用你的包管理器添加，如：

   ```sh
   pnpm add fake-qq-ui@latest
   ```

3. 你可以选择将 FakeQQUI 中的所有组件[注册为全局组件](#注册为全局组件)，
   或者你可以[手动按需导入组件](#手动导入)

### 注册为全局组件

1. 在你项目的入口文件（如 `main.ts`）文件中添加以下内容：

   ```js
   // main.ts
   import { FakeQQUI } from 'fake-qq-ui'

   import 'fake-qq-ui/styles/fake-qq-ui.css' // 导入基础样式（必须）
   import 'fake-qq-ui/styles/light.css' // 导入浅色模式的 CSS（必须）
   import 'fake-qq-ui/styles/dark.css' // 导入深色模式的 CSS（可选）

   // const app = createApp(App)
   app.use(FakeQQUI)
   ```

   > 深色模式仅支持在 html 根元素的 class 中添加 `dark` 类，不支持媒体查询

2. 为全局组件提供编辑器类型支持，下面两种方法二选一即可。

   1. 修改你的 `tsconfig.json` 中的 `types` 字段，添加 `fake-qq-ui/client`，如：

      ```json
      "types": ["node", "fake-qq-ui/client"]
      ```

   2. FakeQQUI 在 `fake-qq-ui/client.d.ts` 中所有组件提供了全局类型支持。你可以在你的 `env.d.ts` 或任意类型声明文件中添加如下内容：

      ```js
      /// <reference types="fake-qq-ui/client" />
      ```

3. 在你的页面中直接使用各个组件，详见[API文档](api.md)

   ```html
   <template>
     <div>
       <q-text name="[bot] 机器人" tag="LV96 机器人" tag-color="purple" is-bot>
         今天天气不错
       </q-text>
     </div>
   </template>
   ```

### 手动导入

1. 在你项目的入口文件（如 `main.ts`）文件中添加以下内容：

   ```js
   // main.ts
   import 'fake-qq-ui/styles/fake-qq-ui.css' // 导入基础样式（必须）
   import 'fake-qq-ui/styles/light.css' // 导入浅色模式的 CSS（必须）
   import 'fake-qq-ui/styles/dark.css' // 导入深色模式的 CSS（可选）
   ```

   > 深色模式仅支持在 html 根元素的 class 中添加 `dark` 类，不支持媒体查询

2. 在你的页面中导入和使用组件，详见[API文档](api.md)

   ```html
   <script setup lang="ts">
   import { QText } from 'fake-qq-ui'
   </script>
   <template>
     <div>
       <q-text name="[bot] 机器人" tag="LV96 机器人" tag-color="purple" is-bot>
         今天天气不错
       </q-text>
     </div>
   </template>
   ```

### 组件 API

参阅 [API.md](./api.md)
