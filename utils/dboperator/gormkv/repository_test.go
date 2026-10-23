package gormkv //nolint:testpackage // 需要访问未导出的内部结构

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"sealdice-core/utils/dboperator/gormkv/codec"
)

func newTestRepo(t *testing.T, db *gorm.DB, namespace string, compression codec.Options) (*Repository, error) {
	t.Helper()
	backend, err := NewBackend(BackendOptions{DB: db, Compression: compression})
	if err != nil {
		return nil, err
	}
	return backend.Namespace(namespace)
}

type queryCaptureLogger struct {
	mu      sync.Mutex
	queries []string
}

func (l *queryCaptureLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }
func (l *queryCaptureLogger) Info(context.Context, string, ...interface{})     {}
func (l *queryCaptureLogger) Warn(context.Context, string, ...interface{})     {}
func (l *queryCaptureLogger) Error(context.Context, string, ...interface{})    {}
func (l *queryCaptureLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.mu.Lock()
	l.queries = append(l.queries, sql)
	l.mu.Unlock()
}

func (l *queryCaptureLogger) reset() {
	l.mu.Lock()
	l.queries = nil
	l.mu.Unlock()
}

func (l *queryCaptureLogger) selectQueries() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	queries := make([]string, 0, len(l.queries))
	for _, query := range l.queries {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
			queries = append(queries, query)
		}
	}
	return queries
}

func (l *queryCaptureLogger) allQueries() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.queries...)
}

func requireSelectedColumns(t *testing.T, query string, required ...string) {
	t.Helper()
	selectClause := strings.SplitN(query, " FROM ", 2)[0]
	if strings.Contains(selectClause, "*") {
		t.Fatalf("query selected all columns: %s", query)
	}
	for _, column := range required {
		if !strings.Contains(selectClause, column) {
			t.Errorf("query did not select %s: %s", column, query)
		}
	}
}

func TestRepositoryQueriesSelectOnlyRequiredColumns(t *testing.T) {
	capture := &queryCaptureLogger{}
	db, err := gorm.Open(sqlite.Open("file:query-projections?mode=memory&cache=shared"), &gorm.Config{Logger: capture})
	if err != nil {
		t.Fatal(err)
	}
	if migrateErr := AutoMigrate(db); migrateErr != nil {
		t.Fatal(migrateErr)
	}
	repo, err := newTestRepo(t, db, "projection", codec.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if putErr := repo.Put(t.Context(), "key", []byte("value"), store.PutOptions{ValueKind: store.ValueKindText}); putErr != nil {
		t.Fatal(putErr)
	}

	capture.reset()
	if putErr := repo.Put(t.Context(), "key", []byte("updated"), store.PutOptions{ValueKind: store.ValueKindText}); putErr != nil {
		t.Fatal(putErr)
	}
	queries := capture.selectQueries()
	if len(queries) != 1 {
		t.Fatalf("Put SELECT queries = %#v", queries)
	}
	requireSelectedColumns(t, queries[0], "raw_key")
	if strings.Contains(strings.SplitN(queries[0], " FROM ", 2)[0], "value_blob") {
		t.Fatalf("Put collision query selected value_blob: %s", queries[0])
	}

	capture.reset()
	record, found, err := repo.Get(t.Context(), "key")
	if err != nil || !found || record.Key != "key" || string(record.Value) != "updated" {
		t.Fatalf("Get record=%#v found=%v err=%v", record, found, err)
	}
	queries = capture.selectQueries()
	if len(queries) != 1 {
		t.Fatalf("Get SELECT queries = %#v", queries)
	}
	requireSelectedColumns(t, queries[0], "value_blob", "value_codec", "metadata_json", "expires_at")

	capture.reset()
	if _, err := repo.List(t.Context(), store.ListOptions{Limit: 10}); err != nil {
		t.Fatal(err)
	}
	queries = capture.selectQueries()
	if len(queries) != 1 {
		t.Fatalf("List SELECT queries = %#v", queries)
	}
	requireSelectedColumns(t, queries[0], "raw_key", "metadata_json", "expires_at")
	if strings.Contains(strings.SplitN(queries[0], " FROM ", 2)[0], "value_blob") {
		t.Fatalf("List query selected value_blob: %s", queries[0])
	}
}

func TestRepositoryStoresRawValuesWithoutImplicitMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err = AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo, err := newTestRepo(t, db, "plugin", codec.Options{})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err = repo.Put(ctx, "key", []byte("value"), store.PutOptions{ValueKind: store.ValueKindText}); err != nil {
		t.Fatal(err)
	}
	record, found, err := repo.Get(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	if !found || string(record.Value) != "value" {
		t.Fatalf("unexpected record: found=%v value=%q", found, record.Value)
	}
	var row kvRecord
	if err = db.Table(DefaultTableName).Where("namespace = ? AND raw_key = ?", "plugin", "key").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if string(row.ValueBlob) != "value" || row.ValueCodec != codec.CodecRaw {
		t.Fatalf("stored raw value = %q codec=%d", row.ValueBlob, row.ValueCodec)
	}

	count, err := repo.Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("unexpected count: %d, %v", count, err)
	}
	if err = repo.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	count, err = repo.Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("unexpected count after clear: %d, %v", count, err)
	}
}

func TestRepositoryStoresEmptyRawValueAsNonNullBlob(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo, err := newTestRepo(t, db, "empty-value", codec.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if err = repo.Put(context.Background(), "empty", []byte{}, store.PutOptions{ValueKind: store.ValueKindText}); err != nil {
		t.Fatalf("put empty value: %v", err)
	}

	var record kvRecord
	if err = db.Where("namespace = ? AND raw_key = ?", "empty-value", "empty").First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.ValueBlob == nil || len(record.ValueBlob) != 0 {
		t.Fatalf("expected non-nil empty blob, got %#v", record.ValueBlob)
	}
	got, found, err := repo.Get(context.Background(), "empty")
	if err != nil {
		t.Fatal(err)
	}
	if !found || got.Value == nil || len(got.Value) != 0 {
		t.Fatalf("expected found empty value, got found=%v value=%#v", found, got.Value)
	}
}

func TestRepositoryPreservesCompressedValueCompatibility(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	compressedRepo, err := newTestRepo(t, db, "compressed", codec.Options{Enabled: true, MinSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	value := []byte(strings.Repeat("repeated-value-", 512))
	if err = compressedRepo.Put(context.Background(), "key", value, store.PutOptions{ValueKind: store.ValueKindText}); err != nil {
		t.Fatal(err)
	}

	var row kvRecord
	if err = db.Table(DefaultTableName).Where("namespace = ? AND raw_key = ?", "compressed", "key").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.ValueCodec != codec.CodecZstd {
		t.Fatalf("stored codec = %d, want zstd", row.ValueCodec)
	}

	plainRepo, err := newTestRepo(t, db, "compressed", codec.Options{})
	if err != nil {
		t.Fatal(err)
	}
	record, found, err := plainRepo.Get(context.Background(), "key")
	if err != nil || !found || string(record.Value) != string(value) {
		t.Fatalf("compressed value readback = %q found=%v err=%v", record.Value, found, err)
	}
}

func TestAutoMigrateCreatesQueryAndExpirationIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:gormkv-indexes?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	assertIndexColumns := func(indexName string, want ...string) {
		t.Helper()
		var rows []struct {
			Name string `gorm:"column:name"`
		}
		query := "PRAGMA index_info('" + indexName + "')"
		if err := db.Raw(query).Scan(&rows).Error; err != nil {
			t.Fatal(err)
		}
		got := make([]string, 0, len(rows))
		for _, row := range rows {
			got = append(got, row.Name)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("index %s columns = %v, want %v", indexName, got, want)
		}
	}

	assertIndexColumns("idx_namespace_raw_key", "namespace", "raw_key")
	assertIndexColumns("idx_namespace_expires_at", "namespace", "expires_at")
}
