# Sealpkg 测试指南

## 目录结构

```
sealdice-core/
├── data/
│   ├── packages/                    # 包源文件存储目录
│   │   ├── packages.json           # 包状态持久化文件
│   │   └── *.sealpkg               # 已安装的包源文件
│   └── extensions/<包ID>/           # 扩展数据目录（扩展包和传统扩展共用）
│       ├── storage.db              # JS 扩展的 storage 数据库（传统扩展）
│       └── _userdata/              # 扩展包用户数据目录
├── cache/
│   └── packages/<包ID>/            # 包解压缓存目录（可从源文件重建）
│       ├── manifest.toml
│       ├── scripts/
│       ├── decks/
│       ├── reply/
│       ├── helpdoc/
│       └── templates/
└── tmp/
    └── test-packages/              # 测试包目录
```

## 测试包列表

| 包 ID | 名称 | 测试内容 |
|-------|------|----------|
| `test@simple-script` | 简单脚本测试包 | JS 脚本加载 |
| `test@config-package` | 配置测试扩展包 | 配置项系统、多种配置类型 |
| `test@deck-package` | 测试牌堆扩展包 | 牌堆加载 |
| `test@reply-package` | 回复测试包 | 自定义回复加载 |
| `test@helpdoc-package` | 帮助文档测试包 | 帮助文档加载 |
| `test@template-package` | 模板测试包 | 游戏模板加载 |
| `test@permission-package` | 权限测试扩展包 | 权限声明、网络访问 |

## 测试包详情

### test@simple-script
- **内容**: `scripts/*.js`
- **功能**: 基础 JS 扩展，注册 `.hello` 命令

### test@config-package
- **内容**: `scripts/*.js`, `decks/*.toml`
- **配置项**:
  - `greeting_message` (string) - 问候消息
  - `max_items` (integer, 1-100) - 最大项目数
  - `enable_debug` (boolean) - 调试模式
  - `response_mode` (enum) - 响应模式
  - `allowed_groups` (array) - 允许的群组
  - `custom_dice` (object) - 自定义骰子配置

### test@reply-package
- **内容**: `reply/*.yaml`
- **功能**: 测试自定义回复的动态加载

### test@helpdoc-package
- **内容**: `helpdoc/*.json`
- **功能**: 测试帮助文档的动态加载
- **帮助条目**: `测试帮助`, `helpdoc测试`

### test@template-package
- **内容**: `templates/*.yaml`
- **功能**: 测试游戏模板的动态加载

## API 端点

### 包管理

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/sd-api/package/list` | 获取所有已安装包列表 |
| POST | `/sd-api/package/install` | 从本地文件安装包 |
| POST | `/sd-api/package/install-url` | 从 URL 安装包 |
| POST | `/sd-api/package/uninstall` | 卸载包 |
| POST | `/sd-api/package/enable` | 启用包 |
| POST | `/sd-api/package/disable` | 禁用包 |
| POST | `/sd-api/package/reload` | 重载指定包资源 |
| POST | `/sd-api/package/reload-all` | 重载所有已启用包资源 |
| GET | `/sd-api/package/config` | 获取包配置 |
| POST | `/sd-api/package/config` | 更新包配置 |

### 请求示例

```bash
# 获取包列表
curl http://127.0.0.1:3211/sd-api/package/list

# 安装包（需要认证）
curl -X POST http://127.0.0.1:3211/sd-api/package/install \
  -H "Authorization: Bearer <token>" \
  -F "file=@tmp/test-packages/reply-package.sealpkg"

# 启用包
curl -X POST http://127.0.0.1:3211/sd-api/package/enable \
  -H "Content-Type: application/json" \
  -d '{"id": "test@reply-package"}'

# 重载包资源
curl -X POST http://127.0.0.1:3211/sd-api/package/reload \
  -H "Content-Type: application/json" \
  -d '{"id": "test@reply-package"}'

# 更新包配置
curl -X POST http://127.0.0.1:3211/sd-api/package/config \
  -H "Content-Type: application/json" \
  -d '{"id": "test@config-package", "config": {"enable_debug": true}}'
```

## 内容类型支持

| 类型 | manifest 字段 | 加载函数 | 动态重载 |
|------|---------------|----------|----------|
| 脚本 | `contents.scripts` | `JsLoadScripts()` | ✅ |
| 牌堆 | `contents.decks` | `DecksDetect()` | ✅ |
| 回复 | `contents.reply` | `ReplyReload()` | ✅ |
| 帮助文档 | `contents.helpdoc` | `HelpManager.Load()` | ✅ |
| 模板 | `contents.template` | `GameSystemTemplateReload()` | ✅ |

## 测试步骤

### 1. 安装测试包

```bash
# 复制测试包到安装目录
cp tmp/test-packages/reply-package.sealpkg data/packages/

# 或通过 API 安装
curl -X POST http://127.0.0.1:3211/sd-api/package/install \
  -F "file=@tmp/test-packages/reply-package.sealpkg"
```

### 2. 验证包加载

```bash
# 检查包列表
curl http://127.0.0.1:3211/sd-api/package/list | jq '.data[].manifest.package.id'

# 检查解压目录
ls -la cache/packages/test@reply-package/
```

### 3. 启用包并测试

```bash
# 启用包
curl -X POST http://127.0.0.1:3211/sd-api/package/enable \
  -d '{"id": "test@reply-package"}'

# 重载资源
curl -X POST http://127.0.0.1:3211/sd-api/package/reload \
  -d '{"id": "test@reply-package"}'
```

### 4. 验证资源加载

- **脚本**: 检查扩展列表，测试注册的命令
- **牌堆**: 使用 `.draw` 命令测试
- **回复**: 发送触发消息测试自定义回复
- **帮助文档**: 使用 `.help <关键词>` 测试
- **模板**: 检查游戏系统模板列表

## 配置项测试

### 通过 JS 读取配置

```javascript
// 在包的 JS 脚本中
let ext = seal.ext.find('扩展名');
let config = ext.getPackageConfig();
console.log(config.greeting_message);
console.log(config.enable_debug);
```

### 通过 API 更新配置

```bash
curl -X POST http://127.0.0.1:3211/sd-api/package/config \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test@config-package",
    "config": {
      "greeting_message": "新的问候语",
      "max_items": 20,
      "enable_debug": true
    }
  }'
```

## 注意事项

1. **包 ID 格式**:
   - 个人包：`author@package`，如 `test@reply-package`
   - 组织包：`@org@package`，如 `@sealdice@official-coc7`
2. **源文件命名**: 包文件名与包 ID 一致，如 `test@reply-package.sealpkg`
3. **缓存目录**: 平铺存储，目录名与包 ID 一致，如 `cache/packages/test@reply-package/`
4. **Storage 位置**: JS 扩展的 storage 存储在全局 `data/extensions/<扩展名>/storage.db`
5. **资源不复制**: 资源文件保留在包目录，通过 `GetEnabledContentDirs()` 动态获取路径
