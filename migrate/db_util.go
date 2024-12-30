//go:build !cgo
// +build !cgo

package migrate

import (
	_ "github.com/glebarez/go-sqlite"
	"github.com/glebarez/sqlite"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/utils"
)

func openDB(path string) (*sqlx.DB, error) {
	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic(err)
	}
	db, err := utils.GetSQLXDB(gdb)
	// db, err := sqlx.Open("sqlite", path)
	if err != nil {
		panic(err)
	}

	// enable WAL mode
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		panic(err)
	}

	return db, err
}
