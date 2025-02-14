package dboperator

import (
	"context"
	"os"
	"sync"

	"gorm.io/gorm"

	"sealdice-core/dice/dao"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
	engine2 "sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/dboperator/engine/mysql"
	"sealdice-core/utils/dboperator/engine/pgsql"
	"sealdice-core/utils/dboperator/engine/sqlite"
	log "sealdice-core/utils/kratos"
)

var (
	engine            engine2.DatabaseOperator
	once              sync.Once
	errEngineInstance error
)

// initEngine 初始化数据库引擎，仅执行一次
func initEngine() {
	dbType := os.Getenv("DB_TYPE")
	switch dbType {
	case dao.SQLITE:
		log.Info("当前选择使用: SQLITE数据库")
		engine = &sqlite.SQLiteEngine{}
	case dao.MYSQL:
		log.Info("当前选择使用: MYSQL数据库")
		engine = &mysql.MYSQLEngine{}
	case dao.POSTGRESQL:
		log.Info("当前选择使用: POSTGRESQL数据库")
		engine = &pgsql.PGSQLEngine{}
	default:
		log.Warn("未配置数据库类型，默认使用: SQLITE数据库")
		engine = &sqlite.SQLiteEngine{}
	}
	// TODO: 使用统一管理的context，以确保在程序关闭时，可以正确销毁数据库的context从而优雅退出
	errEngineInstance = engine.Init(context.Background())
	if errEngineInstance != nil {
		log.Error("数据库引擎初始化失败:", errEngineInstance)
	}
	err := hackMigrator(engine.GetLogDB(constant.WRITE))
	if err != nil {
		log.Errorf("数据库引擎初始化失败: %v", err)
		return
	}
}

// getEngine 获取数据库引擎，确保只初始化一次
func getEngine() (engine2.DatabaseOperator, error) {
	once.Do(initEngine)
	return engine, errEngineInstance
}

// GetDatabaseOperator 初始化数据和日志数据库
func GetDatabaseOperator() (engine2.DatabaseOperator, error) {
	return getEngine()
}

func hackMigrator(logsDB *gorm.DB) error {
	// TODO: 将这段逻辑挪移到Migrator上
	var ids []uint64
	var logItemSums []struct {
		LogID uint64
		Count int64
	}
	logsDB.Model(&model.LogInfo{}).Where("size IS NULL").Pluck("id", &ids)
	if len(ids) > 0 {
		// 根据 LogInfo 表中的 IDs 查找对应的 LogOneItem 记录
		err := logsDB.Model(&model.LogOneItem{}).
			Where("log_id IN ?", ids).
			Group("log_id").
			Select("log_id, COUNT(*) AS count"). // 如果需要求和其他字段，可以使用 Sum
			Scan(&logItemSums).Error
		if err != nil {
			// 错误处理
			log.Infof("Error querying LogOneItem: %v", err)
			return err
		}

		// 2. 更新 LogInfo 表的 Size 字段
		for _, sum := range logItemSums {
			// 将求和结果更新到对应的 LogInfo 的 Size 字段
			err = logsDB.Model(&model.LogInfo{}).
				Where("id = ?", sum.LogID).
				UpdateColumn("size", sum.Count).Error // 或者是 sum.Time 等，如果要是其他字段的求和
			if err != nil {
				// 错误处理
				log.Errorf("Error updating LogInfo: %v", err)
				return err
			}
		}
	}
	return nil
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
