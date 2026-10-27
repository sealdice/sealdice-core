//nolint:testpackage
package dice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	sealdiceLogger "sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
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

func TestLogExportErrorReplyHidesFullPath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "log-exports", "export.txt")
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "create file",
			err: fmt.Errorf("创建日志导出文件失败: %w", &os.PathError{
				Op: "open", Path: filePath, Err: os.ErrPermission,
			}),
			want: "创建日志导出文件失败: open export.txt: permission denied",
		},
		{
			name: "create directory",
			err: fmt.Errorf("创建日志导出目录失败: %w", &os.PathError{
				Op: "mkdir", Path: filepath.Dir(filePath), Err: os.ErrPermission,
			}),
			want: "创建日志导出目录失败: mkdir log-exports: permission denied",
		},
		{
			name: "write file",
			err: fmt.Errorf("写入日志导出临时文件失败: %w", &os.PathError{
				Op: "write", Path: filePath, Err: os.ErrPermission,
			}),
			want: "写入日志导出临时文件失败: write export.txt: permission denied",
		},
		{
			name: "bare filesystem error",
			err:  &os.PathError{Op: "chmod", Path: filePath, Err: os.ErrPermission},
			want: "chmod export.txt: permission denied",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original := test.err.Error()
			if got := logExportErrorReply(test.err); got != test.want {
				t.Fatalf("error reply = %q, want %q", got, test.want)
			}
			if got := test.err.Error(); got != original || !strings.Contains(got, filepath.Dir(filePath)) {
				t.Fatalf("original error should retain the full path for logging: %q", got)
			}
			if !errors.Is(test.err, os.ErrPermission) {
				t.Fatal("original error should retain the filesystem cause")
			}
		})
	}
}

func TestLogExportErrorReplyPreservesOtherErrors(t *testing.T) {
	for _, text := range []string{
		"此log不存在，或条目数为空，名字是否正确？",
		"日志导出超时（10秒限制），请尝试减少数据量或联系管理员",
	} {
		if got := logExportErrorReply(errors.New(text)); got != text {
			t.Fatalf("error reply = %q, want %q", got, text)
		}
	}
}

func TestLogExportFailureHidesPathInGroupReply(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()
	if err := d.DBOperator.GetLogDB(constant.WRITE).AutoMigrate(&model.LogInfo{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	core, logs := observer.New(zapcore.DebugLevel)
	d.Logger = zap.New(core).Sugar()

	// 用同名文件阻止创建导出目录，稳定触发 Windows 和 Unix 上的文件系统错误。
	exportDir := filepath.Join(d.BaseConfig.DataDir, "log-exports")
	if err := os.Remove(exportDir); err != nil {
		t.Fatalf("remove empty export directory: %v", err)
	}
	if err := os.WriteFile(exportDir, []byte("block directory creation"), 0o600); err != nil {
		t.Fatalf("block export directory: %v", err)
	}

	d.ImSession.ExecuteNew(ep, newGroupMsg("QQ-Group:1010", "QQ:999", ".log export shared-export"))
	reply, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("expected export failure reply")
	}
	if !strings.Contains(reply, "创建日志导出目录失败") || !strings.Contains(reply, "log-exports") {
		t.Fatalf("reply should include the failure reason and directory name: %q", reply)
	}
	if strings.Contains(reply, d.BaseConfig.DataDir) {
		t.Fatalf("reply contains the full data directory path: %q", reply)
	}
	entries := logs.FilterMessageSnippet("导出日志失败").All()
	if len(entries) != 1 || !strings.Contains(entries[0].Message, exportDir) {
		t.Fatalf("server log should retain the full export path: %#v", entries)
	}
}
