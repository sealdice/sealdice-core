//go:build linux

package dice

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func replaceFileAtomic(source, target string) error {
	return os.Rename(source, target)
}

func renameRestorePath(source, target string) error {
	return unix.Renameat2(unix.AT_FDCWD, source, unix.AT_FDCWD, target, unix.RENAME_NOREPLACE)
}

func syncRestoreDirectoryPath(directory string) error {
	file, err := os.Open(directory)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}

func isLinkedRestorePath(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
