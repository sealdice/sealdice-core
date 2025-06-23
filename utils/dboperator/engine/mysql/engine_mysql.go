package mysql

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/cache"
	"sealdice-core/utils/constant"
	log "sealdice-core/utils/kratos"
)

type MYSQLEngine struct {
	DSN      string
	DB       *gorm.DB
	dataDB   *gorm.DB
	logsDB   *gorm.DB
	censorDB *gorm.DB
	ctx      context.Context
}

func (s *MYSQLEngine) Close() {
	db, err := s.DB.DB()
	if err != nil {
		log.Errorf("failed to close db: %v", err)
		return
	}
	err = db.Close()
	if err != nil {
		log.Errorf("failed to close db: %v", err)
		return
	}
}

func (s *MYSQLEngine) GetDataDB(_ constant.DBMode) *gorm.DB {
	return s.dataDB
}

func (s *MYSQLEngine) GetLogDB(_ constant.DBMode) *gorm.DB {
	return s.logsDB
}

func (s *MYSQLEngine) GetCensorDB(_ constant.DBMode) *gorm.DB {
	return s.censorDB
}

func (s *MYSQLEngine) Init(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ctx is missing")
	}
	s.ctx = ctx
	s.DSN = os.Getenv("DB_DSN")
	if s.DSN == "" {
		return errors.New("DB_DSN is missing")
	}
	var err error
	s.DB, err = MySQLDBInit(s.DSN)
	if err != nil {
		return err
	}
	s.dataDB, err = s.dataDBInit()
	if err != nil {
		return err
	}
	s.logsDB, err = s.logDBInit()
	if err != nil {
		return err
	}
	s.censorDB, err = s.censorDBInit()
	if err != nil {
		return err
	}
	return nil
}

// DBCheck DB检查
func (s *MYSQLEngine) DBCheck() {
	fmt.Fprintln(os.Stdout, "MYSQL 海豹不提供检查，请自行检查数据库！")
}

// DataDBInit 初始化
func (s *MYSQLEngine) dataDBInit() (*gorm.DB, error) {
	dataContext := context.WithValue(s.ctx, cache.CacheKey, cache.DataDBCacheKey)
	dataDB := s.DB.WithContext(dataContext)
	return dataDB, nil
}

func (s *MYSQLEngine) logDBInit() (*gorm.DB, error) {
	// logs特殊建表
	logsContext := context.WithValue(s.ctx, cache.CacheKey, cache.LogsDBCacheKey)
	logDB := s.DB.WithContext(logsContext)
	return logDB, nil
}

func (s *MYSQLEngine) censorDBInit() (*gorm.DB, error) {
	censorContext := context.WithValue(s.ctx, cache.CacheKey, cache.CensorsDBCacheKey)
	censorDB := s.DB.WithContext(censorContext)
	if err := censorDB.AutoMigrate(&model.CensorLog{}); err != nil {
		return nil, err
	}
	return censorDB, nil
}

func (s *MYSQLEngine) Type() string {
	return "mysql"
}
