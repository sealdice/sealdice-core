package dice

import (
	"strings"

	"sealdice-core/dice/sealpack"
)

// 为方便使用，重新导出 sealpack 中的类型
type (
	PackageState    = sealpack.PackageState
	UninstallMode   = sealpack.UninstallMode
	PackageManifest = sealpack.Manifest
	PackageInfo     = sealpack.PackageInfo
	PackageInstance = sealpack.Instance
	PackageContents = sealpack.Contents
	ConfigSchema    = sealpack.ConfigSchema
	PackageSandbox  = sealpack.Sandbox
	PermissionError = sealpack.PermissionError
	SandboxedFS     = sealpack.SandboxedFS
	SandboxedHTTP   = sealpack.SandboxedHTTP
)

// 重新导出常量
const (
	PackageStateInstalled = sealpack.PackageStateInstalled
	PackageStateEnabled   = sealpack.PackageStateEnabled
	PackageStateDisabled  = sealpack.PackageStateDisabled
	PackageStateError     = sealpack.PackageStateError

	UninstallModeFull     = sealpack.UninstallModeFull
	UninstallModeKeepData = sealpack.UninstallModeKeepData
	UninstallModeDisable  = sealpack.UninstallModeDisable
)

// DependencyError 依赖错误
type DependencyError struct {
	PackageID       string   `json:"packageId"`
	MissingDeps     []string `json:"missingDeps"`
	VersionMismatch []string `json:"versionMismatch"`
}

func (e *DependencyError) Error() string {
	var b strings.Builder
	b.WriteString("包 ")
	b.WriteString(e.PackageID)
	b.WriteString(" 依赖不满足")
	if len(e.MissingDeps) > 0 {
		b.WriteString(", 缺少: ")
		for i, dep := range e.MissingDeps {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(dep)
		}
	}
	if len(e.VersionMismatch) > 0 {
		b.WriteString(", 版本不匹配: ")
		for i, dep := range e.VersionMismatch {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(dep)
		}
	}
	return b.String()
}
