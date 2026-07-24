// upgrade/store/gorm_store.go
// GormStore 把升级记录存在 data.db 的 upgrade_records 表里，不再依赖外部 JSON 文件。
package store

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	upgrade "sealdice-core/utils/upgrader"
)

type GormStore struct {
	db      *gorm.DB
	once    sync.Once
	initErr error
}

// NewGormStore 创建一个以 data.db 为后端的升级记录存储。
// 表的初始化（CREATE TABLE IF NOT EXISTS）延迟到首次使用时执行（sync.Once），
// 此时 data.db 已由 SQLiteEngine 等完成连接初始化，可直接操作。
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

// 初始化 upgrade_records 表。幂等（CREATE TABLE IF NOT EXISTS），三种数据库均支持。
func (gs *GormStore) ensureTable() error {
	gs.once.Do(func() {
		gs.initErr = gs.db.Exec(`
			CREATE TABLE IF NOT EXISTS upgrade_records (
				id        TEXT PRIMARY KEY,
				timestamp TEXT,
				success   INTEGER,
				message   TEXT,
				logs      TEXT
			)
		`).Error
	})
	return gs.initErr
}

// dbRecord 是 upgrade_records 表对应的中间结构体，
// 仅负责序列化/反序列化，不直接暴露给外部。
type dbRecord struct {
	ID        string `gorm:"column:id;primaryKey"`
	Timestamp string `gorm:"column:timestamp"`
	Success   int    `gorm:"column:success"`
	Message   string `gorm:"column:message"`
	Logs      string `gorm:"column:logs"`
}

func (dbRecord) TableName() string { return "upgrade_records" }

func toDBRecord(rec upgrade.UpgradeRecord) dbRecord {
	logsJSON, _ := json.Marshal(rec.Logs)
	return dbRecord{
		ID:        rec.ID,
		Timestamp: rec.Timestamp.Format(time.RFC3339),
		Success:   boolToInt(rec.Success),
		Message:   rec.Message,
		Logs:      string(logsJSON),
	}
}

func fromDBRecord(r dbRecord) upgrade.UpgradeRecord {
	ts, _ := time.Parse(time.RFC3339, r.Timestamp)
	var logs []string
	_ = json.Unmarshal([]byte(r.Logs), &logs)
	return upgrade.UpgradeRecord{
		ID:        r.ID,
		Timestamp: ts,
		Success:   r.Success != 0,
		Message:   r.Message,
		Logs:      logs,
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// IsApplied 判断指定 ID 的迁移是否已记录（不论成功与否，与 JSONStore 语义一致）。
func (gs *GormStore) IsApplied(id string) (bool, error) {
	if err := gs.ensureTable(); err != nil {
		return false, err
	}
	var count int64
	if err := gs.db.Table("upgrade_records").Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("查询升级记录 %s 失败: %w", id, err)
	}
	return count > 0, nil
}

// SaveRecord 写入（或覆盖）一条升级记录。
func (gs *GormStore) SaveRecord(rec upgrade.UpgradeRecord) error {
	if err := gs.ensureTable(); err != nil {
		return err
	}
	dr := toDBRecord(rec)
	if err := gs.db.Table("upgrade_records").Save(&dr).Error; err != nil {
		return fmt.Errorf("保存升级记录 %s 失败: %w", rec.ID, err)
	}
	return nil
}

// LoadRecords 返回所有已保存的升级记录，按时间戳排序。
func (gs *GormStore) LoadRecords() ([]upgrade.UpgradeRecord, error) {
	if err := gs.ensureTable(); err != nil {
		return nil, err
	}
	var rows []dbRecord
	if err := gs.db.Table("upgrade_records").Order("timestamp").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取升级记录失败: %w", err)
	}
	recs := make([]upgrade.UpgradeRecord, 0, len(rows))
	for _, r := range rows {
		recs = append(recs, fromDBRecord(r))
	}
	return recs, nil
}
