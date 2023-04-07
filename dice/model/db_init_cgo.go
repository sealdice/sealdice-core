//go:build cgo
// +build cgo

package model

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func _SQLiteDBInit(path string, useWAL bool) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}

	//_, err = db.Exec("vacuum")
	//if err != nil {
	//	panic(err)
	//}

	// enable WAL mode
	if useWAL {
		_, err = db.Exec("PRAGMA journal_mode=WAL")
		if err != nil {
			panic(err)
		}
	}

	return db, err
}
