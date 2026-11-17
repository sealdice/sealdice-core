//go:build aix || darwin || dragonfly || freebsd || linux

package dice

import (
	"path/filepath"
	"strconv"

	"golang.org/x/sys/unix"
)

type packageDiskStatValue interface {
	~int32 | ~int64 | ~uint32 | ~uint64
}

func packageDiskStatUint64[T packageDiskStatValue](value T) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value)
}

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
	blockSize := packageDiskStatUint64(fsStat.Bsize)
	return packageDiskSpace{
		Volume:    strconv.FormatUint(packageDiskStatUint64(fileStat.Dev), 10),
		Available: packageDiskStatUint64(fsStat.Bavail) * blockSize,
		Total:     packageDiskStatUint64(fsStat.Blocks) * blockSize,
	}, nil
}
