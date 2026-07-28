package v2_test

import (
	"testing"

	"sealdice-core/migrate/v2/v2test"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

func TestFullUpgradeFlow(t *testing.T) {
	op, _ := v2test.NewTestSQLiteEngine(t)
	logDB := op.GetLogDB(constant.WRITE)
	dataDB := op.GetDataDB(constant.WRITE)

	v2test.ExecSQLFile(t, logDB, "../testdata/full_setup_logs.sql")
	v2test.ExecSQLFile(t, dataDB, "../testdata/full_setup_data.sql")

	mgr := v2test.NewTestManager(t, op)
	if err := mgr.ApplyAll(); err != nil {
		t.Fatalf("首次 ApplyAll 失败: %v", err)
	}

	// === logs 侧 ===
	if !logDB.Migrator().HasColumn(&model.LogInfo{}, "size") {
		t.Fatal("升级后 logs 表应包含 size 列")
	}
	sizes := v2test.ScanLogSizes(t, logDB)
	if sizes[1] != 3 {
		t.Fatalf("log 1 的 size 期望 3，实际 %d", sizes[1])
	}
	if sizes[2] != 1 {
		t.Fatalf("log 2 的 size 期望 1，实际 %d", sizes[2])
	}

	var logZero, itemLogZero int64
	logDB.Raw("SELECT COUNT(1) FROM logs WHERE id = 0").Scan(&logZero)
	logDB.Raw("SELECT COUNT(1) FROM log_items WHERE log_id = 0").Scan(&itemLogZero)
	if logZero != 0 || itemLogZero != 0 {
		t.Fatalf("log_id=0 清理不彻底: logs.id=0=%d, log_items.log_id=0=%d", logZero, itemLogZero)
	}

	var idxCount int64
	logDB.Raw("SELECT COUNT(1) FROM sqlite_master WHERE type='index' AND name='idx_log_delete_by_id'").Scan(&idxCount)
	if idxCount != 1 {
		t.Fatalf("idx_log_delete_by_id 索引应存在，实际数量 %d", idxCount)
	}

	// === data 侧 ===
	var oldTables int64
	dataDB.Raw("SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name IN ('attrs_user','attrs_group','attrs_group_user')").Scan(&oldTables)
	if oldTables != 0 {
		t.Fatalf("旧 attrs_* 表应已被删除，剩余 %d 张", oldTables)
	}

	type attrRow struct {
		AttrsType string `gorm:"column:attrs_type"`
		ID        string `gorm:"column:id"`
	}
	var attrs []attrRow
	dataDB.Raw("SELECT id, attrs_type FROM attrs").Scan(&attrs)
	if len(attrs) != 3 {
		t.Fatalf("attrs 表应有 3 条记录，实际 %d (%+v)", len(attrs), attrs)
	}
	typeCount := map[string]int{}
	for _, a := range attrs {
		typeCount[a.AttrsType]++
	}
	if typeCount["user"] != 1 || typeCount["group"] != 1 || typeCount["group_user"] != 1 {
		t.Fatalf("attrs 类型分布不符预期: %+v", typeCount)
	}

	var banCount, banBad int64
	dataDB.Raw("SELECT COUNT(1) FROM ban_info").Scan(&banCount)
	dataDB.Raw("SELECT COUNT(1) FROM ban_info WHERE id = 'ban-bad'").Scan(&banBad)
	if banCount != 1 || banBad != 0 {
		t.Fatalf("ban_info 清理不符预期: total=%d, ban-bad=%d (期望 total=1, ban-bad=0)", banCount, banBad)
	}

	// === 幂等性 ===
	if err := mgr.ApplyAll(); err != nil {
		t.Fatalf("第二次 ApplyAll（幂等）失败: %v", err)
	}
	sizes2 := v2test.ScanLogSizes(t, logDB)
	if sizes2[1] != 3 || sizes2[2] != 1 {
		t.Fatalf("幂等重跑后 size 不应变化: %+v", sizes2)
	}
	var attrsAfter int64
	dataDB.Raw("SELECT COUNT(1) FROM attrs").Scan(&attrsAfter)
	if attrsAfter != 3 {
		t.Fatalf("幂等重跑后 attrs 条数不应变化: %d", attrsAfter)
	}
}
