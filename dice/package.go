package dice

import (
	"sealdice-core/dice/sealpkg"
)

// 为方便使用，重新导出 sealpkg 中的类型
type (
	PackageState     = sealpkg.PackageState
	UninstallMode    = sealpkg.UninstallMode
	PackageManifest  = sealpkg.Manifest
	PackageInfo      = sealpkg.PackageInfo
	PackageInstance  = sealpkg.Instance
	PackageContents  = sealpkg.Contents
	ConfigSchema     = sealpkg.ConfigSchema
	PackageSandbox   = sealpkg.Sandbox
	PermissionError  = sealpkg.PermissionError
	SandboxedFS      = sealpkg.SandboxedFS
	SandboxedHTTP    = sealpkg.SandboxedHTTP
)

// 重新导出常量
const (
	PackageStateInstalled = sealpkg.PackageStateInstalled
	PackageStateEnabled   = sealpkg.PackageStateEnabled
	PackageStateDisabled  = sealpkg.PackageStateDisabled
	PackageStateError     = sealpkg.PackageStateError

	UninstallModeFull     = sealpkg.UninstallModeFull
	UninstallModeKeepData = sealpkg.UninstallModeKeepData
	UninstallModeDisable  = sealpkg.UninstallModeDisable
)

// DependencyError 依赖错误
type DependencyError struct {
	PackageID       string   `json:"packageId"`
	MissingDeps     []string `json:"missingDeps"`
	VersionMismatch []string `json:"versionMismatch"`
}

func (e *DependencyError) Error() string {
	msg := "包 " + e.PackageID + " 依赖不满足"
	if len(e.MissingDeps) > 0 {
		msg += ", 缺少: "
		for i, dep := range e.MissingDeps {
			if i > 0 {
				msg += ", "
			}
			msg += dep
		}
	}
	if len(e.VersionMismatch) > 0 {
		msg += ", 版本不匹配: "
		for i, dep := range e.VersionMismatch {
			if i > 0 {
				msg += ", "
			}
			msg += dep
		}
	}
	return msg
}
