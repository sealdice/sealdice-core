package service

import (
	"sealdice-core/model"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/dbutil"
	engine2 "sealdice-core/utils/dboperator/engine"
)

// BanItemDel 删除指定 ID 的禁用项
func BanItemDel(operator engine2.DatabaseOperator, id string) error {
	db := operator.GetDataDB(constant.WRITE)
	// 使用 GORM 的 Delete 方法删除指定 ID 的记录
	result := db.Where("id = ?", id).Delete(&model.BanInfo{})
	return result.Error // 返回错误
}

// BanItemSave 保存或替换禁用项 这里的[]byte也是json反序列化产物
func BanItemSave(operator engine2.DatabaseOperator, id string, updatedAt int64, banUpdatedAt int64, data []byte) error {
	db := operator.GetDataDB(constant.WRITE)
	// 使用 FirstOrCreate ，这里显然，第一次初始化的时候替换ID，而剩余的时候只换ID以外的数据
	bytePoint := dbutil.BYTE(data)
	if err := db.Where("id = ?", id).Attrs(map[string]any{
		"id":             id,
		"updated_at":     int(updatedAt),
		"ban_updated_at": int(banUpdatedAt), // 只在创建时设置的字段
		"data":           &bytePoint,        // 禁用项数据
	}).
		Assign(map[string]any{
			"updated_at":     int(updatedAt),
			"ban_updated_at": int(banUpdatedAt), // 只在创建时设置的字段
			"data":           &bytePoint,        // 禁用项数据
		}).FirstOrCreate(&model.BanInfo{}).Error; err != nil {
		return err // 返回错误
	}
	return nil // 操作成功，返回 nil
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
