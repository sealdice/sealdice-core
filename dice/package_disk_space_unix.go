//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package dice

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func platformPackageDiskSpace(path string) (packageDiskSpace, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return packageDiskSpace{}, err
	}
	var fsStat unix.Statfs_t
	if err := unix.Statfs(absPath, &fsStat); err != nil {
		return packageDiskSpace{}, err
	}
	var fileStat unix.Stat_t
	if err := unix.Stat(absPath, &fileStat); err != nil {
		return packageDiskSpace{}, err
	}
	return packageDiskSpace{
		Volume:    fmt.Sprintf("%d", uint64(fileStat.Dev)),
		Available: uint64(fsStat.Bavail) * uint64(fsStat.Bsize),
		Total:     uint64(fsStat.Blocks) * uint64(fsStat.Bsize),
	}, nil
}
