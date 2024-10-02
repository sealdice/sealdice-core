//go:build !cgo
// +build !cgo

package model

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TODO: 这个得修啊
func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	//db, err := sqlx.Open("sqlite", path)
	//
	//
	//if err != nil {
	//	panic(err)
	//}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// _, err = db.Exec("vacuum")
	// if err != nil {
	// 	panic(err)
	// }
	// github.com/mattn/go-sqlite3

	// enable WAL mode
	if useWAL {
		err = db.Exec("PRAGMA journal_mode=WAL").Error
		if err != nil {
			panic(err)
		}
	}

	return db, err
}
