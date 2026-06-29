package pgsql

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sealdice-core/logger"
	"sealdice-core/utils/cache"
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

	// GetOtterCacheDB 逻辑保持不变
	cacheDB, err := cache.GetOtterCacheDB(db)
	if err != nil {
		return nil, err
	}

	// 返回数据库连接
	return cacheDB, nil
}
