package database

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/dice/model/database/cache"
)

func PostgresDBInit(dsn string) (*gorm.DB, error) {
	// 构建 PostgreSQL DSN (Data Source Name)

	// 使用 GORM 连接 PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold: time.Second, // 慢 SQL 阈值
				LogLevel:      logger.Info, // 记录所有SQL操作
				Colorful:      true,        // 是否启用彩色打印
			},
		),
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
