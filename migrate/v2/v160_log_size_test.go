package v2

import (
	"testing"

	"gorm.io/gorm"

	"sealdice-core/migrate/v2/v160"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

func TestV160LogSizeRepair_AddsMissingColumnAndRecounts(t *testing.T) {
	op, _ := newTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	// 建一个“没有 size 列”的 logs 表，模拟 V150 历史失误遗留
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
		group_id TEXT,
		removed INTEGER
	)`)
	mustExec(t, logDB, `INSERT INTO logs (id, name) VALUES (1, 'a'), (2, 'b')`)
	// log 1: 3 条，其中 1 条 removed=1 → 可见 2；log 2: 1 条可见
	mustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (1, NULL), (1, 1), (2, NULL)`)

	if logDB.Migrator().HasColumn(&model.LogInfo{}, "size") {
		t.Fatal("前置条件不满足：迁移前 size 列不应存在")
	}

	if err := v160.V160LogSizeRepairMigrate(op, silentLogf); err != nil {
		t.Fatalf("V160LogSizeRepairMigrate 失败: %v", err)
	}

	if !logDB.Migrator().HasColumn(&model.LogInfo{}, "size") {
		t.Fatal("迁移后 size 列应存在")
	}
	sizes := scanLogSizes(t, logDB)
	if sizes[1] != 2 {
		t.Fatalf("log 1 的 size 期望 2，实际 %d", sizes[1])
	}
	if sizes[2] != 1 {
		t.Fatalf("log 2 的 size 期望 1，实际 %d", sizes[2])
	}
}

func TestV160LogSizeRepair_ExistingColumnGetsRecounted(t *testing.T) {
	op, _ := newTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)

	// logs 表已带 size 列，但值全是 0（脏数据），应当被重算
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
		removed INTEGER
	)`)
	mustExec(t, logDB, `INSERT INTO logs (id, name, size) VALUES (1, 'a', 0), (2, 'b', 0)`)
	mustExec(t, logDB, `INSERT INTO log_items (log_id, removed) VALUES (1, NULL), (1, NULL), (2, NULL), (2, NULL), (2, 1)`)

	if err := v160.V160LogSizeRepairMigrate(op, silentLogf); err != nil {
		t.Fatalf("V160LogSizeRepairMigrate 失败: %v", err)
	}

	sizes := scanLogSizes(t, logDB)
	if sizes[1] != 2 || sizes[2] != 2 {
		t.Fatalf("size 重算错误: %v (期望 log1=2, log2=2)", sizes)
	}
}

func TestV160LogSizeRepair_NoLogsTableIsNoOp(t *testing.T) {
	op, _ := newTestSQLiteEngine(t)
	// 全新空库，没有 logs 表
	if err := v160.V160LogSizeRepairMigrate(op, silentLogf); err != nil {
		t.Fatalf("无 logs 表时不应报错: %v", err)
	}
}

// scanLogSizes 扫描 logs 表返回 id -> size 的映射。
func scanLogSizes(t *testing.T, db *gorm.DB) map[int]int {
	t.Helper()
	type row struct {
		ID   int  `gorm:"column:id"`
		Size *int `gorm:"column:size"`
	}
	var rows []row
	if err := db.Raw("SELECT id, size FROM logs ORDER BY id").Scan(&rows).Error; err != nil {
		t.Fatalf("查询 logs.size 失败: %v", err)
	}
	m := make(map[int]int, len(rows))
	for _, r := range rows {
		v := 0
		if r.Size != nil {
			v = *r.Size
		}
		m[r.ID] = v
	}
	return m
}
