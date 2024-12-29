package model

import (
	"gorm.io/gorm"
)

// BanInfo 模型
// GORM STRUCT
type BanInfo struct {
	ID           string `gorm:"primaryKey;column:id"`                                    // 主键列
	BanUpdatedAt int    `gorm:"index:idx_ban_info_ban_updated_at;column:ban_updated_at"` // BanUpdatedAt 列
	UpdatedAt    int    `gorm:"index:idx_ban_info_updated_at;column:updated_at"`         // UpdatedAt 列
	Data         []byte `gorm:"column:data"`                                             // BLOB 类型
}

func (*BanInfo) TableName() string {
	return "ban_info"
}

// BanItemDel 删除指定 ID 的禁用项
func BanItemDel(db *gorm.DB, id string) error {
	// 使用 GORM 的 Delete 方法删除指定 ID 的记录
	result := db.Where("id = ?", id).Delete(&BanInfo{})
	return result.Error // 返回错误
}

// BanItemSave 保存或替换禁用项 这里的[]byte也是json反序列化产物
func BanItemSave(db *gorm.DB, id string, updatedAt int64, banUpdatedAt int64, data []byte) error {
	// 使用 FirstOrCreate ，这里显然，第一次初始化的时候替换ID，而剩余的时候只换ID以外的数据
	if err := db.Where("id = ?", id).Attrs(map[string]any{
		"id":             id,
		"updated_at":     int(updatedAt),
		"ban_updated_at": int(banUpdatedAt), // 只在创建时设置的字段
		"data":           BYTE(data),        // 禁用项数据
	}).
		Assign(map[string]any{
			"updated_at":     int(updatedAt),
			"ban_updated_at": int(banUpdatedAt), // 只在创建时设置的字段
			"data":           BYTE(data),        // 禁用项数据
		}).FirstOrCreate(&BanInfo{}).Error; err != nil {
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
