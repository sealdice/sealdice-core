//go:build !cgo

package migrate

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/embed"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sealdice-core/utils"
)

func openDB(path string) (*sqlx.DB, error) {
	path = fmt.Sprintf("file:%v?_txlock=immediate&_busy_timeout=15000", path)
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
