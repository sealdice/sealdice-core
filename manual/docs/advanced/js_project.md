---
lang: zh-cn
title: 插件的工程化编写
---

# 插件的工程化编写

::: info 本节内容

本节将介绍如何使用 Node.js 项目编译出海豹可使用的插件，面向有前端经验的开发者。

我们假定你了解如何使用前端工具链，你应当具备诸如命令行、Node.js、npm/pnpm 等工具的使用知识。如果你对这些内容感到陌生，请自行了解或转至 [使用单 JS 文件编写](./js_start.md#单-js-文件编写插件)，手册不会介绍这些相关背景知识。

:::

如果你打算使用 TypeScript，或者需要编写大型插件，希望更加工程化以方便维护，可以创建项目使用前端工具链来编译出插件。

海豹提供了相应的 [模板项目](https://github.com/sealdice/sealdice-js-ext-template)。注册扩展和指令的代码已经写好，可以直接编译出一个可直接装载的 JS 扩展文件。

## Clone 或下载模板项目

推荐的流程：

1. 在 [模板项目仓库](https://github.com/sealdice/sealdice-js-ext-template) 点击 Use this template 按钮，使用该模板在自己的 GitHub 上创建一个扩展的仓库，并设置为自己的扩展的名字；
2. `git clone` 到本地，进行开发。

如果不打算使用 GitHub 托管仓库，希望先在本地编写：

1. 在 [模板项目仓库](https://github.com/sealdice/sealdice-js-ext-template) 点击 Code 按钮，在出现的浮窗中选择 Download ZIP，这样就会下载一个压缩包；
2. 解压后进行开发。

## 补全信息

当插件开发完成后（或者开始开发时），你还需要修改几处地方：

- `header.txt`：这个文件是你插件的描述信息；
- `tools/build-config.js`：最开头一行 `var filename = 'sealdce-js-ext.js';` 改成你中意的名字，注意不要与现有的重名。**这决定了编译时输出的插件文件名。**
- （可选）`package.json`：修改其中 `name` `version` `description` 等项目描述信息，不过不修改也不会影响编译。

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

## 目录结构

只列出其中主要的一些文件

- `src`
  - `index.ts`：你的扩展的代码就写在这个文件里。
- `tools`
  - `build-config.js`：一些编译的配置，影响 `index.ts` 编译成 js 文件的方式；
  - `build.js`：在命令 `npm run build` 执行时所运行的脚本，用于读取 `build-config` 并按照配置进行编译。
- `types`
  - `seal.d.ts`：类型文件，海豹核心提供的扩展 API。
- `header.txt`：扩展头信息，会在编译时自动加到目标文件头部；
- `package.json`：命令 `npm install` 时就在安装这个文件里面所指示的依赖包；
- `tsconfig.json`：TypeScript 的配置文件。

## 其他问题

### 我能在项目中引用 npm 包吗？

当然可以，像正常的前端项目一样，你可以在其中引用其他 npm 包，比如模板项目中就为你引入了常用的 `lodash-es`。

一般来说纯 JS 编写的包都是可以引用的，一些强 native 相关的包可能存在兼容性问题，你需要自行尝试。

推荐你尽量使用 esm 格式的包，不过 commonjs 格式的包也是可以使用的，如 `dayjs`。其他格式的支持和更多问题排查，请查阅模板项目所使用的构建工具 esbuild 的文档，`tools/build-config.js` 中即是 esbuild 的配置项。

### 我想使用的 API 没有被自动提示，直接使用被提示错误，如何解决？

`types/seal.d.ts` 文件中维护了海豹提供的 API，但目前来说维护的并不完全。如果你发现有一些存在的 API 未被提示，可以手动在 `types/seal.d.ts` 补上来解决报错。

**有时 `seal.d.ts` 会有更新，可以去模板项目仓库看看有没有最新的，有的话可以替换到你的项目中。也非常欢迎你向模板仓库提 PR 来帮忙完善。**

### 默认输出的插件代码是压缩过的，如何尽量保持产物的可读性？

调整 `tools/build-config.js` 中的选项，关闭 minify：

```js{5}
module.exports = {
  ...
  build: {
    ...
    minify: false,
    ...
  }
}
```
