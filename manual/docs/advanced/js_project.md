---
lang: zh-cn
title: 插件的工程化编写
---

# 插件的工程化编写

::: info 本节内容

本节将介绍如何使用 Node.js 项目编译出海豹可使用的插件，面向有前端经验的开发者。

我们假定你了解如何使用前端工具链，你应当具备诸如命令行、Node.js、npm/pnpm 等工具的使用知识。如果你对这些内容感到陌生，请自行了解或转至 [使用单 JS 文件编写](./js_start.md#单-js-文件编写插件），手册不会介绍相关内容。

:::

如果你打算使用 TypeScript，或者需要编写大型插件，希望更加工程化以方便维护，可以创建项目使用前端工具链来编译出插件。

海豹提供了相应的 [模板项目](https://github.com/sealdice/sealdice-js-ext-template)。注册扩展和指令的代码已经写好，可以直接编译出一个可直接装载的 JS 扩展文件。

## Clone 或下载项目

推荐的流程：

1. 在 [模板项目仓库](https://github.com/sealdice/sealdice-js-ext-template) 点击 Use this template 按钮，使用该模板在自己的 GitHub 上创建一个扩展的仓库，并设置为自己的扩展的名字；
2. `git clone` 到本地，进行开发。

如果不打算使用 GitHub 托管仓库，希望先在本地编写：

1. 在 [模板项目仓库](https://github.com/sealdice/sealdice-js-ext-template) 点击 Code 按钮，在出现的浮窗中选择 Download ZIP，这样就会下载一个压缩包；
2. 解压后进行开发。

## 使用和编译

在确认你所使用的包管理器后，在命令行使用如下命令安装依赖：

::: tabs key:npm

== npm

```bash
npm install
```

== pnpm

```bash
pnpm install
```

:::

当你写好了代码，需要工程编译为插件的单 js 文件以便上传到海豹骰时，在命令行使用如下命令：

::: tabs key:npm

== npm

```bash
npm run build
```

== pnpm

```bash
pnpm run build
```

:::

编译成功的 js 文件在 `dist` 目录下，默认的名字是 `sealdce-js-ext.js`。

## 补全信息

当插件开发完成后（或者开始开发时），你还需要修改几处地方：

- `header.txt`：这个文件是你插件的描述信息；
- `tools/build-config.js`：最开头一行 `var filename = 'sealdce-js-ext.js';` 改成你中意的名字，注意不要与现有的重名。

## 目录结构

只列出其中主要的一些文件

- `src`
  - `index.ts`：你的扩展的代码就写在这个文件里。
- `tools`
  - `build-config`：一些编译的配置，影响 `index.ts` 编译成 js 文件的方式；
  - `build.js`：在命令 `npm run build` 执行时所运行的脚本，用于读取 `build-config` 并按照配置进行编译。
- `types`
  - `seal.d.ts`：类型文件，海豹核心提供的扩展 API。
- `header.txt`：扩展头信息，会在编译时自动加到目标文件头部；
- `package.json`：命令 `npm install` 时就在安装这个文件里面所指示的依赖包；
- `tsconfig.json`：typescript 的配置。
