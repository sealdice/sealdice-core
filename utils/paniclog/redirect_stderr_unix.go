// Copied from https://github.com/rclone/rclone/tree/master/fs/log
// Log the panic under unix to the log file

//go:build !windows && !solaris && !plan9 && !js

package paniclog

import (
	"os"

	"golang.org/x/sys/unix"

	"sealdice-core/logger"
)

// redirectStderr to the file passed in
func redirectStderr(f *os.File) {
	err := unix.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
	if err != nil {
		logger.M().Fatalf("Failed to redirect stderr to file: %v", err)
	}
}
