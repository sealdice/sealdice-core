//go:build windows

package dice

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func platformPackageDiskSpace(path string) (packageDiskSpace, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return packageDiskSpace{}, err
	}
	pathPtr, err := windows.UTF16PtrFromString(absPath)
	if err != nil {
		return packageDiskSpace{}, err
	}
	var available uint64
	var total uint64
	var totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(pathPtr, &available, &total, &totalFree); err != nil {
		return packageDiskSpace{}, err
	}
	return packageDiskSpace{
		Volume:    strings.ToUpper(filepath.VolumeName(absPath)),
		Available: available,
		Total:     total,
	}, nil
}
