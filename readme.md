# SealDice

![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)
![Core](https://img.shields.io/badge/SealDice-Core-blue)

海豹 TRPG 骰点核心，开源跑团辅助工具，支持 QQ/Kook/Discord 等。

轻量 · 易用 · 全能

## 文档

见 [使用手册](https://sealdice.github.io/sealdice-manual-next/)。

## SealDice Project

- [核心](https://github.com/sealdice/sealdice-core)（本仓库）：Go 后端代码仓库，也作为海豹的主仓库，Bug 可反馈在该仓库的 issue 中；
- [UI](https://github.com/sealdice/sealdice-ui)：前端代码，基于 Vue3 + ElementPlus 开发；
- [手册](https://github.com/sealdice/sealdice-manual-next)：官方手册源码，由 VitePress 驱动；
- [构建](https://github.com/sealdice/sealdice-build)：自动构建仓库，用于自动化发布海豹的每日构建包与 Release；
- [Android](https://github.com/sealdice/sealdice-android)：Android 应用源码；
- ……

注：如无特殊说明，所有代码文件均遵循 MIT 开源协议

## Core 开发环境搭建

### golang 开发环境

编译的 golang 版本为 1.25.0。在 [构建](https://github.com/sealdice/sealdice-build) 仓库中采用对 go 进行修补的方式以支持 Windows 7 等低版本系统。

因部分依赖库的需求，可能需要配置国内镜像，个人使用 <https://goproxy.cn/> 镜像。

#### 代码格式化

本项目要求所有代码使用 goimports 进行格式化。这一行为已经设定在本项目的编辑器配置文件中。

#### 在本地配置 LINTER

本项目使用 golangci-lint 工具进行静态分析。

此工具对于代码开发**不是**必要的。但是，本项目的 CI 流程中配置了 linter 检查，不符合规范的代码不能被合入。

因此，强烈推荐开发者在本地安装此工具，请参考[这份文档](https://golangci-lint.run/welcome/install/#local-installation)。分析器的相关配置位于 `.golangci.yml` 文件中。

你可能需要调整编辑器的相关配置，使用 golangci-lint 为默认的分析工具，并开启自动检查。

> 对于 Visual Studio Code，列出以下配置项供参考：
>
> 1. `go.lintTool` 选择 golangci-lint
> 2. `go.lintFlags` 添加一项 `--fast`
> 3. `go.lintOnSave` **不能**选择 file，因为只分析单个文件会导致无法正确解析符号引用
>
> 以上配置没有写入项目的统一设置，以允许开发者不本地使用 golangci-lint

### 编译运行

下面的命令默认从仓库根目录执行，即包含 `go.mod`、`main.go`、`ui/` 的目录。

#### 使用 `go-task`

你可以安装 [go-task](https://taskfile.dev/installation) 以执行预置好的任务。安装后可执行：

```bash
# 初次编译运行（包括安装依赖和相关工具）
task install run 

# 后续编译运行
task run
```

#### 手动执行

你也可以按照以下步骤手动进行编译运行：

##### 拉取代码并配置数据文件

使用 git 拉取项目代码

从已发布的海豹二进制包中，解压 `data`、`lagrange` 两个目录到代码目录下。

同时需要在项目的 `static/frontend` 下放置用于打包进 core 的 ui 静态资源文件，可手动提供，也可通过命令自动从 github 拉取：

```bash
go generate ./...
```

放置静态资源大致如下：

```text
static
│
└─frontend
   │  CNAME
   │  favicon.svg
   │  index.html
   │
   └─assets
```

##### 运行编译命令

打开项目，或使用终端访问项目目录，运行：

```bash
go mod download
go install github.com/pointlander/peg@v1.0.1
go build
```

或者直接使用：

```shell
go run .
```

启动项目，大功告成！

### 新版 V2 UI / OpenAPI 开发流程

本仓库同时包含 Go 后端和新版管理前端：

- 后端入口：仓库根目录的 `main.go`
- 前端目录：`ui/`
- OpenAPI 描述文件：`ui/openapi.json`
- 前端生成代码：`ui/src/api/generated/`
- 前端开发服务器：默认 `http://127.0.0.1:5175`
- 后端 UI/API 服务：默认监听 `0.0.0.0:3211`，本机访问使用 `http://127.0.0.1:3211`

`ui/openapi.json` 和 `ui/src/api/generated/` 是生成产物，已被 `.gitignore` 忽略，不要手工编辑，也不要提交。

#### 0. 安装可复现的本地依赖

后端需要 Go，当前项目编译版本见上文“golang 开发环境”。前端需要 Node.js 和 pnpm，版本约束写在 `ui/package.json`：

```bash
node --version
pnpm --version
go version
```

第一次准备仓库时执行：

```bash
cd /path/to/sealdice-core-newui
go mod download
pnpm --dir ui install
```

如果需要使用 `task` 统一执行构建任务，还需要安装 [go-task](https://taskfile.dev/installation)。不使用 `task` 时，下面所有步骤都可以直接用 `go`、`pnpm` 手动复现。

#### 1. 编译后端

只编译 Go 后端，不构建新版前端：

```bash
cd /path/to/sealdice-core-newui
go build .
```

该命令会在当前目录生成 `sealdice-core`（Windows 下为 `sealdice-core.exe`）。如果只是临时运行，可以跳过二进制产物，直接使用：

```bash
cd /path/to/sealdice-core-newui
go run .
```

如果要把新版 V2 UI 一起构建并嵌入到后端静态资源中，使用：

```bash
cd /path/to/sealdice-core-newui
task build-with-v2ui
```

`task build-with-v2ui` 会先执行 `test-and-lint` 依赖，再执行构建。完整展开后包含：

```bash
go test ./...
go vet ./...
goimports -w .
golangci-lint run
mkdir -p temp
go build -o temp/openapi-gen .
./temp/openapi-gen --gen-openapi=./ui/openapi.json
pnpm --dir ui run generate-client
pnpm --dir ui run build:embed:prepared
go build .
```

因此它会先完成 Go 测试和 lint，再刷新 OpenAPI、生成前端 API 客户端、构建可嵌入的 V2 UI，并最终编译后端。注意 `goimports -w .` 会直接格式化 Go 文件；如果只想避免自动格式化，请使用下面的手动命令链。

#### 2. 生成 `ui/openapi.json`

后端提供专用参数 `--gen-openapi`，用于生成 Huma v2 OpenAPI JSON 后退出。

推荐直接使用前端脚本：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run generate-openapi
```

该脚本等价于：

```bash
cd /path/to/sealdice-core-newui
go run . --gen-openapi=./ui/openapi.json
```

执行成功后，应出现或更新：

```text
ui/openapi.json
```

如果只想确认该文件是否生成：

```bash
cd /path/to/sealdice-core-newui
test -f ui/openapi.json && echo "ui/openapi.json exists"
```

#### 3. 让前端生成对应 TypeScript API 代码

前端使用 `@hey-api/openapi-ts` 读取 `ui/openapi.json`，并把 TypeScript 类型、axios client、SDK、Vue Query options 生成到 `ui/src/api/generated/`。

如果已经有 `ui/openapi.json`，只生成前端代码：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run generate-client
```

如果要从后端重新生成 OpenAPI，再生成前端代码，使用完整命令：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run generate-api
```

`pnpm --dir ui run generate-api` 等价于：

```bash
cd /path/to/sealdice-core-newui/ui
pnpm run generate-openapi
pnpm run generate-client
```

执行成功后，应出现或更新：

```text
ui/openapi.json
ui/src/api/generated/
```

生成代码后建议立刻做一次前端类型检查：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run type-check
```

#### 4. 运行后端

本地开发推荐显式指定后端监听地址，避免不同平台或已有配置影响端口：

```bash
cd /path/to/sealdice-core-newui
go run . --address=127.0.0.1:3211
```

也可以先编译再运行：

```bash
cd /path/to/sealdice-core-newui
go build .
./sealdice-core --address=127.0.0.1:3211
```

Windows PowerShell 下运行编译产物：

```powershell
cd C:\path\to\sealdice-core-newui
go build .
.\sealdice-core.exe --address=127.0.0.1:3211
```

后端启动后，浏览器访问：

```text
http://127.0.0.1:3211
```

V2 UI 的内置访问路径是：

```text
http://127.0.0.1:3211/v2ui/
```

如果没有构建并嵌入 V2 UI，`/v2ui/` 会显示占位页面；这不影响 API 调试。API 路径仍然是同一个后端服务下的：

```text
http://127.0.0.1:3211/sd-api/v2
```

#### 5. 让前端调试正确连接后端

前端开发态通过 Vite dev server 启动，页面地址默认是：

```text
http://127.0.0.1:5175
```

前端代码本身使用同源 API 地址。开发时，Vite 会把这些同源请求代理到后端。代理目标由 `ui/vite.config.ts` 读取，优先级如下：

1. `VITE_API_PROXY_TARGET`
2. `DEV_PROXY_SERVER`
3. `VITE_API_BASE_URL`
4. 默认值 `http://localhost:3211`

推荐在两个终端中分别运行后端和前端。

终端 A，运行后端：

```bash
cd /path/to/sealdice-core-newui
go run . --address=127.0.0.1:3211
```

终端 B，运行前端开发服务器：

```bash
cd /path/to/sealdice-core-newui
VITE_API_PROXY_TARGET=http://127.0.0.1:3211 pnpm --dir ui run dev
```

Windows PowerShell 下设置环境变量并启动前端：

```powershell
cd C:\path\to\sealdice-core-newui
$env:VITE_API_PROXY_TARGET = "http://127.0.0.1:3211"
pnpm --dir ui run dev
```

启动后访问：

```text
http://127.0.0.1:5175
```

前端调试时不要直接把页面打开到 `http://127.0.0.1:3211/v2ui/`，那是后端内置静态资源路径；开发模式应访问 Vite 的 `5175` 端口。Vite 会代理以下路径到后端：

```text
/api
/sd-api
/openapi.json
/docs
/schemas
```

如果后端改了端口，例如：

```bash
cd /path/to/sealdice-core-newui
go run . --address=127.0.0.1:4000
```

前端也必须使用相同端口作为代理目标：

```bash
cd /path/to/sealdice-core-newui
VITE_API_PROXY_TARGET=http://127.0.0.1:4000 pnpm --dir ui run dev
```

#### 6. 前端构建与检查

仅做类型检查：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run type-check
```

仅做 Vite 生产构建，要求已经存在 `ui/src/api/generated/`：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run build-only
```

完整前端构建，会自动重新生成 OpenAPI 和前端 API 代码：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run build
```

如需构建可嵌入 Go 后端的前端产物：

```bash
cd /path/to/sealdice-core-newui
pnpm --dir ui run build:embed
```

## 重点

### 从哪开始看

从 `main.go` 开始，这里海豹分出了几个线程，一个启动核心并提供服务，另一个提供 ui 的 http 服务。

可以顺藤摸瓜了解海豹如何启动，如何提供服务，如何响应指令。指令响应的部分写在 `im_session.go` 中

注意有部分代码还在构思中，实际并未使用，例如 `CharacterTemplate`，请阅读时先 Find Usage 加以区分

### 重要数据结构

`dice.go` 中的 `Dice` 结构体存放着各种核心配置，每个 `Dice` 实例是一个骰子，而每个骰子下面可以挂靠多个端点 (EndPoint)。端点即交互渠道，例如一个 QQ 账号是一个端点。

所有的端点由 `IMSession` 来统一管理，同样的，这个类也负责接收和分发指令。

可能你会注意到有 `IMSession` 和 `IMSessionLegacy`，只看前一个就行，`IMSessionLegacy` 对应的是 0.99.13 的上古版本之前的数据结构，仅用于升级配置文件。

`GroupInfo` 是群组信息

`GroupPlayerInfo` 是玩家信息

### 为海豹添加更多平台支持

海豹使用叫做 `PlatformAdapter` 的接口来接入平台，只需将接口全部实现，再创建一个 `EndPointInfo` 塞入当前用户的 `IMSession` 对象之中即可。

注：每次在 UI 上添加 QQ 账号，其实就是创建了一个 `EndPointInfo` 对象，并制定 Adapter 为 `PlatformAdapterQQOnebot`

目前实现的两个 adapter，一个对应 onebot 协议，主要用于 QQ，另一个对应 http，用于 UI 后台的测试窗口。

观察 `PlatformAdapterHttp` 如何运作起来是一个很好的切入点，因为他非常简单。

### 改动扩展模块，如 dnd5e，coc7 等

对应 `dice/ext_xxx.go` 系列文件

推荐从 `ext_template.go` 入手，以 `ext_dnd5e.go` 为参考，因为这个模块书写时间较晚，相对较为完善。

### 暂不建议修改的地方

#### 表达式解析器

`dice/roll.peg` 是海豹的骰点指令文法文件

`dice/rollvm.go` 是骰点指令虚拟机

1.5 后，已经替换使用 dicescript (RollVM V2) 作为表达式解释器，现有版本不宜轻动。

关于 dicescript 的信息，请移步 <https://github.com/sealdice/dicescript>

而出于兼容性的考虑，V1 版本的解释器将继续保留，直到 2.0 版本。
