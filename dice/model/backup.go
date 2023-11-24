package model

import (
	"github.com/jmoiron/sqlx"
)

func Vacuum(db *sqlx.DB, path string) error {
	_, err := db.Exec("vacuum into $1", path)
	return err
}

func FlushWAL(db *sqlx.DB) error {
	_, _ = db.Exec("PRAGMA shrink_memory")
	_, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	return err
}
