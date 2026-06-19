package storylog

import (
	"strings"
	"testing"
	"time"
)

func TestBuildLogBackupFilenameKeepsCleanName(t *testing.T) {
	name, notice := buildLogBackupFilename("QQ-Group:12345", "normal-log", time.Unix(0, 0))
	if notice != "" {
		t.Fatalf("unexpected notice %q", notice)
	}
	if !strings.Contains(name, "normal-log") {
		t.Fatalf("filename %q does not contain original name", name)
	}
	if !strings.HasSuffix(name, ".zip") {
		t.Fatalf("filename %q does not end with .zip", name)
	}
}

func TestBuildLogBackupFilenameFallsBackToHashForInvalidName(t *testing.T) {
	name, notice := buildLogBackupFilename("QQ-Group:12345", "..", time.Unix(0, 0))
	if notice == "" {
		t.Fatal("expected notice for invalid filename")
	}
	if strings.Contains(name, "..") {
		t.Fatalf("filename %q still contains invalid name", name)
	}
}

func TestBuildLogBackupFilenameFallsBackToHashForLongName(t *testing.T) {
	longName := strings.Repeat("测", 200)
	name, notice := buildLogBackupFilename("QQ-Group:12345", longName, time.Unix(0, 0))
	if notice == "" {
		t.Fatal("expected notice for long filename")
	}
	if len([]byte(name)) > maxExportFilenameBytes {
		t.Fatalf("filename too long: %d bytes", len([]byte(name)))
	}
}

func TestBuildTempPatternFallsBackForLongPrefix(t *testing.T) {
	pattern, notice := buildTempPattern(strings.Repeat("prefix-", 200))
	if notice == "" {
		t.Fatal("expected notice for long temp prefix")
	}
	if len([]byte(pattern)) > maxTempPatternBytes {
		t.Fatalf("pattern too long: %d bytes", len([]byte(pattern)))
	}
	if !strings.Contains(pattern, "*") {
		t.Fatalf("pattern %q does not contain wildcard", pattern)
	}
}
