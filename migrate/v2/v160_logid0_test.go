package v2_test

import (
	"testing"

	v160 "sealdice-core/migrate/v2/v160"
	"sealdice-core/migrate/v2/v2test"
	"sealdice-core/utils/constant"
)

func TestV160LogIDZeroClean_DeletesAndRecounts(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	v2test.MustExec(t, logDB, `CREATE TABLE logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		group_id TEXT,
		created_at INTEGER,
		updated_at INTEGER,
		size INTEGER
	)`)
	v2test.MustExec(t, logDB, `CREATE TABLE log_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		log_id INTEGER,
		group_id TEXT,
		removed INTEGER
	)`)
	v2test.MustExec(t, logDB, `INSERT INTO logs (id, name, size) VALUES (1, 'a', 99), (0, NULL, 0)`)
	v2test.MustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (1, NULL), (0, NULL), (0, NULL)`)

	if err := v160.V160LogIDZeroCleanMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("V160LogIDZeroCleanMigrate 失败: %v", err)
	}

	var logZero, itemLogZero int64
	logDB.Raw("SELECT COUNT(1) FROM logs WHERE id = 0").Scan(&logZero)
	logDB.Raw("SELECT COUNT(1) FROM log_items WHERE log_id = 0").Scan(&itemLogZero)
	if logZero != 0 || itemLogZero != 0 {
		t.Fatalf("log_id=0 清理不彻底: logs.id=0=%d, log_items.log_id=0=%d", logZero, itemLogZero)
	}

	sizes := v2test.ScanLogSizes(t, logDB)
	if sizes[1] != 2 {
		t.Fatalf("清理后 log 1 的 size 期望 2，实际 %d", sizes[1])
	}
}

func TestV160LogIDZeroClean_TolerantOfMissingSizeColumn(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	v2test.MustExec(t, logDB, `CREATE TABLE logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		group_id TEXT,
		created_at INTEGER,
		updated_at INTEGER
	)`)
	v2test.MustExec(t, logDB, `CREATE TABLE log_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		log_id INTEGER,
		removed INTEGER
	)`)
	v2test.MustExec(t, logDB, `INSERT INTO logs (id, name) VALUES (1, 'a'), (0, NULL)`)
	v2test.MustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (0, NULL)`)

	if err := v160.V160LogIDZeroCleanMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("size 列缺失时，008 迁移不应报错: %v", err)
	}

	var logZero, itemLogZero int64
	logDB.Raw("SELECT COUNT(1) FROM logs WHERE id = 0").Scan(&logZero)
	logDB.Raw("SELECT COUNT(1) FROM log_items WHERE log_id = 0").Scan(&itemLogZero)
	if logZero != 0 || itemLogZero != 0 {
		t.Fatalf("size 列缺失时，清理仍应完成: logs.id=0=%d, log_items.log_id=0=%d", logZero, itemLogZero)
	}
}

func TestV160LogIDZeroClean_NothingToDoIsNoOp(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	v2test.MustExec(t, logDB, `CREATE TABLE logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, group_id TEXT,
		created_at INTEGER, updated_at INTEGER, size INTEGER
	)`)
	v2test.MustExec(t, logDB, `CREATE TABLE log_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT, log_id INTEGER, removed INTEGER
	)`)
	v2test.MustExec(t, logDB, `INSERT INTO logs (id, name, size) VALUES (1, 'a', 1)`)
	v2test.MustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL)`)

	if err := v160.V160LogIDZeroCleanMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("无 log_id=0 数据时不应报错: %v", err)
	}
	sizes := v2test.ScanLogSizes(t, logDB)
	if sizes[1] != 1 {
		t.Fatalf("log 1 的 size 应保持不变为 1，实际 %d", sizes[1])
	}
}
