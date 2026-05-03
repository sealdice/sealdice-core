package sealpkg //nolint:testpackage

import (
	"archive/zip"
	"os"
	"path/filepath"
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
	archivePath := filepath.Join(t.TempDir(), "duplicate.sealpkg")
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

func createArchiveForTest(t *testing.T, files map[string]string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "test.sealpkg")
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
