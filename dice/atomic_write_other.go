//go:build !windows

package dice

import "os"

func replaceFileAtomically(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
