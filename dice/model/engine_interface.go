package model

import "gorm.io/gorm"

// DatabaseOperator 本来是单独放了个文件夹的，但是由于现在所有的model都和处理逻辑在一起，如果放在单独文件夹必然会循环依赖
// 只能放在外面
type DatabaseOperator interface {
	Init() error
	DBCheck()
	DataDBInit() (*gorm.DB, error)
	LogDBInit() (*gorm.DB, error)
	CensorDBInit() (*gorm.DB, error)
}
