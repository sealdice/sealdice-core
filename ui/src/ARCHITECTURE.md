# UI 架构说明

这份说明面向第一次接手 `ui` 的开发者。项目是 SealDice 新版管理后台，目标是用一个可复现、可生成、以 V2 OpenAPI 为数据源的 Vue 3 前端替代旧管理界面。

## 总体数据流

```txt
后端 Huma V2 OpenAPI
  -> pnpm run generate-api
  -> src/api/generated/*
  -> src/api/client.ts 注入 baseUrl / token / 错误处理
  -> 页面或 feature composable 使用 Vue Query
  -> Naive UI 组件渲染
```

`src/api/generated` 是提交到仓库的生成客户端。新 clone 后可以直接 `pnpm run type-check` 和 `pnpm run build-only`，不依赖开发者先本地生成。

## 目录职责

- `api/`：API 边界。`generated/` 不手改；`client.ts` 负责全局 fetch client 配置；`download.ts` 放文件下载这种生成器不方便表达的薄封装。
- `features/`：业务域逻辑。页面共享的状态、实时事件适配、上传控制器、认证状态都放这里，避免页面直接拼协议细节。
- `components/shared/`：跨页面复用组件。shared 组件只通过 props / emits / `defineModel` 接收数据，不直接发请求。
- `components/<domain>/`：某个业务页的拆分组件，例如 JS 扩展页的 list/config/data 视图。
- `layouts/`：页面壳。`default` 与 `wide` 都保留后台导航，只调整内容区宽度；`plain` 用于轻量流程页。
- `pages/`：路由入口。页面负责组合 query、mutation、feature composable 和展示布局，不承载全局基础设施。
- `router/`：文件路由的补充配置。路由由 `vue-router/auto-routes` 生成，标题、布局、菜单在这里集中维护。

## 状态管理原则

服务端状态优先使用 Vue Query，稳定数据通过 OpenAPI generated client 获取。客户端本地状态优先使用局部 `ref/reactive`；只有主题、token、侧边栏等 UI 偏好才允许持久化。

认证状态只有一个来源：`features/auth/state.ts` 中的 `token`。页面不要直接读写 localStorage，也不要恢复旧版 `t` token。

实时数据统一通过 `features/realtime/client.ts` 建立连接，再由业务 feature 订阅事件并转换成页面状态。例如首页日志使用 `features/base/logStream.ts`，连接管理使用 `features/connect/realtime.ts`。

## 新页面开发顺序

1. 确认后端 V2 OpenAPI 已覆盖接口，运行 `pnpm run generate-api` 更新 `src/api/generated`。
2. 在 `pages/` 新建路由入口。
3. 在 `router/routeMeta.ts` 配置标题和 layout。
4. 在 `router/navigation.ts` 接入菜单。
5. 将复杂业务状态抽到 `features/<domain>/`，不要堆进页面。
6. 跨页面复用组件才放 `components/shared/`。
7. 运行 `pnpm run type-check` 和 `pnpm run build-only`。

## 维护约定

- 不手改 `src/api/generated`。需要变更 API 类型时改后端 OpenAPI，再重新生成。
- 不在页面里实现全局错误处理、token 注入、401 清理，这些属于 `api/client.ts`。
- 不在页面里直接创建 WebSocket/SSE，这些属于 `features/realtime/client.ts`。
- 不把老前端的大 store 模式搬进来。每个业务域维护自己的最小状态。
