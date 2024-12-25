package model

import (
	"os"

	"gorm.io/gorm"

	log "sealdice-core/utils/kratos"
)

func DatabaseInit() (dataDB *gorm.DB, logsDB *gorm.DB, err error) {
	dbType := os.Getenv("DB_TYPE")
	var engine DatabaseOperator
	switch dbType {
	case SQLITE:
		engine = &SQLiteEngine{}
	case MYSQL:
		// TODO
		log.Warn("当前配置默认使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	case POSTGRESQL:
		engine = &PGSQLEngine{}
	default:

		log.Warn("当前配置默认使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	}
	err = engine.Init()
	if err != nil {
		return nil, nil, err
	}
	dataDB, err = engine.DataDBInit()
	if err != nil {
		return nil, nil, err
	}
	logsDB, err = engine.LogDBInit()
	if err != nil {
		return nil, nil, err
	}
	return dataDB, logsDB, nil
}

func DBCheck() {
	dbType := os.Getenv("DB_TYPE")
	var engine DatabaseOperator
	switch dbType {
	case SQLITE:
		log.Info("当前选择使用:SQLITE数据库")
		engine = &SQLiteEngine{}
	case MYSQL:
		// TODO
		log.Warn("当前配置：MYSQL未能实现，默认使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	case POSTGRESQL:
		log.Info("当前选择使用:PGSQL 数据库")
		engine = &PGSQLEngine{}
	default:
		log.Warn("当前配置默认使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	}
	err := engine.Init()
	if err != nil {
		return
	}
	engine.DBCheck()
}

func CensorDBInit() (censorDB *gorm.DB, err error) {
	dbType := os.Getenv("DB_TYPE")
	var engine DatabaseOperator
	switch dbType {
	case SQLITE:
		engine = &SQLiteEngine{}
	case MYSQL:
		// TODO
		log.Warn("当前配置默认使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	case POSTGRESQL:
		engine = &PGSQLEngine{}
	default:
		log.Warn("当前配置默认使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	}
	err = engine.Init()
	if err != nil {
		return nil, err
	}
	init, err := engine.CensorDBInit()
	if err != nil {
		return nil, err
	}
	return init, nil
}
