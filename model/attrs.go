package model

// AttributesItemModel 新版人物卡。说明一下，这里带s的原因是attrs指的是一个map
// 补全GORM缺少部分
type AttributesItemModel struct {
	Id        string `json:"id" gorm:"column:id"`                                                           // 如果是群内，那么是类似 QQ-Group:12345-QQ:678910，群外是nanoid
	Data      []byte `json:"data" gorm:"column:data"`                                                       // 序列化后的卡数据，理论上[]byte不会进入字符串缓存，要更好些？
	AttrsType string `json:"attrsType" gorm:"column:attrs_type;index:idx_attrs_attrs_type_id;default:NULL"` // 分为: 角色卡(character)、组内用户(group_user)、群组(group)、用户(user)

	// 这些是群组内置卡专用的，其实就是替代了绑卡关系表，作为群组内置卡时，这个字段用于存放绑卡关系
	BindingSheetId string `json:"bindingSheetId" gorm:"column:binding_sheet_id;default:'';index:idx_attrs_binding_sheet_id"` // 绑定的卡片ID

	// 这些是角色卡专用的
	Name      string `json:"name" gorm:"column:name"`                                    // 卡片名称
	OwnerId   string `json:"ownerId" gorm:"column:owner_id;index:idx_attrs_owner_id_id"` // 若有明确归属，就是对应的UniformID
	SheetType string `json:"sheetType" gorm:"column:sheet_type"`                         // 卡片类型，如dnd5e coc7
	// 手动定义bool类的豹存方式
	IsHidden bool `json:"isHidden" gorm:"column:is_hidden;type:bool"` // 隐藏的卡片不出现在 pc list 中

	// 通用属性
	CreatedAt int64 `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt int64 `json:"updatedAt" gorm:"column:updated_at"`

	// 下面的属性并非数据库字段，而是用于内存中的缓存
	BindingGroupsNum int64 `json:"bindingGroupNum" gorm:"-"` // 当前绑定中群数
}

// 兼容旧版本数据库
func (*AttributesItemModel) TableName() string {
	return "attrs"
}

func (m *AttributesItemModel) IsDataExists() bool {
	return len(m.Data) > 0
}
