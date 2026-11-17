package pgsql

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sealdice-core/logger"
)

func PostgresDBInit(dsn string) (*gorm.DB, error) {
	// 使用 GORM 连接 PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.DefaultSealLogger,
	})
	if err != nil {
		return nil, err
	}

	// 返回数据库连接
	return db, nil
}
