package sealpkg

import (
	"path/filepath"
	"testing"
)

func TestValidatePackageID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "ascii", id: "alice/awesome-dice"},
		{name: "mixed", id: "alice_123/my-package"},
		{name: "chinese-author", id: "木落/awesome-dice"},
		{name: "chinese-package", id: "alice/奇幻牌堆"},
		{name: "all-chinese", id: "海豹/扩展包"},
		{name: "empty", id: "", wantErr: true},
		{name: "missing-slash", id: "alice", wantErr: true},
		{name: "too-many-segments", id: "alice/pkg/v2", wantErr: true},
		{name: "empty-author", id: "/pkg", wantErr: true},
		{name: "empty-package", id: "alice/", wantErr: true},
		{name: "space", id: "alice/my package", wantErr: true},
		{name: "dot", id: "./pkg", wantErr: true},
		{name: "dotdot", id: "alice/..", wantErr: true},
		{name: "backslash", id: `alice\pkg`, wantErr: true},
		{name: "at-format", id: "alice@pkg", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePackageID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePackageID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestParsePackageID(t *testing.T) {
	author, pkg, err := ParsePackageID("木落/奇幻牌堆")
	if err != nil {
		t.Fatalf("ParsePackageID() error = %v", err)
	}
	if author != "木落" || pkg != "奇幻牌堆" {
		t.Fatalf("ParsePackageID() = (%q, %q)", author, pkg)
	}
}

func TestPackageIDToSafePath(t *testing.T) {
	got := PackageIDToSafePath("作者/扩展")
	want := filepath.Join("作者", "扩展")
	if got != want {
		t.Fatalf("PackageIDToSafePath() = %q, want %q", got, want)
	}
}

func TestPackageVersionHelpers(t *testing.T) {
	if got := PackageVersionToFileName("1.2.3"); got != "1.2.3.sealpkg" {
		t.Fatalf("PackageVersionToFileName() = %q", got)
	}
	if got := FileNameToPackageVersion("1.2.3.sealpkg"); got != "1.2.3" {
		t.Fatalf("FileNameToPackageVersion() = %q", got)
	}
}
