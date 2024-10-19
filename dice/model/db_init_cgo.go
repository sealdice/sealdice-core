//go:build cgo
// +build cgo

package model

import (
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TODO：重构整个Init方案，采用高级配置读取的方式
func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	open, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
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
