package api //nolint:testpackage

import (
	"path/filepath"
	"testing"

	"sealdice-core/dice"
)

func TestResolveBackupFilePathLimitsBackupNames(t *testing.T) {
	const validName = "bak_260101_010203_r1_145a8990.zip"

	path, ok := resolveBackupFilePath(validName)
	if !ok {
		t.Fatal("expected a generated backup filename to be accepted")
	}
	root, err := filepath.Abs(dice.BackupDir)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if relative != validName {
		t.Fatalf("resolved backup path escapes its root: %q", relative)
	}

	for _, name := range []string{
		"secret.txt",
		"bak_260101_010203_r1_deadbeef.zip",
		"..",
		"../" + validName,
		`..\` + validName,
		`C:\secret.txt`,
		"/tmp/" + validName,
	} {
		if _, ok := resolveBackupFilePath(name); ok {
			t.Errorf("unexpectedly accepted backup name %q", name)
		}
	}
}
