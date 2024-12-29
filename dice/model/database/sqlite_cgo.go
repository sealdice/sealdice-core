//go:build cgo
// +build cgo

package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/dice/model/database/cache"
)

func SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	open, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	// Enable Cache Mode
	open, err = cache.GetBuntCacheDB(open)
	if err != nil {
		return nil, err
	}
	// enable WAL mode
	if useWAL {
		err = open.Exec("PRAGMA journal_mode=WAL").Error
		if err != nil {
			panic(err)
		}
	}

	return open, err
}
