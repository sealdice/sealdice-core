# sealdice-manual-next

Vuepress2 驱动的海豹骰全新官方文档。

## 编写

文档写在 `docs` 下的 `.md` 文件中，按文件夹分组。

如需调整导航栏和侧边栏，需要修改 `docs/.vuepress/navbar.ts` 和 `docs/.vuepress/sidebar.ts` 中的配置。

## 排版

文档排版应当遵循 [中文文案排版指北](https://github.com/sparanoid/chinese-copywriting-guidelines) 的规范。

## 本地调试

```bash
pnpm install
pnpm run docs:dev
```

运行以上命令启动本地文档服务便于调试。

## Vuepress2 和 Markdown 扩展

文档使用 Vuepress2 驱动，文档见 [Vuepress2 文档](https://v2.vuepress.vuejs.org/zh/)。

同时使用了 `vuepress-plugin-md-enhance` 插件为 Markdown 提供更多扩展语法，文档见 [vuepress-plugin-md-enhance 文档](https://plugin-md-enhance.vuejs.press/zh/guide)。