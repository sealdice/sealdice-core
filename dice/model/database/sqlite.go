//go:build !cgo
// +build !cgo

package database

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/dice/model/database/cache"
)

func SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	// https://github.com/glebarez/sqlite/issues/52 尚未遇见问题，可以先考虑不使用
	// sqlDB, _ := db.DB()
	// sqlDB.SetMaxOpenConns(1)
	if err != nil {
		return nil, err
	}
	// Enable Cache Mode
	db, err = cache.GetBuntCacheDB(db)
	if err != nil {
		return nil, err
	}
	// enable WAL mode
	if useWAL {
		err = db.Exec("PRAGMA journal_mode=WAL").Error
		if err != nil {
			return nil, err
		}
	}
	return db, err
}
