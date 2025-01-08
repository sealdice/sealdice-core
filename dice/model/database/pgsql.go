package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/dice/model/database/cache"
)

func PostgresDBInit(dsn string) (*gorm.DB, error) {
	// 构建 PostgreSQL DSN (Data Source Name)

	// 使用 GORM 连接 PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// GetBuntCacheDB 逻辑保持不变
	cacheDB, err := cache.GetBuntCacheDB(db)
	if err != nil {
		return nil, err
	}

	// 返回数据库连接
	return cacheDB, nil
}
