package v2_test

import (
	"testing"

	v161 "sealdice-core/migrate/v2/v161"
	"sealdice-core/migrate/v2/v2test"
	"sealdice-core/utils/constant"
)

func TestV161LogUpdatedAtRepair_RebuildsUpdatedAtFromLatestLogItemTime(t *testing.T) {
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
		time INTEGER,
		removed INTEGER
	)`)
	v2test.MustExec(t, logDB, `INSERT INTO logs (id, name, created_at, updated_at, size) VALUES
		(1, 'log-one', 100, 9999, 0),
		(2, 'log-two', 200, 9999, 0),
		(3, 'log-three', 300, 9999, 0)`)
	v2test.MustExec(t, logDB, `INSERT INTO log_items (log_id, group_id, time, removed) VALUES
		(1, 'QQ-Group:1', 101, NULL),
		(1, 'QQ-Group:1', 104, NULL),
		(1, 'QQ-Group:1', 103, 1),
		(2, 'QQ-Group:1', 201, NULL)`)

	if err := v161.V161LogUpdatedAtRepairMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("V161LogUpdatedAtRepairMigrate 失败: %v", err)
	}

	type row struct {
		ID        int   `gorm:"column:id"`
		UpdatedAt int64 `gorm:"column:updated_at"`
	}
	var rows []row
	if err := logDB.Raw("SELECT id, updated_at FROM logs ORDER BY id").Scan(&rows).Error; err != nil {
		t.Fatalf("查询 logs.updated_at 失败: %v", err)
	}

	got := map[int]int64{}
	for _, item := range rows {
		got[item.ID] = item.UpdatedAt
	}
	if got[1] != 104 {
		t.Fatalf("log 1 updated_at = %d, want 104", got[1])
	}
	if got[2] != 201 {
		t.Fatalf("log 2 updated_at = %d, want 201", got[2])
	}
	if got[3] != 300 {
		t.Fatalf("log 3 updated_at = %d, want 300", got[3])
	}
}

func TestV161LogUpdatedAtRepair_NoLogsTableIsNoOp(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	if err := v161.V161LogUpdatedAtRepairMigrate(op, v2test.SilentLogf); err != nil {
		t.Fatalf("无 logs 表时不应报错: %v", err)
	}
}

func TestV161LogUpdatedAtRepair_MissingLogItemsTableErrors(t *testing.T) {
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
	v2test.MustExec(t, logDB, `INSERT INTO logs (id, name, created_at, updated_at, size) VALUES (1, 'a', 100, 9999, 0)`)

	if err := v161.V161LogUpdatedAtRepairMigrate(op, v2test.SilentLogf); err == nil {
		t.Fatal("logs 存在而 log_items 缺失时应返回错误")
	}
}
