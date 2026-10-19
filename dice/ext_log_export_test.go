//nolint:testpackage
package dice

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	sealdiceLogger "sealdice-core/logger"
)

func TestGetLogTxtUsesSharedLogExportDirectory(t *testing.T) {
	mockDB := newLogAliasTestDB(t)
	const groupID = "QQ-Group:1010"
	const logName = "shared-export"
	appendTestLog(t, mockDB, groupID, logName)

	dataDir := t.TempDir()
	ctx := &MsgContext{
		Dice: &Dice{
			BaseConfig: BaseConfig{DataDir: dataDir},
			DBOperator: mockDB,
			Logger:     sealdiceLogger.M(),
		},
	}

	logFile, notice, err := GetLogTxt(ctx, groupID, logName, "export")
	if err != nil {
		t.Fatalf("GetLogTxt: %v", err)
	}
	if notice != "" {
		t.Fatalf("unexpected notice: %q", notice)
	}

	wantDir := filepath.Join(dataDir, "log-exports")
	if got := filepath.Dir(logFile); got != wantDir {
		t.Fatalf("log directory = %q, want %q", got, wantDir)
	}
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read exported log: %v", err)
	}
	if !strings.Contains(string(content), "tester(user)") || !strings.Contains(string(content), "line") {
		t.Fatalf("exported log does not contain the source entry: %q", content)
	}
	if _, err := os.Stat(logFile); err != nil {
		t.Fatalf("exported log should remain available after GetLogTxt: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(logFile)
		if err != nil {
			t.Fatalf("stat exported log: %v", err)
		}
		if got := info.Mode().Perm(); got != 0o644 {
			t.Fatalf("exported log permissions = %#o, want 0644", got)
		}
	}
}

func TestLogExportFileURIEscapesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log with #.txt")
	uri := logExportFileURI(path)
	if !strings.HasPrefix(uri, "files://") {
		t.Fatalf("URI = %q, want files:// scheme", uri)
	}
	if strings.Contains(uri, " ") || strings.Contains(uri, "#") {
		t.Fatalf("URI contains unescaped path characters: %q", uri)
	}
}
