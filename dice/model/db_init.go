//go:build !cgo
// +build !cgo

package model

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// _SQLiteDBInit 初始化 SQLite 数据库连接
// 警告:这个替代品的封装应该有建表问题，修正之前请谨慎使用它
// 非CGO的另一个替代品使用了WASM方案：https://github.com/ncruces/go-sqlite3/tree/main/gormlite
func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		// 注意，这里虽然是Info,但实际上打印就变成了Debug.
		Logger: logger.Default.LogMode(logger.Info),
	})
	// https://github.com/glebarez/sqlite/issues/52 尚未遇见问题，可以先考虑不使用
	// sqlDB, _ := db.DB()
	// sqlDB.SetMaxOpenConns(1)
	if err != nil {
		return nil, err
	}
	// Enable Cache Mode
	db, err = GetBuntCacheDB(db)
	if err != nil {
		return nil, err
	}
	// enable WAL mode
	if useWAL {
		err = db.Exec("PRAGMA journal_mode=WAL").Error
		if err != nil {
			return nil, err
		}
	}
	return db, err
}

// _MySQLDBInit 初始化 MySQL 数据库连接 暂时不用它
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
//
//	// 返回数据库连接
//	return db, nil
// }
