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
	DB  *gorm.DB
}

func (s *PGSQLEngine) Init() error {
	s.DSN = os.Getenv("DB_DSN")
	if s.DSN == "" {
		return errors.New("DB_DSN is missing")
	}
	var err error
	s.DB, err = database.PostgresDBInit(s.DSN)
	if err != nil {
		return err
	}
	return nil
}

// DBCheck DB检查
func (s *PGSQLEngine) DBCheck() {
	fmt.Fprintln(os.Stdout, "PostGRESQL 海豹不提供检查，请自行检查数据库！")
}

// DataDBInit 初始化
func (s *PGSQLEngine) DataDBInit() (*gorm.DB, error) {
	// data建表
	err := s.DB.AutoMigrate(
		&GroupPlayerInfoBase{},
		&GroupInfo{},
		&BanInfo{},
		&EndpointInfo{},
		&AttributesItemModel{},
	)
	if err != nil {
		return nil, err
	}
	return s.DB, nil
}

func (s *PGSQLEngine) LogDBInit() (*gorm.DB, error) {
	// logs建表
	if err := s.DB.AutoMigrate(&LogInfo{}, &LogOneItem{}); err != nil {
		return nil, err
	}
	return s.DB, nil
}

func (s *PGSQLEngine) CensorDBInit() (*gorm.DB, error) {
	// 创建基本的表结构，并通过标签定义索引
	if err := s.DB.AutoMigrate(&CensorLog{}); err != nil {
		return nil, err
	}
	return s.DB, nil
}
