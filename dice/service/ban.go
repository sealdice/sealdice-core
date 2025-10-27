package service

import (
	"errors"

	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	engine2 "sealdice-core/utils/dboperator/engine"
)

// BanItemDel 删除指定 ID 的禁用项
func BanItemDel(operator engine2.DatabaseOperator, id string) error {
	db := operator.GetDataDB(constant.WRITE)
	// 使用 GORM 的 Delete 方法删除指定 ID 的记录
	result := db.Where("id = ?", id).Delete(&model.BanInfo{})
	return result.Error // 返回错误
}

func BanItemSave(operator engine2.DatabaseOperator, id string, updatedAt int64, banUpdatedAt int64, data []byte) error {
	db := operator.GetDataDB(constant.WRITE)
	readDB := operator.GetDataDB(constant.READ)

	var banInfo model.BanInfo

	// 先尝试查找记录
	result := readDB.Where("id = ?", id).First(&banInfo)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 记录不存在，创建新记录
			newBanInfo := model.BanInfo{
				ID:           id,
				UpdatedAt:    int(updatedAt),
				BanUpdatedAt: int(banUpdatedAt),
				Data:         data,
			}
			if err := db.Create(&newBanInfo).Error; err != nil {
				return err
			}
		} else {
			// 其他查询错误
			return result.Error
		}
	} else {
		// 记录存在，更新记录
		updates := map[string]interface{}{
			"updated_at":     updatedAt,
			"ban_updated_at": banUpdatedAt,
			"data":           data,
		}
		if err := db.Model(&model.BanInfo{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

// BanItemList 列出所有禁用项并调用回调函数处理
func BanItemList(operator engine2.DatabaseOperator, callback func(id string, banUpdatedAt int64, data []byte)) error {
	var items []model.BanInfo
	db := operator.GetDataDB(constant.READ)
	// 使用 GORM 查询所有禁用项
	if err := db.Order("ban_updated_at DESC").Find(&items).Error; err != nil {
		return err // 返回错误
	}

	// 遍历每个禁用项并调用回调函数
	for _, item := range items {
		callback(item.ID, int64(item.BanUpdatedAt), item.Data) // 确保类型一致
	}
	return nil // 操作成功，返回 nil
}
