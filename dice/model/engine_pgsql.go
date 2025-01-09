package model

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"

	"sealdice-core/dice/model/database"
)

type PGSQLEngine struct {
	DSN string
	DB  *gorm.DB
	ctx context.Context
}

func (s *PGSQLEngine) Init(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ctx is missing")
	}
	s.ctx = ctx
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
	dataContext := context.WithValue(s.ctx, "gorm_cache", "data-db::")
	dataDB := s.DB.WithContext(dataContext)
	err := dataDB.AutoMigrate(
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
	// logs建表
	logsContext := context.WithValue(s.ctx, "gorm_cache", "logs-db::")
	logDB := s.DB.WithContext(logsContext)
	if err := logDB.AutoMigrate(&LogInfo{}, &LogOneItem{}); err != nil {
		return nil, err
	}
	return logDB, nil
}

func (s *PGSQLEngine) CensorDBInit() (*gorm.DB, error) {
	censorContext := context.WithValue(s.ctx, "gorm_cache", "censor-db::")
	censorDB := s.DB.WithContext(censorContext)
	// 创建基本的表结构，并通过标签定义索引
	if err := censorDB.AutoMigrate(&CensorLog{}); err != nil {
		return nil, err
	}
	return censorDB, nil
}
