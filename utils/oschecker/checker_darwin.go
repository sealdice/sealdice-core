//go:build darwin
// +build darwin

package oschecker

import (
	mac "github.com/wobsoriano/go-macos-version"
)

// TODO: MacOS 如何提示用户？
func OldVersionCheck() (bool, string) {
	version, err := mac.MacOSVersion()
	if err != nil {
		return true, "MAC_UNKNOWN"
	}
	judge, err := mac.IsMacOSVersion(">10.10")
	if err != nil {
		return true, "MAC_UNKNOWN"
	}
	return judge, version
}
