package v150

import (
	"fmt"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
)

// V150FixGroupInfoMigrate 针对GroupInfo特调，修正GroupInfo里面有问题的数据
func V150FixGroupInfoMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	db := dboperator.GetDataDB(constant.WRITE)
	res := db.Where(`
        NOT (
            (created_at IS NULL OR CAST(created_at AS INTEGER) > 0)
            AND (updated_at IS NULL OR CAST(updated_at AS INTEGER) > 0)
            AND data IS NOT NULL
        )
    `).Delete(&model.GroupInfo{})

	if res.Error != nil {
		logf(fmt.Sprintf("删除出现意外错误，错误为 %s", res.Error.Error()))
		return res.Error
	}

	logf(fmt.Sprintf("数据修复 - GroupInfo表，删除了 %d 条不符合条件的记录", res.RowsAffected))
	return nil
}
