package mysql

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils/cache"
	"sealdice-core/utils/constant"
)

type MYSQLEngine struct {
	DSN         string
	DB          *gorm.DB
	dataDB      *gorm.DB
	logsDB      *gorm.DB
	censorDB    *gorm.DB
	ctx         context.Context
	cachePlugin *cache.Plugin
}

func (s *MYSQLEngine) Close() {
	log := zap.S().Named(logger.LogKeyDatabase)

	if s.cachePlugin != nil {
		s.cachePlugin.Close()
		s.cachePlugin = nil
	}

	db, err := s.DB.DB()
	if err != nil {
		log.Errorf("failed to close db: %v", err)
		return
	}
	if err = db.Close(); err != nil {
		log.Errorf("failed to close db: %v", err)
	}
}

func (s *MYSQLEngine) GetDataDB(_ constant.DBMode) *gorm.DB { return s.dataDB }
func (s *MYSQLEngine) GetLogDB(_ constant.DBMode) *gorm.DB  { return s.logsDB }
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
	s.DB, s.cachePlugin, err = MySQLDBInit(s.DSN)
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

func (s *MYSQLEngine) DBCheck() {
	fmt.Fprintln(os.Stdout, "MySQL integrity checks are not implemented; check the database manually.")
}

func (s *MYSQLEngine) dataDBInit() (*gorm.DB, error) {
	dataContext := context.WithValue(s.ctx, cache.CacheKey, cache.DataDBCacheKey)
	return s.DB.WithContext(dataContext), nil
}

func (s *MYSQLEngine) logDBInit() (*gorm.DB, error) {
	logsContext := context.WithValue(s.ctx, cache.CacheKey, cache.LogsDBCacheKey)
	return s.DB.WithContext(logsContext), nil
}

func (s *MYSQLEngine) censorDBInit() (*gorm.DB, error) {
	censorContext := context.WithValue(s.ctx, cache.CacheKey, cache.CensorsDBCacheKey)
	censorDB := s.DB.WithContext(censorContext)
	if err := censorDB.AutoMigrate(&model.CensorLog{}); err != nil {
		return nil, err
	}
	return censorDB, nil
}

func (s *MYSQLEngine) Type() string { return "mysql" }
