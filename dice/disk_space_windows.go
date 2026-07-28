//go:build windows

package dice

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func diskSpace(path string) (uint64, uint64, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, err
	}
	ptr, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		return 0, 0, err
	}
	var freeAvailable, total uint64
	if err = windows.GetDiskFreeSpaceEx(ptr, &freeAvailable, &total, nil); err != nil {
		return 0, 0, err
	}
	return freeAvailable, total, nil
}
