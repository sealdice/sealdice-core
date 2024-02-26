# SealDice

![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)

海豹TRPG骰点核心

项目的二进制构建见 [sealdice-build](https://github.com/sealdice/sealdice-build/releases) 子项目

注: 如无特殊说明，所有代码文件均遵循MIT开源协议

## 开发环境搭建

### 1. golang 开发环境

编译的 golang 版本为 1.20。使用更新版本时需注意不要使用新版本引入的新函数。

因部分依赖库的需求，可能需要配置国内镜像，个人使用 <https://goproxy.cn/> 镜像。

#### 代码格式化

本项目要求所有代码使用 goimports 进行格式化。这一行为已经设定在本项目的编辑器配置文件中。

#### 在本地配置 LINTER

本项目使用 golangci-lint 工具进行静态分析。

此工具对于代码开发**不是**必要的。但是，本项目的 CI 流程中配置了 linter 检查，不符合规范的代码不能被合入。

因此，强烈推荐开发者在本地安装此工具，请参考[这份文档](https://golangci-lint.run/usage/install/#local-installation)。分析器的相关配置位于 `.golangci.yml` 文件中。

你可能需要调整编辑器的相关配置，使用 golangci-lint 为默认的分析工具，并开启自动检查。

> 对于 Visual Studio Code，列出以下配置项供参考：
>
> 1. `go.lintTool` 选择 golangci-lint
> 2. `go.lintFlags` 添加一项 `--fast`
> 3. `go.lintOnSave` **不能**选择 file，因为只分析单个文件会导致无法正确解析符号引用
>
> 以上配置没有写入项目的统一设置，以允许开发者不本地使用 golangci-lint

### 2. 拉取代码并配置数据文件

使用git拉取项目代码

从已发布的海豹二进制包中，解压 `data`、`gocqhttp` 两个目录到代码目录下。

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

### 3. 编译运行

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

## 重点

### 从哪开始看

从 main.go 开始，这里海豹分出了几个线程，一个启动核心并提供服务，另一个提供ui的http服务。

可以顺藤摸瓜了解海豹如何启动，如何提供服务，如何响应指令。指令响应的部分写在im_session.go中

注意有部分代码还在构思中，实际并未使用，例如 CharacterTemplate，请阅读时先Find Usage加以区分

### 重要数据结构

dice.go 中的 Dice 结构体存放着各种核心配置，每个Dice实例是一个骰子，而每个骰子下面可以挂靠多个端点(EndPoint)。端点即交互渠道，例如一个QQ账号是一个端点。

所有的端点由 IMSession 来统一管理，同样的，这个类也负责接收和分发指令。

可能你会注意到有 IMSession 和 IMSessionLegacy，只看前一个就行，IMSessionLegacy对应的是0.99.13的上古版本之前的数据结构，仅用于升级配置文件。

GroupInfo 是群组信息

GroupPlayerInfo 是玩家信息

### 为海豹添加更多平台支持

海豹使用叫做 PlatformAdapter 的接口来接入平台，只需将接口全部实现，再创建一个 EndPointInfo 塞入当前用户的 IMSession 对象之中即可。

注: 每次在UI上添加QQ账号，其实就是创建了一个EndPointInfo对象，并制定Adapter为PlatformAdapterQQOnebot

目前实现的两个adapter，一个对应onebot协议，主要用于QQ，另一个对应http，用于UI后台的测试窗口。

观察 PlatformAdapterHttp 如何运作起来是一个很好的切入点，因为他非常简单。

### 改动扩展模块，如dnd5e，coc7等

对应 dice/ext_xxx.go 系列文件

推荐从 ext_template.go 入手，以 ext_dnd5e.go 为参考，因为这个模块书写时间较晚，相对较为完善。

### 暂不建议修改的地方

#### 1. 表达式解析器

dice/roll.peg 是海豹的骰点指令文法文件

dice/rollvm.go 是骰点指令虚拟机

预计将于1.5版本上线重构后的 dicescript (RollVM V2) 作为表达式解释器

请移步 <https://github.com/sealdice/dicescript>

而出于兼容性的考虑，V1版本的解释器将继续保留，直到2.0版本。

#### 2. 角色卡

角色数据相关，包括存取机制和相关API，这部分代码将于1.5版本进行重制。
