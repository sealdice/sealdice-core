package dboperator

import (
	"context"
	"errors"
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
func newEngine() operator.DatabaseOperator {
	log := zap.S().Named(logger.LogKeyDatabase)

	dbType := os.Getenv("DB_TYPE")
	switch dbType {
	case constant.SQLITE:
		log.Info("当前选择使用: SQLITE数据库")
		return &sqlite.SQLiteEngine{}
	case constant.MYSQL:
		log.Info("当前选择使用: MYSQL数据库")
		return &mysql.MYSQLEngine{}
	case constant.POSTGRESQL:
		log.Info("当前选择使用: POSTGRESQL数据库")
		return &pgsql.PGSQLEngine{}
	default:
		log.Warn("未配置数据库类型，默认使用: SQLITE数据库")
		return &sqlite.SQLiteEngine{}
	}
}

// NewDatabaseOperator 创建一个独立的数据库引擎实例。
// Runtime 重建必须使用新实例，不能复用已关闭的包级单例。
func NewDatabaseOperator(ctx context.Context) (operator.DatabaseOperator, error) {
	engineInstance := newEngine()
	if engineInstance == nil {
		return nil, errors.New("database engine factory returned nil")
	}
	if err := engineInstance.Init(ctx); err != nil {
		engineInstance.Close()
		return nil, err
	}
	return engineInstance, nil
}

func initEngine() {
	engine, errEngineInstance = NewDatabaseOperator(context.Background())
	if errEngineInstance != nil {
		zap.S().Named(logger.LogKeyDatabase).Error("数据库引擎初始化失败:", errEngineInstance)
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
