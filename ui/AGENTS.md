# SealDice UI Agent 指南

你是这个仓库的代码代理，服务于 `SealDice` 新版管理前端（Vue 3 + TypeScript + Naive UI）。
目标不是“随便改到能跑”，而是做出可维护、可验证、尽量贴合现有架构的改动。

## 基本原则

- 先理解上下文，再动手。
- 优先沿用现有结构、命名和实现方式，不要引入无必要的新范式。
- 任何会改变行为的任务，先给出简短方案，再实施。
- 用户的现有改动永远优先，未明确要求时不要回滚、覆盖或重置它们。
- 只改任务相关文件，不做顺手重构。
- 默认用中文沟通，表述要简洁、直接、可执行。

## 项目事实

- 前端栈：Vue 3、TypeScript、Naive UI、Vue Query、Tailwind CSS v4。
- 路由：`vue-router/auto-routes` 文件路由，Hash 模式。
- API：后端 OpenAPI 生成客户端是构建期产物，不提交到 `src/api/generated/`。
- 状态：服务端状态优先用 Vue Query，局部 UI 状态优先用 `ref` / `reactive`。
- 认证：只认 `src/features/auth/state.ts` 中的 `token`。
- 实时连接：统一走 `src/features/realtime/client.ts`，不要在页面里直接散写 WebSocket / SSE。

## 目录边界

- `src/api/generated/`：只允许从 `openapi.json` 重新生成，不手改、不提交。
- `src/api/`：API 边界与全局 client 配置。
- `src/features/`：业务域逻辑、状态、query、mutation、适配器。
- `src/components/<domain>/`：业务组件拆分。
- `src/components/shared/`：跨页面复用组件，只通过 props / emits / `defineModel` 交互。
- `src/pages/`：路由入口，只负责组装页面，不承载全局基础设施。
- `src/router/`：路由、菜单、标题、布局等集中配置。
- `src/layouts/`：页面布局壳。

## 全流程

### 1. 接任务

- 先判断任务属于：新功能、修 bug、重构、页面调整、API 变更、测试修复中的哪一种。
- 如果需求不清楚，先问一个最关键的问题，不要一次抛很多问题。
- 如果任务较大，先拆成可独立验证的小步。

### 2. 看上下文

- 先读 `README.md`、`src/ARCHITECTURE.md`，再看相关 `feature` / `component` / `page`。
- 搜索相似实现，优先复用同类模式。
- 关注是否已经有对应测试、工具函数、约定写法。

### 3. 选方案

- 对于会影响结构或行为的任务，先给出 2-3 个方案，并说明取舍。
- 默认推荐最小改动、最少风险、最符合现有架构的方案。
- 如果是视觉或交互改动，优先贴合现有 Naive UI 风格和后台壳层，不要自创一套设计语言。

### 4. 实施

- 用 `Composition API + <script setup lang="ts">`。
- 复杂业务逻辑下沉到 `src/features/<domain>/`，页面保持薄。
- 复用组件放 `src/components/shared/`，业务组件放对应模块目录。
- 需要修改多个文件时，按最小闭环顺序提交改动。
- 编辑文件时优先用 `apply_patch`。

### 5. 验证

- 新逻辑优先补测试，特别是纯函数、状态机、query 适配、解析器和权限/错误处理。
- 完成后至少跑与任务相关的验证；通常优先级是：
  - `pnpm run type-check`
  - `pnpm run test`
  - `pnpm run build-only`
  - `pnpm run lint`
- 如果任务涉及 API 变化，先更新后端 OpenAPI，再运行：
  - `pnpm run generate-api`

### 6. 收尾

- 检查 `git status`，确认只包含本次任务相关改动。
- 总结时明确说明改了什么、怎么验证、还有哪些风险或未覆盖项。

## 关键禁区

- 不手改或提交 `src/api/generated/`。
- 不在页面里直接创建全局级网络连接或错误处理。
- 不把状态无脑堆进页面组件。
- 不为了“顺手”重构 unrelated 代码。
- 不覆盖用户已有改动。

## 常用命令

- 开发：`pnpm dev`
- 类型检查：`pnpm run type-check`
- 测试：`pnpm run test`
- 构建：`pnpm run build-only`
- 统一构建：`pnpm run build`
- 生成 API：`pnpm run generate-api`

## 交付标准

- 代码能解释清楚边界和数据流。
- 改动能通过相关验证。
- 说明里能让接手的人知道哪里改了、为什么这么改、还剩什么风险。
