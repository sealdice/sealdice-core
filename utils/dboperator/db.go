package dboperator

import (
	"context"
	"errors"
	"os"

	"go.uber.org/zap"

	"sealdice-core/logger"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/dboperator/engine/mysql"
	"sealdice-core/utils/dboperator/engine/pgsql"
	"sealdice-core/utils/dboperator/engine/sqlite"
)

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

// DBCheck 检查数据库状态
func DBCheck() {
	log := zap.S().Named(logger.LogKeyDatabase)
	dbEngine, err := NewDatabaseOperator(context.Background())
	if err != nil {
		log.Error("数据库引擎初始化失败:", err)
		return
	}
	defer dbEngine.Close()
	dbEngine.DBCheck()
}
