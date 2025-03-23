package service

import (
	"errors"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	engine2 "sealdice-core/utils/dboperator/engine"
)

var ErrEndpointInfoUIDEmpty = errors.New("user id is empty")

func Query(operator engine2.DatabaseOperator, e *model.EndpointInfo) error {
	db := operator.GetDataDB(constant.READ)
	if len(e.UserID) == 0 {
		return ErrEndpointInfoUIDEmpty
	}
	if db == nil {
		return errors.New("db is nil")
	}

	err := db.Model(&model.EndpointInfo{}).
		Where("user_id = ?", e.UserID).
		Select("cmd_num", "cmd_last_time", "online_time", "updated_at").
		Limit(1).
		Find(&e).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func Save(operator engine2.DatabaseOperator, e *model.EndpointInfo) error {
	db := operator.GetDataDB(constant.WRITE)
	// 检查 user_id 是否为空
	if len(e.UserID) == 0 {
		return ErrEndpointInfoUIDEmpty
	}
	// 使用 FirstOrCreate 来插入或更新
	if err := db.Where("user_id = ?", e.UserID).Assign(
		"cmd_num", e.CmdNum,
		"cmd_last_time", e.CmdLastTime,
		"online_time", e.OnlineTime,
		"updated_at", e.UpdatedAt,
	). // 奇怪的想法，映射回自身？
		FirstOrCreate(&e).Error; err != nil {
		// 处理查询或创建时的错误
		return err
	}
	return nil
}
