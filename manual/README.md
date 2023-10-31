# sealdice-manual-next

VuePress2 驱动的海豹骰全新官方使用手册。

当前手册预览可见 [海豹手册](https://sealdice.github.io/sealdice-manual-next/)。

## 编写文档

文档写在 `docs` 下的 `.md` 文件中，按文件夹分组。

如需调整导航栏和侧边栏，需要修改 `docs/.vuepress/navbar.ts` 和 `docs/.vuepress/sidebar.ts` 中的配置。

## 排版

文档排版应当遵循 [中文文案排版指北](https://github.com/sparanoid/chinese-copywriting-guidelines) 的规范。

使用 AutoCorrect 插件可以提供相关帮助。
- [VS Code](https://marketplace.visualstudio.com/items?itemName=huacnlee.autocorrect)
- [JetBrains](https://plugins.jetbrains.com/plugin/20244-autocorrect)

## 本地调试

```bash
pnpm install
pnpm run docs:dev
```

运行以上命令启动本地文档服务便于调试。

## 将自己的分支发布为 GitHub Pages

[主仓库](https://github.com/sealdice/sealdice-manual-next) 的 [Pages](https://sealdice.github.io/sealdice-manual-next/) 自动追踪主仓库的 main 分支。

如果你希望请其他人预览修改的效果，或者有其他需求，需要将自己的分支也发布为 GitHub Pages，参考以下步骤：

1. 在你的 fork 仓库，进入 Actions 选项卡；
   - 如果是新 fork 从未运行过 Actions 的仓库，你可能需要点击一个绿色的「I understand my workflows, go ahead and enable them」按钮；
2. 在左侧边栏选择 docs；
3. 在 runs 列表的上方应有一个 banner，内容为「This workflow has a workflow_dispatch event trigger」，选择它右边的「Run workflow」；
4. 在弹出的下拉菜单中选择你的分支，运行 workflow；
5. 等待 workflow 完成，这时你的仓库应多出一个 gh-pages 分支；
6. 进入 Settings 选项卡，左边栏选择 Code and automation 下的 Pages；
7. Source 选择「Deploy from a branch」，Branch 选择「gh-pages」，点击保存；
   - 如果需要，你可在下方的 Custom domain 填写一个自定义的域名，否则，你的 GitHub 用户名将出现在域名中；
8. 回到 Actions，应有一个新的名为 pages-build-deployment 的 workflow 正在运行；
9. 等待 workflow 完成，左边栏选择 Deployments，应能看到一个对应你分支的 Pages 链接；
10. 如需更新你 Pages 的内容，你只需重新进行第 3, 4 两步；发布 Pages 的 workflow 会自动触发。

## VuePress2 和 Markdown 扩展

手册使用 VuePress2 驱动，文档见 [VuePress2 文档](https://v2.vuepress.vuejs.org/zh/)，主题为 [vuepress-theme-hope](https://theme-hope.vuejs.press/zh/)。

同时使用了 `vuepress-plugin-md-enhance` 插件为 Markdown 提供更多扩展语法，文档见 [vuepress-plugin-md-enhance 文档](https://plugin-md-enhance.vuejs.press/zh/guide)。
