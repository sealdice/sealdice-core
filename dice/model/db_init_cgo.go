//go:build cgo
// +build cgo

package model

import (
	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	open, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	// Enable Cache Mode
	open, err = GetBuntCacheDB(open)
	if err != nil {
		return nil, err
	}
	// enable WAL mode
	if useWAL {
		err = open.Exec("PRAGMA journal_mode=WAL").Error
		if err != nil {
			panic(err)
		}
	}

	return open, err
}

// _MySQLDBInit 初始化 MySQL 数据库连接 测试专用
// func _MySQLDBInit(user, password, host, dbName string) (*gorm.DB, error) {
//	// 构建 MySQL DSN (Data Source Name)
//	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, dbName)
//
//	// 使用 GORM 连接 MySQL
//	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.New(
//		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
//		logger.Config{
//			SlowThreshold: time.Second, // 慢 SQL 阈值
//			LogLevel:      logger.Info, // 记录所有SQL操作
//			Colorful:      true,        // 是否启用彩色打印
//		},
//	)})
//	if err != nil {
//		return nil, err
//	}
//	cacheDB, err := GetBuntCacheDB(db)
//	if err != nil {
//		return nil, err
//	}
//	// 返回数据库连接
//	return cacheDB, nil
// }
