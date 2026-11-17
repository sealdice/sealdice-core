# UI 浏览器截图复现指南

最后验证：2026-08-10  
适用仓库：`sealdice-core-newui`  
适用范围：连接真实 SealDice 后端，对 `ui/` 桌面端页面进行可重复的浏览器检查与截图

## 1. 已验证方案

本仓库当前验证稳定的方案是：

- 后端：仓库根目录启动的真实 SealDice Core，监听 `127.0.0.1:3211`。
- 前端：`ui/` 的 Vite 开发服务器，监听 `127.0.0.1:5175`，通过 Vite proxy 访问后端。
- 自动化：`agent-browser 0.31.1`。
- 浏览器：`agent-browser install` 安装的 Chrome for Testing；本次实际版本为 149。
- 字体：对每个浏览器命令显式设置 `FONTCONFIG_FILE=/etc/fonts/fonts.conf`。
- 会话：使用独立 namespace，并为浅色、深色各开一个 session。

Firefox 和 Vivaldi 均安装在当前机器上，不应描述为“不可用”。本次没有继续以它们作为自动截图基线：`agent-browser` 默认的 Chrome for Testing 能稳定提供无头模式、隔离会话、可访问树、直接截图和 DOM 求值，复现成本最低。Vivaldi/Firefox 若用于人工复核可以另行启动，但不应与本流程混用后再比较像素结果。

## 2. 为什么必须连接正确后端

Vite 页面壳能加载，不代表审查环境正确。页面初始化还会调用：

```text
POST /sd-api/v2/base/login
GET  /sd-api/v2/base/overview
GET  /sd-api/v2/base/security-check
GET  /sd-api/v2/config/advanced
```

默认没有设置密码时，前端仍会执行正常登录和安全检查。后端未启动、代理指向错误实例或实例数据不对时，菜单虽然可能出现，但账号、群组、插件、牌堆、备份和设置页面会为空、报错或展示错误状态。因此不要用静态 mock、只启动 Vite，或直接根据页面壳判断截图有效。

后端从当前工作目录读取 `data/`。要复现某一套业务数据，必须在对应的数据目录/仓库根目录启动正确实例；启动前确认不会误用另一份 `data/`。

## 3. 一次性准备

### 3.1 项目依赖

从仓库根目录执行：

```bash
go version
node --version
pnpm --version
go mod download
pnpm --dir ui install
```

Node 版本约束以 `ui/package.json` 为准，目前为 `^20.19.0 || >=22.12.0`。

### 3.2 安装截图工具

```bash
npm install -g agent-browser
agent-browser install
agent-browser --version
agent-browser doctor
```

Linux 缺少 Chrome 运行库时改用：

```bash
agent-browser install --with-deps
```

不要依赖系统的 `google-chrome`、Firefox 或 Vivaldi 路径；`agent-browser install` 会准备与 CLI 匹配的 Chrome for Testing。

### 3.3 验证中文字体

当前 JetBrains Remote 会设置类似下面的变量：

```text
FONTCONFIG_PATH=~/.cache/JetBrains/GoLand2026.1/tmp/jbrd-fontconfig-...
```

该配置面向 JetBrains Runtime，子进程直接继承后可能让无头 Chrome 把中文错误匹配到 `Fira Code`，即使系统实际已经安装微软雅黑和 Noto CJK。先对比：

```bash
fc-match 'Microsoft YaHei'
FONTCONFIG_FILE=/etc/fonts/fonts.conf fc-match 'Microsoft YaHei'
```

正确结果应类似：

```text
msyh.ttc: "Microsoft YaHei" "Regular"
```

只在终端运行 `fc-list :lang=zh-cn` 不能证明浏览器使用了正确配置。后续每条 `agent-browser` 命令都要带 `FONTCONFIG_FILE=/etc/fonts/fonts.conf`，或使用第 5 节的包装函数。

## 4. 启动前后端

### 4.1 检查端口是否已有正确进程

```bash
ss -ltnp 'sport = :3211'
ss -ltnp 'sport = :5175'
```

如果端口已被正确实例占用，直接复用，不要再启动第二份后端。不能确认进程及其数据目录时，先查看完整命令行：

```bash
ps -eo pid,args --width 240 | rg '3211|5175|vite|sealdice'
```

### 4.2 启动真实后端

在终端 A、仓库根目录执行：

```bash
go run . --address=127.0.0.1:3211
```

如果希望先编译，避免每次启动重新编译：

```bash
go build -o /tmp/sealdice-core-newui-dev .
/tmp/sealdice-core-newui-dev --address=127.0.0.1:3211
```

本次截图曾用 `--address 0.0.0.0:3211` 方便远程环境访问，但仅本机截图时优先绑定 `127.0.0.1`。

### 4.3 启动 Vite

在终端 B、仓库根目录执行：

```bash
VITE_API_PROXY_TARGET=http://127.0.0.1:3211 pnpm --dir ui run dev
```

`ui/package.json` 已将端口固定为 5175 并启用 `strictPort`。开发模式应访问 Vite，而不是后端内置静态页面：

```text
正确：http://127.0.0.1:5175
不要：http://127.0.0.1:3211/
```

### 4.4 基础连通性检查

```bash
curl -fsS -o /dev/null http://127.0.0.1:3211/
curl -fsS -o /dev/null http://127.0.0.1:5175/
```

这两条只验证进程和 HTTP。真实 API、认证和代理是否正确，还要在浏览器打开后按第 6 节检查。

## 5. 建立可复用的浏览器命令

在准备截图的终端定义：

```bash
export AB_NAMESPACE=sealdice-ui-review
export AB_SESSION=light

ab() {
  FONTCONFIG_FILE=/etc/fonts/fonts.conf agent-browser \
    --namespace "$AB_NAMESPACE" \
    --session "$AB_SESSION" \
    "$@"
}
```

namespace 隔离守护进程、socket 和恢复状态；session 隔离浏览器上下文及 `localStorage`。不要省略 session 后在同一个上下文中来回切主题，否则截图主题容易受上一页状态污染。

检查函数：

```bash
ab session
ab open http://127.0.0.1:5175
ab set viewport 1440 1000
```

前端使用 Hash Router，直接打开页面时 URL 形如：

```bash
ab open 'http://127.0.0.1:5175/#/mod/js'
```

## 6. 首次进入与环境确认

### 6.1 等待页面完成初始化

```bash
ab wait '.sd-content-pane'
ab wait 800
ab snapshot -i
```

不要只用固定睡眠后立即批量截图。先看到菜单、面包屑和页面主要控件，再继续。

### 6.2 关闭首次安全提示

新 session 可能显示“我已知晓！”按钮。先通过 `snapshot -i` 确认，再点击：

```bash
ab find role button click --name '我已知晓！'
```

该提示会遮挡页面；未关闭时不应截图。可访问树中的 ref 会在页面变化后失效，如使用 `@eN` 操作，点击后必须重新 snapshot。

### 6.3 验证真实后端请求

```bash
ab network requests --filter sd-api
ab errors
```

至少应看到以下请求返回 200：

```text
POST http://127.0.0.1:5175/sd-api/v2/base/login
GET  http://127.0.0.1:5175/sd-api/v2/base/overview
GET  http://127.0.0.1:5175/sd-api/v2/base/security-check
```

还应在页面看到真实业务数据，例如账号状态、日志、群组、牌堆或备份记录。请求 200 但数据与目标实例不符时，仍然不能开始正式截图。

切换页面前可以清空网络记录，便于定位当前页请求：

```bash
ab network requests --clear
```

### 6.4 解锁“高级设置”菜单

“高级设置”默认隐藏。每个 session 都需要点击侧栏品牌标题 8 次：

```bash
ab eval 'for (let i = 0; i < 8; i += 1) document.querySelector(".brand-title")?.click(); "done"'
```

然后展开“综合设置”并重新 snapshot：

```bash
ab find text '综合设置' click --exact
ab snapshot -i
```

输出中应出现“高级设置”。这是前端临时状态，不会修改后端配置；刷新或新建 session 后可能需要重做。

## 7. 管理浅色和深色会话

### 7.1 浅色会话

```bash
export AB_SESSION=light
ab open http://127.0.0.1:5175
ab set viewport 1440 1000
```

检查实际主题，而不是假定新 session 一定为浅色：

```bash
ab eval 'document.documentElement.classList.contains("dark") ? "dark" : "light"'
```

若输出为 `dark`：

```bash
ab find role button click --name '切换到亮色模式'
```

### 7.2 深色会话

```bash
export AB_SESSION=dark
ab open http://127.0.0.1:5175
ab set viewport 1440 1000
ab snapshot -i
```

按需关闭首次安全提示、解锁高级设置。若当前输出为 `light`，执行：

```bash
ab find role button click --name '切换到深色模式'
```

验证：

```bash
ab eval '({ theme: document.documentElement.classList.contains("dark") ? "dark" : "light", background: getComputedStyle(document.body).backgroundColor })'
```

不要在复用 session 时无条件点击主题按钮；它是 toggle，会把已经正确的主题切反。

## 8. 页面导航与标准截图

### 8.1 建立归档目录

```bash
export SHOT_ROOT="$PWD/docs/ui-review-assets/$(date +%F)"
mkdir -p "$SHOT_ROOT"/{light,dark}/{1440x1000,1280x800}
```

命名规则：

```text
route-slug--state.png
```

示例：

```text
home--default.png
mod-helpdoc--item.png
misc-base-setting--platform.png
custom-text-entertainment--default-full.png
```

### 8.2 打开路由

```bash
export AB_SESSION=light
ab open 'http://127.0.0.1:5175/#/mod/helpdoc'
ab wait '.sd-content-pane'
ab wait 500
ab snapshot -i
```

中文路由需要整体加引号，例如：

```bash
ab open 'http://127.0.0.1:5175/#/custom-text/娱乐'
```

### 8.3 截取默认态

```bash
ab set viewport 1440 1000
ab screenshot "$SHOT_ROOT/light/1440x1000/mod-helpdoc--default.png"

ab set viewport 1280 800
ab screenshot "$SHOT_ROOT/light/1280x800/mod-helpdoc--default.png"
```

切换深色 session 后重复：

```bash
export AB_SESSION=dark
ab open 'http://127.0.0.1:5175/#/mod/helpdoc'
ab wait '.sd-content-pane'
ab wait 500
ab set viewport 1440 1000
ab screenshot "$SHOT_ROOT/dark/1440x1000/mod-helpdoc--default.png"
```

### 8.4 截取页签/筛选状态

先 snapshot，优先使用语义定位：

```bash
ab snapshot -i
ab find role tab click --name '词条'
ab wait 300
ab screenshot "$SHOT_ROOT/light/1440x1000/mod-helpdoc--item.png"
```

找不到稳定语义角色时，使用 snapshot 产生的 ref：

```bash
ab snapshot -i
ab click @e42
ab snapshot -i
```

ref 在导航、页签切换、弹窗打开和动态重渲染后都会失效，不要把 `@e42` 等编号写进长期脚本。

截图审查只切换导航、页签、筛选和本地展示模式。保存、删除、清理、恢复、导入、上传等会改变后端数据的操作不得为了“补状态”而点击。

## 9. Naive UI 内部滚动页的完整高度截图

### 9.1 为什么 `screenshot --full` 不够

本应用的主内容不是 document 滚动，而是：

```css
.sd-content-pane .n-scrollbar-container
```

因此：

```bash
ab screenshot --full page.png
```

只会按 document 高度截图。document 仍是 `1440×1000` 时，即使内部内容高达 4000px，结果仍可能只有 1000px，看起来像“完整截图成功”，实际下半页缺失。

先比较：

```bash
ab eval '(() => { const el = document.querySelector(".sd-content-pane .n-scrollbar-container"); return { viewportHeight: innerHeight, documentHeight: document.documentElement.scrollHeight, clientHeight: el?.clientHeight, contentHeight: el?.scrollHeight }; })()'
```

当 `contentHeight > clientHeight` 且 `documentHeight == clientHeight` 时，必须使用下面的 viewport 扩高方案。

### 9.2 已验证的完整高度方案

先恢复目标宽度并计算内部内容高度：

```bash
ab set viewport 1440 1000
ab eval 'document.querySelector(".sd-content-pane .n-scrollbar-container")?.scrollTo({ top: 0, left: 0 }); "top"'

HEIGHT="$(ab eval 'Math.max(1000, Math.ceil(document.querySelector(".sd-content-pane .n-scrollbar-container")?.scrollHeight || document.documentElement.scrollHeight))')"
printf 'content height: %s\n' "$HEIGHT"
```

扩高 viewport 后使用普通截图，不再加 `--full`：

```bash
ab set viewport 1440 "$HEIGHT"
ab screenshot "$SHOT_ROOT/light/1440x1000/home--default-full.png"
ab set viewport 1440 1000
```

本次验证中该方法正确生成过：

- 首页约 4341px（日志继续增长后高度会变化）。
- “娱乐”文案 7887px。
- 牌堆 2149px。
- 关于页 4194px。

完整截图后必须立即恢复标准 viewport，否则后续 `1280×800` 截图会继承错误高度。

### 9.3 极端长页

若 `HEIGHT` 超过约 10000px，优先修正页面本身的无边界列表；仅为取证时，可固定 `1440×1000`，分段修改内部容器的 `scrollTop` 并截图，文件名追加 `--part-01`。分段间保留约 100px 重叠，避免遗漏边界内容。不要无限增大 viewport，超大位图会增加浏览器内存和 PNG 编码时间。

## 10. 当前页面和主状态清单

完整截图索引以 [视觉审查补充报告](ui-visible-pages-visual-review-2026-08-10.md) 为准。重新审查时至少覆盖：

| 页面组 | 路由/状态 |
| --- | --- |
| 顶层 | `/`、`/connect`、`/about` |
| 自定义文案 | `/custom-text/COC`、`DND`、`其它`、`娱乐`、`日志`、`核心`；全部、未修改、修改过、指定分组、旧版 |
| 扩展功能 | `/mod/reply`、`deck`、`package`、`story`、`js`、`helpdoc`、`censor` |
| 插件包 | 已安装、商店、管理 |
| 剧情卡 | 列表、清理、备份 |
| JS 插件 | 列表、控制台、配置、数据 |
| 帮助文档 | 文件、词条 |
| 敏感词 | 设置、词库、日志 |
| 综合设置 | `/misc/base-setting`、`group`、`ban`、`dice-public`、`backup`、`advanced-setting` |
| 基础设置 | Master 与通知、行为与消息、刷屏警告、访问与安全、平台特殊配置、游戏与扩展、维护与低频 |
| 黑白名单 | 列表、配置 |
| 备份 | 设置、文件 |
| 辅助工具 | `/tool/test` 私聊/群聊、`/tool/resource`、`/tool/profile` |

默认态、非默认一级状态均应在双主题、双桌面宽度下截图。每个页面另外保留一张浅色 1440px 完整高度图，用于发现页面级长滚动和栅格失衡。

## 11. 截图后校验

### 11.1 数量和目录

```bash
find "$SHOT_ROOT" -type f -name '*.png' | wc -l

for dir in "$SHOT_ROOT"/{light,dark}/{1440x1000,1280x800}; do
  printf '%s ' "$dir"
  find "$dir" -maxdepth 1 -type f -name '*.png' | wc -l
done
```

2026-08-10 的基线是 301 张：

```text
light/1440x1000  94
light/1280x800   69
dark/1440x1000   69
dark/1280x800    69
```

### 11.2 PNG 尺寸

当前环境没有 ImageMagick `identify`，使用系统 `file`：

```bash
find "$SHOT_ROOT" -type f -name '*.png' ! -name '*-full.png' -exec file {} +
```

标准图只应出现 `1440 x 1000` 或 `1280 x 800`。完整高度图应为 1440px 宽：

```bash
find "$SHOT_ROOT/light/1440x1000" -type f -name '*-full.png' -exec file {} +
```

检查可疑空白文件：

```bash
find "$SHOT_ROOT" -type f -name '*.png' -size -10000c -print
```

有输出时必须人工查看。2026-08-10 基线中最小 PNG 为 52855 bytes。

### 11.3 页面状态和错误

每个页面至少抽查：

```bash
ab get url
ab snapshot -i
ab errors
ab network requests --filter sd-api
```

确认事项：

- URL 与文件名对应。
- 主标题、菜单当前项、页签状态正确。
- API 请求不是 401、404、500。
- 中文不是方框、乱码或 Fira Code 风格的错误 fallback。
- 浅色/深色实际生效，不仅文件夹名称不同。
- 不存在加载遮罩、首次安全提示、打开的菜单或 Tooltip 意外遮挡内容。

## 12. 常见问题

### 12.1 中文字体明明存在，截图仍异常

原因通常不是字体缺失，而是 JetBrains Remote 的 `FONTCONFIG_PATH` 被浏览器继承。对每条 `agent-browser` 命令设置：

```bash
FONTCONFIG_FILE=/etc/fonts/fonts.conf agent-browser ...
```

关闭旧 session 后重新启动；已启动浏览器不会因终端环境变化自动重建字体配置。

### 12.2 页面能打开，但内容为空或异常

依次检查：

```bash
curl -fsS -o /dev/null http://127.0.0.1:3211/
curl -fsS -o /dev/null http://127.0.0.1:5175/
ab network requests --filter sd-api
ab errors
```

重点确认 Vite 启动时的 `VITE_API_PROXY_TARGET`、后端进程工作目录和实际数据实例。默认无密码不代表不需要后端认证请求。

### 12.3 `--full` 仍只有 1000px 高

这是 Naive UI 内部滚动容器导致的预期现象。使用第 9 节的 `.n-scrollbar-container.scrollHeight` + viewport 扩高方案。

### 12.4 主题截图反了

session 会保留 `localStorage`。先执行：

```bash
ab eval 'document.documentElement.classList.contains("dark") ? "dark" : "light"'
```

然后按按钮的当前可访问名称切换，不要无条件 toggle。必要时关闭该 session，换一个新 namespace/session 重建。

### 12.5 找不到“高级设置”

每个新 session 点击 `.brand-title` 8 次，再展开“综合设置”。只打开 `/misc/advanced-setting` URL 不等于已经验证菜单可见性。

### 12.6 snapshot 的 `@eN` 突然无效

refs 只对当前可访问树有效。导航、动态加载、页签切换、弹窗打开后重新运行：

```bash
ab snapshot -i
```

长期步骤优先使用 `find role`、`find text`、`find label` 等语义定位。

### 12.7 底部出现圆形下箭头

开发模式启用了 `vite-plugin-vue-devtools`。底部居中圆形下箭头的可访问名称为 `Toggle devtools panel`，不属于产品 UI，不应记为产品缺陷。正式产品截图应使用 production build；开发态审查报告中需明确排除该浮标。

### 12.8 浏览器会话卡住或残留

先关闭当前 session：

```bash
ab close
```

需要清理当前 namespace 的全部 session 时：

```bash
FONTCONFIG_FILE=/etc/fonts/fonts.conf agent-browser --namespace "$AB_NAMESPACE" close --all
```

再运行：

```bash
agent-browser doctor
agent-browser --namespace "$AB_NAMESPACE" session list
```

不要把 Vivaldi/Firefox 的 GUI 进程和 `agent-browser` Chrome session 当成同一个故障域。

## 13. 收尾

完成截图后分别关闭会话：

```bash
export AB_SESSION=light
ab close

export AB_SESSION=dark
ab close
```

前后端若是本次临时启动，在各自终端用 `Ctrl+C` 正常退出。不要使用 `kill -9` 终止正在写入数据的后端。

最终提交范围通常只应包含审查报告和 `docs/ui-review-assets/`。开始前后均运行 `git status --short`，不要把后端运行产生的数据变化、生成 API、依赖安装产物或其他人的工作区改动混入截图任务。
