package utils

import (
	"strings"
	"testing"
)

func TestFilenameSafeReadableBasic(t *testing.T) {
	got := FilenameSafeReadable("  团务/log:name\t2026?  ", "fallback", 64)
	want := "团务_log_name_2026"
	if got != want {
		t.Fatalf("unexpected sanitized filename: got %q want %q", got, want)
	}
}

func TestFilenameSafeReadableFallbackAndReservedNames(t *testing.T) {
	if got := FilenameSafeReadable("", "  ", 32); got != "file" {
		t.Fatalf("unexpected empty fallback result: %q", got)
	}
	if got := FilenameSafeReadable("CON", "fallback", 32); got != "_CON" {
		t.Fatalf("unexpected reserved name handling: %q", got)
	}
}

func TestFilenameSafeReadableTruncatesByBytes(t *testing.T) {
	input := strings.Repeat("超长日志名", 20)
	got := FilenameSafeReadable(input, "fallback", 48)
	if len(got) > 48 {
		t.Fatalf("filename was not truncated to byte limit: len=%d value=%q", len(got), got)
	}
	if !strings.Contains(got, "-") {
		t.Fatalf("expected truncated filename to include hash suffix: %q", got)
	}
}

func TestTempFilePatternFromNamePreservesExtension(t *testing.T) {
	got := TempFilePatternFromName(`C:\temp\危险/日志:*?名.png`, "temp", 48)
	if !strings.HasSuffix(got, ".png") {
		t.Fatalf("expected pattern to preserve extension: %q", got)
	}
	if strings.ContainsAny(got, `/:"<>|\`) {
		t.Fatalf("pattern still contains illegal filename characters: %q", got)
	}
	if !strings.Contains(got, "-*") {
		t.Fatalf("pattern should contain temp wildcard: %q", got)
	}
}
