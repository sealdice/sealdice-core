package engine

import (
	"context"

	"gorm.io/gorm"

	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine/mysql"
	"sealdice-core/utils/dboperator/engine/pgsql"
	"sealdice-core/utils/dboperator/engine/sqlite"
)

// DatabaseOperator 本来是单独放了个文件夹的，但是由于现在所有的model都和处理逻辑在一起，如果放在单独文件夹必然会循环依赖
// 为了完善逻辑，去除Init，改为Init后，使用函数获取readDB和WriteDB
type DatabaseOperator interface {
	Init(ctx context.Context) error
	DBCheck()
	GetDataDB(mode constant.DBMode) *gorm.DB
	GetLogDB(mode constant.DBMode) *gorm.DB
	GetCensorDB(mode constant.DBMode) *gorm.DB
	Close()
}

// 实现检查 copied from platform
var (
	_ DatabaseOperator = (*sqlite.SQLiteEngine)(nil)
	_ DatabaseOperator = (*mysql.MYSQLEngine)(nil)
	_ DatabaseOperator = (*pgsql.PGSQLEngine)(nil)
)
