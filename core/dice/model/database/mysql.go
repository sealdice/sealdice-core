package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/dice/model/database/cache"
)

func MySQLDBInit(dsn string) (*gorm.DB, error) {
	// 构建 MySQL DSN (Data Source Name)
	// 使用 GORM 连接 MySQL
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info)})
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
