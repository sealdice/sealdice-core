//go:build !cgo
// +build !cgo

package model

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var MYSQL *gorm.DB

// TODO：重构整个Init方案，采用高级配置读取的方式
func _SQLiteDBInit(path string, useWAL bool) (*gorm.DB, error) {
	// 防止重复初始化连接
	if MYSQL != nil {
		return MYSQL, nil
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
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
	//init, err := _MySQLDBInit("root", "password", "127.0.0.1", "sealdice")
	//if err != nil {
	//	return nil, err
	//}
	//MYSQL = init
	return db, err
}

// _MySQLDBInit 初始化 MySQL 数据库连接
func _MySQLDBInit(user, password, host, dbName string) (*gorm.DB, error) {
	// 构建 MySQL DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, dbName)

	// 使用 GORM 连接 MySQL
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second, // 慢 SQL 阈值
			LogLevel:      logger.Info, // 记录所有SQL操作
			Colorful:      true,        // 是否启用彩色打印
		},
	)})
	if err != nil {
		return nil, err
	}

	// 返回数据库连接
	return db, nil
}
