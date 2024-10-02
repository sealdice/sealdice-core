package model

import (
	"errors"

	"gorm.io/gorm"
)

var ErrEndpointInfoUIDEmpty = errors.New("user id is empty")

// 仅修改为gorm格式
type EndpointInfo struct {
	UserID      string `gorm:"column:user_id;primaryKey"`
	CmdNum      int64  `gorm:"column:cmd_num;"`
	CmdLastTime int64  `gorm:"column:cmd_last_time;"`
	OnlineTime  int64  `gorm:"column:online_time;"`
	UpdatedAt   int64  `gorm:"column:updated_at;"`
}

func (EndpointInfo) TableName() string {
	return "endpoint_info"
}

func (e *EndpointInfo) Query(db *gorm.DB) error {
	if len(e.UserID) == 0 {
		return ErrEndpointInfoUIDEmpty
	}
	if db == nil {
		return errors.New("db is nil")
	}

	err := db.Table("endpoint_info").
		Where("user_id = ?", e.UserID).
		Select("cmd_num", "cmd_last_time", "online_time", "updated_at").
		Scan(&e).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func (e *EndpointInfo) Save(db *gorm.DB) error {
	// 检查 user_id 是否为空
	if len(e.UserID) == 0 {
		return ErrEndpointInfoUIDEmpty
	}

	// 检查数据库连接是否为 nil
	if db == nil {
		return errors.New("db is nil")
	}

	// 直接使用 Save 函数
	// Save 会根据主键（user_id）进行插入或更新
	err := db.Table("endpoint_info").Save(&e).Error
	if err != nil {
		return err
	}

	return nil
}
