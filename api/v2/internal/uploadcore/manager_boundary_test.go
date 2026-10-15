package uploadcore_test

import (
	"os"
	"path/filepath"
	"testing"

	"sealdice-core/api/v2/internal/uploadcore"
)

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
			if err := uploadcore.ReplaceFileAtomic(staged, target); err == nil {
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
	if err := uploadcore.ReplaceFileAtomic(staged, link); err == nil {
		t.Fatal("symlink target accepted")
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "keep" {
		t.Fatalf("symlink referent changed: %q, %v", data, err)
	}
}
