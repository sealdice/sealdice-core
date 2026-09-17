package sqlite_test

import (
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/dboperator/engine/sqlite"
)

func autoVacuumMode(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var mode int
	if err := db.Raw("PRAGMA auto_vacuum").Row().Scan(&mode); err != nil {
		t.Fatalf("query auto_vacuum: %v", err)
	}
	return mode
}

func freelistCount(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var n int
	if err := db.Raw("PRAGMA freelist_count").Row().Scan(&n); err != nil {
		t.Fatalf("query freelist_count: %v", err)
	}
	return n
}

func openEngine(t *testing.T, path string) (*gorm.DB, *gorm.DB) {
	t.Helper()
	readDB, writeDB, err := sqlite.SQLiteDBRWInit(path)
	if err != nil {
		t.Fatalf("SQLiteDBRWInit: %v", err)
	}
	t.Cleanup(func() {
		if db, e := readDB.DB(); e == nil {
			_ = db.Close()
		}
		if db, e := writeDB.DB(); e == nil {
			_ = db.Close()
		}
	})
	return readDB, writeDB
}

func forceAutoVacuumNone(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Connection(func(tx *gorm.DB) error {
		if e := tx.Exec("PRAGMA auto_vacuum=0").Error; e != nil {
			return e
		}
		return tx.Exec("VACUUM;").Error
	}); err != nil {
		t.Fatalf("force auto_vacuum=NONE: %v", err)
	}
}

func TestFreshDBAutoVacuumIncremental(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	_, writeDB := openEngine(t, path)

	if got := autoVacuumMode(t, writeDB); got != 2 {
		t.Fatalf("fresh DB auto_vacuum = %d, want 2 (INCREMENTAL)", got)
	}

	if err := writeDB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := writeDB.Exec("INSERT INTO t (v) VALUES (?)", "x").Error; err != nil {
		t.Fatal(err)
	}
	if got := autoVacuumMode(t, writeDB); got != 2 {
		t.Errorf("after table creation auto_vacuum = %d, want 2 (setting did not persist)", got)
	}
}

func TestExistingDBWithTablesNotConverted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")

	_, writeDB := openEngine(t, path)
	if err := writeDB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	forceAutoVacuumNone(t, writeDB)
	if got := autoVacuumMode(t, writeDB); got != 0 {
		t.Fatalf("setup: auto_vacuum = %d, want 0", got)
	}
	if db, e := writeDB.DB(); e == nil {
		_ = db.Close()
	}

	// 已有表的库重新打开时不应被静默转换。
	_, writeDB2 := openEngine(t, path)
	if got := autoVacuumMode(t, writeDB2); got != 0 {
		t.Errorf("existing DB auto_vacuum = %d, want 0 (must not be silently converted)", got)
	}
}

func TestConvertToIncrementalAutoVacuum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	_, writeDB := openEngine(t, path)
	if err := writeDB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	forceAutoVacuumNone(t, writeDB)
	if got := autoVacuumMode(t, writeDB); got != 0 {
		t.Fatalf("setup: auto_vacuum = %d, want 0", got)
	}

	if err := sqlite.ConvertToIncrementalAutoVacuum(writeDB); err != nil {
		t.Fatalf("ConvertToIncrementalAutoVacuum: %v", err)
	}
	if got := autoVacuumMode(t, writeDB); got != 2 {
		t.Errorf("after convert auto_vacuum = %d, want 2", got)
	}
}

func TestReclaimIncrementalVacuum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	_, writeDB := openEngine(t, path)
	if err := writeDB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil {
		t.Fatal(err)
	}

	blob := strings.Repeat("x", 512)
	tx := writeDB.Begin()
	for range 2000 {
		if err := tx.Exec("INSERT INTO t (v) VALUES (?)", blob).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := writeDB.Exec("DELETE FROM t").Error; err != nil {
		t.Fatal(err)
	}

	if got := autoVacuumMode(t, writeDB); got != 2 {
		t.Fatalf("auto_vacuum = %d, want 2", got)
	}
	before := freelistCount(t, writeDB)
	if before == 0 {
		t.Fatalf("expected free pages after delete, got 0")
	}

	reclaimed, err := sqlite.ReclaimIncrementalVacuum(writeDB)
	if err != nil {
		t.Fatalf("ReclaimIncrementalVacuum: %v", err)
	}
	after := freelistCount(t, writeDB)
	if after >= before {
		t.Errorf("freelist_count = %d after reclaim, want < %d", after, before)
	}
	if reclaimed != before-after {
		t.Errorf("reclaimed = %d, want before-after = %d", reclaimed, before-after)
	}
}

func TestReclaimIncrementalVacuumNoopWhenNotIncremental(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	_, writeDB := openEngine(t, path)
	if err := writeDB.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	forceAutoVacuumNone(t, writeDB)
	if got := autoVacuumMode(t, writeDB); got != 0 {
		t.Fatalf("setup: auto_vacuum = %d, want 0", got)
	}

	reclaimed, err := sqlite.ReclaimIncrementalVacuum(writeDB)
	if err != nil {
		t.Fatalf("ReclaimIncrementalVacuum: %v", err)
	}
	if reclaimed != 0 {
		t.Errorf("reclaimed = %d, want 0 (must be a no-op when not incremental)", reclaimed)
	}
}
