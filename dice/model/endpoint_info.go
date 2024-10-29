package model

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	err := db.Model(&EndpointInfo{}).
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
	// 检查user_id冲突时更新，否则进行创建
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"cmd_num", "cmd_last_time", "online_time", "updated_at",
		}),
	}).Create(e)

	return result.Error
}
