package dice

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
)

const (
	packageDiskReserve       uint64 = 10 << 20
	packageDiskCheckInterval uint64 = 8 << 20
)

type packageDiskSpace struct {
	Volume    string
	Available uint64
	Total     uint64
}

var probePackageDiskSpace = platformPackageDiskSpace

type packageDiskRequirement struct {
	path  string
	bytes uint64
}

func (pm *PackageManager) checkPackageDiskRequirements(requirements ...packageDiskRequirement) error {
	type volumeRequirement struct {
		path      string
		available uint64
		required  uint64
	}
	volumes := map[string]*volumeRequirement{}
	for _, requirement := range requirements {
		if requirement.path == "" {
			continue
		}
		space, err := probePackageDiskSpace(requirement.path)
		if err != nil {
			pm.warnDiskProbe(requirement.path, err)
			continue
		}
		entry := volumes[space.Volume]
		if entry == nil {
			entry = &volumeRequirement{path: requirement.path, available: space.Available}
			volumes[space.Volume] = entry
		}
		if math.MaxUint64-entry.required < requirement.bytes {
			return errors.New("扩展包所需磁盘空间计算溢出")
		}
		entry.required += requirement.bytes
	}

	for _, volume := range volumes {
		if math.MaxUint64-volume.required < packageDiskReserve {
			return errors.New("扩展包所需磁盘空间计算溢出")
		}
		required := volume.required + packageDiskReserve
		if volume.available < required {
			return fmt.Errorf("磁盘空间不足: %s 需要 %s（含 10 MiB 预留），当前可用 %s",
				filepath.Clean(volume.path), formatPackageBytes(required), formatPackageBytes(volume.available))
		}
	}
	return nil
}

func (pm *PackageManager) warnDiskProbe(path string, err error) {
	if pm != nil && pm.parent != nil && pm.parent.Logger != nil {
		pm.parent.Logger.Warnf("无法查询扩展包目录磁盘空间 %s，将继续操作: %v", filepath.Clean(path), err)
	}
}

type packageDiskGuard struct {
	pm            *PackageManager
	path          string
	expectedTotal uint64
	written       uint64
	nextCheck     uint64
	probeFailed   bool
}

func newPackageDiskGuard(pm *PackageManager, path string, expectedTotal uint64) *packageDiskGuard {
	return &packageDiskGuard{
		pm:            pm,
		path:          path,
		expectedTotal: expectedTotal,
	}
}

func (g *packageDiskGuard) BeforeWrite(chunk uint64) error {
	if math.MaxUint64-g.written < chunk {
		return errors.New("扩展包写入大小计算溢出")
	}
	projected := g.written + chunk
	if projected < g.nextCheck && (g.expectedTotal == 0 || projected < g.expectedTotal) {
		g.written = projected
		return nil
	}

	if !g.probeFailed {
		space, err := probePackageDiskSpace(g.path)
		if err != nil {
			g.pm.warnDiskProbe(g.path, err)
			g.probeFailed = true
		} else {
			remaining := uint64(0)
			if g.expectedTotal > projected {
				remaining = g.expectedTotal - projected
			}
			writeBudget := chunk
			if g.expectedTotal == 0 && writeBudget < packageDiskCheckInterval {
				writeBudget = packageDiskCheckInterval
			}
			if math.MaxUint64-packageDiskReserve < writeBudget {
				return errors.New("扩展包所需磁盘空间计算溢出")
			}
			reservedWrite := packageDiskReserve + writeBudget
			if math.MaxUint64-remaining < reservedWrite {
				return errors.New("扩展包所需磁盘空间计算溢出")
			}
			required := remaining + reservedWrite
			if space.Available < required {
				return fmt.Errorf("磁盘空间不足: %s 继续写入至少需要 %s，当前可用 %s",
					filepath.Clean(g.path), formatPackageBytes(required), formatPackageBytes(space.Available))
			}
		}
	}
	g.written = projected
	if math.MaxUint64-projected < packageDiskCheckInterval {
		g.nextCheck = math.MaxUint64
	} else {
		g.nextCheck = projected + packageDiskCheckInterval
	}
	return nil
}

func formatPackageBytes(value uint64) string {
	const (
		kiB = 1 << 10
		miB = 1 << 20
		giB = 1 << 30
	)
	switch {
	case value >= giB:
		return fmt.Sprintf("%.2f GiB", float64(value)/giB)
	case value >= miB:
		return fmt.Sprintf("%.2f MiB", float64(value)/miB)
	case value >= kiB:
		return fmt.Sprintf("%.2f KiB", float64(value)/kiB)
	default:
		return fmt.Sprintf("%d B", value)
	}
}
