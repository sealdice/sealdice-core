//go:build !cgo
// +build !cgo

package model

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// _SQLiteDBInit 初始化 SQLite 数据库连接
func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// err = db.Exec("vacuum").Error
	// if err != nil {
	// 	panic(err)
	// }
	// github.com/glebarez/sqlite

	// enable WAL mode
	if useWAL {
		err = db.Exec("PRAGMA journal_mode=WAL").Error
		if err != nil {
			panic(err)
		}
	}
	fmt.Println(db)
	return db, err
}

// _MySQLDBInit 初始化 MySQL 数据库连接 暂时不用它
//func _MySQLDBInit(user, password, host, dbName string) (*gorm.DB, error) {
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
//}
