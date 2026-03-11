//go:build cgo

package dice

// openTestGormDB opens a plain file-backed SQLite database without any caching
// plugin.  It is used only in tests to avoid the background goroutines started
// by the otter cache, which would otherwise trigger goleak failures.
//
// This file is compiled when cgo is enabled; its !cgo counterpart lives in
// execute_new_db_nocgo_test.go.

import (
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openTestGormDB(path string) (*gorm.DB, error) {
	return gorm.Open(gormsqlite.Open(path), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
}
