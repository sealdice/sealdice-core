package v2

import (
	"testing"

	"sealdice-core/migrate/v2/v160"
	"sealdice-core/utils/constant"
)

func TestV160LogIDZeroClean_DeletesAndRecounts(t *testing.T) {
	op, _ := newTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	mustExec(t, logDB, `CREATE TABLE logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		group_id TEXT,
		created_at INTEGER,
		updated_at INTEGER,
		size INTEGER
	)`)
	mustExec(t, logDB, `CREATE TABLE log_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		log_id INTEGER,
		group_id TEXT,
		removed INTEGER
	)`)
	mustExec(t, logDB, `INSERT INTO logs (id, name, size) VALUES (1, 'a', 99), (0, NULL, 0)`)
	mustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (1, NULL), (0, NULL), (0, NULL)`)

	if err := v160.V160LogIDZeroCleanMigrate(op, silentLogf); err != nil {
		t.Fatalf("V160LogIDZeroCleanMigrate 失败: %v", err)
	}

	// id=0 的日志与 log_id=0 的条目应被删除
	var logZero, itemLogZero int64
	logDB.Raw("SELECT COUNT(1) FROM logs WHERE id = 0").Scan(&logZero)
	logDB.Raw("SELECT COUNT(1) FROM log_items WHERE log_id = 0").Scan(&itemLogZero)
	if logZero != 0 || itemLogZero != 0 {
		t.Fatalf("log_id=0 清理不彻底: logs.id=0=%d, log_items.log_id=0=%d", logZero, itemLogZero)
	}

	// log 1 的 size 应被重算为 2（原本被设为脏值 99）
	sizes := scanLogSizes(t, logDB)
	if sizes[1] != 2 {
		t.Fatalf("清理后 log 1 的 size 期望 2，实际 %d", sizes[1])
	}
}

// 验证：当 size 列不存在（V150 失误遗留）时，008 迁移不会因重算失败而报错。
// 此时清理仍会完成，size 列由 010 迁移负责补建与重算。
func TestV160LogIDZeroClean_TolerantOfMissingSizeColumn(t *testing.T) {
	op, _ := newTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	// 注意：logs 表没有 size 列
	mustExec(t, logDB, `CREATE TABLE logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		group_id TEXT,
		created_at INTEGER,
		updated_at INTEGER
	)`)
	mustExec(t, logDB, `CREATE TABLE log_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		log_id INTEGER,
		removed INTEGER
	)`)
	mustExec(t, logDB, `INSERT INTO logs (id, name) VALUES (1, 'a'), (0, NULL)`)
	mustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (0, NULL)`)

	// 关键断言：不应报错
	if err := v160.V160LogIDZeroCleanMigrate(op, silentLogf); err != nil {
		t.Fatalf("size 列缺失时，008 迁移不应报错: %v", err)
	}

	// 清理仍应生效
	var logZero, itemLogZero int64
	logDB.Raw("SELECT COUNT(1) FROM logs WHERE id = 0").Scan(&logZero)
	logDB.Raw("SELECT COUNT(1) FROM log_items WHERE log_id = 0").Scan(&itemLogZero)
	if logZero != 0 || itemLogZero != 0 {
		t.Fatalf("size 列缺失时，清理仍应完成: logs.id=0=%d, log_items.log_id=0=%d", logZero, itemLogZero)
	}
}

func TestV160LogIDZeroClean_NothingToDoIsNoOp(t *testing.T) {
	op, _ := newTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	mustExec(t, logDB, `CREATE TABLE logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, group_id TEXT,
		created_at INTEGER, updated_at INTEGER, size INTEGER
	)`)
	mustExec(t, logDB, `CREATE TABLE log_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT, log_id INTEGER, removed INTEGER
	)`)
	mustExec(t, logDB, `INSERT INTO logs (id, name, size) VALUES (1, 'a', 1)`)
	mustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL)`)

	if err := v160.V160LogIDZeroCleanMigrate(op, silentLogf); err != nil {
		t.Fatalf("无 log_id=0 数据时不应报错: %v", err)
	}
	sizes := scanLogSizes(t, logDB)
	if sizes[1] != 1 {
		t.Fatalf("log 1 的 size 应保持不变为 1，实际 %d", sizes[1])
	}
}
