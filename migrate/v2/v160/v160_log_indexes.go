package v160

import (
	"fmt"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

func V160LogRawMsgIDIndexMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	db := dboperator.GetLogDB(constant.WRITE)
	if !db.Migrator().HasTable(&model.LogOneItem{}) {
		return nil
	}

	switch dboperator.Type() {
	case constant.MYSQL:
		if !db.Migrator().HasIndex(&model.LogOneItemHookMySQL{}, "idx_log_delete_by_id") {
			if err := db.Exec("CREATE INDEX idx_log_delete_by_id ON log_items(group_id(20), raw_msg_id(20), id)").Error; err != nil {
				return err
			}
			logf("数据修复 - LogItems表，已为(group_id, raw_msg_id, id)创建复合索引")
		} else {
			logf("数据修复 - LogItems表，复合索引已存在，无需处理")
		}
	default:
		if !db.Migrator().HasIndex(&model.LogOneItem{}, "idx_log_delete_by_id") {
			if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_log_delete_by_id ON log_items(group_id, raw_msg_id, id)").Error; err != nil {
				return err
			}
			logf("数据修复 - LogItems表，已为(group_id, raw_msg_id, id)创建复合索引")
		} else {
			logf("数据修复 - LogItems表，复合索引已存在，无需处理")
		}
	}

	return nil
}

var V160LogRawMsgIDIndexMigration = upgrade.Upgrade{
	ID: "008a_V160LogRawMsgIDIndexMigration",
	Description: `
# 升级说明
为日志消息回查补齐(group_id, raw_msg_id, id)复合索引
`,
	Apply: func(logf func(string), operator operator.DatabaseOperator) error {
		logf(fmt.Sprintf("[INFO] V160日志索引修复开始 type=%s", operator.Type()))
		err := V160LogRawMsgIDIndexMigrate(operator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V160日志索引修复处置完毕")
		return nil
	},
}
