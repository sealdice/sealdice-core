package model

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
