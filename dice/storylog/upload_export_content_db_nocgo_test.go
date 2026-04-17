//go:build !cgo

package storylog

import (
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openContentTestGormDB(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open("file:"+path+"?_txlock=immediate&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
}
