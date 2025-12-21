package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils/cache"
	"sealdice-core/utils/constant"
)

type SQLiteEngine struct {
	DataDir string
	ctx     context.Context
	// 用于控制readList和writeList的读写锁
	mu        sync.RWMutex
	readList  map[dbName]*gorm.DB
	writeList map[dbName]*gorm.DB
}

func (s *SQLiteEngine) Type() string {
	return "sqlite"
}

// 定义一个基于 string 的新类型 dbName
type dbName string

const (
	LogsDBKey    dbName = "logs"
	DataDBKey    dbName = "data"
	CensorsDBKey dbName = "censor"
)

func (s *SQLiteEngine) Close() {
	log := zap.S().Named(logger.LogKeyDatabase)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 关闭 readList 中的连接
	for name, db := range s.readList {
		sqlDB, err := db.DB()
		if err != nil {
			log.Errorf("failed to get sql.DB for %s: %v", name, err)
			continue
		}
		if err = sqlDB.Close(); err != nil {
			log.Errorf("failed to close db %s: %v", name, err)
		}
	}

	// 关闭 writeList 中的连接
	for name, db := range s.writeList {
		sqlDB, err := db.DB()
		if err != nil {
			log.Errorf("failed to get sql.DB for %s: %v", name, err)
			continue
		}
		if err = sqlDB.Close(); err != nil {
			log.Errorf("failed to close db %s: %v", name, err)
		}
	}
}

func (s *SQLiteEngine) getDBByModeAndKey(mode constant.DBMode, key dbName) *gorm.DB {
	// 取读者锁，从而允许同时获取大量的DB
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch mode {
	case constant.WRITE:
		return s.writeList[key]
	case constant.READ:
		return s.writeList[key]
	default:
		// 默认获取写的，牺牲性能为代价，防止多个写
		return s.writeList[key]
	}
}

func (s *SQLiteEngine) GetDataDB(mode constant.DBMode) *gorm.DB {
	return s.getDBByModeAndKey(mode, DataDBKey)
}

func (s *SQLiteEngine) GetLogDB(mode constant.DBMode) *gorm.DB {
	return s.getDBByModeAndKey(mode, LogsDBKey)
}

func (s *SQLiteEngine) GetCensorDB(mode constant.DBMode) *gorm.DB {
	return s.getDBByModeAndKey(mode, CensorsDBKey)
}

const defaultDataDir = "./data/default"

func (s *SQLiteEngine) Init(ctx context.Context) error {
	log := zap.S().Named(logger.LogKeyDatabase)
	if ctx == nil {
		return errors.New("ctx is missing")
	}
	s.ctx = ctx
	s.DataDir = os.Getenv("DATADIR")
	if s.DataDir == "" {
		log.Debug("未能发现SQLITE定义位置，使用默认data地址")
		s.DataDir = defaultDataDir
	}
	// 检查s.DataDir是否存在，不存在则新建
	if _, err := os.Stat(s.DataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(s.DataDir, 0755); err != nil {
			// 打个日志
			log.Errorf("创建数据库文件目录失败，请检查是否有可写入权限：%v", err)
			return err
		}
	}
	// map初始化
	s.readList = make(map[dbName]*gorm.DB)
	s.writeList = make(map[dbName]*gorm.DB)
	err := s.dataDBInit()
	if err != nil {
		return err
	}
	err = s.LogDBInit()
	if err != nil {
		return err
	}
	err = s.CensorDBInit()
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLiteEngine) DBCheck() {
	dataDir := s.DataDir
	checkDB := func(db *gorm.DB) bool {
		rows, err := db.Raw("PRAGMA integrity_check").Rows()
		if err != nil {
			return false
		}
		defer rows.Close()
		var ok bool
		for rows.Next() {
			var s string
			if errR := rows.Scan(&s); errR != nil {
				ok = false
				break
			}
			fmt.Fprintln(os.Stdout, s)
			if s == "ok" {
				ok = true
			}
		}

		if errR := rows.Err(); errR != nil {
			ok = false
		}
		return ok
	}

	var ok1, ok2, ok3 bool
	var dataDB *gorm.DB
	var logsDB *gorm.DB
	var censorDB *gorm.DB
	var err error

	dbDataPath, _ := filepath.Abs(filepath.Join(dataDir, "data.db"))
	dataDB, err = SQLiteDBInit(dbDataPath, false)
	if err != nil {
		fmt.Fprintln(os.Stdout, "数据库 data.db 无法打开")
	} else {
		ok1 = checkDB(dataDB)
		db, _ := dataDB.DB()
		// 关闭
		db.Close()
	}

	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = SQLiteDBInit(dbDataLogsPath, false)
	if err != nil {
		fmt.Fprintln(os.Stdout, "数据库 data-logs.db 无法打开")
	} else {
		ok2 = checkDB(logsDB)
		db, _ := logsDB.DB()
		// 关闭db
		db.Close()
	}

	dbDataCensorPath, _ := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	censorDB, err = SQLiteDBInit(dbDataCensorPath, false)
	if err != nil {
		fmt.Fprintln(os.Stdout, "数据库 data-censor.db 无法打开")
	} else {
		ok3 = checkDB(censorDB)
		db, _ := censorDB.DB()
		// 关闭db
		db.Close()
	}

	fmt.Fprintln(os.Stdout, "数据库检查结果：")
	fmt.Fprintln(os.Stdout, "data.db:", ok1)
	fmt.Fprintln(os.Stdout, "data-logs.db:", ok2)
	fmt.Fprintln(os.Stdout, "data-censor.db:", ok3)
}

// 初始化
func (s *SQLiteEngine) dataDBInit() error {
	dbDataPath, _ := filepath.Abs(filepath.Join(s.DataDir, "data.db"))
	readDB, writeDB, err := SQLiteDBRWInit(dbDataPath)
	if err != nil {
		return err
	}
	// 添加并设置context
	dataContext := context.WithValue(s.ctx, cache.CacheKey, cache.DataDBCacheKey)
	readDB = readDB.WithContext(dataContext)
	writeDB = writeDB.WithContext(dataContext)
	s.readList[DataDBKey] = readDB
	s.writeList[DataDBKey] = writeDB
	return nil
}

func (s *SQLiteEngine) LogDBInit() error {
	dbDataLogsPath, _ := filepath.Abs(filepath.Join(s.DataDir, "data-logs.db"))
	readDB, writeDB, err := SQLiteDBRWInit(dbDataLogsPath)
	if err != nil {
		return err
	}
	// 添加并设置context
	logsContext := context.WithValue(s.ctx, cache.CacheKey, cache.LogsDBCacheKey)
	readDB = readDB.WithContext(logsContext)
	writeDB = writeDB.WithContext(logsContext)
	s.readList[LogsDBKey] = readDB
	s.writeList[LogsDBKey] = writeDB
	return nil
}

func (s *SQLiteEngine) CensorDBInit() error {
	path, err := filepath.Abs(filepath.Join(s.DataDir, "data-censor.db"))
	if err != nil {
		return err
	}
	readDB, writeDB, err := SQLiteDBRWInit(path)
	if err != nil {
		return err
	}
	// 添加并设置context
	censorContext := context.WithValue(s.ctx, cache.CacheKey, cache.CensorsDBCacheKey)
	readDB = readDB.WithContext(censorContext)
	writeDB = writeDB.WithContext(censorContext)
	
	// 创建 CensorLog 表结构
	if err := writeDB.AutoMigrate(&model.CensorLog{}); err != nil {
		return err
	}
	
	s.readList[CensorsDBKey] = readDB
	s.writeList[CensorsDBKey] = writeDB
	return nil
}
