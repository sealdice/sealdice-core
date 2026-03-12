//go:build !cgo

//nolint:testpackage
package dice

// openTestGormDB opens a plain in-memory (or file-backed) SQLite database
// without any caching plugin.  It is used only in tests to avoid the
// background goroutines started by the otter cache, which would otherwise
// trigger goleak failures.
//
// This file is compiled when cgo is disabled; its cgo counterpart lives in
// execute_new_db_cgo_test.go.

import (
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
)

func openTestGormDB(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open("file:"+path+"?_txlock=immediate&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
}
