//go:build !windows && !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris

package dice

import "errors"

func platformPackageDiskSpace(string) (packageDiskSpace, error) {
	return packageDiskSpace{}, errors.New("当前平台不支持磁盘空间探测")
}
