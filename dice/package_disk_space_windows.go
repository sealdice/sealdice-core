//go:build windows

package dice

import (
	"fmt"
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
	volumeName := strings.ToUpper(filepath.VolumeName(absPath))
	volume := volumeName
	root := absPath
	if volumeName != "" {
		root = volumeName + `\`
	} else if index := strings.Index(absPath, `\`); index >= 0 {
		root = absPath[:index+1]
	}
	if rootPtr, rootErr := windows.UTF16PtrFromString(root); rootErr == nil {
		var serial uint32
		if windows.GetVolumeInformation(rootPtr, nil, 0, &serial, nil, nil, nil, 0) == nil {
			volume = fmt.Sprintf("%s:%08X", volumeName, serial)
		}
	}
	return packageDiskSpace{
		Volume:    volume,
		Available: available,
		Total:     total,
	}, nil
}
