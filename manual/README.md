# sealdice-manual-next

VitePress 驱动的海豹骰官方使用手册。

当前手册预览可见 [海豹手册](https://sealdice.github.io/sealdice-manual-next/)。

## 编写文档

文档写在 `docs` 下的 `.md` 文件中，按文件夹分组。

如需调整导航栏和侧边栏，则涉及到修改 `docs/.vitepress/theme.ts` 和 `docs/.vitepress/catalogue.ts` 中的配置。

## Lint

提交前请务必使用 markdownlint 进行检查：

```shell
pnpm install
pnpm run lint:md
pnpm run lint:md:fix // 检查并进行部分自动修复
```

## 排版

文档排版应当遵循 [中文文案排版指北](https://github.com/sparanoid/chinese-copywriting-guidelines) 的规范。

使用 AutoCorrect 插件可以提供相关帮助。

- [VS Code](https://marketplace.visualstudio.com/items?itemName=huacnlee.autocorrect)
- [JetBrains](https://plugins.jetbrains.com/plugin/20244-autocorrect)

## 标注功能版本

对于新加入的功能，需要在标题和正文开头都注明加入的版本。

段落标题的标签应当加在尽可能高等级的标题上，使用以下标签附在标题末尾：

```markdown
## 新加入的功能 <Badge type="tip" text="vA.B.C"/>
```

在紧随以上标题的第一段正文开头，也应该做出文字叙述，例如：

> 从 &lt;Badge type="tip" text="vA.B.C"/&gt; 起，海豹支持某新加入的功能……

## 本地调试

```bash
pnpm install
pnpm run docs:dev
```

运行以上命令启动本地文档服务便于调试。

## 将自己的分支发布为 GitHub Pages

[主仓库](https://github.com/sealdice/sealdice-manual-next) 的 [Pages](https://sealdice.github.io/sealdice-manual-next/) 自动追踪主仓库的 main 分支。

如果你希望请其他人预览修改的效果，或者有其他需求，需要将自己的分支也发布为 GitHub Pages。

我们已经为 fork 仓库配置了 GitHub Actions 自动部署，在你启用后，会将最新推送的 commit 进行部署。如果你是首次使用，请参考以下步骤：

1. 在你的 fork 仓库，进入 Actions 选项卡，你需要点击一个绿色的「I understand my workflows, go ahead and enable them」按钮；
   - 如果你没有看到这个按钮，而是列出了一些 workflow，这说明你已启用过 Actions，可以跳过下面的步骤；
2. 在左侧边栏选择 docs；
3. 在 runs 列表的上方应有一个 banner，内容为「This workflow has a workflow_dispatch event trigger」，选择它右边的「Run workflow」；
4. 在弹出的下拉菜单中选择你的分支，运行 workflow；
5. 等待 workflow 完成，这时你的仓库应多出一个 gh-pages 分支；
6. 进入 Settings 选项卡，左边栏选择 Code and automation 下的 Pages；
7. Source 选择「Deploy from a branch」，Branch 选择「gh-pages」，点击保存；
   - 如果需要，你可在下方的 Custom domain 填写一个自定义的域名，否则，你的 GitHub 用户名将出现在域名中；
8. 回到 Actions，应有一个新的名为 pages-build-deployment 的 workflow 正在运行；
9. 等待 workflow 完成，左边栏选择 Deployments，应能看到一个对应你分支的 Pages 链接；
10. 完成以上步骤后，你每次推送新的 commit 到自己的 fork 仓库，都会自动执行构建和部署，**无需再次手动操作**。

## VitePress

手册使用 VitePress 驱动，文档见 [VitePress](https://vitepress.dev/zh/)。
