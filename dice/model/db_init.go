//go:build !cgo
// +build !cgo

package model

import (
	_ "github.com/glebarez/go-sqlite"
	"github.com/jmoiron/sqlx"
)

func _SQLiteDBInit(path string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		panic(err)
	}

	//_, err = db.Exec("vacuum")
	//if err != nil {
	//	panic(err)
	//}

	// enable WAL mode
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		panic(err)
	}

	return db, err
}
