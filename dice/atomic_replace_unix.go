//go:build !windows

package dice

import "os"

func replaceFileAtomic(source, target string) error {
	return os.Rename(source, target)
}
