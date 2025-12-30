//go:build !windows

package oschecker

// 我们没有对其他的系统进行筛查的打算。
func OldVersionCheck() (bool, string) {
	return false, "NOTHING"
}
