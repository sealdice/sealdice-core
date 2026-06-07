# UI 架构说明

这份说明面向第一次接手 `ui` 的开发者。项目是 SealDice 新版管理后台，目标是用一个可复现、可生成、以 V2 OpenAPI 为数据源的 Vue 3 前端替代旧管理界面。

## 总体架构

```txt
后端 Huma V2 OpenAPI
  -> pnpm run generate-api
  -> openapi.json
  -> src/api/generated/*
  -> src/api/client.ts 注入 baseUrl / token / 错误处理 / 会话清理
  -> feature/composable 组织业务状态
  -> page 组装页面
  -> 组件负责展示
```

`openapi.json` 和 `src/api/generated` 是构建期产物，不提交到仓库。正式构建会先生成它们；本地开发如果缺失或 API 变化，先运行 `pnpm run generate-api`。

这个项目的核心思路是“公共能力集中，业务逻辑分层，页面保持薄”：

- 公共能力放在 `api/`、`features/`、`router/`、`components/app-shell/` 和少量全局 `layouts/` 中。
- 业务页面只负责把这些能力拼起来。
- 真正跨页面复用的 UI 才进入 `components/shared/`。

## 目录职责

- `src/api/`：API 边界。
  - `generated/` 由 OpenAPI 生成，不手改、不提交。
  - `client.ts` 负责全局 axios client 配置、token 注入、错误反馈、401 会话清理。
  - `config.ts` 负责 API baseUrl。
  - `download.ts` 放文件下载这类生成器不方便表达的薄封装。
- `src/features/`：业务域逻辑。
  - 页面共享的状态、query、mutation、实时事件适配、上传控制器、认证状态都放这里。
  - 这一层负责“把后端接口和页面状态翻译成可用的业务模型”。
- `src/components/app-shell/`：后台应用壳层组件。
  - 包括侧边栏、面包屑、搜索、主题切换、解锁弹窗、未保存提示。
  - 这些组件可以依赖 router、auth、theme、unsaved changes 等应用级 feature。
- `src/components/shared/`：跨页面复用组件。
  - 只通过 props / emits / `defineModel` 接收数据。
  - 不直接发请求，不保存业务域状态。
- `src/components/<domain>/`：某个业务页的拆分组件。
  - 例如 JS 扩展页的 list/config/data 视图、指令测试页的聊天窗口。
- `src/layouts/`：页面壳。
  - `default` 与 `wide` 都保留后台导航，只调整内容区宽度。
  - `plain` 用于轻量流程页。
- `src/pages/`：路由入口。
  - 页面负责组合 query、mutation、feature composable 和展示布局。
  - 页面不承载全局基础设施。
- `src/router/`：路由补充配置。
  - 路由由 `vue-router/auto-routes` 生成。
  - 标题、布局、菜单、动态导航语义集中维护。

## 公共能力的位置

这部分是本项目最重要的“不要放错地方”的约定。

- HTTP 客户端统一在 `src/api/client.ts`。
  - baseURL、凭据、Bearer token、token 滚动更新、网络/会话级错误提示、401 清理都在这里。
  - 页面和 feature 不要重复写这些逻辑。
- 登录态统一在 `src/features/auth/state.ts`。
  - 当前唯一 token 源是这里的 `token`。
  - 不要直接在页面里读写 localStorage。
  - 不要恢复旧版 `t` token。
- Vue Query 全局配置在 `src/queryClient.ts`。
  - 这里定义默认缓存、重试和错误策略。
  - 业务层不要重复创建一套全局 query client。
- 实时连接统一在 `src/features/realtime/client.ts`。
  - WebSocket / SSE 的建连、降级、重连、事件分发都在这里。
  - 业务 feature 只负责订阅事件并把 payload 转成自己的状态。
- 路由语义统一在 `src/router/navigation.ts`、`src/router/routeMeta.ts`、`src/router/navigationModel.ts`。
  - 新页面一般只需要先补导航模型，标题、布局、面包屑等会一起派生。
- 未保存提示统一在 `src/features/unsavedChanges/`。
  - 页面只注册 source，不自己实现全局拦截。
- App 壳布局宽度逻辑统一在 `src/components/app-shell/appShellLayout.ts`。
  - `default`/`wide` 的差异是内容区宽度，不是另一套页面体系。

## 状态管理原则

服务端状态优先使用 Vue Query，稳定数据通过 OpenAPI generated client 获取。客户端本地状态优先使用局部 `ref/reactive`；只有主题、token、侧边栏等 UI 偏好才允许持久化。

页面层应该尽量只做三件事：

1. 读取 feature/composable 暴露出来的状态。
2. 触发 mutation 或本地交互。
3. 渲染组件。

不要把以下内容塞进页面：

- 全局错误处理。
- token 注入和清理。
- 实时连接管理。
- 大量跨页面共享状态。
- 复杂的数据转换规则。

## 一个简单功能的完整流程

下面以 `/tool/test` 的“指令测试”页面为例。

### 1. 路由进入

- 用户访问 `/tool/test`。
- 页面文件是 `src/pages/tool/test.vue`。
- 页面标题、布局等由 `src/router/navigation.ts` 和 `src/router/routeMeta.ts` 派生。

### 2. 页面初始化

- 页面里通过 `useToolTest()` 获取业务状态。
- 视图只负责渲染：
  - 模式切换
  - 消息窗口
  - 输入框
  - 快捷操作
  - 错误提示

### 3. feature 组织业务状态

- `src/features/toolTest/useToolTest.ts` 是核心。
- 它负责：
  - 管理当前模式 `private/group`
  - 管理输入框内容
  - 拉取指令列表
  - 发送测试消息
  - 轮询 pending 消息
  - 触发牌堆 / JS / 帮助文档重载
- `src/features/toolTest/model.ts` 负责纯数据模型和纯函数：
  - 初始消息
  - 追加自发消息
  - 追加 pending 消息
  - 生成命令联想项

### 4. 数据来源

- 指令列表来自 `getSdApiV2ToolTestCommands()`。
- 待处理消息来自 `getSdApiV2ToolTestMessagesPending()`。
- 发送消息来自 `postSdApiV2ToolTestMessages()`。
- 快捷操作来自：
  - `postSdApiV2DeckReload()`
  - `postSdApiV2JsReload()`
  - `postSdApiV2HelpdocReload()`

### 5. 状态流转

- 有 token 时，`watch(hasAccessToken)` 会自动加载指令并启动轮询。
- 用户发送消息时：
  - 先把自己的消息追加到本地会话里。
  - 再发请求给后端。
  - 成功后继续拉取 pending 消息。
  - 失败时追加 tip 提示。
- 轮询失败时会停止轮询，并把错误显示到页面。

### 6. 视图层展示

- `src/components/tool-test/ToolTestChatWindow.vue` 负责聊天窗口展示。
- 页面只传 `title` 和 `messages`。
- 这样以后如果聊天窗口样式变化，业务逻辑不用跟着改。

### 7. 这个流程体现的分层

- `pages/` 负责组合。
- `features/` 负责业务状态。
- `components/` 负责展示。
- `api/` 负责请求边界。
- `queryClient.ts` 负责缓存策略。

## 新页面开发顺序

1. 确认后端 V2 OpenAPI 已覆盖接口，运行 `pnpm run generate-api` 更新本地 `openapi.json` 和 `src/api/generated`。
2. 在 `pages/` 新建路由入口。
3. 在 `router/navigation.ts` 增加导航语义。
4. 由 `routeMeta` 派生页面标题和 layout。
5. 将复杂业务状态抽到 `features/<domain>/`，不要堆进页面。
6. 跨页面复用组件才放 `components/shared/`。
7. 运行 `pnpm run type-check` 和 `pnpm run build-only`。

## 维护约定

- 不手改或提交 `src/api/generated/`。需要变更 API 类型时改后端 OpenAPI，再重新生成本地产物。
- 不在页面里实现全局错误处理、token 注入、401 清理，这些属于 `src/api/client.ts`。
- 不在页面里直接创建 WebSocket / SSE，这些属于 `src/features/realtime/client.ts`。
- 不把老前端的大 store 模式搬进来。每个业务域维护自己的最小状态。
- 不把公共能力散落到页面和组件里。能下沉就下沉到对应 feature / api / router / app-shell。

## 常见落点速查

- API 客户端配置：`src/api/client.ts`
- 认证 token：`src/features/auth/state.ts`
- Vue Query 配置：`src/queryClient.ts`
- 实时连接：`src/features/realtime/client.ts`
- 路由菜单语义：`src/router/navigation.ts`
- 路由派生元信息：`src/router/routeMeta.ts`
- App 壳宽度逻辑：`src/components/app-shell/appShellLayout.ts`
- 未保存拦截：`src/features/unsavedChanges/`
