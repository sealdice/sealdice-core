package gormkv //nolint:testpackage // 需要访问未导出的内部结构

import (
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja_nodejs/cloudflarekv/store"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackendCreatesIsolatedNamespaceRepositories(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	backend, err := NewBackend(BackendOptions{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if migrateErr := backend.AutoMigrate(); migrateErr != nil {
		t.Fatal(migrateErr)
	}

	alpha, err := backend.Namespace("alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := backend.Namespace("beta")
	if err != nil {
		t.Fatal(err)
	}

	if err = alpha.Put(t.Context(), "key", []byte("alpha"), store.PutOptions{ValueKind: store.ValueKindText}); err != nil {
		t.Fatal(err)
	}
	if err = beta.Put(t.Context(), "key", []byte("beta"), store.PutOptions{ValueKind: store.ValueKindText}); err != nil {
		t.Fatal(err)
	}

	alphaRecord, found, err := alpha.Get(t.Context(), "key")
	if err != nil || !found || string(alphaRecord.Value) != "alpha" {
		t.Fatalf("alpha record = %#v, found=%v, err=%v", alphaRecord, found, err)
	}
	betaRecord, found, err := beta.Get(t.Context(), "key")
	if err != nil || !found || string(betaRecord.Value) != "beta" {
		t.Fatalf("beta record = %#v, found=%v, err=%v", betaRecord, found, err)
	}

	if err := alpha.Clear(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, found, err := alpha.Get(t.Context(), "key"); err != nil || found {
		t.Fatalf("alpha record remains after clear: found=%v err=%v", found, err)
	}
	if _, found, err := beta.Get(t.Context(), "key"); err != nil || !found {
		t.Fatalf("beta record was affected by alpha clear: found=%v err=%v", found, err)
	}
}

func TestBackendPurgesExpiredRowsInBoundedBatches(t *testing.T) {
	capture := &queryCaptureLogger{}
	db, err := gorm.Open(sqlite.Open("file:purge-expired?mode=memory&cache=shared"), &gorm.Config{Logger: capture})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	backend, err := NewBackend(BackendOptions{DB: db, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if migrateErr := backend.AutoMigrate(); migrateErr != nil {
		t.Fatal(migrateErr)
	}
	alpha, err := backend.Namespace("alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := backend.Namespace("beta")
	if err != nil {
		t.Fatal(err)
	}
	expired := now.Add(-time.Minute)
	future := now.Add(time.Hour)
	for _, item := range []struct {
		repo       *Repository
		key        string
		expiration *time.Time
	}{
		{alpha, "expired-1", &expired},
		{alpha, "expired-2", &expired},
		{beta, "expired-3", &expired},
		{beta, "live", &future},
	} {
		if putErr := item.repo.Put(t.Context(), item.key, []byte("value"), store.PutOptions{
			Expiration: item.expiration,
			ValueKind:  store.ValueKindText,
		}); putErr != nil {
			t.Fatal(putErr)
		}
	}

	capture.reset()
	first, err := backend.PurgeExpired(t.Context(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if first.Deleted != 2 || !first.HasMore {
		t.Fatalf("first purge = %#v", first)
	}
	queries := capture.selectQueries()
	if len(queries) != 1 {
		t.Fatalf("purge SELECT queries = %#v", queries)
	}
	requireSelectedColumns(t, queries[0], "id")
	var deleteQuery string
	for _, query := range capture.allQueries() {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "DELETE") {
			deleteQuery = query
			break
		}
	}
	if !strings.Contains(deleteQuery, "expires_at") {
		t.Fatalf("purge DELETE did not recheck expiration: %s", deleteQuery)
	}

	second, err := backend.PurgeExpired(t.Context(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if second.Deleted != 1 || second.HasMore {
		t.Fatalf("second purge = %#v", second)
	}
	var remaining int64
	if err := db.Table(DefaultTableName).Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("remaining rows = %d", remaining)
	}
}
