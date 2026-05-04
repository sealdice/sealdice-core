package storylog

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/parquet-go/parquet-go"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

type exportFixtureLine struct {
	Nickname    string
	IMUserID    string
	Time        int64
	Message     string
	CommandID   int64
	CommandInfo map[string]interface{}
	RawMsgID    string
	UniformID   string
}

const contentTestBaseTimestamp int64 = 1_700_000_000

type contentTestDatabaseOperator struct {
	db *gorm.DB
}

func (m *contentTestDatabaseOperator) Init(_ context.Context) error           { return nil }
func (m *contentTestDatabaseOperator) Type() string                           { return "content-test-sqlite" }
func (m *contentTestDatabaseOperator) DBCheck()                               {}
func (m *contentTestDatabaseOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return m.db }
func (m *contentTestDatabaseOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return m.db }
func (m *contentTestDatabaseOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return m.db }
func (m *contentTestDatabaseOperator) Close() {
	if sqlDB, err := m.db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

type contentTestCountingReader struct {
	reader io.Reader
	n      int64
}

func (r *contentTestCountingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.n += int64(n)
	return n, err
}

type contentTestUploadCapture struct {
	t               *testing.T
	expectedVersion StoryVersion
	expectedClient  string
	expectedURL     string

	mu           sync.Mutex
	requestCount int
	uploadedSize int64
	err          error
}

func (c *contentTestUploadCapture) failf(format string, args ...interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.err = fmt.Errorf(format, args...)
	}
}

func (c *contentTestUploadCapture) setUploadedSize(n int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.uploadedSize = n
}

func (c *contentTestUploadCapture) snapshot() (int, int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requestCount, c.uploadedSize, c.err
}

func (c *contentTestUploadCapture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		c.failf("unexpected method: %s", r.Method)
		http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		c.failf("create multipart reader: %v", err)
		http.Error(w, "invalid multipart body", http.StatusBadRequest)
		return
	}

	var (
		clientValue  string
		versionValue string
		fileBytes    int64
		sawFile      bool
	)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.failf("read multipart part: %v", err)
			http.Error(w, "read multipart part failed", http.StatusBadRequest)
			return
		}

		switch part.FormName() {
		case "client":
			data, readErr := io.ReadAll(part)
			if readErr != nil {
				c.failf("read client field: %v", readErr)
				http.Error(w, "read client field failed", http.StatusBadRequest)
				return
			}
			clientValue = string(data)
		case "version":
			data, readErr := io.ReadAll(part)
			if readErr != nil {
				c.failf("read version field: %v", readErr)
				http.Error(w, "read version field failed", http.StatusBadRequest)
				return
			}
			versionValue = string(data)
		case "file":
			sawFile = true
			switch c.expectedVersion {
			case StoryVersionV1:
				fileBytes, err = validateContentTestZlibJSONPart(part)
			case StoryVersionV105:
				fileBytes, err = validateContentTestParquetPart(part)
			default:
				err = fmt.Errorf("unexpected version %d", c.expectedVersion)
			}
			if err != nil {
				c.failf("validate upload body: %v", err)
				http.Error(w, "invalid upload payload", http.StatusBadRequest)
				return
			}
		default:
			if _, readErr := io.Copy(io.Discard, part); readErr != nil {
				c.failf("discard field %q: %v", part.FormName(), readErr)
				http.Error(w, "discard multipart field failed", http.StatusBadRequest)
				return
			}
		}
	}

	if !sawFile {
		c.failf("missing file field")
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	if clientValue != contentTestExpectedClientForVersion(c.expectedVersion) {
		c.failf("unexpected client field: got %q want %q", clientValue, c.expectedClient)
		http.Error(w, "unexpected client field", http.StatusBadRequest)
		return
	}
	if versionValue != fmt.Sprintf("%d", c.expectedVersion) {
		c.failf("unexpected version field: got %q want %d", versionValue, c.expectedVersion)
		http.Error(w, "unexpected version field", http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	c.requestCount++
	c.mu.Unlock()
	c.setUploadedSize(fileBytes)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": c.expectedURL})
}

func validateContentTestZlibJSONPart(part *multipart.Part) (int64, error) {
	countingPart := &contentTestCountingReader{reader: part}
	zlibReader, err := zlib.NewReader(countingPart)
	if err != nil {
		return 0, fmt.Errorf("create zlib reader: %w", err)
	}
	defer func() { _ = zlibReader.Close() }()

	decoder := json.NewDecoder(zlibReader)
	token, err := decoder.Token()
	if err != nil {
		return 0, fmt.Errorf("read first json token: %w", err)
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return 0, fmt.Errorf("unexpected first json token: %v", token)
	}
	if _, err := io.Copy(io.Discard, zlibReader); err != nil {
		return 0, fmt.Errorf("drain zlib payload: %w", err)
	}
	if countingPart.n == 0 {
		return 0, fmt.Errorf("empty uploaded payload")
	}
	return countingPart.n, nil
}

func validateContentTestParquetPart(part *multipart.Part) (int64, error) {
	countingPart := &contentTestCountingReader{reader: part}
	var header [4]byte
	if _, err := io.ReadFull(countingPart, header[:]); err != nil {
		return 0, fmt.Errorf("read parquet header: %w", err)
	}
	if string(header[:]) != "PAR1" {
		return 0, fmt.Errorf("unexpected parquet magic: %q", string(header[:]))
	}
	if _, err := io.Copy(io.Discard, countingPart); err != nil {
		return 0, fmt.Errorf("drain parquet payload: %w", err)
	}
	return countingPart.n, nil
}

func newContentTestDatabaseOperator(t *testing.T) *contentTestDatabaseOperator {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "storylog-content.db")
	db, err := openContentTestGormDB(dbPath)
	if err != nil {
		t.Fatalf("open test sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.LogInfo{}, &model.LogOneItem{}); err != nil {
		t.Fatalf("auto migrate log tables: %v", err)
	}

	return &contentTestDatabaseOperator{db: db}
}

func contentTestExpectedClientForVersion(version StoryVersion) string {
	if version == StoryVersionV105 {
		return "Parquet"
	}
	return "SealDice"
}

func TestUploadV1ExportArtifacts(t *testing.T) {
	fixtures := []exportFixtureLine{
		{
			Nickname:    "alpha",
			IMUserID:    "QQ:1001",
			Time:        contentTestBaseTimestamp,
			Message:     "first message",
			CommandID:   11,
			CommandInfo: map[string]interface{}{"phase": "open"},
			RawMsgID:    "raw-alpha",
			UniformID:   "QQ:1001",
		},
		{
			Nickname:    "beta",
			IMUserID:    "QQ:1002",
			Time:        contentTestBaseTimestamp + 1,
			Message:     "second message\nwith newline",
			CommandID:   12,
			CommandInfo: map[string]interface{}{"phase": "close"},
			RawMsgID:    "raw-beta",
			UniformID:   "QQ:1002",
		},
	}

	zipEntries := runUploadAndReadZip(t, StoryVersionV1, fixtures)

	expectedTXT := expectedTXTContent(fixtures)
	if got := string(zipEntries[ExportTxtFilename]); got != expectedTXT {
		t.Fatalf("unexpected TXT export:\n%s", got)
	}

	var payload struct {
		Items   []model.LogOneItem `json:"items"`
		Version StoryVersion       `json:"version"`
	}
	if err := json.Unmarshal(zipEntries[ExportJsonFilename], &payload); err != nil {
		t.Fatalf("unmarshal exported json: %v", err)
	}
	if payload.Version != StoryVersionV1 {
		t.Fatalf("unexpected json version: got %d want %d", payload.Version, StoryVersionV1)
	}
	if len(payload.Items) != len(fixtures) {
		t.Fatalf("unexpected json item count: got %d want %d", len(payload.Items), len(fixtures))
	}
	if payload.Items[1].Message != fixtures[1].Message {
		t.Fatalf("unexpected second message: got %q want %q", payload.Items[1].Message, fixtures[1].Message)
	}
}

func TestUploadV105ExportArtifacts(t *testing.T) {
	fixtures := []exportFixtureLine{
		{
			Nickname:    "gamma",
			IMUserID:    "QQ:2001",
			Time:        contentTestBaseTimestamp + 10,
			Message:     "parquet one",
			CommandID:   21,
			CommandInfo: map[string]interface{}{"phase": "one"},
			RawMsgID:    "raw-gamma",
			UniformID:   "QQ:2001",
		},
		{
			Nickname:    "delta",
			IMUserID:    "QQ:2002",
			Time:        contentTestBaseTimestamp + 11,
			Message:     "parquet two",
			CommandID:   22,
			CommandInfo: map[string]interface{}{"phase": "two"},
			RawMsgID:    "raw-delta",
			UniformID:   "QQ:2002",
		},
	}

	zipEntries := runUploadAndReadZip(t, StoryVersionV105, fixtures)

	expectedTXT := expectedTXTContent(fixtures)
	if got := string(zipEntries[ExportTxtFilename]); got != expectedTXT {
		t.Fatalf("unexpected TXT export:\n%s", got)
	}

	parquetRows, err := parquet.Read[model.LogOneItemParquet](
		bytes.NewReader(zipEntries[ExportParquetFilename]),
		int64(len(zipEntries[ExportParquetFilename])),
	)
	if err != nil {
		t.Fatalf("read exported parquet: %v", err)
	}
	if len(parquetRows) != len(fixtures) {
		t.Fatalf("unexpected parquet row count: got %d want %d", len(parquetRows), len(fixtures))
	}
	if parquetRows[0].Message != fixtures[0].Message {
		t.Fatalf("unexpected first parquet message: got %q want %q", parquetRows[0].Message, fixtures[0].Message)
	}
	if parquetRows[1].UniformID != fixtures[1].UniformID {
		t.Fatalf("unexpected second parquet uniform id: got %q want %q", parquetRows[1].UniformID, fixtures[1].UniformID)
	}
}

func TestUploadV1UnsafeLogNameStillExports(t *testing.T) {
	fixtures := []exportFixtureLine{
		{
			Nickname:    "unsafe",
			IMUserID:    "QQ:3001",
			Time:        contentTestBaseTimestamp + 20,
			Message:     "unsafe log name payload",
			CommandID:   31,
			CommandInfo: map[string]interface{}{"phase": "unsafe"},
			RawMsgID:    "raw-unsafe",
			UniformID:   "QQ:3001",
		},
	}

	logName := strings.Repeat("危险/日志:*?名\n", 20) + "CON"
	zipEntries := runUploadAndReadZipWithName(t, StoryVersionV1, logName, fixtures)
	if got := string(zipEntries[ExportTxtFilename]); got != expectedTXTContent(fixtures) {
		t.Fatalf("unexpected TXT export for unsafe name:\n%s", got)
	}
}

func TestUploadV105UnsafeLogNameStillExports(t *testing.T) {
	fixtures := []exportFixtureLine{
		{
			Nickname:    "unsafe-v105",
			IMUserID:    "QQ:3002",
			Time:        contentTestBaseTimestamp + 21,
			Message:     "unsafe log name parquet payload",
			CommandID:   32,
			CommandInfo: map[string]interface{}{"phase": "unsafe-v105"},
			RawMsgID:    "raw-unsafe-v105",
			UniformID:   "QQ:3002",
		},
	}

	logName := strings.Repeat("超长\\parquet|日志:?名\t", 20) + "LPT1"
	zipEntries := runUploadAndReadZipWithName(t, StoryVersionV105, logName, fixtures)
	if got := string(zipEntries[ExportTxtFilename]); got != expectedTXTContent(fixtures) {
		t.Fatalf("unexpected TXT export for unsafe V105 name:\n%s", got)
	}
}

func runUploadAndReadZip(t *testing.T, version StoryVersion, fixtures []exportFixtureLine) map[string][]byte {
	t.Helper()
	return runUploadAndReadZipWithName(t, version, fmt.Sprintf("content-log-%d", version), fixtures)
}

func runUploadAndReadZipWithName(t *testing.T, version StoryVersion, logName string, fixtures []exportFixtureLine) map[string][]byte {
	t.Helper()

	operator := newContentTestDatabaseOperator(t)
	defer operator.Close()

	groupID := fmt.Sprintf("content-group-%d", version)
	seedExportFixtureLog(t, operator.db, groupID, logName, fixtures)

	capture := &contentTestUploadCapture{
		t:               t,
		expectedVersion: version,
		expectedClient:  contentTestExpectedClientForVersion(version),
		expectedURL:     fmt.Sprintf("http://example.test/content/%d", version),
	}
	server := httptest.NewServer(capture)
	defer server.Close()

	exportDir := t.TempDir()
	env := UploadEnv{
		Dir:       exportDir,
		Db:        operator,
		Log:       zap.NewNop().Sugar(),
		Backends:  []string{server.URL},
		Version:   version,
		LogName:   logName,
		UniformID: "QQ:test-user",
		GroupID:   groupID,
	}

	url, err := Upload(env)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if url != capture.expectedURL {
		t.Fatalf("unexpected upload url: got %q want %q", url, capture.expectedURL)
	}

	requestCount, uploadedSize, handlerErr := capture.snapshot()
	if handlerErr != nil {
		t.Fatalf("upload handler validation failed: %v", handlerErr)
	}
	if requestCount != 1 {
		t.Fatalf("unexpected request count: got %d want 1", requestCount)
	}
	if uploadedSize <= 0 {
		t.Fatalf("uploaded payload was empty")
	}

	entries := readSingleZip(t, exportDir)
	if _, ok := entries[ExportReadmeFilename]; !ok {
		t.Fatalf("zip missing %s", ExportReadmeFilename)
	}
	if _, ok := entries[ExportTxtFilename]; !ok {
		t.Fatalf("zip missing %s", ExportTxtFilename)
	}
	if version == StoryVersionV1 {
		if _, ok := entries[ExportJsonFilename]; !ok {
			t.Fatalf("zip missing %s", ExportJsonFilename)
		}
	} else {
		if _, ok := entries[ExportParquetFilename]; !ok {
			t.Fatalf("zip missing %s", ExportParquetFilename)
		}
	}
	return entries
}

func seedExportFixtureLog(t *testing.T, db *gorm.DB, groupID string, logName string, fixtures []exportFixtureLine) {
	t.Helper()

	logInfo := model.LogInfo{
		Name:      logName,
		GroupID:   groupID,
		CreatedAt: fixtures[0].Time,
		UpdatedAt: fixtures[len(fixtures)-1].Time,
	}
	if err := db.Create(&logInfo).Error; err != nil {
		t.Fatalf("create log info: %v", err)
	}

	items := make([]model.LogOneItem, 0, len(fixtures))
	for _, fixture := range fixtures {
		items = append(items, model.LogOneItem{
			LogID:       logInfo.ID,
			GroupID:     groupID,
			Nickname:    fixture.Nickname,
			IMUserID:    fixture.IMUserID,
			Time:        fixture.Time,
			Message:     fixture.Message,
			CommandID:   fixture.CommandID,
			CommandInfo: fixture.CommandInfo,
			RawMsgID:    fixture.RawMsgID,
			UniformID:   fixture.UniformID,
		})
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("create log items: %v", err)
	}
	if err := db.Model(&model.LogInfo{}).Where("id = ?", logInfo.ID).Update("size", len(items)).Error; err != nil {
		t.Fatalf("update log size: %v", err)
	}
}

func expectedTXTContent(fixtures []exportFixtureLine) string {
	var buf bytes.Buffer
	for _, fixture := range fixtures {
		buf.WriteString(FormatLogTxtLine(fixture.Nickname, fixture.IMUserID, fixture.Time, fixture.Message))
	}
	return buf.String()
}

func readSingleZip(t *testing.T, dir string) map[string][]byte {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read export dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected export file count: got %d want 1", len(entries))
	}

	zipPath := filepath.Join(dir, entries[0].Name())
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = reader.Close() }()

	result := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", file.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read zip entry %s: %v", file.Name, err)
		}
		result[file.Name] = data
	}
	return result
}
