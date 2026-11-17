package dice //nolint:testpackage

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckPackageDiskRequirementsCombinesSameVolume(t *testing.T) {
	originalProbe := probePackageDiskSpace
	t.Cleanup(func() { probePackageDiskSpace = originalProbe })

	available := packageDiskReserve + 300
	probePackageDiskSpace = func(string) (packageDiskSpace, error) {
		return packageDiskSpace{Volume: "same", Available: available, Total: available}, nil
	}
	pm := &PackageManager{}
	if err := pm.checkPackageDiskRequirements(
		packageDiskRequirement{path: "source", bytes: 100},
		packageDiskRequirement{path: "cache", bytes: 200},
	); err != nil {
		t.Fatalf("checkPackageDiskRequirements() error = %v", err)
	}

	available = packageDiskReserve + 299
	err := pm.checkPackageDiskRequirements(
		packageDiskRequirement{path: "source", bytes: 100},
		packageDiskRequirement{path: "cache", bytes: 200},
	)
	if err == nil || !strings.Contains(err.Error(), "磁盘空间不足") {
		t.Fatalf("checkPackageDiskRequirements() error = %v, want insufficient-space rejection", err)
	}
}

func TestCheckPackageDiskRequirementsChecksSeparateVolumes(t *testing.T) {
	originalProbe := probePackageDiskSpace
	t.Cleanup(func() { probePackageDiskSpace = originalProbe })

	probePackageDiskSpace = func(path string) (packageDiskSpace, error) {
		available := packageDiskReserve + 100
		if path == "cache" {
			available = packageDiskReserve + 199
		}
		return packageDiskSpace{Volume: path, Available: available, Total: available}, nil
	}
	pm := &PackageManager{}
	err := pm.checkPackageDiskRequirements(
		packageDiskRequirement{path: "source", bytes: 100},
		packageDiskRequirement{path: "cache", bytes: 200},
	)
	if err == nil || !strings.Contains(err.Error(), "cache") {
		t.Fatalf("checkPackageDiskRequirements() error = %v, want cache-volume rejection", err)
	}
}

func TestCheckPackageDiskRequirementsContinuesWhenProbeFails(t *testing.T) {
	originalProbe := probePackageDiskSpace
	t.Cleanup(func() { probePackageDiskSpace = originalProbe })
	probePackageDiskSpace = func(string) (packageDiskSpace, error) {
		return packageDiskSpace{}, errors.New("unsupported")
	}
	pm := &PackageManager{}
	if err := pm.checkPackageDiskRequirements(packageDiskRequirement{path: "source", bytes: 1 << 40}); err != nil {
		t.Fatalf("checkPackageDiskRequirements() error = %v, want fail-open behavior", err)
	}
}

func TestPackageDiskGuardKeepsReserve(t *testing.T) {
	originalProbe := probePackageDiskSpace
	t.Cleanup(func() { probePackageDiskSpace = originalProbe })
	available := packageDiskReserve + packageDiskCheckInterval
	probePackageDiskSpace = func(string) (packageDiskSpace, error) {
		return packageDiskSpace{Volume: "same", Available: available, Total: available}, nil
	}
	guard := newPackageDiskGuard(&PackageManager{}, "source", 0)
	if err := guard.BeforeWrite(32); err != nil {
		t.Fatalf("BeforeWrite() error = %v", err)
	}
	available = packageDiskReserve
	guard.nextCheck = 0
	if err := guard.BeforeWrite(1); err == nil {
		t.Fatal("BeforeWrite() error = nil, want reserve rejection")
	}
}
