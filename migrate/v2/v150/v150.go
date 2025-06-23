package v150

import (
	"sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

var V150UpgradeAttrsMigration = upgrade.Upgrade{
	ID: "006_V150UpgradeAttrsMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: `
# 升级说明
本次升级实现了从旧版数据库结构到新版（v1.5.0）的迁移，主要将分散的attrs_user、attrs_group和attrs_group_user表合并为统一的attrs表，同时转换角色卡数据格式并维护绑定关系，最终删除旧表完成升级。
`,
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] V150数据库重构升级开始")
		err := V150AttrsMigrate(operator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V150数据库重构升级完毕")
		return nil
	},
}

var V150FixGroupInfoMigration = upgrade.Upgrade{
	ID: "007_V150FixGroupInfoMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: `
# 升级说明
清理掉在Ver1.4.6内，SQLX会导致的GroupInfo出现列错乱的问题。
`,
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] V150 GroupInfo数据库修复开始")
		err := V150FixGroupInfoMigrate(operator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V150 GroupInfo数据库修复开始")
		return nil
	},
}
