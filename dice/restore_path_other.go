//go:build !windows

package dice

import "os"

func isLinkedRestorePath(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
