package model

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"sealdice-core/utils"

	ds "github.com/sealdice/dicescript"
)

const (
	AttrsTypeCharacter = "character"
	AttrsTypeGroupUser = "group_user"
	AttrsTypeGroup     = "group"
	AttrsTypeUser      = "user"
)

// 注: 角色表有用sheet也有用sheets的，这里数据结构中使用sheet

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
	IsHidden  bool   `json:"isHidden" gorm:"column:is_hidden"`                           // 隐藏的卡片不出现在 pc list 中

	// 通用属性
	CreatedAt int64 `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt int64 `json:"updatedAt" gorm:"column:updated_at"`

	// 下面的属性并非数据库字段，而是用于内存中的缓存
	BindingGroupsNum int64 `json:"bindingGroupNum" gorm:"-"` // 当前绑定中群数
}

// 兼容旧版本数据库
func (AttributesItemModel) TableName() string {
	return "attrs"
}

func (m *AttributesItemModel) IsDataExists() bool {
	return m.Data != nil && len(m.Data) > 0
}

// TOOD: 下面这个表记得添加 unique 索引

// PlatformMappingModel 虚拟ID - 平台用户ID 映射表
type PlatformMappingModel struct {
	Id       string `json:"id" gorm:"column:id"`               // 虚拟ID，格式为 U:nanoid 意为 User / Uniform / Universal
	IMUserID string `json:"IMUserID" gorm:"column:im_user_id"` // IM平台的用户ID
}

func AttrsGetById(db *gorm.DB, id string) (*AttributesItemModel, error) {
	// 这里必须使用AttributesItemModel结构体，如果你定义一个只有ID属性的结构体去接收，居然能接收到值，这样就会豹错
	var item AttributesItemModel
	err := db.Table("attrs").
		Select("id, data, COALESCE(attrs_type, '') as attrs_type, binding_sheet_id, name, owner_id, sheet_type, is_hidden, created_at, updated_at").
		Where("id = ?", id).
		First(&item).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &item, nil
}

// AttrsGetBindingSheetIdByGroupId 获取当前正在绑定的ID
func AttrsGetBindingSheetIdByGroupId(db *gorm.DB, id string) (string, error) {
	// 这里必须使用AttributesItemModel结构体，如果你定义一个只有ID属性的结构体去接收，居然能接收到值，这样就会豹错
	var item AttributesItemModel
	err := db.Table("attrs").
		Select("binding_sheet_id").
		Where("id = ?", id).
		First(&item).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	return item.BindingSheetId, nil
}

func AttrsGetIdByUidAndName(db *gorm.DB, userId string, name string) (string, error) {
	// 这里必须使用AttributesItemModel结构体，如果你定义一个只有ID属性的结构体去接收，居然能接收到值，这样就会豹错
	var item AttributesItemModel
	// 使用 GORM 查询 attrs 表，选择 id 字段
	err := db.Table("attrs").
		Select("id").
		Where("owner_id = ? AND name = ?", userId, name).
		First(&item).Error
	// 如果有错误，但是错误不是找不到记录的情况下：
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	return item.Id, nil // 返回找到的 id
}

func AttrsPutById(db *gorm.DB, id string, data []byte, name, sheetType string) error {
	now := time.Now().Unix() // 获取当前时间
	// 定义用于查询的条件
	conditions := map[string]any{
		"id": id,
	}
	// 要么更新，要么创建，查询条件来源于conditions
	if err := db. // 如果是新创建的，则这些赋值为：
			Attrs(map[string]any{
			"is_hidden":        true,
			"binding_sheet_id": "",
			"created_at":       now,
		}).
		// 如果是更新的，则需要被更新的为：
		Assign(map[string]any{
			"data":       data,
			"updated_at": now,
			"name":       name,
			"sheet_type": sheetType,
		}).FirstOrCreate(&AttributesItemModel{}, conditions).Error; err != nil {
		return err // 返回错误
	}
	return nil // 操作成功，返回 nil
}

func AttrsDeleteById(db *gorm.DB, id string) error {
	// 使用 GORM 的 Delete 方法删除指定 id 的记录
	if err := db.Where("id = ?", id).Delete(&AttributesItemModel{}).Error; err != nil {
		return err // 返回错误
	}
	return nil // 操作成功，返回 nil
}

func AttrsCharGetBindingList(db *gorm.DB, id string) ([]string, error) {
	// 定义一个切片用于存储结果
	var lst []string

	// 使用 GORM 查询绑定的 id 列表
	if err := db.Table("attrs").
		Select("id").
		Where("binding_sheet_id = ?", id).
		Find(&lst).Error; err != nil {
		return nil, err // 返回错误
	}

	return lst, nil // 返回结果切片
}

func AttrsCharUnbindAll(db *gorm.DB, id string) (int64, error) {
	// 使用 GORM 更新绑定的记录，将 binding_sheet_id 设为空字符串
	result := db.Model(&AttributesItemModel{}).
		Where("binding_sheet_id = ?", id).
		Update("binding_sheet_id", "")

	if result.Error != nil {
		return 0, result.Error // 返回错误
	}
	return result.RowsAffected, nil // 返回受影响的行数
}

// AttrsNewItem 新建一个角色卡/属性容器
func AttrsNewItem(db *gorm.DB, item *AttributesItemModel) (*AttributesItemModel, error) {
	id := utils.NewID()                       // 生成新的 ID
	now := time.Now().Unix()                  // 获取当前时间
	item.CreatedAt, item.UpdatedAt = now, now // 设置创建和更新时间

	if item.Id == "" {
		item.Id = id // 如果 ID 为空，则赋值新生成的 ID
	}

	// 使用 GORM 的 Create 方法插入新记录
	if err := db.Create(item).Error; err != nil {
		return nil, err // 返回错误
	}
	return item, nil // 返回新创建的项
}

func AttrsBindCharacter(db *gorm.DB, charId string, id string) error {
	// 将新字典值转换为 JSON
	json, err := ds.NewDictVal(nil).V().ToJSON()
	if err != nil {
		return err // 返回错误
	}

	// 插入新的属性记录
	item := AttributesItemModel{
		Id:             id,
		Data:           json,
		IsHidden:       true,
		BindingSheetId: "",
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	if err := db.Create(&item).Error; err != nil {
		return err // 返回错误
	}

	// 更新指定 id 的绑定记录
	result := db.Model(&AttributesItemModel{}).
		Where("id = ?", id).
		Update("binding_sheet_id", charId)

	if result.Error != nil {
		return result.Error // 返回错误
	}

	if result.RowsAffected == 0 {
		return errors.New("群信息不存在: " + id) // 如果没有记录被更新，返回错误
	}
	return nil // 操作成功，返回 nil
}

func AttrsGetCharacterListByUserId(db *gorm.DB, userId string) ([]*AttributesItemModel, error) {
	var items []*AttributesItemModel

	// 构建子查询
	subQuery := db.Table("attrs").
		Select("count(id)").
		Where("binding_sheet_id = t1.id")

	// 主查询
	db.Table("attrs as t1").
		Select("t1.id, t1.name, t1.sheet_type, (?) as binding_count", subQuery).
		Where("t1.owner_id = ?", userId).
		Where("t1.is_hidden = ?", false).
		Find(&items)

	return items, nil // 返回角色列表
}
