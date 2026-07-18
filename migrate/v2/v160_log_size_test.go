package v2_test

import (
	"testing"

	v160 "sealdice-core/migrate/v2/v160"
	"sealdice-core/migrate/v2/v2test"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

func TestV160LogSizeRepair_AddsMissingColumnAndRecounts(t *testing.T) {
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
		group_id TEXT,
		removed INTEGER
	)`)
	v2test.MustExec(t, logDB, `INSERT INTO logs (id, name) VALUES (1, 'a'), (2, 'b')`)
	v2test.MustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (1, NULL), (1, 1), (2, NULL)`)

	if logDB.Migrator().HasColumn(&model.LogInfo{}, "size") {
		t.Fatal("前置条件不满足：迁移前 size 列不应存在")
	}

	if err := v160.V160LogSizeRepairMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("V160LogSizeRepairMigrate 失败: %v", err)
	}

	if !logDB.Migrator().HasColumn(&model.LogInfo{}, "size") {
		t.Fatal("迁移后 size 列应存在")
	}
	sizes := v2test.ScanLogSizes(t, logDB)
	if sizes[1] != 2 {
		t.Fatalf("log 1 的 size 期望 2，实际 %d", sizes[1])
	}
	if sizes[2] != 1 {
		t.Fatalf("log 2 的 size 期望 1，实际 %d", sizes[2])
	}
}

func TestV160LogSizeRepair_ExistingColumnGetsRecounted(t *testing.T) {
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
		removed INTEGER
	)`)
	v2test.MustExec(t, logDB, `INSERT INTO logs (id, name, size) VALUES (1, 'a', 0), (2, 'b', 0)`)
	v2test.MustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (1, NULL), (2, NULL), (2, NULL), (2, 1)`)

	if err := v160.V160LogSizeRepairMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("V160LogSizeRepairMigrate 失败: %v", err)
	}

	sizes := v2test.ScanLogSizes(t, logDB)
	if sizes[1] != 2 || sizes[2] != 2 {
		t.Fatalf("size 重算错误: %v (期望 log1=2, log2=2)", sizes)
	}
}

func TestV160LogSizeRepair_NoLogsTableIsNoOp(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	if err := v160.V160LogSizeRepairMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("无 logs 表时不应报错: %v", err)
	}
}
