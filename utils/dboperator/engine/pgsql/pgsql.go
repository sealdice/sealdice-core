package pgsql

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/utils/cache"
)

func PostgresDBInit(dsn string) (*gorm.DB, *cache.Plugin, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
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
