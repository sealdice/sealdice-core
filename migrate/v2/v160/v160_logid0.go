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

	itemResult := db.Where("log_id = 0").Delete(&model.LogOneItem{})
	if itemResult.Error != nil {
		return itemResult.Error
	}

	logResult := db.Where("id = 0").Delete(&model.LogInfo{})
	if logResult.Error != nil {
		return logResult.Error
	}

	if itemResult.RowsAffected > 0 {
		logf(fmt.Sprintf("数据修复 - LogItems表，删除了 %d 条记录", itemResult.RowsAffected))
	} else {
		logf("数据修复 - LogItems表，没有需要删除的记录")
	}

	if logResult.RowsAffected > 0 {
		logf(fmt.Sprintf("数据修复 - Logs表，删除了 %d 条记录", logResult.RowsAffected))
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
