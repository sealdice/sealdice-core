package model

import (
	"fmt"

	"gorm.io/gorm"
)

// Vacuum 执行数据库的 vacuum 操作
func Vacuum(db *gorm.DB, path string) error {
	// 检查数据库驱动是否为 SQLite
	if db.Dialector.Name() != "sqlite" {
		fmt.Println("非SQLITE，跳过运行")
		return nil
	}

	// 使用 GORM 执行 vacuum 操作，并将数据库保存到指定路径
	err := db.Exec("VACUUM INTO ?", path).Error
	return err // 返回错误
}

// FlushWAL 执行 WAL 日志的检查点和内存收缩
func FlushWAL(db *gorm.DB) error {
	// 检查数据库驱动是否为 SQLite
	if db.Dialector.Name() != "sqlite" {
		fmt.Println("非SQLITE，跳过运行")
		return nil
	}

	// 执行 WAL 检查点操作
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE);").Error; err != nil {
		return err // 返回错误
	}
	// 执行内存收缩操作
	err := db.Exec("PRAGMA shrink_memory;").Error
	return err // 返回错误
}
