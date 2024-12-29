package model

import (
	"os"
	"sync"

	"gorm.io/gorm"

	log "sealdice-core/utils/kratos"
)

var (
	engine            DatabaseOperator
	once              sync.Once
	errEngineInstance error
)

// initEngine 初始化数据库引擎，仅执行一次
func initEngine() {
	dbType := os.Getenv("DB_TYPE")
	switch dbType {
	case SQLITE:
		log.Info("当前选择使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	case MYSQL:
		log.Info("当前选择使用: MYSQL数据库")
		engine = &MYSQLEngine{}
	case POSTGRESQL:
		log.Info("当前选择使用: POSTGRESQL数据库")
		engine = &PGSQLEngine{}
	default:
		log.Warn("未配置数据库类型，默认使用: SQLITE数据库")
		engine = &SQLiteEngine{}
	}

	errEngineInstance = engine.Init()
	if errEngineInstance != nil {
		log.Error("数据库引擎初始化失败:", errEngineInstance)
	}
}

// getEngine 获取数据库引擎，确保只初始化一次
func getEngine() (DatabaseOperator, error) {
	once.Do(initEngine)
	return engine, errEngineInstance
}

// DatabaseInit 初始化数据和日志数据库
func DatabaseInit() (dataDB *gorm.DB, logsDB *gorm.DB, err error) {
	engine, err = getEngine()
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
	// TODO: 将这段逻辑挪移到Migrator上
	var ids []uint64
	var logItemSums []struct {
		LogID uint64
		Count int64
	}
	logsDB.Model(&LogInfo{}).Where("size IS NULL").Pluck("id", &ids)
	if len(ids) > 0 {
		// 根据 LogInfo 表中的 IDs 查找对应的 LogOneItem 记录
		err = logsDB.Model(&LogOneItem{}).
			Where("log_id IN ?", ids).
			Group("log_id").
			Select("log_id, COUNT(*) AS count"). // 如果需要求和其他字段，可以使用 Sum
			Scan(&logItemSums).Error
		if err != nil {
			// 错误处理
			log.Infof("Error querying LogOneItem: %v", err)
			return nil, nil, err
		}

		// 2. 更新 LogInfo 表的 Size 字段
		for _, sum := range logItemSums {
			// 将求和结果更新到对应的 LogInfo 的 Size 字段
			err = logsDB.Model(&LogInfo{}).
				Where("id = ?", sum.LogID).
				UpdateColumn("size", sum.Count).Error // 或者是 sum.Time 等，如果要是其他字段的求和
			if err != nil {
				// 错误处理
				log.Errorf("Error updating LogInfo: %v", err)
				return nil, nil, err
			}
		}
	}
	return dataDB, logsDB, nil
}

// DBCheck 检查数据库状态
func DBCheck() {
	dbEngine, err := getEngine()
	if err != nil {
		log.Error("数据库引擎获取失败:", err)
		return
	}

	dbEngine.DBCheck()
}

// CensorDBInit 初始化敏感词数据库
func CensorDBInit() (censorDB *gorm.DB, err error) {
	censorEngine, err := getEngine()
	if err != nil {
		return nil, err
	}

	return censorEngine.CensorDBInit()
}
