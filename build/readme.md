# SealDice

## 简介

海豹骰工程代码合并仓库，用于实现全平台自动出包。

使用 git submodule 机制整合以下四个仓库的代码
- [sealdice-core](https://github.com/sealdice/sealdice-core)：海豹核心，即海豹的后端工程代码；
- [sealdice-ui](https://github.com/sealdice/sealdice-ui)：海豹的前端工程代码；
- [sealdice-android](https://github.com/sealdice/sealdice-android)：海豹的 Android 工程代码；
- [sealdice-builtins](https://github.com/sealdice/sealdice-builtins)：其他海豹骰子所需的资源文件仓库，包括牌堆、helpdoc 等；
- [go-cqhttp](https://github.com/sealdice/go-cqhttp)：go-cqhttp 的 fork。

克隆该项目时需要使用 `git clone --recursive` 命令以将子模块代码一并拉取。

## 细节

### 自动更新

通过 dependabot 实现自动检查子模块更新，bot 提交的 pr 会被 action 自动批准合并。

dependabot 的配置在 [dependabot.yml](.github/dependabot.yml)，自动批准合并的工作流在 [auto-approve-and-mr.yml](.github/workflows/auto-approve-and-mr.yml)

允许自动合并需要配置相关权限，在仓库的 Settings > Actions > Workflow permissions 中设置。

### 自动构建

工作流为 [auto-build.yml](.github/workflows/auto-build.yml)，相关 jobs 功能：
- `commit-num-check`：用于检查 24 小时内是否有新 commit，没有则每天自动触发的构建不打包；
- `resources-download`：下载资源文件，牌堆、helpdoc、gocghttp 等；
- `gocqhttp-build`,`gocqhttp-android-build`：自动编译所需平台的 gocqhttp，android 端需要使用 NDK；
- `ui-build`：ui自动构建；
- `core-build`,`core-darwin-build`,`core-android-build`：core 的自动构建，分别为 windows&linux macos 和 android；
- `pc-pack`：windows & linux & macos 三端的打包，会组装 helpdoc、gocqhttp 等资源文件；
- `android-build`：android apk 的打包，目前只打包 debug 版本，也会组装资源文件；
- `clear-temp-artifact`：清理产物，保证 artifacts 整洁。

## 关于 issue 和 pull request

你可以通过 fork 本项目并提交 pull request 的形式贡献代码