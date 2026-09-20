//nolint:testpackage
package dice

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	sealdiceLogger "sealdice-core/logger"
)

func TestLogExportStagesOneBotFileInDataTemp(t *testing.T) {
	mockDB := newLogAliasTestDB(t)
	const groupID = "QQ-Group:1007"
	const logName = "shared-volume-log"
	appendTestLog(t, mockDB, groupID, logName)

	dataDir := filepath.Join(t.TempDir(), "data", "default")
	d := &Dice{
		BaseConfig: BaseConfig{DataDir: dataDir},
		DBOperator: mockDB,
		Logger:     sealdiceLogger.M(),
	}
	ctx := &MsgContext{Dice: d}
	logFile, _, err := GetLogTxt(ctx, groupID, logName, "export")
	if err != nil {
		t.Fatalf("GetLogTxt: %v", err)
	}
	tempDir := filepath.Join(dataDir, "temp")
	if got := filepath.Dir(logFile); got != tempDir {
		t.Fatalf("log directory = %q, want %q", got, tempDir)
	}
	sourceData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log export: %v", err)
	}

	pa := &PlatformAdapterGocq{}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{Session: &IMSession{Parent: d}}, Adapter: pa}
	pa.EndPoint = ep
	ctx.EndPoint = ep
	pa.sendFileToGroup(ctx, groupID, logFile, "skip", tempDir)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("read temp directory: %v", err)
	}
	var stagedFile string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "temp-") {
			if stagedFile != "" {
				t.Fatal("multiple staged upload files")
			}
			stagedFile = filepath.Join(tempDir, entry.Name())
		}
	}
	if stagedFile == "" {
		t.Fatal("OneBot upload file was not staged in data temp directory")
	}
	stagedData, err := os.ReadFile(stagedFile)
	if err != nil {
		t.Fatalf("read staged upload: %v", err)
	}
	if !bytes.Equal(stagedData, sourceData) {
		t.Fatal("staged upload differs from exported log")
	}
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(stagedFile)
		if statErr != nil {
			t.Fatalf("stat staged upload: %v", statErr)
		}
		if got := info.Mode().Perm(); got != 0o644 {
			t.Fatalf("staged upload permissions = %#o, want 0644", got)
		}
	}
	if err := os.Remove(logFile); err != nil {
		t.Fatalf("remove source export: %v", err)
	}
	if _, err := os.Stat(stagedFile); err != nil {
		t.Fatalf("staged upload must survive source cleanup: %v", err)
	}
}
