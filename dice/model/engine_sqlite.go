package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	"sealdice-core/dice/model/database"
	log "sealdice-core/utils/kratos"
)

type SQLiteEngine struct {
	DataDir string
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

func (s *SQLiteEngine) Init() error {
	s.DataDir = os.Getenv("DATADIR")
	if s.DataDir == "" {
		log.Debug("未能发现SQLITE定义位置，使用默认data地址")
		s.DataDir = defaultDataDir
	}
	return nil
}

// DB检查 BUG FIXME
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
	dataDB, err = database.SQLiteDBInit(dbDataPath, false)
	if err != nil {
		fmt.Fprintln(os.Stdout, "数据库 data.db 无法打开")
	} else {
		ok1 = checkDB(dataDB)
		db, _ := dataDB.DB()
		// 关闭
		db.Close()
	}

	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = database.SQLiteDBInit(dbDataLogsPath, false)
	if err != nil {
		fmt.Fprintln(os.Stdout, "数据库 data-logs.db 无法打开")
	} else {
		ok2 = checkDB(logsDB)
		db, _ := logsDB.DB()
		// 关闭db
		db.Close()
	}

	dbDataCensorPath, _ := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	censorDB, err = database.SQLiteDBInit(dbDataCensorPath, false)
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
func (s *SQLiteEngine) DataDBInit() (*gorm.DB, error) {
	dbDataPath, _ := filepath.Abs(filepath.Join(s.DataDir, "data.db"))
	dataDB, err := database.SQLiteDBInit(dbDataPath, true)
	if err != nil {
		return nil, err
	}
	// 特殊情况建表语句处置
	tx := dataDB.Begin()
	// 检查是否有这个影响的注释
	var count int64
	err = dataDB.Raw("SELECT count(*) FROM `sqlite_master` WHERE tbl_name = 'attrs' AND `sql` LIKE '%这个方法太严格了%'").Count(&count).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if count > 0 {
		log.Warn("数据库 attrs 表结构为前置测试版本150,重建中")
		// 创建临时表
		err = tx.Exec(createSql).Error
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		// 迁移数据
		err = tx.Exec("INSERT INTO `attrs__temp` SELECT * FROM `attrs`").Error
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		// 删除旧的表
		err = tx.Exec("DROP TABLE `attrs`").Error
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		// 改名
		err = tx.Exec("ALTER TABLE `attrs__temp` RENAME TO `attrs`").Error
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		tx.Commit()
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

func (s *SQLiteEngine) LogDBInit() (*gorm.DB, error) {
	dbDataLogsPath, _ := filepath.Abs(filepath.Join(s.DataDir, "data-logs.db"))
	logsDB, err := database.SQLiteDBInit(dbDataLogsPath, true)
	if err != nil {
		return nil, err
	}
	// logs建表
	if err = logsDB.AutoMigrate(&LogInfo{}); err != nil {
		return nil, err
	}

	itemsAutoMigrate := false
	// 用于确认是否需要重建LogOneItem数据库
	if logsDB.Migrator().HasTable(&LogOneItem{}) {
		if err = logItemsSQLiteMigrate(logsDB); err != nil {
			return nil, err
		}
	} else {
		itemsAutoMigrate = true
	}
	if itemsAutoMigrate {
		if err = logsDB.AutoMigrate(&LogOneItem{}); err != nil {
			return nil, err
		}
	}
	return logsDB, nil
}

func (s *SQLiteEngine) CensorDBInit() (*gorm.DB, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = defaultDataDir
	}
	path, err := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	if err != nil {
		return nil, err
	}
	censorDB, err := database.SQLiteDBInit(path, true)
	if err != nil {
		return nil, err
	}
	// 创建基本的表结构，并通过标签定义索引
	if err = censorDB.AutoMigrate(&CensorLog{}); err != nil {
		return nil, err
	}
	return censorDB, nil
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
	err = stmt.Parse(&LogOneItem{})
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
