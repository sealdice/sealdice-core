package service

import (
	"time"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	engine2 "sealdice-core/utils/dboperator/engine"
)

// BanScoreLogAppend 添加一条怒气值变更记录
func BanScoreLogAppend(operator engine2.DatabaseOperator, log *model.BanScoreLog) error {
	db := operator.GetDataDB(constant.WRITE)
	log.CreatedAt = time.Now().Unix()
	return db.Create(log).Error
}

// QueryBanScoreLog 分页查询参数
type QueryBanScoreLog struct {
	PageNum    int    `query:"pageNum"`    // 当前页码
	PageSize   int    `query:"pageSize"`   // 每页条数
	UserID     string `query:"userId"`     // 用户ID
	ChangeType string `query:"changeType"` // 变更类型
	IsWarning  *bool  `query:"isWarning"`  // 是否触发警告
	IsBanned   *bool  `query:"isBanned"`   // 是否触发黑名单
}

// BanScoreLogGetPage 分页查询怒气值变更日志
func BanScoreLogGetPage(operator engine2.DatabaseOperator, params QueryBanScoreLog) (int64, []model.BanScoreLog, error) {
	db := operator.GetDataDB(constant.READ)
	var total int64
	var logs []model.BanScoreLog

	// 构建查询
	query := db.Model(&model.BanScoreLog{})

	// 添加过滤条件
	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.ChangeType != "" {
		query = query.Where("change_type = ?", params.ChangeType)
	}
	if params.IsWarning != nil {
		query = query.Where("is_warning = ?", *params.IsWarning)
	}
	if params.IsBanned != nil {
		query = query.Where("is_banned = ?", *params.IsBanned)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// 分页查询
	if err := query.
		Order("created_at DESC").
		Limit(params.PageSize).
		Offset((params.PageNum - 1) * params.PageSize).
		Find(&logs).Error; err != nil {
		return 0, nil, err
	}

	return total, logs, nil
}
