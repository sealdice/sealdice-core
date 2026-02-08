package v160

import (
	"fmt"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

// V160LogIDZeroCleanMigrate 清理 log_id = 0 的日志行与日志表残留
func V160LogIDZeroCleanMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	db := dboperator.GetLogDB(constant.WRITE)

	var itemIDs []uint64
	db.Model(&model.LogOneItem{}).
		Where("log_id = 0").
		Pluck("id", &itemIDs)
	itemResult := db.Where("log_id = 0").Delete(&model.LogOneItem{})

	var logIDs []uint64
	db.Model(&model.LogInfo{}).
		Where("id = 0").
		Pluck("id", &logIDs)
	logResult := db.Where("id = 0").Delete(&model.LogInfo{})

	if len(itemIDs) > 0 {
		logf(fmt.Sprintf("数据修复 - LogItems表，删除了 %d 条记录，ID列表: %v", itemResult.RowsAffected, itemIDs))
	} else {
		logf("数据修复 - LogItems表，没有需要删除的记录")
	}

	if len(logIDs) > 0 {
		logf(fmt.Sprintf("数据修复 - Logs表，删除了 %d 条记录，ID列表: %v", logResult.RowsAffected, logIDs))
	} else {
		logf("数据修复 - Logs表，没有需要删除的记录")
	}

	return nil
}

var V160LogIDZeroCleanMigration = upgrade.Upgrade{
	ID: "001_V160LogIDZeroCleanMigration",
	Description: `
# 升级说明
清理 log_id = 0 的错误日志数据
`,
	Apply: func(logf func(string), operator operator.DatabaseOperator) error {
		logf("[INFO] V160清理log_id=0数据开始")
		err := V160LogIDZeroCleanMigrate(operator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V160清理log_id=0数据处置完毕")
		return nil
	},
}
