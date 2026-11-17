//nolint:testpackage
package dice

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sealdiceLogger "sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

func TestLogSendToBackendReuploadsMissingCachedURL(t *testing.T) {
	mockDB := newLogAliasTestDB(t)
	groupID := "QQ-Group:1004"
	logName := "expired-upload"
	appendTestLog(t, mockDB, groupID, logName)

	oldURL := "https://logs.example/?key=Old1#123456"
	db := mockDB.GetLogDB(constant.WRITE)
	if err := db.Model(&model.LogInfo{}).
		Where("group_id = ? AND name = ?", groupID, logName).
		UpdateColumns(map[string]interface{}{
			"upload_url":  oldURL,
			"upload_time": time.Now().Add(time.Minute).Unix(),
		}).Error; err != nil {
		t.Fatalf("seed cached upload: %v", err)
	}

	var probeCount atomic.Int32
	var uploadCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/dice/api/load_data":
			probeCount.Add(1)
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && r.URL.Path == "/dice/api/log":
			uploadCount.Add(1)
			_, _ = w.Write([]byte(`{"url":"https://logs.example/?key=New1#654321"}`))
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	oldBackends := BackendUrls
	BackendUrls = []string{server.URL}
	t.Cleanup(func() { BackendUrls = oldBackends })

	ctx := &MsgContext{
		Dice: &Dice{
			BaseConfig: BaseConfig{DataDir: t.TempDir()},
			DBOperator: mockDB,
			Logger:     sealdiceLogger.M(),
		},
		EndPoint: &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "UI:1000"}},
	}

	_, gotURL, _, err := LogSendToBackend(ctx, groupID, logName)
	if err != nil {
		t.Fatalf("LogSendToBackend: %v", err)
	}
	if gotURL != "https://logs.example/?key=New1#654321" {
		t.Fatalf("url = %q, want newly uploaded URL", gotURL)
	}
	if got := probeCount.Load(); got != 1 {
		t.Fatalf("probe requests = %d, want 1", got)
	}
	if got := uploadCount.Load(); got != 1 {
		t.Fatalf("upload requests = %d, want 1", got)
	}
}

func TestLogSendToBackendFallsBackToCachedURLWhenProbeAndUploadFail(t *testing.T) {
	mockDB := newLogAliasTestDB(t)
	groupID := "QQ-Group:1005"
	logName := "uncertain-upload"
	appendTestLog(t, mockDB, groupID, logName)

	oldURL := "https://logs.example/?key=Old2#123456"
	db := mockDB.GetLogDB(constant.WRITE)
	if err := db.Model(&model.LogInfo{}).
		Where("group_id = ? AND name = ?", groupID, logName).
		UpdateColumns(map[string]interface{}{
			"upload_url":  oldURL,
			"upload_time": time.Now().Add(time.Minute).Unix(),
		}).Error; err != nil {
		t.Fatalf("seed cached upload: %v", err)
	}

	var uploadCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/custom/path/load_data":
			http.Error(w, "temporary failure", http.StatusInternalServerError)
		case r.Method == http.MethodPut && r.URL.Path == "/custom/path/log":
			uploadCount.Add(1)
			http.Error(w, "upload failed", http.StatusInternalServerError)
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	ctx := &MsgContext{
		Dice: &Dice{
			BaseConfig: BaseConfig{DataDir: t.TempDir()},
			DBOperator: mockDB,
			Logger:     sealdiceLogger.M(),
			AdvancedConfig: AdvancedConfig{
				Enable:             true,
				StoryLogBackendUrl: server.URL + "/custom/path/log",
			},
		},
		EndPoint: &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "UI:1000"}},
	}

	_, gotURL, notice, err := LogSendToBackend(ctx, groupID, logName)
	if err != nil {
		t.Fatalf("LogSendToBackend: %v", err)
	}
	if gotURL != oldURL {
		t.Fatalf("url = %q, want cached URL %q", gotURL, oldURL)
	}
	if !strings.Contains(notice, "重新上传日志失败") || !strings.Contains(notice, "可能仍然有效") {
		t.Fatalf("notice = %q, want upload failure and uncertain cached URL warning", notice)
	}
	if got := uploadCount.Load(); got != 1 {
		t.Fatalf("upload requests = %d, want 1", got)
	}
}
