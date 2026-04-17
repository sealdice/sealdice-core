//go:build cgo

package storylog

import (
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openContentTestGormDB(path string) (*gorm.DB, error) {
	return gorm.Open(gormsqlite.Open(path), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
}
