//nolint:testpackage // 需要访问未导出的 DSN 与连接池配置
package sqlite

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDSNTxLock(t *testing.T) {
	if got := readDBDSN("/tmp/x.db"); !strings.Contains(got, "_txlock=deferred") {
		t.Errorf("readDBDSN = %q, want it to contain _txlock=deferred", got)
	}
	if got := writeDBDSN("/tmp/x.db"); !strings.Contains(got, "_txlock=immediate") {
		t.Errorf("writeDBDSN = %q, want it to contain _txlock=immediate", got)
	}
}

func TestPoolConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	readDB, writeDB, err := SQLiteDBRWInit(path)
	if err != nil {
		t.Fatalf("SQLiteDBRWInit: %v", err)
	}
	defer func() {
		if sqlDB, dbErr := readDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		if sqlDB, dbErr := writeDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	}()

	readSQL, err := readDB.DB()
	if err != nil {
		t.Fatalf("read DB(): %v", err)
	}
	if got := readSQL.Stats().MaxOpenConnections; got < 2 || got > 4 {
		t.Errorf("read MaxOpenConnections = %d, want within [2,4]", got)
	}

	writeSQL, err := writeDB.DB()
	if err != nil {
		t.Fatalf("write DB(): %v", err)
	}
	if got := writeSQL.Stats().MaxOpenConnections; got != 1 {
		t.Errorf("write MaxOpenConnections = %d, want 1", got)
	}
}
