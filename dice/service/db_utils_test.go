package service_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/dice/service"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine/sqlite"
)

type fakeOperator struct {
	db  *gorm.DB
	typ string
}

func (f *fakeOperator) Init(context.Context) error           { return nil }
func (f *fakeOperator) Type() string                         { return f.typ }
func (f *fakeOperator) DBCheck()                             {}
func (f *fakeOperator) GetDataDB(constant.DBMode) *gorm.DB   { return f.db }
func (f *fakeOperator) GetLogDB(constant.DBMode) *gorm.DB    { return f.db }
func (f *fakeOperator) GetCensorDB(constant.DBMode) *gorm.DB { return f.db }
func (f *fakeOperator) Close()                               {}

func freelistCount(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var n int
	if err := db.Raw("PRAGMA freelist_count").Row().Scan(&n); err != nil {
		t.Fatalf("query freelist_count: %v", err)
	}
	return n
}

func prepareDeletedRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	blob := strings.Repeat("x", 512)
	tx := db.Begin()
	for range 2000 {
		if err := tx.Exec("INSERT INTO t (v) VALUES (?)", blob).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("DELETE FROM t").Error; err != nil {
		t.Fatal(err)
	}
}

func TestDBIncrementalVacuumReclaims(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	readDB, writeDB, err := sqlite.SQLiteDBRWInit(path)
	if err != nil {
		t.Fatalf("SQLiteDBRWInit: %v", err)
	}
	defer func() {
		if db, e := readDB.DB(); e == nil {
			_ = db.Close()
		}
		if db, e := writeDB.DB(); e == nil {
			_ = db.Close()
		}
	}()

	prepareDeletedRows(t, writeDB)
	before := freelistCount(t, writeDB)
	if before == 0 {
		t.Fatal("expected free pages after delete")
	}

	service.DBIncrementalVacuum(&fakeOperator{db: writeDB, typ: "sqlite"})

	if after := freelistCount(t, writeDB); after != 0 {
		t.Errorf("freelist_count = %d after DBIncrementalVacuum, want 0 (all free pages should be reclaimed)", after)
	}
}
