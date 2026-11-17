package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/utils/dboperator/engine/sqlite"
)

func assertPragma(t *testing.T, conn *sql.Conn, ctx context.Context, label, pragma string, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRowContext(ctx, "PRAGMA "+pragma).Scan(&got); err != nil {
		t.Fatalf("%s: query PRAGMA %s: %v", label, pragma, err)
	}
	if got != want {
		t.Errorf("%s: PRAGMA %s = %d, want %d (new connection did not receive connection-local pragma)", label, pragma, got, want)
	}
}

// 连接级 pragma（busy_timeout/synchronous/cache_size）必须在连接池后续新建的
// 每个连接上都生效，而不是只作用于 SetDefaultPragmas 当时借用的那一个连接。
func TestConnLocalPragmasAppliedToAllConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")

	readDB, writeDB, err := sqlite.SQLiteDBRWInit(path)
	if err != nil {
		t.Fatalf("SQLiteDBRWInit: %v", err)
	}
	defer func() {
		if sqlDB, err := readDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		if sqlDB, err := writeDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	checkPool := func(db *gorm.DB, label string) {
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("%s: DB(): %v", label, err)
		}
		sqlDB.SetMaxOpenConns(4)

		const n = 4
		ctx := t.Context()
		conns := make([]*sql.Conn, n)
		errs := make([]error, n)

		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				conns[i], errs[i] = sqlDB.Conn(ctx)
			}(i)
		}
		close(start)
		wg.Wait()

		if stats := sqlDB.Stats(); stats.OpenConnections != n {
			t.Fatalf("%s: OpenConnections = %d, want %d (could not force distinct connections)", label, stats.OpenConnections, n)
		}
		for i := range n {
			if errs[i] != nil {
				t.Fatalf("%s: Conn(): %v", label, errs[i])
			}
			conn := conns[i]
			defer conn.Close()
			assertPragma(t, conn, ctx, label, "cache_size", -4096)
			assertPragma(t, conn, ctx, label, "busy_timeout", 15000)
			assertPragma(t, conn, ctx, label, "synchronous", 1)
			assertPragma(t, conn, ctx, label, "journal_size_limit", 67108864)
		}
	}

	checkPool(readDB, "read")
	checkPool(writeDB, "write")
}
