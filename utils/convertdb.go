package utils

import (
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
)

// GetSQLXDB 将 GORM 的 *gorm.DB 转换为 *sqlx.DB，并自动获取驱动名称，用于需要sqlx的场景
func GetSQLXDB(db *gorm.DB) (*sqlx.DB, error) {
	// 获取底层的 *sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 获取 GORM 使用的驱动名称
	driverName := db.Dialector.Name()

	// 使用 sqlx.NewDb 传递现有的 *sql.DB 和驱动名称
	sqlxDB := sqlx.NewDb(sqlDB, driverName)

	return sqlxDB, nil
}
