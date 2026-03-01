package v160

import (
	"fmt"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

// V160LogIDZeroCleanMigrate 清理 log_id = 0 的日志行与日志表残留
func V160LogIDZeroCleanMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	db := dboperator.GetLogDB(constant.WRITE)
	migrator := db.Migrator()

	// 初始化以及正常没有logid为0时不执行migration
	if !migrator.HasTable(&model.LogInfo{}) || !migrator.HasTable(&model.LogOneItem{}) {
		return nil
	}

	var hasLogIDZero bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM logs WHERE id = 0 LIMIT 1)").Scan(&hasLogIDZero).Error; err != nil {
		return err
	}

	var hasItemLogIDZero bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM log_items WHERE log_id = 0 LIMIT 1)").Scan(&hasItemLogIDZero).Error; err != nil {
		return err
	}

	if !hasLogIDZero && !hasItemLogIDZero {
		return nil
	}

	itemResult := db.Where("log_id = 0").Delete(&model.LogOneItem{})
	if itemResult.Error != nil {
		return itemResult.Error
	}

	logResult := db.Where("id = 0").Delete(&model.LogInfo{})
	if logResult.Error != nil {
		return logResult.Error
	}

	// log_id=0 清理后，回填剩余日志(size 仅需更新 id>0 的有效日志行)
	recountResult := db.Model(&model.LogInfo{}).Where("id > 0").Update("size", gorm.Expr(
		"(SELECT COUNT(1) FROM log_items WHERE log_items.log_id = logs.id AND log_items.removed IS NULL)",
	))
	if recountResult.Error != nil {
		return recountResult.Error
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
	logf(fmt.Sprintf("数据修复 - Logs表，重算了 %d 条size字段", recountResult.RowsAffected))

	return nil
}

var V160LogIDZeroCleanMigration = upgrade.Upgrade{
	ID: "008_V160LogIDZeroCleanMigration",
	Description: `
# 升级说明
清理 log_id = 0 的错误日志数据
`,
	Apply: func(logf func(string), dbOperator operator.DatabaseOperator) error {
		logf("[INFO] V160清理log_id=0数据开始")
		err := V160LogIDZeroCleanMigrate(dbOperator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V160清理log_id=0数据处置完毕")
		return nil
	},
}
