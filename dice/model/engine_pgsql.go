package model

import (
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"

	"sealdice-core/dice/model/database"
)

type PGSQLEngine struct {
	DSN string
}

func (s *PGSQLEngine) Init() error {
	s.DSN = os.Getenv("DB_DSN")
	if s.DSN == "" {
		return errors.New("DB_DSN is missing")
	}
	return nil
}

// DB检查
func (s *PGSQLEngine) DBCheck() {
	fmt.Fprintln(os.Stdout, "PostGRESQL 海豹不提供检查，请自行检查数据库！")
}

// 初始化
func (s *PGSQLEngine) DataDBInit() (*gorm.DB, error) {
	dataDB, err := database.PostgresDBInit(s.DSN)
	if err != nil {
		return nil, err
	}
	// data建表
	err = dataDB.AutoMigrate(
		&GroupPlayerInfoBase{},
		&GroupInfo{},
		&BanInfo{},
		&EndpointInfo{},
		&AttributesItemModel{},
	)
	if err != nil {
		return nil, err
	}
	return dataDB, nil
}

func (s *PGSQLEngine) LogDBInit() (*gorm.DB, error) {
	logsDB, err := database.PostgresDBInit(s.DSN)
	if err != nil {
		return nil, err
	}
	// logs建表
	if err = logsDB.AutoMigrate(&LogInfo{}); err != nil {
		return nil, err
	}
	return logsDB, nil
}

func (s *PGSQLEngine) CensorDBInit() (*gorm.DB, error) {
	censorDB, err := database.PostgresDBInit(s.DSN)
	if err != nil {
		return nil, err
	}
	// 创建基本的表结构，并通过标签定义索引
	if err = censorDB.AutoMigrate(&CensorLog{}); err != nil {
		return nil, err
	}
	return censorDB, nil
}
