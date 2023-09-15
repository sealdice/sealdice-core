# SealDice

![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)

海豹TRPG骰点核心

注: 如无特殊说明，所有代码文件均遵循MIT开源协议

## 开发环境搭建

#### 1. golang 开发环境

需求golang版本为1.16以上，推荐为1.18或更新版本。

因部分依赖库的需求，可能需要配置国内镜像，个人使用 https://goproxy.cn/ 镜像


#### 2. 拉取代码并配置数据文件

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

#### 3. 编译运行

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

1.

dice/roll.peg 是海豹的骰点指令文法文件

dice/rollvm.go 是骰点指令虚拟机

这部分代码正在进行大规模的重构，并抽出作为通用模块。

请移步 https://github.com/sealdice/dicescript

2. 

角色数据相关，目前角色卡的实现有点绕，同样是计划进行较大规模的修改，包括角色卡模板、技能名称规范化等机制。

另外还计划在未来更换保存数据的嵌入式数据库，目前使用bblot，而这个库对断电丢失数据不太友好。

