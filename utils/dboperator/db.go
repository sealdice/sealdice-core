package dboperator

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"

	"sealdice-core/logger"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/dboperator/engine/mysql"
	"sealdice-core/utils/dboperator/engine/pgsql"
	"sealdice-core/utils/dboperator/engine/sqlite"
)

var (
	engine            operator.DatabaseOperator
	once              sync.Once
	errEngineInstance error
)

// initEngine 初始化数据库引擎，仅执行一次
func initEngine() {
	log := zap.S().Named(logger.LogKeyDatabase)

	dbType := os.Getenv("DB_TYPE")
	switch dbType {
	case constant.SQLITE:
		log.Info("当前选择使用: SQLITE数据库")
		engine = &sqlite.SQLiteEngine{}
	case constant.MYSQL:
		log.Info("当前选择使用: MYSQL数据库")
		engine = &mysql.MYSQLEngine{}
	case constant.POSTGRESQL:
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
}

// getEngine 获取数据库引擎，确保只初始化一次
func getEngine() (operator.DatabaseOperator, error) {
	once.Do(initEngine)
	return engine, errEngineInstance
}

// GetDatabaseOperator 初始化数据和日志数据库
func GetDatabaseOperator() (operator.DatabaseOperator, error) {
	return getEngine()
}

// DBCheck 检查数据库状态
func DBCheck() {
	log := zap.S().Named(logger.LogKeyDatabase)
	dbEngine, err := getEngine()
	if err != nil {
		log.Error("数据库引擎获取失败:", err)
		return
	}
	dbEngine.DBCheck()
}
