package v151

import (
	"fmt"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
)

// V151GORMCleanMigrate 删除包含NULL的数据
func V151GORMCleanMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	db := dboperator.GetDataDB(constant.WRITE)

	// 处理 BanInfo 表
	var banIDs []string
	db.Model(&model.BanInfo{}).
		Where("data IS NULL OR data = '' OR length(data) = 0").
		Pluck("id", &banIDs) // 先获取所有要删除的ID

	banResult := db.Where("data IS NULL OR data = '' OR length(data) = 0").Delete(&model.BanInfo{})

	// 处理 AttributesItemModel 表
	var attrIDs []string
	db.Model(&model.AttributesItemModel{}).
		Where("data IS NULL OR data = '' OR length(data) = 0").
		Pluck("id", &attrIDs)

	attrResult := db.Where("data IS NULL OR data = '' OR length(data) = 0").Delete(&model.AttributesItemModel{})

	// 打印日志
	if len(banIDs) > 0 {
		logf(fmt.Sprintf("数据修复 - BanInfo表，删除了 %d 条记录，ID列表: %v", banResult.RowsAffected, banIDs))
	} else {
		logf("数据修复 - BanInfo表，没有需要删除的记录")
	}

	if len(attrIDs) > 0 {
		logf(fmt.Sprintf("数据修复 - AttributesItemModel表，删除了 %d 条记录，ID列表: %v", attrResult.RowsAffected, attrIDs))
	} else {
		logf("数据修复 - AttributesItemModel表，没有需要删除的记录")
	}

	return nil
}
