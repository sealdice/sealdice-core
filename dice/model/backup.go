package model

import (
	"github.com/jmoiron/sqlx"
)

func Backup(db *sqlx.DB, path string) error {
	_, err := db.Exec("vacuum into $1", path)
	return err
}
