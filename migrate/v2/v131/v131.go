package v131

import (
	"sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

var V131ConfigUpdateMigration = upgrade.Upgrade{
	ID: "003_V131ConfigUpdateMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: `
# 升级说明

本次升级将旧版(v1.3.1)配置文件serve.yaml中的几个已弃用配置项（包括骰主帮助信息、使用协议、骰子状态附加文本和抽牌列表文本）迁移到新版的自定义文案配置文件text-template.yaml中。

迁移时会保留原始配置值，创建必要的配置结构，并在迁移成功后从原配置文件中删除这些旧项，同时会备份原始自定义文案文件。
`,
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] V131配置文件迁移自定义文案转移升级开始")
		err := V131DeprecatedConfig2CustomText()
		if err != nil {
			// 非常尴尬的是这里的err设置，设置是故意忽略的
			logf("[WARN] 发生错误，但可能并不影响升级，错误内容为" + err.Error() + "不放心可以让专业人员查看")
			return nil
		}
		logf("[INFO] V131配置文件迁移自定义文案升级完成")
		return nil
	},
}
