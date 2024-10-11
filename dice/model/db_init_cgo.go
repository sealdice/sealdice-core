//go:build cgo
// +build cgo

package model

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TODO：重构整个Init方案，采用高级配置读取的方式
func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	db, err := sqlx.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	open, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: db.DB,
	}))
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
