package sqlite

import (
	"database/sql"

	"gorm.io/gorm"
)

// ensureIncrementalAutoVacuum 对全新的空库设置 auto_vacuum=INCREMENTAL。
// 已有表的库不会被静默转换（需要 --vacuum 全量 VACUUM）。
func ensureIncrementalAutoVacuum(pool *sql.DB) error {
	var tables int
	if err := pool.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables); err != nil {
		return err
	}
	if tables > 0 {
		return nil
	}
	_, err := pool.Exec("PRAGMA auto_vacuum=INCREMENTAL")
	return err
}

// ConvertToIncrementalAutoVacuum 将数据库转换为 INCREMENTAL 模式并立即全量 VACUUM。
// 必须在同一条连接上设置 pragma 并执行 VACUUM，否则会转回 NONE。
func ConvertToIncrementalAutoVacuum(db *gorm.DB) error {
	return db.Connection(func(tx *gorm.DB) error {
		if err := tx.Exec("PRAGMA auto_vacuum=INCREMENTAL").Error; err != nil {
			return err
		}
		if err := tx.Exec("VACUUM;").Error; err != nil {
			return err
		}
		return tx.Exec("PRAGMA optimize;").Error
	})
}

// ReclaimIncrementalVacuum 在数据库为 INCREMENTAL 时回收空闲页，
// 返回本次实际回收（freelist 减少）的页数。
func ReclaimIncrementalVacuum(db *gorm.DB) (int, error) {
	var mode int
	if err := db.Raw("PRAGMA auto_vacuum").Row().Scan(&mode); err != nil {
		return 0, err
	}
	if mode != 2 { // 2 = INCREMENTAL
		return 0, nil
	}

	var before int
	if err := db.Raw("PRAGMA freelist_count").Row().Scan(&before); err != nil {
		return 0, err
	}

	// PRAGMA incremental_vacuum 是逐行返回的语句，每一行代表回收一页。
	// 必须遍历完整个结果集，否则只会回收第一页。
	rows, err := db.Raw("PRAGMA incremental_vacuum;").Rows()
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var after int
	if err := db.Raw("PRAGMA freelist_count").Row().Scan(&after); err != nil {
		return 0, err
	}
	if err := db.Exec("PRAGMA optimize;").Error; err != nil {
		return 0, err
	}
	return before - after, nil
}
