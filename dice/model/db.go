package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"

	log "sealdice-core/utils/kratos"
)

func DBCheck(dataDir string) {
	checkDB := func(db *gorm.DB) bool {
		rows, err := db.Exec("PRAGMA integrity_check").Rows()
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
	dataDB, err = _SQLiteDBInit(dbDataPath, false)
	if err != nil {
		fmt.Fprintln(os.Stdout, "数据库 data.db 无法打开")
	} else {
		ok1 = checkDB(dataDB)
		db, _ := dataDB.DB()
		// 关闭
		db.Close()
	}

	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = _SQLiteDBInit(dbDataLogsPath, false)
	if err != nil {
		fmt.Fprintln(os.Stdout, "数据库 data-logs.db 无法打开")
	} else {
		ok2 = checkDB(logsDB)
		db, _ := logsDB.DB()
		// 关闭db
		db.Close()
	}

	dbDataCensorPath, _ := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	censorDB, err = _SQLiteDBInit(dbDataCensorPath, false)
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

var createSql = `
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
    updated_at INTEGER  default 0
);
`

func SQLiteDBInit(dataDir string) (dataDB *gorm.DB, logsDB *gorm.DB, err error) {
	dbDataPath, _ := filepath.Abs(filepath.Join(dataDir, "data.db"))
	dataDB, err = _SQLiteDBInit(dbDataPath, true)
	if err != nil {
		return nil, nil, err
	}
	// 特殊情况建表语句处置
	if strings.Contains(dataDB.Dialector.Name(), "sqlite") {
		tx := dataDB.Begin()
		// 检查是否有这个影响的注释
		var count int64
		err = dataDB.Raw("SELECT count(*) FROM `sqlite_master` WHERE tbl_name = 'attrs' AND `sql` LIKE '%这个方法太严格了%'").Count(&count).Error
		if err != nil {
			tx.Rollback()
			return nil, nil, err
		}
		if count > 0 {
			log.Warn("数据库 attrs 表结构为前置测试版本150,重建中")
			// 创建临时表
			err = tx.Exec(createSql).Error
			if err != nil {
				tx.Rollback()
				return nil, nil, err
			}
			// 迁移数据
			err = tx.Exec("INSERT INTO `attrs__temp` SELECT * FROM `attrs`").Error
			if err != nil {
				tx.Rollback()
				return nil, nil, err
			}
			// 删除旧的表
			err = tx.Exec("DROP TABLE `attrs`").Error
			if err != nil {
				tx.Rollback()
				return nil, nil, err
			}
			// 改名
			err = tx.Exec("ALTER TABLE `attrs__temp` RENAME TO `attrs`").Error
			if err != nil {
				tx.Rollback()
				return nil, nil, err
			}
			tx.Commit()
		}
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
		return nil, nil, err
	}
	err = dataDB.Exec("VACUUM").Error
	if err != nil {
		return nil, nil, err
	}
	logsDB, err = LogDBInit(dataDir)
	return
}

// LogDBInit SQLITE初始化
func LogDBInit(dataDir string) (logsDB *gorm.DB, err error) {
	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = _SQLiteDBInit(dbDataLogsPath, true)
	if err != nil {
		return
	}
	// logs建表
	if err = logsDB.AutoMigrate(&LogInfo{}, &LogOneItem{}); err != nil {
		return nil, err
	}
	err = logsDB.Exec("VACUUM").Error
	if err != nil {
		return nil, err
	}
	return logsDB, nil
}

func SQLiteCensorDBInit(dataDir string) (censorDB *gorm.DB, err error) {
	path, err := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	if err != nil {
		return nil, err
	}
	censorDB, err = _SQLiteDBInit(path, true)
	if err != nil {
		return nil, err
	}
	// 创建基本的表结构，并通过标签定义索引
	if err = censorDB.AutoMigrate(&CensorLog{}); err != nil {
		return nil, err
	}
	err = censorDB.Exec("VACUUM").Error
	if err != nil {
		return nil, err
	}
	return censorDB, nil
}
