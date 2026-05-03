package mysql

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/utils/cache"
)

func MySQLDBInit(dsn string) (*gorm.DB, *cache.Plugin, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, err
	}

	cacheDB, plugin, err := cache.AttachOtterCache(db)
	if err != nil {
		return nil, nil, err
	}
	return cacheDB, plugin, nil
}
