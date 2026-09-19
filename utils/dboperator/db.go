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
	closeOnce         sync.Once
	errEngineInstance error
	// rootCtx 是数据库引擎的统一上下文，在 Close 时被取消。
	rootCtx, cancelRoot = context.WithCancel(context.Background())
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
	errEngineInstance = engine.Init(rootCtx)
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

// Context 返回数据库引擎的统一上下文，在 Close 时被取消。
func Context() context.Context {
	return rootCtx
}

// Close 取消数据库引擎上下文并关闭所有数据库连接，用于程序退出时优雅释放资源。
func Close() {
	closeOnce.Do(func() {
		if cancelRoot != nil {
			cancelRoot()
		}
		if engine != nil {
			engine.Close()
		}
	})
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
