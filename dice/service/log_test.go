package service

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
)

type logInfoTestOperator struct {
	db     *gorm.DB
	dbType string
}

func (o *logInfoTestOperator) Init(_ context.Context) error           { return nil }
func (o *logInfoTestOperator) Type() string                           { return o.dbType }
func (o *logInfoTestOperator) DBCheck()                               {}
func (o *logInfoTestOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return o.db }
func (o *logInfoTestOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return o.db }
func (o *logInfoTestOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return o.db }
func (o *logInfoTestOperator) Close()                                 {}

func TestLogGetInfoUsesSQLiteSequence(t *testing.T) {
	db := newLogInfoTestDB(t)
	seedLogInfoTestDB(t, db)

	if err := db.Exec("UPDATE sqlite_sequence SET seq = ? WHERE name = ?", 99, "logs").Error; err != nil {
		t.Fatalf("update logs sqlite sequence: %v", err)
	}
	if err := db.Exec("UPDATE sqlite_sequence SET seq = ? WHERE name = ?", 123, "log_items").Error; err != nil {
		t.Fatalf("update log_items sqlite sequence: %v", err)
	}

	got, err := LogGetInfo(&logInfoTestOperator{db: db, dbType: constant.SQLITE})
	if err != nil {
		t.Fatalf("LogGetInfo() error = %v", err)
	}

	want := []int{99, 123, 99, 123}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LogGetInfo() = %v, want %v", got, want)
	}
}

func TestLogGetInfoFallsBackToMaxIDWhenSQLiteSequenceMissing(t *testing.T) {
	db := newLogInfoTestDB(t)
	seedLogInfoTestDB(t, db)

	if err := db.Exec("DELETE FROM sqlite_sequence WHERE name IN (?, ?)", "logs", "log_items").Error; err != nil {
		t.Fatalf("delete sqlite sequence: %v", err)
	}

	got, err := LogGetInfo(&logInfoTestOperator{db: db, dbType: constant.SQLITE})
	if err != nil {
		t.Fatalf("LogGetInfo() error = %v", err)
	}

	want := []int{3, 4, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LogGetInfo() = %v, want %v", got, want)
	}
}

func newLogInfoTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "log-info.db"))
	db, err := openLogInfoTestDB(dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.LogInfo{}, &model.LogOneItem{}); err != nil {
		t.Fatalf("migrate log tables: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

func seedLogInfoTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	for i := 1; i <= 3; i++ {
		info := &model.LogInfo{
			Name:      fmt.Sprintf("log-%d", i),
			GroupID:   "group",
			CreatedAt: int64(i),
			UpdatedAt: int64(i),
		}
		if err := db.Create(info).Error; err != nil {
			t.Fatalf("create log info %d: %v", i, err)
		}
	}
	for i := 1; i <= 4; i++ {
		item := &model.LogOneItem{
			LogID:    1,
			GroupID:  "group",
			Nickname: fmt.Sprintf("nick-%d", i),
			Time:     int64(i),
			Message:  fmt.Sprintf("message-%d", i),
		}
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("create log item %d: %v", i, err)
		}
	}
	if err := db.Where("id = ?", 2).Delete(&model.LogInfo{}).Error; err != nil {
		t.Fatalf("delete log info gap: %v", err)
	}
	if err := db.Where("id = ?", 2).Delete(&model.LogOneItem{}).Error; err != nil {
		t.Fatalf("delete log item gap: %v", err)
	}
}
