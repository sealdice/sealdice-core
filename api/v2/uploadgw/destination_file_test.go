package uploadgw //nolint:testpackage // Exercise the gateway's file replacement boundary.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceFileAtomicCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, ".staged")
	dst := filepath.Join(dir, "target.json")
	if err := os.WriteFile(staged, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write staged: %v", err)
	}
	if err := replaceFileAtomic(staged, dst); err != nil {
		t.Fatalf("ReplaceFileAtomic: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("dst content = %q, want %q", got, "hello")
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("staged file should be moved away, stat err = %v", err)
	}
}

func TestReplaceFileAtomicOverwritesExistingFileAndCleansBackup(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "target.json")
	if err := os.WriteFile(dst, []byte("old"), 0o600); err != nil {
		t.Fatalf("write dst: %v", err)
	}
	staged := filepath.Join(dir, ".staged")
	if err := os.WriteFile(staged, []byte("new-content"), 0o644); err != nil {
		t.Fatalf("write staged: %v", err)
	}
	if err := replaceFileAtomic(staged, dst); err != nil {
		t.Fatalf("ReplaceFileAtomic: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(got) != "new-content" {
		t.Fatalf("dst content = %q, want %q", got, "new-content")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".backup-") {
			t.Fatalf("unexpected leftover backup: %s", e.Name())
		}
	}
}

func TestReplaceFileAtomicPreservesDirectoryTargets(t *testing.T) {
	for _, name := range []string{".", "..", "package-dir"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "data", "scripts", name)
			if err := os.MkdirAll(target, 0o755); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(target, "keep.txt")
			if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}
			staged := filepath.Join(dir, "staged")
			if err := os.WriteFile(staged, []byte("upload"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := replaceFileAtomic(staged, target); err == nil {
				t.Fatal("directory target accepted")
			}
			if data, err := os.ReadFile(marker); err != nil || string(data) != "keep" {
				t.Fatalf("target changed: %q, %v", data, err)
			}
			if _, err := os.Stat(staged); err != nil {
				t.Fatalf("staged file consumed on rejection: %v", err)
			}
		})
	}
}

func TestReplaceFileAtomicRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target, link, staged := filepath.Join(dir, "target"), filepath.Join(dir, "link"), filepath.Join(dir, "staged")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, []byte("upload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := replaceFileAtomic(staged, link); err == nil {
		t.Fatal("symlink target accepted")
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "keep" {
		t.Fatalf("symlink referent changed: %q, %v", data, err)
	}
}
