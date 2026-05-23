# SealDice 管理前端

SealDice 的新版管理后台 UI，基于 Vue 3 + TypeScript + Naive UI 构建，通过 V2 OpenAPI 接口与后端通信。

架构接手说明见 [`src/ARCHITECTURE.md`](src/ARCHITECTURE.md)。它描述了目录边界、数据流、认证、实时通道和新页面开发顺序。

## 技术栈

- Vue 3 + TypeScript
- Naive UI
- Vue Query（服务端状态）
- Tailwind CSS v4
- @hey-api/openapi-ts（API 客户端生成）
- vue-router/auto-routes（文件路由）

## 快速开始

```sh
pnpm install
pnpm dev
```

开发时页面请求保持同源，由 Vite 代理转发到后端（默认 `http://127.0.0.1:3005`），可通过环境变量覆盖：

```sh
VITE_API_PROXY_TARGET=http://127.0.0.1:3005 pnpm dev
```

前端路由使用 Hash 模式，页面地址形如 `/#/mod/js`。

## 生成 API 客户端

`openapi.json` 和 `src/api/generated/` 都是构建期产物，不提交到仓库，也不要手改。

本地需要刷新 API 客户端时运行：

```sh
pnpm run generate-api
```

该命令会先调用当前后端生成 `openapi.json`，再用 `@hey-api/openapi-ts` 生成 `src/api/generated/` 下的 TypeScript 客户端。正式构建命令会自动执行这一步。

## 构建与检查

```sh
pnpm run type-check    # TypeScript 类型检查
pnpm run build-only    # Vite 生产构建，要求已存在 generated API
pnpm run build         # generate-api + type-check + build
pnpm run lint          # ESLint
```

## 目录结构

```txt
src/
  api/              # OpenAPI 生成客户端 + 配置
  components/       # 业务组件（按模块分子目录）
    shared/         # 跨页面复用组件
    js/             # JS 扩展相关组件
    story/          # 跑团日志相关组件
    ...
  features/         # 业务模块逻辑（composable、状态、类型）
    auth/           # 认证状态与会话管理
    upload/         # 通用分片上传控制器
    theme/          # 主题切换
    ...
  layouts/          # 页面布局壳
  pages/            # 路由页面（文件路由自动注册）
    mod/            # 功能模块页（牌堆、JS、日志、自定义回复…）
    misc/           # 系统设置页（基础设置、群组、黑白名单…）
    tool/           # 工具页（指令测试、资源管理）
    custom-text/    # 自定义文案动态路由
  router/           # 路由配置、菜单、进度条
```

## 页面清单

| 路径 | 页面 | 状态 |
|------|------|------|
| `/` | 主页 | ✅ |
| `/connect` | 连接管理 | ✅ |
| `/mod/deck` | 牌堆管理 | ✅ |
| `/mod/story` | 跑团日志 | ✅ |
| `/mod/js` | JS 扩展 | ✅ |
| `/mod/reply` | 自定义回复 | ✅ |
| `/custom-text/:category` | 自定义文案 | ✅ |
| `/mod/helpdoc` | 帮助文档 | ✅ |
| `/mod/censor` | 拦截管理 | ✅ |
| `/misc/base-setting` | 基本设置 | ✅ |
| `/misc/group` | 群组管理 | ✅ |
| `/misc/ban` | 黑白名单 | ✅ |
| `/misc/dice-public` | 公骰设置 | ✅ |
| `/misc/backup` | 备份 | ✅ |
| `/misc/advanced-setting` | 高级设置 | ✅ |
| `/tool/test` | 指令测试 | ✅ |
| `/tool/resource` | 资源管理 | ✅ |
| `/about` | 关于 | ✅ |
