//go:build linux

package dice

import (
	"os"

	"golang.org/x/sys/unix"
)

func replaceFileAtomic(source, target string) error {
	return os.Rename(source, target)
}

func renameRestorePath(source, target string) error {
	return unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, target, unix.RENAME_NOREPLACE)
}
