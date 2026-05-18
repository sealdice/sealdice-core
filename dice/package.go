package dice

import (
	"strings"
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
