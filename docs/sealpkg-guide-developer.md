# 海豹扩展包开发指南

本文档面向希望为海豹骰开发扩展包的开发者。

## 概述

海豹扩展包（`.sealpkg`）是一种 ZIP 格式的打包文件，可以包含脚本、牌堆、自定义回复等多种资源。与传统的散装文件分发方式相比，扩展包具有以下优势：

- **一键安装**：用户无需手动复制文件到各个目录
- **版本管理**：支持版本号、依赖声明、更新检查
- **权限隔离**：脚本只能访问声明的权限，保护用户安全
- **配置界面**：可定义配置项 schema，自动生成配置 UI
- **干净卸载**：卸载时不会残留文件

## 快速开始

### 1. 创建目录结构

```
my-extension/
├── manifest.toml      # 必需：包元数据
├── scripts/           # JS/TS 脚本
│   └── main.js
├── decks/             # 牌堆文件
│   └── mytable.toml
├── reply/             # 自定义回复
├── helpdoc/           # 帮助文档
└── assets/            # 静态资源（图片等）
```

### 2. 编写 manifest.toml

```toml
[package]
id = "你的名字/my-extension"       # 唯一标识符（必须为 "作者/包名" 格式）
name = "我的扩展包"                  # 显示名称
version = "1.0.0"                   # 语义化版本号
authors = ["你的名字"]
license = "MIT"
description = "这是一个示例扩展包"
homepage = "https://github.com/yourname/my-extension"
keywords = ["COC", "工具"]

[package.seal]
min_version = "1.4.6"              # 最低海豹版本要求

[dependencies]
# 依赖其他扩展包（可选）
# "com.seal.coc7" = ">=1.0.0"

[permissions]
network = false                    # 是否需要网络访问
file_read = ["assets/*"]           # 可读取的文件路径
file_write = []                    # 可写入的文件路径（默认只能写 _userdata/）

[contents]
scripts = ["scripts/*.js"]
decks = ["decks/*.toml"]
```

### 3. 编写脚本

脚本格式与现有的海豹 JS 插件相同，使用 UserScript 头部声明元数据：

```javascript
// ==UserScript==
// @name         我的扩展
// @version      1.0.0
// @author       你的名字
// @description  扩展功能描述
// ==/UserScript==

let ext = seal.ext.find("my-extension");
if (!ext) {
  ext = seal.ext.new("my-extension", "你的名字", "1.0.0");
  seal.ext.register(ext);
}

// 你的代码...
```

### 4. 打包

将目录打包为 ZIP 文件，并将扩展名改为 `.sealpkg`：

```bash
cd my-extension
zip -r ../my-extension.sealpkg .
```

或使用任意压缩工具，确保 `manifest.toml` 在压缩包根目录。

## manifest.toml 完整参考

### [package] 基础信息

| 字段 | 必需 | 说明 |
|-----|------|------|
| `id` | 是 | 唯一标识符，必须为 `作者/包名` 格式 |
| `name` | 是 | 显示名称 |
| `version` | 是 | 语义化版本号（如 `1.0.0`、`1.2.3-beta`） |
| `authors` | 否 | 作者列表 |
| `license` | 否 | 许可证（如 `MIT`、`GPL-3.0`、`CC-BY-NC-4.0`） |
| `description` | 否 | 简短描述 |
| `homepage` | 否 | 主页链接 |
| `repository` | 否 | 代码仓库链接 |
| `keywords` | 否 | 关键词列表，用于搜索 |

### [package.seal] 海豹版本要求

```toml
[package.seal]
min_version = "1.4.6"    # 最低版本
max_version = "2.0.0"    # 最高版本（可选）
```

### [dependencies] 依赖声明

声明对其他扩展包的依赖：

```toml
[dependencies]
"com.seal.coc7" = ">=1.0.0"           # 大于等于 1.0.0
"com.example.base" = "^2.0.0"         # 兼容 2.x.x
"com.example.utils" = "~1.2.0"        # 兼容 1.2.x
"com.example.core" = "1.0.0 - 2.0.0"  # 范围
```

版本约束语法遵循 [semver](https://semver.org/) 规范。

### [permissions] 权限声明

```toml
[permissions]
# 网络权限
network = true                         # 允许网络访问
network_hosts = ["api.example.com"]    # 限制可访问的域名（可选）

# 文件权限
file_read = ["assets/*", "data/**"]    # 可读取的路径模式
file_write = ["_userdata/*"]           # 可写入的路径模式

# 高级权限
dangerous = false                       # 危险操作（如执行命令）
http_server = false                     # 启动 HTTP 服务
ipc = ["com.example.other"]            # 允许与其他扩展通信
```

**路径模式说明**：
- `*` 匹配当前目录下的所有文件
- `**` 匹配任意深度的目录和文件
- 路径相对于扩展包安装目录

**安全提示**：只声明实际需要的权限，权限越少用户越放心安装。

### [contents] 资源清单

```toml
[contents]
scripts = ["scripts/*.js", "scripts/*.ts"]
decks = ["decks/*.toml", "decks/*.json"]
reply = ["reply/*.yaml"]
helpdoc = ["helpdoc/*.yaml"]
template = ["templates/*.yaml"]
```

### [config] 用户配置项

定义可由用户配置的选项，海豹会自动生成配置界面：

```toml
[config.api_key]
type = "string"
title = "API 密钥"
description = "第三方服务的 API 密钥"
default = ""
secret = true                          # 敏感信息，UI 中隐藏显示

[config.max_results]
type = "integer"
title = "最大结果数"
description = "每次查询返回的最大数量"
default = 10
min = 1
max = 100

[config.enable_feature]
type = "boolean"
title = "启用高级功能"
default = false

[config.mode]
type = "string"
title = "运行模式"
enum = ["simple", "advanced", "expert"]
default = "simple"

[config.tags]
type = "array"
title = "标签列表"
items = { type = "string" }
default = ["默认标签"]
```

**支持的类型**：
- `string` - 字符串
- `integer` - 整数
- `number` - 浮点数
- `boolean` - 布尔值
- `array` - 数组
- `object` - 嵌套对象

## 在脚本中访问配置

```javascript
// 获取扩展包配置
const config = seal.ext.getPackageConfig();
const apiKey = config.api_key;
const maxResults = config.max_results;

// 或者使用存储 API（传统方式仍可用）
ext.storageGet("key");
ext.storageSet("key", "value");
```

## 访问包内资源

```javascript
// 读取包内文件
const data = seal.ext.readPackageFile("assets/data.json");
const obj = JSON.parse(data);

// 写入用户数据（只能写入 _userdata/ 目录）
seal.ext.writePackageFile("_userdata/cache.json", JSON.stringify(cache));
```

## 最佳实践

### 1. ID 命名规范

使用 `作者/包名` 格式：
- `sealdice/coc7` - 官方 COC7 扩展
- `username/my-extension` - 用户扩展
- `organization/tool-name` - 组织扩展

### 2. 版本号规范

遵循语义化版本：
- `MAJOR.MINOR.PATCH`（如 `1.2.3`）
- 重大变更增加 MAJOR
- 新功能增加 MINOR
- Bug 修复增加 PATCH

### 3. 权限最小化

只申请必要的权限：
```toml
[permissions]
network = true
network_hosts = ["api.example.com"]  # 限制具体域名而非开放所有
```

### 4. 提供默认配置

为所有配置项设置合理的默认值，确保开箱即用。

### 5. 错误处理

在脚本中妥善处理权限不足的情况：
```javascript
try {
  const resp = await fetch("https://api.example.com/data");
} catch (e) {
  if (e.message.includes("没有") && e.message.includes("权限")) {
    seal.replyToSender(ctx, msg, "此功能需要网络权限，请在扩展包设置中启用");
  }
}
```

## 发布到扩展商店

1. 确保 `manifest.toml` 信息完整
2. 测试扩展包在全新安装时能正常工作
3. 准备好图标和截图
4. 提交到海豹扩展商店审核

## 常见问题

### Q: 如何调试扩展包？

开发时可以直接将文件放在 `data/packages/你的包ID/` 目录下，无需每次打包。

### Q: 扩展包和传统 JS 插件有什么区别？

扩展包是传统插件的超集，可以包含脚本、牌堆、回复等多种资源，并提供权限隔离和配置管理。传统插件仍然支持。

### Q: 如何处理扩展包更新？

用户安装新版本时，`_userdata/` 目录会保留，用户配置不会丢失。
