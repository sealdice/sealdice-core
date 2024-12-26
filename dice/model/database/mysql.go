package database

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/dice/model/database/cache"
)

func MySQLDBInit(dsn string) (*gorm.DB, error) {
	// 构建 MySQL DSN (Data Source Name)
	// 使用 GORM 连接 MySQL
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second, // 慢 SQL 阈值
			LogLevel:      logger.Info, // 记录所有SQL操作
			Colorful:      true,        // 是否启用彩色打印
		},
	)})
	if err != nil {
		return nil, err
	}
	// 存疑，MYSQL是否需要使用缓存
	cacheDB, err := cache.GetBuntCacheDB(db)
	if err != nil {
		return nil, err
	}
	// 返回数据库连接
	return cacheDB, nil
}
