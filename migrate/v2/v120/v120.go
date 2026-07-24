package v120

import (
	"os"

	"sealdice-core/utils"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

var V120Migration = upgrade.Upgrade{
	ID: "001_V120Migration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: `
### 🆕 升级说明

版本V120升级

#### 1. 配置文件结构化迁移

- 新增支持将旧版 serve.yaml 中的配置数据迁移至 SQLite 数据库；
- 涉及迁移内容包括群组信息表（group_info、group_player_info）及属性类数据表（如 attrs_group、attrs_group_user、attrs_user、ban_info）；
- 有助于后续的管理与查询操作；
- 升级过程中将自动保留原始配置文件为 serve.yaml.old，以供回溯查验。

#### 2. 日志系统数据库化改造

- 支持将旧版 BoltDB 日志格式结构化迁移到 SQL 数据库；
- 新增 logs 与 log_items 两张日志相关表，并建立索引以优化查询性能；
- 提供完整的数据迁移逻辑，实现历史日志的平滑过渡；
- 为后续实现日志管理、检索与上传等功能奠定基础。
`,
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		const bdbPath = "./data/default/data.bdb"
		logf("[INFO] 尝试检查是否为V120版本升级到新版本")
		if _, err := os.Stat(bdbPath); err != nil {
			logf("[INFO] V120升级已经被应用过或版本为新版本，无需应用升级")
			return nil // 没有旧数据库，无需迁移
		}

		// 若新版 attrs 表已存在，说明 V150 已执行过，data.bdb 中的旧数据早已迁移完成。
		// 跳过可避免"V120 重建旧表 → V150 重跑时 convertToNew 生成重复角色卡"。
		if operator.GetDataDB(constant.WRITE).Migrator().HasTable("attrs") {
			logf("[INFO] 新版 attrs 表已存在，data.bdb 数据已迁移，将旧文件重命名为 data.bdb.migrated 作为备份")
			_ = os.Remove(bdbPath + ".migrated") // Windows 上 Rename 不覆盖已有目标，先清理
			_ = os.Rename(bdbPath, bdbPath+".migrated")
			return nil
		}

		// 尝试升级 TODO: 历史遗留的SQLX，如果改动怕升级失败，不改动吧又看不到日志
		dataDB, err := utils.GetSQLXDB(operator.GetDataDB(constant.WRITE))
		if err != nil {
			return err
		}
		logDB, err := utils.GetSQLXDB(operator.GetLogDB(constant.WRITE))
		if err != nil {
			return err
		}
		err = ConvertServe(dataDB)
		if err != nil {
			return err
		}
		err = ConvertLogs(logDB)
		if err != nil {
			return err
		}
		// 迁移成功，重命名 data.bdb 为备份，后续启动不再触发本迁移
		logf("[INFO] V120 升级完成，将 data.bdb 重命名为 data.bdb.migrated 作为备份")
		_ = os.Remove(bdbPath + ".migrated") // Windows 上 Rename 不覆盖已有目标，先清理
		_ = os.Rename(bdbPath, bdbPath+".migrated")
		return nil
	},
}

var V120LogMessageMigration = upgrade.Upgrade{
	ID:          "002_V120LogMessageMigration", // TODO：需要合理的生成逻辑，这个等提交了PR再后续讨论
	Description: "V120到V131内，有一个被应用的数据库修正，旨在将错误的message字段类型修改为正确的",
	Apply: func(logf func(string), operator engine.DatabaseOperator) error {
		logf("[INFO] 尝试检查数据库状态")
		if operator.Type() != "sqlite" {
			logf("[INFO] 仅sqlite数据库部分情况下需要处理，您无需处理")
			return nil
		}
		logDB, err := utils.GetSQLXDB(operator.GetLogDB(constant.WRITE))
		if err != nil {
			return err
		}
		err = LogItemFixDatatype(logDB)
		if err != nil {
			logf("[INFO] 升级失败，请检查数据库状态")
			return nil
		}
		logf("[INFO] 升级完毕")
		return nil
	},
}
