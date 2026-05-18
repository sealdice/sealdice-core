# ui

This template should help get you started developing with Vue 3 in Vite.

## Recommended IDE Setup

[VSCode](https://code.visualstudio.com/) + [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (and disable Vetur).

## Type Support for `.vue` Imports in TS

TypeScript cannot handle type information for `.vue` imports by default, so we replace the `tsc` CLI with `vue-tsc` for type checking. In editors, we need [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) to make the TypeScript language service aware of `.vue` types.

## Customize configuration

See [Vite Configuration Reference](https://vite.dev/config/).

## Project Setup

```sh
pnpm install
```

### Compile and Hot-Reload for Development

```sh
pnpm dev
```

开发时页面请求保持同源，由 Vite 代理转发到后端，不需要在页面代码里写后端端口。

前端路由使用 Hash 模式，页面地址形如 `/#/signin`。

默认代理到 `http://127.0.0.1:3005`，也可以通过环境变量覆盖：

```sh
VITE_API_PROXY_TARGET=http://127.0.0.1:3005 pnpm dev
```

当前会代理这些入口：

```txt
/api
/openapi.json
/docs
/schemas
```

### Type-Check, Compile and Minify for Production

```sh
pnpm build
```

### Lint with [ESLint](https://eslint.org/)

```sh
pnpm lint
```
