# 海豹扩展包系统技术文档

本文档面向海豹骰核心开发者，介绍扩展包系统的内部实现。

## 架构概览

```
dice/
├── sealpkg/                    # 独立包（无外部依赖）
│   ├── types.go                # 数据类型定义
│   ├── manifest.go             # manifest 解析和验证
│   ├── sandbox.go              # 权限沙箱实现
│   └── config.go               # 配置 schema 验证
├── package.go                  # 类型重导出 + DependencyError
└── package_manager.go          # PackageManager 实现
```

### 设计原则

1. **隔离性**：`sealpkg` 包不依赖 `dice` 包，可独立测试和复用
2. **兼容性**：与现有的 JS 插件、牌堆系统共存
3. **安全性**：通过权限沙箱限制脚本能力
4. **可扩展**：配置 schema 支持自动 UI 生成

## 核心数据结构

### sealpkg.Manifest

对应 `manifest.toml` 文件：

```go
type Manifest struct {
    Package      PackageInfo             `toml:"package"`
    Dependencies map[string]string       `toml:"dependencies"`  // pkgID -> semver constraint
    Permissions  Permissions             `toml:"permissions"`
    Contents     Contents                `toml:"contents"`
    Config       map[string]ConfigSchema `toml:"config"`
}
```

### sealpkg.Instance

运行时的包实例：

```go
type Instance struct {
    Manifest     *Manifest
    State        PackageState           // installed/enabled/disabled/error
    InstallTime  time.Time
    InstallPath  string                 // cache/packages/<id>/ 运行时缓存
    SourcePath   string                 // data/packages/<id>.sealpkg 源文件
    UserDataPath string                 // data/extensions/<id>/_userdata/ 用户数据
    Config       map[string]interface{} // 用户配置值
    ErrText      string
}
```

### PackageManager

包管理器，挂载在 `Dice` 上：

```go
type PackageManager struct {
    lock                   *sync.RWMutex
    parent                 *Dice
    packages               map[string]*sealpkg.Instance
    dependencyGraph        map[string][]string  // A -> [B,C] A依赖B,C
    reverseDependencyGraph map[string][]string  // A -> [B,C] B,C依赖A
}
```

## 目录结构

```
data/
├── packages/                         # 扩展包源文件目录
│   ├── author@package.sealpkg        # 源文件
│   └── packages.json                 # 包管理器状态持久化
└── extensions/                       # 扩展用户数据（传统扩展和扩展包共用）
    └── author@package/               # 按包ID隔离
        ├── storage.db                # 传统扩展 storage（如使用）
        └── _userdata/                # 扩展包用户数据（卸载时可保留）
            └── cache.json            # 用户数据文件

cache/
└── packages/                         # 运行时缓存（可从源文件重建）
    └── author@package/               # 解压后的包内容
        ├── manifest.toml             # 包元数据
        ├── scripts/                  # 脚本目录
        ├── decks/                    # 牌堆目录
        ├── reply/                    # 回复配置
        ├── helpdoc/                  # 帮助文档
        └── assets/                   # 静态资源
```

## 核心流程

### 安装流程

```
Install(pkgPath)
  ├─ zip.OpenReader(pkgPath)
  ├─ 读取并解析 manifest.toml
  ├─ 验证 manifest 格式
  ├─ 检查海豹版本兼容性
  ├─ 检查依赖是否满足
  ├─ 如已安装旧版本
  │   ├─ 比较版本号
  │   └─ 禁用旧版本
  ├─ 复制源文件到 data/packages/<id>.sealpkg
  ├─ 解压到 cache/packages/<id>/
  ├─ 创建用户数据目录 data/extensions/<id>/_userdata/
  ├─ 初始化默认配置
  ├─ 注册到 packages map
  ├─ 重建依赖图
  └─ 保存状态到 packages.json
```

### 启用流程

```
Enable(pkgID)
  ├─ 检查包是否存在
  ├─ 检查依赖是否满足
  ├─ 递归启用依赖的包
  ├─ 设置 State = enabled
  ├─ 加载包内资源        // TODO
  │   ├─ 加载脚本
  │   ├─ 加载牌堆
  │   ├─ 加载回复配置
  │   └─ 加载帮助文档
  └─ 保存状态
```

### 禁用流程

```
Disable(pkgID)
  ├─ 检查是否有已启用的包依赖此包
  │   └─ 如有则拒绝禁用
  ├─ 设置 State = disabled
  ├─ 卸载包内资源        // TODO
  └─ 保存状态
```

### 卸载流程

```
Uninstall(pkgID, mode)
  ├─ 检查反向依赖
  ├─ 如已启用则先禁用
  ├─ 根据 mode:
  │   ├─ full: 删除缓存目录、源文件、用户数据
  │   ├─ keep_data: 删除缓存目录和源文件，保留 data/extensions/<id>/_userdata/
  │   └─ disable_only: 仅设置状态
  ├─ 从 packages map 中移除
  ├─ 重建依赖图
  └─ 保存状态
```

## 权限沙箱

### Sandbox 结构

```go
type Sandbox struct {
    PackageID    string
    Permissions  *Permissions
    BasePath     string  // 包缓存路径 (cache/packages/<id>/)
    UserDataPath string  // 用户数据路径 (data/extensions/<id>/_userdata/)
}
```

### 权限检查方法

| 方法 | 检查内容 |
|-----|---------|
| `CheckNetworkPermission(url)` | network + network_hosts 白名单 |
| `CheckFileReadPermission(path)` | file_read 路径模式匹配 |
| `CheckFileWritePermission(path)` | file_write 路径模式（默认仅 _userdata/） |
| `CheckDangerousPermission(op)` | dangerous 标志 |
| `CheckHTTPServerPermission()` | http_server 标志 |
| `CheckIPCPermission(targetPkg)` | ipc 白名单 |

### 路径模式匹配

```go
// 支持的模式:
"scripts/*"      // 匹配 scripts/ 下的文件
"assets/**"      // 匹配 assets/ 下任意深度
"*.json"         // 匹配根目录下的 json 文件
```

### 沙箱化 API

```go
// SandboxedFS - 沙箱化文件系统
fs := sealpkg.NewSandboxedFS(sandbox)
fs.ReadFile(path)   // 检查 file_read 权限
fs.WriteFile(path)  // 检查 file_write 权限

// SandboxedHTTP - 沙箱化网络访问
http := sealpkg.NewSandboxedHTTP(sandbox)
http.Get(url)       // 检查 network 权限 + 域名白名单
```

## 配置系统

### ConfigSchema

类 JSON Schema 的配置定义：

```go
type ConfigSchema struct {
    Type        string                  // string/integer/number/boolean/array/object
    Title       string                  // 显示标题
    Description string                  // 描述
    Default     interface{}             // 默认值
    Secret      bool                    // 敏感信息标记
    Min, Max    *float64                // 数值范围
    Enum        []interface{}           // 枚举值
    Items       *ConfigSchema           // 数组元素类型
    Properties  map[string]ConfigSchema // 对象属性
}
```

### 配置验证

```go
// 验证单个值
sealpkg.ValidateConfigValue(key, value, schema)

// 验证整个配置
sealpkg.ValidateConfig(config, schemas)

// 初始化默认配置
sealpkg.InitDefaultConfig(schemas)

// 合并配置（升级时保留用户设置）
sealpkg.MergeConfig(defaults, userConfig)
```

## 依赖管理

### 版本约束

使用 semver 库解析版本约束：

```go
sealpkg.CheckDependencyConstraint(">=1.0.0", "1.2.3")  // true
sealpkg.CheckDependencyConstraint("^2.0.0", "3.0.0")   // false
```

### 依赖图

```go
// 正向依赖：A 依赖哪些包
dependencyGraph["A"] = ["B", "C"]

// 反向依赖：哪些包依赖 A
reverseDependencyGraph["A"] = ["X", "Y"]
```

用途：
- 启用时递归启用依赖
- 禁用/卸载时检查是否被依赖

## 集成点（TODO）

### 1. Dice 初始化

```go
func (d *Dice) Init() {
    // ...
    d.PackageManager = NewPackageManager(d)
    d.PackageManager.Init()
    d.PackageManager.LoadAllEnabled()
}
```

### 2. 脚本加载集成

修改 `JsLoadScripts()` 添加包内脚本：

```go
func (d *Dice) JsLoadScripts() {
    // 现有逻辑...

    // 加载扩展包内的脚本
    for _, pkg := range d.PackageManager.GetEnabled() {
        scriptsPath := filepath.Join(pkg.InstallPath, "scripts")
        // 遍历加载，设置 PackageID
    }
}
```

### 3. 牌堆加载集成

修改 `DecksDetect()` 添加包内牌堆。

### 4. JS API 注入

为包内脚本注入沙箱化 API：

```go
func (d *Dice) JsLoadScriptRaw(jsInfo *JsScriptInfo) {
    if jsInfo.PackageID != "" {
        sandbox := d.PackageManager.GetSandbox(jsInfo.PackageID)
        // 注入沙箱化的 fetch, fs 等
    }
}
```

### 5. API 端点

| 方法 | 路径 | 描述 |
|-----|------|------|
| GET | `/package/list` | 列出已安装的包 |
| POST | `/package/install` | 上传安装 |
| POST | `/package/install-url` | URL 安装 |
| DELETE | `/package/uninstall` | 卸载 |
| POST | `/package/enable` | 启用 |
| POST | `/package/disable` | 禁用 |
| GET | `/package/:id/config` | 获取配置 |
| PUT | `/package/:id/config` | 更新配置 |
| GET | `/package/:id/config-schema` | 获取配置 schema |

## 测试

### 单元测试

```go
// sealpkg 包可独立测试
func TestParseManifest(t *testing.T) {
    data := []byte(`
[package]
id = "test"
name = "Test"
version = "1.0.0"
`)
    m, err := sealpkg.ParseManifest(data)
    assert.NoError(t, err)
    assert.Equal(t, "test", m.Package.ID)
}

func TestSandboxPermission(t *testing.T) {
    s := sealpkg.NewSandbox("test", &sealpkg.Permissions{
        Network: false,
    }, "/tmp/test")

    err := s.CheckNetworkPermission("https://example.com")
    assert.Error(t, err)
}
```

### 集成测试

1. 创建测试扩展包
2. 安装、启用、禁用、卸载
3. 验证资源正确加载/卸载
4. 验证权限检查生效

## 未来扩展

1. **签名验证**：对扩展包进行签名，验证来源
2. **沙箱增强**：考虑 WebAssembly 或独立进程隔离
3. **热更新**：不重启更新扩展
4. **依赖自动安装**：从商店自动下载缺失的依赖
5. **资源冲突检测**：检测多个包注册相同指令
