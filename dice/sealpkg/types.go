// Package sealpkg 提供海豹骰扩展包(.sealpkg)的核心类型定义和工具
package sealpkg

import (
	"time"
)

// PackageState 扩展包状态
type PackageState string

const (
	PackageStateInstalled PackageState = "installed" // 已安装（未启用）
	PackageStateEnabled   PackageState = "enabled"   // 已启用
	PackageStateDisabled  PackageState = "disabled"  // 已禁用
	PackageStateError     PackageState = "error"     // 错误状态
)

// UninstallMode 卸载模式
type UninstallMode string

const (
	UninstallModeFull     UninstallMode = "full"         // 完全删除
	UninstallModeKeepData UninstallMode = "keep_data"    // 保留用户数据
	UninstallModeDisable  UninstallMode = "disable_only" // 仅禁用
)

// Manifest manifest.toml 对应结构
type Manifest struct {
	Package      PackageInfo             `toml:"package" json:"package"`
	Dependencies map[string]string       `toml:"dependencies" json:"dependencies"`
	Permissions  Permissions             `toml:"permissions" json:"permissions"`
	Contents     Contents                `toml:"contents" json:"contents"`
	Config       map[string]ConfigSchema `toml:"config" json:"config"`
}

// PackageInfo 包基础信息
type PackageInfo struct {
	ID          string          `toml:"id" json:"id"`
	Name        string          `toml:"name" json:"name"`
	Version     string          `toml:"version" json:"version"`
	Authors     []string        `toml:"authors" json:"authors"`
	License     string          `toml:"license" json:"license"`
	Description string          `toml:"description" json:"description"`
	Homepage    string          `toml:"homepage" json:"homepage"`
	Repository  string          `toml:"repository" json:"repository"`
	Keywords    []string        `toml:"keywords" json:"keywords"`
	Seal        SealRequirement `toml:"seal" json:"seal"`
}

// SealRequirement 海豹版本要求
type SealRequirement struct {
	MinVersion string `toml:"min_version" json:"minVersion"`
	MaxVersion string `toml:"max_version" json:"maxVersion"`
}

// Permissions 权限声明
type Permissions struct {
	// 网络权限
	Network      bool     `toml:"network" json:"network"`
	NetworkHosts []string `toml:"network_hosts" json:"networkHosts"` // 允许访问的域名白名单

	// 文件权限
	FileRead  []string `toml:"file_read" json:"fileRead"`   // 可读取的路径模式
	FileWrite []string `toml:"file_write" json:"fileWrite"` // 可写入的路径模式

	// 系统权限
	Dangerous  bool `toml:"dangerous" json:"dangerous"`    // 危险操作（如exec）
	HTTPServer bool `toml:"http_server" json:"httpServer"` // 启动HTTP服务

	// 扩展间通信
	IPC []string `toml:"ipc" json:"ipc"` // 允许通信的包ID列表
}

// Contents 包含的资源清单
type Contents struct {
	Scripts  []string `toml:"scripts" json:"scripts"`
	Decks    []string `toml:"decks" json:"decks"`
	Reply    []string `toml:"reply" json:"reply"`
	Helpdoc  []string `toml:"helpdoc" json:"helpdoc"`
	Template []string `toml:"template" json:"template"`
}

// ConfigSchema 配置项Schema（类JSON Schema）
type ConfigSchema struct {
	Type        string                  `toml:"type" json:"type"`
	Title       string                  `toml:"title" json:"title"`
	Description string                  `toml:"description" json:"description"`
	Default     interface{}             `toml:"default" json:"default"`
	Secret      bool                    `toml:"secret" json:"secret"` // 敏感信息标记

	// 数值约束
	Min *float64 `toml:"min" json:"min,omitempty"`
	Max *float64 `toml:"max" json:"max,omitempty"`

	// 枚举约束
	Enum []interface{} `toml:"enum" json:"enum,omitempty"`

	// 数组类型的子项
	Items *ConfigSchema `toml:"items" json:"items,omitempty"`

	// 对象类型的属性
	Properties map[string]ConfigSchema `toml:"properties" json:"properties,omitempty"`
}

// Instance 已安装的扩展包实例
type Instance struct {
	Manifest     *Manifest              `json:"manifest"`
	State        PackageState           `json:"state"`
	InstallTime  time.Time              `json:"installTime"`
	InstallPath  string                 `json:"installPath"`  // cache/packages/<id>/ 运行时缓存
	SourcePath   string                 `json:"sourcePath"`   // 原始 .sealpkg 路径
	UserDataPath string                 `json:"userDataPath"` // data/extensions/<id>/_userdata/ 用户数据
	Config       map[string]interface{} `json:"config"`       // 用户配置值
	ErrText      string                 `json:"errText"`

	// PendingReload 待重载的内容类型列表
	// 当包状态变更（启用/禁用）后设置，重载后清空
	// UI 可根据此字段显示"需要重载"提示
	PendingReload []string `json:"pendingReload,omitempty"`
}

// Persistence 包管理器持久化数据
type Persistence struct {
	Packages map[string]*InstancePersist `json:"packages"`
}

// InstancePersist 持久化的包实例数据
type InstancePersist struct {
	State        PackageState           `json:"state"`
	InstallTime  time.Time              `json:"installTime"`
	InstallPath  string                 `json:"installPath"`
	SourcePath   string                 `json:"sourcePath"`
	UserDataPath string                 `json:"userDataPath"`
	Config       map[string]interface{} `json:"config"`
}

// OperationResult 包操作结果
type OperationResult struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	ReloadNeeded bool     `json:"reloadNeeded"` // 是否需要重载
	ReloadHints  []string `json:"reloadHints"`  // 重载提示列表
}

// ReloadResult 重载操作结果
type ReloadResult struct {
	Success       bool              `json:"success"`
	Message       string            `json:"message"`
	ReloadedItems map[string]string `json:"reloadedItems"` // 资源类型 -> 重载结果信息
	NeedRestart   bool              `json:"needRestart"`   // 是否需要重启
	RestartHints  []string          `json:"restartHints"`  // 需要重启的资源类型
}

// 常量定义
const (
	// Extension .sealpkg 文件扩展名
	Extension = ".sealpkg"

	// PackagesDir 扩展包安装目录名
	PackagesDir = "packages"

	// StateFile 包管理器状态文件名
	StateFile = "packages.json"

	// UserDataDir 用户数据目录名
	UserDataDir = "_userdata"

	// ConfigFile 用户配置文件名
	ConfigFile = "config.json"

	// ManifestFile manifest 文件名
	ManifestFile = "manifest.toml"
)
