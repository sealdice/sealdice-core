package service

import (
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
	engine2 "sealdice-core/utils/dboperator/engine"
)

// GroupInfoListGet 使用 GORM 实现，遍历 group_info 表中的数据并调用回调函数
func GroupInfoListGet(operator engine2.DatabaseOperator, callback func(id string, updatedAt int64, data []byte)) error {
	db := operator.GetDataDB(constant.READ)
	// 创建一个保存查询结果的结构体
	var results []struct {
		ID        string `gorm:"column:id"`         // 字段 id
		UpdatedAt *int64 `gorm:"column:updated_at"` // 由于可能存在 NULL，定义为指针类型
		Data      []byte `gorm:"column:data"`       // 字段 data
	}

	// 使用 GORM 查询 group_info 表中的 id, updated_at, data 列
	err := db.Model(&model.GroupInfo{}).Select("id, updated_at, data").Find(&results).Error
	if err != nil {
		// 如果查询发生错误，返回错误信息
		return err
	}

	// 遍历查询结果
	for _, result := range results {
		var updatedAt int64

		// 如果 updatedAt 是 NULL，需要跳过该字段
		if result.UpdatedAt != nil {
			updatedAt = *result.UpdatedAt
		}

		// 调用回调函数，传递 id, updatedAt, data
		callback(result.ID, updatedAt, result.Data)
	}

	// 返回 nil 表示操作成功
	return nil
}

// GroupInfoSave 保存群组信息
func GroupInfoSave(operator engine2.DatabaseOperator, groupID string, updatedAt int64, data []byte) error {
	// 使用写数据库
	db := operator.GetDataDB(constant.WRITE)
	// 使用 gorm 的 Upsert 功能实现插入或更新
	groupInfo := model.GroupInfo{
		ID:        groupID,
		UpdatedAt: &updatedAt,
		Data:      data,
	}
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at", "data"}),
	}).Create(&groupInfo)
	return result.Error
}

// GroupPlayerNumGet 获取指定群组的玩家数量
func GroupPlayerNumGet(operator engine2.DatabaseOperator, groupID string) (int64, error) {
	// 使用读数据库
	db := operator.GetDataDB(constant.READ)
	var count int64

	// 使用 GORM 的 Table 方法指定表名进行查询
	// db.Table("表名").Where("条件").Count(&count) 是通用的 GORM 用法
	// 将 group_id 作为查询条件
	err := db.Model(&model.GroupPlayerInfoBase{}).Where("group_id = ?", groupID).Count(&count).Error
	if err != nil {
		// 如果查询出现错误，返回错误信息
		return 0, err
	}

	// 返回统计的数量
	return count, nil
}

// GroupPlayerInfoGet 获取指定群组中的玩家信息
func GroupPlayerInfoGet(operator engine2.DatabaseOperator, groupID string, playerID string) *model.GroupPlayerInfoBase {
	// 使用读数据库
	db := operator.GetDataDB(constant.READ)
	var ret model.GroupPlayerInfoBase

	// 使用 GORM 查询数据并绑定到结构体中
	// db.Table("表名").Where("条件").First(&ret) 查询一条数据并映射到结构体
	result := db.Model(&model.GroupPlayerInfoBase{}).
		Where("group_id = ? AND user_id = ?", groupID, playerID).
		Select("name, last_command_time, auto_set_name_template, dice_side_num").
		Limit(1).
		Find(&ret)
	err := result.Error
	// 如果查询发生错误，打印错误并返回 nil
	if err != nil {
		zap.S().Named(logger.LogKeyDatabase).Errorf("error getting group player info: %s", err.Error())
		return nil
	}

	if result.RowsAffected == 0 {
		return nil
	}

	// 将 playerID 赋值给结构体中的 UserID 字段
	ret.UserID = playerID

	// 返回查询结果
	return &ret
}

func GroupPlayerInfoSave(operator engine2.DatabaseOperator, info *model.GroupPlayerInfoBase) error {
	db := operator.GetDataDB(constant.WRITE)
	// 1. 先查询记录是否存在
	var existing model.GroupPlayerInfoBase
	err := db.Where("user_id = ? AND group_id = ?", info.UserID, info.GroupID).First(&existing).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 记录不存在 → 插入新记录
			return db.Create(info).Error
		}
		return err
	}
	// 2. 记录存在 → 更新指定字段
	updates := map[string]interface{}{
		"name":                   info.Name,
		"last_command_time":      info.LastCommandTime,
		"auto_set_name_template": info.AutoSetNameTemplate,
		"dice_side_num":          info.DiceSideNum, // 0值也会被更新
		"updated_at":             info.UpdatedAt,
	}
	return db.Model(&model.GroupPlayerInfoBase{}).
		Where("user_id = ? AND group_id = ?", info.UserID, info.GroupID).
		Updates(updates).Error
}
