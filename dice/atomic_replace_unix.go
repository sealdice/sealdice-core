//go:build !windows && !linux

package dice

import (
	"errors"
	"os"
)

func replaceFileAtomic(source, target string) error {
	return os.Rename(source, target)
}

func renameRestorePath(source, target string) error {
	if _, err := os.Lstat(target); err == nil {
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(source, target)
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
