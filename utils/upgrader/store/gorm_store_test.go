package store_test

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	upgrade "sealdice-core/utils/upgrader"
	store "sealdice-core/utils/upgrader/store"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	return db
}

func TestGormStore_SaveAndCheck(t *testing.T) {
	db := newTestDB(t)
	gs := store.NewGormStore(db)

	// 初始状态：未应用
	applied, err := gs.IsApplied("001_test")
	if err != nil {
		t.Fatalf("IsApplied 失败: %v", err)
	}
	if applied {
		t.Fatal("001_test 不应已应用")
	}

	// 保存一条记录
	rec := upgrade.UpgradeRecord{
		ID:        "001_test",
		Timestamp: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Success:   true,
		Message:   "成功",
		Logs:      []string{"[INFO] 开始", "[INFO] 完成"},
	}
	if err = gs.SaveRecord(rec); err != nil {
		t.Fatalf("SaveRecord 失败: %v", err)
	}

	// 保存后应已应用
	applied, err = gs.IsApplied("001_test")
	if err != nil {
		t.Fatalf("IsApplied 失败: %v", err)
	}
	if !applied {
		t.Fatal("001_test 应已应用")
	}

	// LoadRecords 一致性
	recs, err := gs.LoadRecords()
	if err != nil {
		t.Fatalf("LoadRecords 失败: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("期望 1 条记录，实际 %d", len(recs))
	}
	r := recs[0]
	if r.ID != "001_test" || !r.Success || r.Message != "成功" {
		t.Fatalf("记录字段不一致: %+v", r)
	}
	if len(r.Logs) != 2 || r.Logs[0] != "[INFO] 开始" || r.Logs[1] != "[INFO] 完成" {
		t.Fatalf("Logs 反序列化失败: %v", r.Logs)
	}
	if r.Timestamp.UTC() != rec.Timestamp {
		t.Fatalf("Timestamp 不一致: %v vs %v", r.Timestamp, rec.Timestamp)
	}
}

func TestGormStore_FailedRecord(t *testing.T) {
	db := newTestDB(t)
	gs := store.NewGormStore(db)

	// 失败记录也应算作"已应用"（与 JSONStore 语义一致）
	rec := upgrade.UpgradeRecord{
		ID:        "002_fail",
		Timestamp: time.Now(),
		Success:   false,
		Message:   "列不存在",
		Logs:      []string{"[ERROR] failed"},
	}
	if err := gs.SaveRecord(rec); err != nil {
		t.Fatalf("SaveRecord 失败: %v", err)
	}
	applied, _ := gs.IsApplied("002_fail")
	if !applied {
		t.Fatal("失败记录也应算作已应用")
	}
}

func TestGormStore_MultipleRecords(t *testing.T) {
	db := newTestDB(t)
	gs := store.NewGormStore(db)

	for _, id := range []string{"001_a", "003_c", "002_b"} {
		_ = gs.SaveRecord(upgrade.UpgradeRecord{
			ID:        id,
			Timestamp: time.Now(),
			Success:   true,
			Message:   "ok",
		})
	}

	// IsApplied 多条
	for _, id := range []string{"001_a", "002_b", "003_c"} {
		ap, _ := gs.IsApplied(id)
		if !ap {
			t.Fatalf("%s 应已应用", id)
		}
	}
	ap, _ := gs.IsApplied("999_no")
	if ap {
		t.Fatal("999_no 不应已应用")
	}

	// LoadRecords 返回全部
	recs, _ := gs.LoadRecords()
	if len(recs) != 3 {
		t.Fatalf("期望 3 条，实际 %d", len(recs))
	}
}

func TestGormStore_IdempotentSave(t *testing.T) {
	db := newTestDB(t)
	gs := store.NewGormStore(db)

	rec := upgrade.UpgradeRecord{
		ID:        "001_same",
		Timestamp: time.Now(),
		Success:   true,
		Message:   "ok",
		Logs:      []string{"第一次"},
	}
	_ = gs.SaveRecord(rec)

	// 再次 SaveRecord 同一 ID（UPDATE 语义，不报错）
	rec.Message = "updated"
	rec.Logs = []string{"第二次"}
	if err := gs.SaveRecord(rec); err != nil {
		t.Fatalf("重复 SaveRecord 应成功（覆盖写）: %v", err)
	}

	recs, _ := gs.LoadRecords()
	if len(recs) != 1 {
		t.Fatalf("应只有 1 条记录，实际 %d", len(recs))
	}
	if recs[0].Message != "updated" || recs[0].Logs[0] != "第二次" {
		t.Fatalf("记录应为最新版本: %+v", recs[0])
	}
}

func TestGormStore_EmptyLogs(t *testing.T) {
	db := newTestDB(t)
	gs := store.NewGormStore(db)

	_ = gs.SaveRecord(upgrade.UpgradeRecord{
		ID:        "001_nologs",
		Timestamp: time.Now(),
		Success:   true,
		Message:   "ok",
		Logs:      nil, // nil Logs
	})
	recs, _ := gs.LoadRecords()
	if len(recs[0].Logs) != 0 {
		t.Fatalf("空 Logs 应为 nil 或空切片: %v", recs[0].Logs)
	}
}
