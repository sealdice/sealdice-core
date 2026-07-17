package v160

import (
	"fmt"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
	upgrade "sealdice-core/utils/upgrader"
)

// 背景：V150 的历史升级存在失误——在某些历史版本中，V150 的“建表 + 计算 size”逻辑尚未就位
// 就已经被应用并记录为“已升级”。这导致 logs 表可能根本没有 size 列，或 size 列虽存在但从未被正确计算过。
// 由于升级框架会把 V150 记为已完成、不会再次执行，因此这里用一条新的 V160 迁移来兜底：
//  1. 检测 logs 表是否有 size 列；没有则补建（size 列的定义参见 V150：logs 表上记录该日志条目数的 INTEGER 列）。
//  2. 进行一次 size 列的全量重算（计算方式与 V150 的 calculateLogSize 一致：按 log_id 统计 log_items 行数）。
//
// 关于 size 的语义：此处与 V160 的 log_id=0 清理迁移保持一致——只统计 removed IS NULL 的可见条目，
// 这也与运行期 LogAppend(+1)/LogMarkDelete(-1) 的不变式相符。V150 旧版 calculateLogSize 统计的是全部行
// （不区分 removed），属已知的旧实现差异，本迁移统一到“仅统计可见条目”的口径。

// V160LogSizeRepairMigrate 修复 logs.size 列：缺失则补建，并全量重算每条日志的条目数。
func V160LogSizeRepairMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	db := dboperator.GetLogDB(constant.WRITE)

	// 没有 logs 表（例如全新库或尚未初始化日志库）时，无需处理
	if !db.Migrator().HasTable(&model.LogInfo{}) {
		return nil
	}

	migrator := db.Migrator()

	// 步骤 1：检测 size 列，缺失则补建
	columnCreated := false
	if !migrator.HasColumn(&model.LogInfo{}, "size") {
		if err := migrator.AddColumn(&model.LogInfo{}, "Size"); err != nil {
			return fmt.Errorf("为 logs 表补建 size 列失败: %w", err)
		}
		columnCreated = true
		logf("数据修复 - Logs表，检测到缺失 size 列，已补建")
	}

	// 步骤 2：全量重算 size
	// size = 该日志下 removed IS NULL 的条目数；log_items 表不存在时，所有日志的 size 归零。
	// 用裸 SQL（而非 gorm.Model().Update()）以绕开 GORM “无 WHERE 的批量更新”保护，
	// 这里确实需要更新全部行。该相关子查询与 008 迁移的重算口径一致，三种数据库均支持。
	var rowsAffected int64
	if migrator.HasTable(&model.LogOneItem{}) {
		res := db.Exec("UPDATE logs SET size = (SELECT COUNT(1) FROM log_items WHERE log_items.log_id = logs.id AND log_items.removed IS NULL)")
		if res.Error != nil {
			return res.Error
		}
		rowsAffected = res.RowsAffected
	} else {
		res := db.Exec("UPDATE logs SET size = 0")
		if res.Error != nil {
			return res.Error
		}
		rowsAffected = res.RowsAffected
	}

	if columnCreated {
		logf(fmt.Sprintf("数据修复 - Logs表，已补建 size 列并重算了 %d 条记录", rowsAffected))
	} else {
		logf(fmt.Sprintf("数据修复 - Logs表，size 列已存在，重算了 %d 条记录", rowsAffected))
	}
	return nil
}

var V160LogSizeRepairMigration = upgrade.Upgrade{
	ID: "010_V160LogSizeRepairMigration",
	Description: `
# 升级说明
兜底修复 V150 历史升级失误：若 logs 表缺少 size 列则补建，并对所有日志全量重算 size（该日志下未删除的条目数）。
`,
	Apply: func(logf func(string), dbOperator operator.DatabaseOperator) error {
		logf("[INFO] V160 logs.size 修复开始")
		err := V160LogSizeRepairMigrate(dbOperator, logf)
		if err != nil {
			return err
		}
		logf("[INFO] V160 logs.size 修复处置完毕")
		return nil
	},
}
