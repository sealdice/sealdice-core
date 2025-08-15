// Copied from https://github.com/rclone/rclone/tree/master/fs/log
// Log the panic to the log file - for oses which can't do this

//go:build !windows && !darwin && !dragonfly && !freebsd && !linux && !nacl && !netbsd && !openbsd

package paniclog

import (
	"os"

	"sealdice-core/logger"
)

// redirectStderr to the file passed in
func redirectStderr(f *os.File) {
	// 安卓当前还暂时没有什么头绪，看上去rclone也没头绪。
	logger.M().Error("Can't redirect stderr to file")
}
