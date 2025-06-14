package v141

import (
	"sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

var V141ConfigUpdateMigration = upgrade.Upgrade{
	ID: "004_V141ConfigUpdateMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: `
# 升级说明

将旧版(v1.4.1)配置文件serve.yaml中的两个已弃用配置项进行重命名：  
将 customReplenishRate 重命名为 personalReplenishRate  

将 customBurst 重命名为 personalBurst  

如果配置文件不存在，则直接跳过；否则读取文件内容，解析为 YAML，检查并替换旧字段名，最后将修改后的配置写回文件。整个过程确保向后兼容性，同时更新配置结构。
`,
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] V141已弃用配置项进行重命名升级开始")
		err := V141DeprecatedConfigRename()
		if err != nil {
			// 非常尴尬的是这里的err设置，设置是故意忽略的
			logf("[WARN] 发生错误，但可能并不影响升级，错误内容为" + err.Error() + "不放心可以让专业人员查看")
			return nil
		}
		logf("[INFO] V141已弃用配置项进行重命名升级完成")
		return nil
	},
}
