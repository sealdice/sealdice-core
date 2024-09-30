//go:build cgo
// +build cgo

package model

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	// TODO: Pinenutn:这里先不完全修改，回头再说，这样应该能直接兼容
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
