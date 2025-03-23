package sqlite

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/cache"
	"sealdice-core/utils/constant"
	log "sealdice-core/utils/kratos"
)

type SQLiteEngine struct {
	DataDir string
	ctx     context.Context
	// 用于控制readList和writeList的读写锁
	mu        sync.RWMutex
	readList  map[dbName]*gorm.DB
	writeList map[dbName]*gorm.DB
}

// 定义一个基于 string 的新类型 dbName
type dbName string

const (
	LogsDBKey    dbName = "logs"
	DataDBKey    dbName = "data"
	CensorsDBKey dbName = "censor"
)

func (s *SQLiteEngine) Close() {
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

const createSql = `
CREATE TABLE attrs__temp (
    id TEXT PRIMARY KEY,
    data BYTEA,
    attrs_type TEXT,
    binding_sheet_id TEXT default '',
    name TEXT default '',
    owner_id TEXT default '',
    sheet_type TEXT default '',
    is_hidden BOOLEAN default FALSE,
    created_at INTEGER default 0,
    updated_at INTEGER default 0
);
`

func (s *SQLiteEngine) Init(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ctx is missing")
	}
	s.ctx = ctx
	s.DataDir = os.Getenv("DATADIR")
	if s.DataDir == "" {
		log.Debug("未能发现SQLITE定义位置，使用默认data地址")
		s.DataDir = defaultDataDir
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

	// 检查是否有这个影响的注释
	var count int64
	err = readDB.Raw("SELECT count(*) FROM `sqlite_master` WHERE tbl_name = 'attrs' AND `sql` LIKE '%这个方法太严格了%'").Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		// 特殊情况建表语句处置
		err = writeDB.Transaction(func(tx *gorm.DB) error {
			log.Warn("数据库 attrs 表结构为前置测试版本150,重建中")
			// 创建临时表
			err = tx.Exec(createSql).Error
			if err != nil {
				return err
			}
			// 迁移数据
			err = tx.Exec("INSERT INTO `attrs__temp` SELECT * FROM `attrs`").Error
			if err != nil {
				return err
			}
			// 删除旧的表
			err = tx.Exec("DROP TABLE `attrs`").Error
			if err != nil {
				return err
			}
			// 改名
			err = tx.Exec("ALTER TABLE `attrs__temp` RENAME TO `attrs`").Error
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// data建表
	err = writeDB.AutoMigrate(
		&model.GroupPlayerInfoBase{},
		&model.GroupInfo{},
		&model.BanInfo{},
		&model.EndpointInfo{},
		&model.AttributesItemModel{},
	)
	if err != nil {
		return err
	}
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
	// logs建表
	if err = writeDB.AutoMigrate(&model.LogInfo{}); err != nil {
		return err
	}

	itemsAutoMigrate := false
	// 用于确认是否需要重建LogOneItem数据库
	if writeDB.Migrator().HasTable(&model.LogOneItem{}) {
		if err = logItemsSQLiteMigrate(writeDB); err != nil {
			return err
		}
	} else {
		itemsAutoMigrate = true
	}
	if itemsAutoMigrate {
		if err = writeDB.AutoMigrate(&model.LogOneItem{}); err != nil {
			return err
		}
	}
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
	// 创建基本的表结构，并通过标签定义索引
	if err = writeDB.AutoMigrate(&model.CensorLog{}); err != nil {
		return err
	}
	s.readList[CensorsDBKey] = readDB
	s.writeList[CensorsDBKey] = writeDB
	return nil
}

func logItemsSQLiteMigrate(db *gorm.DB) error {
	type DBColumn struct {
		Name string
		Type string
	}

	// 获取当前列信息
	var currentColumns []DBColumn
	err := db.Raw("PRAGMA table_info(log_items)").Scan(&currentColumns).Error
	if err != nil {
		return err
	}

	// 获取模型定义的列信息
	var modelColumns []DBColumn
	stmt := &gorm.Statement{DB: db}
	err = stmt.Parse(&model.LogOneItem{})
	if err != nil {
		return err
	}
	for _, field := range stmt.Schema.Fields {
		if field.DBName != "" {
			x := db.Migrator().FullDataTypeOf(field)
			col := strings.SplitN(x.SQL, " ", 2)[0]
			modelColumns = append(modelColumns, DBColumn{field.DBName, strings.ToLower(col)})
		}
	}

	// 比较列是否有变化
	needMigrate := false
	if len(currentColumns) != len(modelColumns) {
		needMigrate = true
	} else {
		columnMap := make(map[string]string)
		for _, col := range currentColumns {
			columnMap[col.Name] = strings.ToLower(col.Type)
		}

		for _, col := range modelColumns {
			newType := col.Type
			currentType := columnMap[col.Name]

			// 特殊处理 is_dice 列,允许 bool 或 numeric 类型
			if col.Name == "is_dice" {
				if currentType != "bool" && currentType != "numeric" {
					needMigrate = true
					break
				}
				continue
			}

			if currentType != newType {
				needMigrate = true
				break
			}
		}
	}

	// 如果需要迁移则执行
	if needMigrate {
		log.Info("现在进行log_items表的迁移，如果数据库较大，会花费较长时间，请耐心等待")
		log.Info("若是迁移后观察到数据库体积显著膨胀，可以关闭骰子使用 sealdice-core --vacuum 进行数据库整理，这同样会花费较长时间")
		return db.AutoMigrate()
	}

	return nil
}
