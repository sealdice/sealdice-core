package uploadcore_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sealdice-core/api/v2/internal/uploadcore"
)

func TestReplaceFileAtomicCreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	staged := filepath.Join(dir, ".staged")
	dst := filepath.Join(dir, "target.json")
	if err := os.WriteFile(staged, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write staged: %v", err)
	}
	if err := uploadcore.ReplaceFileAtomic(staged, dst); err != nil {
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
	if err := uploadcore.ReplaceFileAtomic(staged, dst); err != nil {
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
