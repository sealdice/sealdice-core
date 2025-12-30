package v151

import (
	"sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

var V151GORMCleanMigration = upgrade.Upgrade{
	ID: "007_V151GORMCleanMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: `
# 升级说明
删除了因为GORM更新导致逻辑失效后，错误插入的部分数据
`,
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] V151数据库清理错误插入数据开始")
		err := V151GORMCleanMigrate(operator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V151数据库清理错误插入数据处置完毕")
		return nil
	},
}
