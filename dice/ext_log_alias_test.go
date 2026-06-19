package dice

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	sealdiceLogger "sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

func newLogAliasTestDB(t *testing.T) *mockDatabaseOperator {
	t.Helper()

	mockDB, err := newMockDatabaseOperator(filepath.Join(t.TempDir(), "logs.db"))
	if err != nil {
		t.Fatalf("newMockDatabaseOperator: %v", err)
	}
	t.Cleanup(mockDB.Close)

	db := mockDB.GetLogDB(constant.WRITE)
	if err := db.AutoMigrate(&model.LogInfo{}, &model.LogOneItem{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return mockDB
}

func appendTestLog(t *testing.T, mockDB *mockDatabaseOperator, groupID, logName string) {
	t.Helper()

	ok := LogAppend(&MsgContext{Dice: &Dice{DBOperator: mockDB}}, groupID, logName, &model.LogOneItem{
		Nickname: "tester",
		IMUserID: "user",
		Message:  "line",
	})
	if !ok {
		t.Fatalf("LogAppend failed for %q", logName)
	}
}

func TestBuildLogNameAliasIndexUsesWrappedKeysAndResolvesInput(t *testing.T) {
	mockDB := newLogAliasTestDB(t)
	groupID := "QQ-Group:1001"

	appendTestLog(t, mockDB, groupID, "normal-log")
	appendTestLog(t, mockDB, groupID, " spaced\nname ")

	index, err := buildLogNameAliasIndex(mockDB, groupID)
	if err != nil {
		t.Fatalf("buildLogNameAliasIndex: %v", err)
	}
	if len(index.entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(index.entries))
	}

	for _, entry := range index.entries {
		if len(entry.Key) < 8 {
			t.Fatalf("key %q shorter than 8 chars", entry.Key)
		}
		if entry.Name == "normal-log" {
			if got, ok := index.Resolve(entry.Key); !ok || got != entry.Name {
				t.Fatalf("Resolve(%q) = (%q, %v), want (%q, true)", entry.Key, got, ok, entry.Name)
			}
		}
	}

	if got := formatLogNameForDisplay(" spaced\nname "); got != strconv.Quote(" spaced\nname ") {
		t.Fatalf("formatLogNameForDisplay() = %q", got)
	}
	if got := formatLogNameListLine(logNameAliasEntry{Key: "deadbeef", Name: "normal-log"}); got != "- 【deadbeef】: normal-log" {
		t.Fatalf("formatLogNameListLine() = %q", got)
	}
}

func TestBuildLogNameAliasEntriesExtendsConflictingPrefixes(t *testing.T) {
	buckets := map[byte][]string{}
	for i := 0; i < 4096; i++ {
		name := fmt.Sprintf("log-%d", i)
		hash := logNameHashHex(name)
		prefix := hash[0]
		buckets[prefix] = append(buckets[prefix], name)
		if len(buckets[prefix]) >= 2 {
			break
		}
	}

	var names []string
	for _, candidates := range buckets {
		if len(candidates) >= 2 {
			names = candidates[:2]
			break
		}
	}
	if len(names) != 2 {
		t.Fatal("failed to generate colliding 1-char hash prefixes")
	}

	entries, err := buildLogNameAliasEntries(names, 1)
	if err != nil {
		t.Fatalf("buildLogNameAliasEntries: %v", err)
	}

	keys := map[string]struct{}{}
	grew := false
	for _, entry := range entries {
		if _, exists := keys[entry.Key]; exists {
			t.Fatalf("duplicate key %q", entry.Key)
		}
		keys[entry.Key] = struct{}{}
		if len(entry.Key) > 1 {
			grew = true
		}
	}
	if !grew {
		t.Fatal("expected at least one conflicting prefix to grow beyond min length")
	}
}

func TestResolveLogNameForGroupPrefersExactNameBeforeAlias(t *testing.T) {
	mockDB := newLogAliasTestDB(t)
	groupID := "QQ-Group:1002"

	appendTestLog(t, mockDB, groupID, "abcdef12")
	appendTestLog(t, mockDB, groupID, "other-log")

	index, err := buildLogNameAliasIndex(mockDB, groupID)
	if err != nil {
		t.Fatalf("buildLogNameAliasIndex: %v", err)
	}

	var alias string
	for _, entry := range index.entries {
		if entry.Name == "other-log" {
			alias = entry.Key
			break
		}
	}
	if alias == "" {
		t.Fatal("expected alias for other-log")
	}
	if strings.EqualFold(alias, "abcdef12") {
		t.Fatal("test setup produced ambiguous exact-name collision")
	}

	got, err := resolveLogNameForGroup(mockDB, groupID, "abcdef12")
	if err != nil {
		t.Fatalf("resolveLogNameForGroup exact: %v", err)
	}
	if got != "abcdef12" {
		t.Fatalf("resolve exact = %q, want %q", got, "abcdef12")
	}
}

func TestLogSendToBackendAcceptsAliasKey(t *testing.T) {
	mockDB := newLogAliasTestDB(t)
	groupID := "QQ-Group:1003"
	logName := "special name"
	appendTestLog(t, mockDB, groupID, logName)

	index, err := buildLogNameAliasIndex(mockDB, groupID)
	if err != nil {
		t.Fatalf("buildLogNameAliasIndex: %v", err)
	}
	var alias string
	for _, entry := range index.entries {
		if entry.Name == logName {
			alias = entry.Key
			break
		}
	}
	if alias == "" {
		t.Fatal("expected alias for special name")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dice/api/log" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if got := r.FormValue("name"); got != logName {
			t.Fatalf("uploaded name = %q, want %q", got, logName)
		}
		_, _ = w.Write([]byte(`{"url":"https://example.com/log"}`))
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

	unofficial, url, notice, err := LogSendToBackend(ctx, groupID, alias)
	if err != nil {
		t.Fatalf("LogSendToBackend: %v", err)
	}
	if unofficial {
		t.Fatal("expected official backend path")
	}
	if notice != "" {
		t.Fatalf("unexpected notice %q", notice)
	}
	if url != "https://example.com/log" {
		t.Fatalf("url = %q", url)
	}
}
