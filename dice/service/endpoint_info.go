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
	// 检查 UserID 是否为空
	if len(e.UserID) == 0 {
		return ErrEndpointInfoUIDEmpty
	}
	// 直接使用 Save() 方法（自动判断插入或更新）
	if err := db.Save(e).Error; err != nil {
		return err
	}
	return nil
}
