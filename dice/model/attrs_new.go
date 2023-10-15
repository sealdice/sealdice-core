package model

import "github.com/jmoiron/sqlx"

// 新版人物卡
type AttributesItemModel struct {
	Id   string `json:"id" db:"id"`     // 如果是群内，那么是类似 QQ-Group:12345-QQ:678910，群外是nanoid
	Data string `json:"data" db:"data"` // 序列化后的卡数据

	// 这些是群组内置卡专用的，其实就是少创建了一个绑卡关系表
	BindingSheetId string `json:"bindingSheetId" db:"binding_sheet_id"` // 绑定的卡片ID

	// 这些是角色卡专用的
	Nickname  string `json:"nickname" db:"nickname"`    // 卡片名称
	OwnerId   string `json:"ownerId" db:"owner_id"`     // 若有明确归属，就是对应的UniformID
	SheetType string `json:"sheetType" db:"sheet_type"` // 卡片类型，如dnd5e coc7
	IsHidden  bool   `json:"isHidden" db:"is_hidden"`   // 隐藏的卡片不出现在 pc list 中

	// 通用属性
	CreatedAt int64 `json:"createdAt" db:"created_at"`
	UpdatedAt int64 `json:"updatedAt" db:"updated_at"`
}

// TOOD: 下面这个表记得添加 unique 索引

// PlatformMappingModel 虚拟ID - 平台用户ID 映射表
type PlatformMappingModel struct {
	Id       string `json:"id" db:"id"`               // 虚拟ID，格式为 U:nanoid 意为 User / Uniform / Universal
	IMUserID string `json:"IMUserID" db:"im_user_id"` // IM平台的用户ID
}

func AttrsGetById(db *sqlx.DB, id string) (*AttributesItemModel, error) {
	var item AttributesItemModel
	err := db.Get(&item, "select * from attrs where id = $1", id)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func AttrsGetBindingSheetId(db *sqlx.DB, id string) (string, error) {
	var item AttributesItemModel
	err := db.Get(&item, "select binding_sheet_id from attrs where id = $1", id)
	if err != nil {
		return "", err
	}
	return item.BindingSheetId, nil
}
