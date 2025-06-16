package v144

import (
	"sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

var V144RemoveOldHelpDocMigration = upgrade.Upgrade{
	ID: "005_V144RemoveOldHelpDocMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: `
# 升级说明
这段代码用于清理旧版帮助文档文件，在验证新旧文件哈希值匹配后，安全删除旧版"蜜瓜包-怪物之锤查询.json"文件。
`,
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] V144清理旧版帮助文档升级开始")
		err := V144RemoveOldHelpdoc()
		if err != nil {
			// 非常尴尬的是这里的err设置，设置是故意忽略的
			logf("[WARN] 发生错误，但可能并不影响升级，错误内容为" + err.Error() + "不放心可以让专业人员查看")
			return err
		}
		logf("[INFO] V144清理旧版帮助文档升级完成")
		return nil
	},
}
