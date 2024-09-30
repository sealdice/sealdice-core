package model

import "gorm.io/gorm"

// BanItemDel 删除指定 ID 的禁用项
func BanItemDel(db *gorm.DB, id string) error {
	// 使用 GORM 的 Delete 方法删除指定 ID 的记录
	result := db.Where("id = ?", id).Delete(&BanInfo{})
	return result.Error // 返回错误
}

// BanItemSave 保存或替换禁用项
func BanItemSave(db *gorm.DB, id string, updatedAt int64, banUpdatedAt int64, data []byte) error {
	// 定义一个新的禁用项
	item := BanInfo{
		ID:           id,
		UpdatedAt:    int(updatedAt),    // 确保类型一致
		BanUpdatedAt: int(banUpdatedAt), // 确保类型一致
		Data:         data,
	}

	// 使用 GORM 的 Save 方法保存或替换记录
	return db.Save(&item).Error // 返回错误
}

// BanItemList 列出所有禁用项并调用回调函数处理
func BanItemList(db *gorm.DB, callback func(id string, banUpdatedAt int64, data []byte)) error {
	var items []BanInfo

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
