package sealpack //nolint:testpackage

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectArchiveRejectsMissingInfoFile(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"scripts/main.js": "console.log('x')",
	})
	if _, err := InspectArchive(archivePath); err == nil {
		t.Fatal("InspectArchive() error = nil, want missing info.toml rejection")
	}
}

func TestInspectArchiveRejectsNestedInfoFile(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"pkg/info.toml": minimalManifestForArchiveTest("alice/demo", "1.0.0"),
	})
	if _, err := InspectArchive(archivePath); err == nil {
		t.Fatal("InspectArchive() error = nil, want nested info.toml rejection")
	}
}

func TestInspectArchiveRejectsUnsupportedTopLevelDirectory(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml":     minimalManifestForArchiveTest("alice/demo", "1.0.0"),
		"misc/file.txt": "nope",
	})
	if _, err := InspectArchive(archivePath); err == nil {
		t.Fatal("InspectArchive() error = nil, want unsupported top-level directory rejection")
	}
}

func TestInspectArchiveRejectsDuplicateEntries(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "duplicate.sealpack")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	for _, content := range []string{
		minimalManifestForArchiveTest("alice/demo", "1.0.0"),
		minimalManifestForArchiveTest("alice/demo", "1.0.1"),
	} {
		w, err := zw.Create("info.toml")
		if err != nil {
			t.Fatalf("Create(info.toml) error = %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("Write(info.toml) error = %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := InspectArchive(archivePath); err == nil {
		t.Fatal("InspectArchive() error = nil, want duplicate entry rejection")
	}
}

func TestInspectArchiveFallsBackToReadmeFile(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml":    minimalManifestForArchiveTest("alice/demo", "1.0.0"),
		"README.md":    "# Demo",
		"scripts/a.js": "console.log('x')",
	})

	archiveInfo, err := InspectArchive(archivePath)
	if err != nil {
		t.Fatalf("InspectArchive() error = %v", err)
	}
	if archiveInfo.Manifest.Store.Readme != "README.md" {
		t.Fatalf("Manifest.Store.Readme = %q, want README.md", archiveInfo.Manifest.Store.Readme)
	}
}

func TestInspectArchiveReportsUncompressedSize(t *testing.T) {
	manifest := minimalManifestForArchiveTest("alice/demo", "1.0.0")
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml":       manifest,
		"scripts/a.js":    "console.log('x')",
		"assets/icon.png": "image-data",
	})
	info, err := InspectArchive(archivePath)
	if err != nil {
		t.Fatalf("InspectArchive() error = %v", err)
	}
	want := uint64(len(manifest) + len("console.log('x')") + len("image-data"))
	if info.UncompressedSize != want {
		t.Fatalf("UncompressedSize = %d, want %d", info.UncompressedSize, want)
	}
}

func TestInspectArchiveRejectsOversizedManifest(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml": strings.Repeat("x", int(MaxManifestSize)+1),
	})
	if _, err := InspectArchive(archivePath); err == nil || !strings.Contains(err.Error(), "info.toml") {
		t.Fatalf("InspectArchive() error = %v, want manifest size rejection", err)
	}
}

func TestInspectArchiveRejectsTooManyDeclaredEntriesBeforeZipParsing(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml": minimalManifestForArchiveTest("alice/demo", "1.0.0"),
	})
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	index := bytes.LastIndex(data, []byte{'P', 'K', 0x05, 0x06})
	if index < 0 {
		t.Fatal("EOCD not found")
	}
	binary.LittleEndian.PutUint16(data[index+8:index+10], maxArchiveEntries+1)
	binary.LittleEndian.PutUint16(data[index+10:index+12], maxArchiveEntries+1)
	if err := os.WriteFile(archivePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := InspectArchive(archivePath); err == nil || !strings.Contains(err.Error(), "too many entries") {
		t.Fatalf("InspectArchive() error = %v, want entry count rejection", err)
	}
}

func TestInspectArchivePreflightMatchesZip64DirectorySizeCompatibility(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml": minimalManifestForArchiveTest("alice/demo", "1.0.0"),
	})
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	endIndex := bytes.LastIndex(data, []byte{'P', 'K', 0x05, 0x06})
	if endIndex < 0 {
		t.Fatal("EOCD not found")
	}

	end := append([]byte(nil), data[endIndex:]...)
	directorySize := binary.LittleEndian.Uint32(end[12:16])
	directoryOffset := binary.LittleEndian.Uint32(end[16:20])
	binary.LittleEndian.PutUint32(end[12:16], math.MaxUint16)

	zip64EndOffset := uint64(endIndex)
	zip64End := make([]byte, directory64EndLen)
	binary.LittleEndian.PutUint32(zip64End[0:4], directory64EndSignature)
	binary.LittleEndian.PutUint64(zip64End[4:12], directory64EndLen-12)
	binary.LittleEndian.PutUint16(zip64End[12:14], 45)
	binary.LittleEndian.PutUint16(zip64End[14:16], 45)
	binary.LittleEndian.PutUint64(zip64End[24:32], maxArchiveEntries+1)
	binary.LittleEndian.PutUint64(zip64End[32:40], maxArchiveEntries+1)
	binary.LittleEndian.PutUint64(zip64End[40:48], uint64(directorySize))
	binary.LittleEndian.PutUint64(zip64End[48:56], uint64(directoryOffset))

	locator := make([]byte, directory64LocatorLen)
	binary.LittleEndian.PutUint32(locator[0:4], directory64LocatorSignature)
	binary.LittleEndian.PutUint64(locator[8:16], zip64EndOffset)
	binary.LittleEndian.PutUint32(locator[16:20], 1)

	data = append(data[:endIndex], zip64End...)
	data = append(data, locator...)
	data = append(data, end...)
	if err := os.WriteFile(archivePath, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := InspectArchive(archivePath); err == nil || !strings.Contains(err.Error(), "too many entries") {
		t.Fatalf("InspectArchive() error = %v, want ZIP64 entry count rejection", err)
	}
}

func TestInspectArchiveRejectsLongEntryPath(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml": minimalManifestForArchiveTest("alice/demo", "1.0.0"),
		"assets/" + strings.Repeat("x", maxArchivePathSize): "x",
	})
	if _, err := InspectArchive(archivePath); err == nil || !strings.Contains(err.Error(), "path is too long") {
		t.Fatalf("InspectArchive() error = %v, want path length rejection", err)
	}
}

func TestInspectArchiveHonorsCancellation(t *testing.T) {
	archivePath := createArchiveForTest(t, map[string]string{
		"info.toml": minimalManifestForArchiveTest("alice/demo", "1.0.0"),
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := InspectArchiveContext(ctx, archivePath); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("InspectArchiveContext() error = %v, want cancellation", err)
	}
}

func createArchiveForTest(t *testing.T, files map[string]string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "test.sealpack")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%s) error = %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("Write(%s) error = %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return archivePath
}

func minimalManifestForArchiveTest(pkgID, version string) string {
	return "[package]\n" +
		"id = \"" + pkgID + "\"\n" +
		"name = \"Demo\"\n" +
		"version = \"" + version + "\"\n"
}
