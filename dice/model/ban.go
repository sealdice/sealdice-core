package model

import "gorm.io/gorm"

// BanInfo 模型
// GORM STRUCT
type BanInfo struct {
	ID           string `gorm:"primaryKey"`
	BanUpdatedAt int    `gorm:"index:idx_ban_info_ban_updated_at"`
	UpdatedAt    int    `gorm:"index:idx_ban_info_updated_at"`
	Data         []byte // 使用[]byte表示BLOB类型
}

func (BanInfo) TableName() string {
	return "ban_info"
}

// BanItemDel 删除指定 ID 的禁用项
func BanItemDel(db *gorm.DB, id string) error {
	// 使用 GORM 的 Delete 方法删除指定 ID 的记录
	result := db.Where("id = ?", id).Delete(&BanInfo{})
	return result.Error // 返回错误
}

// BanItemSave 保存或替换禁用项
func BanItemSave(db *gorm.DB, id string, updatedAt int64, banUpdatedAt int64, data []byte) error {
	// 定义用于查找的条件
	conditions := map[string]any{
		"id": id,
	}

	// 使用 FirstOrCreate 查找或创建新的禁用项
	if err := db.Attrs(map[string]any{
		"ban_updated_at": int(banUpdatedAt), // 只在创建时设置的字段
	}).
		Assign(map[string]any{
			"updated_at": int(updatedAt), // 更新时覆盖的字段
			"data":       data,           // 禁用项数据
		}).FirstOrCreate(&BanInfo{}, conditions).Error; err != nil {
		return err // 返回错误
	}

	return nil // 操作成功，返回 nil
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
