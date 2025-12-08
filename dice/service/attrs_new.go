package service

import (
	"errors"
	"fmt"
	"time"

	ds "github.com/sealdice/dicescript"
	"gorm.io/gorm/clause"

	"sealdice-core/model"
	"sealdice-core/utils"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/dbutil"
	engine2 "sealdice-core/utils/dboperator/engine"
)

const (
	AttrsTypeCharacter = "character"
	AttrsTypeGroupUser = "group_user"
	AttrsTypeGroup     = "group"
	AttrsTypeUser      = "user"
)

// 注: 角色表有用sheet也有用sheets的，这里数据结构中使用sheet

// TOOD: 下面这个表记得添加 unique 索引

// PlatformMappingModel 虚拟ID - 平台用户ID 映射表
type PlatformMappingModel struct {
	Id       string `gorm:"column:id"         json:"id"`       // 虚拟ID，格式为 U:nanoid 意为 User / Uniform / Universal
	IMUserID string `gorm:"column:im_user_id" json:"IMUserID"` // IM平台的用户ID
}

func AttrsGetById(operator engine2.DatabaseOperator, id string) (*model.AttributesItemModel, error) {
	// 这里必须使用model.AttributesItemModel结构体，如果你定义一个只有ID属性的结构体去接收，居然能接收到值，这样就会豹错
	db := operator.GetDataDB(constant.READ)
	var item model.AttributesItemModel
	err := db.Model(&model.AttributesItemModel{}).
		Select("id, data, COALESCE(attrs_type, '') as attrs_type, binding_sheet_id, name, owner_id, sheet_type, is_hidden, created_at, updated_at").
		Where("id = ?", id).
		Limit(1).
		// 使用Find，如果找不到不会豹错，而是提示RowsAffected = 0，此处返回空对象本身就是预期正常的行为
		Find(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// AttrsGetBindingSheetIdByGroupId 获取当前正在绑定的ID
func AttrsGetBindingSheetIdByGroupId(operator engine2.DatabaseOperator, id string) (string, error) {
	// 这里必须使用model.AttributesItemModel结构体，如果你定义一个只有ID属性的结构体去接收，居然能接收到值，这样就会豹错
	var item model.AttributesItemModel
	db := operator.GetDataDB(constant.READ)
	err := db.Model(&model.AttributesItemModel{}).
		Select("binding_sheet_id").
		Where("id = ?", id).
		Limit(1).
		// 使用Find，如果找不到不会豹错，而是提示RowsAffected = 0，此处返回id=""就是预期正常的行为
		Find(&item).Error
	if err != nil {
		return "", err
	}
	return item.BindingSheetId, nil
}

func AttrsGetIdByUidAndName(operator engine2.DatabaseOperator, userId string, name string) (string, error) {
	// 这里必须使用model.AttributesItemModel结构体
	// 如果你定义一个只有ID属性的结构体去接收，居然有概率能接收到值，这样就会和之前的行为不一致了
	var item model.AttributesItemModel
	db := operator.GetDataDB(constant.READ)
	err := db.Model(&model.AttributesItemModel{}).
		Select("id").
		Where("owner_id = ? AND name = ?", userId, name).
		Limit(1).
		// 使用Find，如果找不到不会豹错，而是提示RowsAffected = 0，此处返回空对象的id=""就是预期正常的行为
		Find(&item).Error
	if err != nil {
		return "", err
	}
	return item.Id, nil
}

func AttrsPutById(operator engine2.DatabaseOperator, id string, data []byte, name, sheetType string) error {
	db := operator.GetDataDB(constant.WRITE)
	now := time.Now().Unix() // 获取当前时间
	// 这里的原本逻辑是：第一次全量创建，第二次修改部分属性
	// 所以使用了Attrs和Assign配合使用
	bytePoint := dbutil.BYTE(data)
	if err := db.Where("id = ?", id).
		Attrs(map[string]any{
			// 第一次全量建表
			"id": id,
			// 使用BYTE规避无法插入的问题
			"data":             &bytePoint,
			"is_hidden":        true,
			"binding_sheet_id": "",
			"name":             name,
			"sheet_type":       sheetType,
			"created_at":       now,
			"updated_at":       now,
		}).
		// 如果是更新的情况，更新下面这部分，则需要被更新的为：
		Assign(map[string]any{
			"data":       &bytePoint,
			"updated_at": now,
			"name":       name,
			"sheet_type": sheetType,
		}).FirstOrCreate(&model.AttributesItemModel{}).Error; err != nil {
		return err // 返回错误
	}
	return nil // 操作成功，返回 nil
}

type AttributesBatchUpsertModel struct {
	Id        string `json:"id"`
	Data      []byte `json:"data"`
	Name      string `json:"name"`
	SheetType string `json:"sheetType"`
}

// AttrsPutsByIDBatch 特殊入库函数
// https://github.com/go-gorm/gorm/issues/6047 混合的更新疑似仍然有问题，所以手动拆分逻辑
func AttrsPutsByIDBatch(operator engine2.DatabaseOperator, saveList []*AttributesBatchUpsertModel) error {
	writeDB := operator.GetDataDB(constant.WRITE)
	now := time.Now().Unix()

	// 构建要插入/更新的记录列表
	records := make([]*model.AttributesItemModel, 0, len(saveList))
	for _, value := range saveList {
		records = append(records, &model.AttributesItemModel{
			Id:             value.Id,
			Data:           value.Data,
			IsHidden:       true,
			BindingSheetId: "",
			Name:           value.Name,
			SheetType:      value.SheetType,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	// 使用GORM Upsert进行批量插入/更新
	err := writeDB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "name", "sheet_type", "updated_at"}),
	}).Create(&records).Error

	return err
}
func AttrsDeleteById(operator engine2.DatabaseOperator, id string) error {
	db := operator.GetDataDB(constant.WRITE)
	// 使用 GORM 的 Delete 方法删除指定 id 的记录
	if err := db.Where("id = ?", id).Delete(&model.AttributesItemModel{}).Error; err != nil {
		return err // 返回错误
	}
	return nil // 操作成功，返回 nil
}

func AttrsCharGetBindingList(operator engine2.DatabaseOperator, id string) ([]string, error) {
	db := operator.GetDataDB(constant.READ)
	// 定义一个切片用于存储结果
	var lst []string

	// 使用 GORM 查询绑定的 id 列表
	if err := db.Model(&model.AttributesItemModel{}).
		Select("id").
		Where("binding_sheet_id = ?", id).
		Find(&lst).Error; err != nil {
		return nil, err // 返回错误
	}

	return lst, nil // 返回结果切片
}

func AttrsCharUnbindAll(operator engine2.DatabaseOperator, id string) (int64, error) {
	db := operator.GetDataDB(constant.WRITE)
	// 使用 GORM 更新绑定的记录，将 binding_sheet_id 设为空字符串
	result := db.Model(&model.AttributesItemModel{}).
		Where("binding_sheet_id = ?", id).
		Update("binding_sheet_id", "")

	if result.Error != nil {
		return 0, result.Error // 返回错误
	}
	return result.RowsAffected, nil // 返回受影响的行数
}

// AttrsNewItem 新建一个角色卡/属性容器
func AttrsNewItem(operator engine2.DatabaseOperator, item *model.AttributesItemModel) (*model.AttributesItemModel, error) {
	db := operator.GetDataDB(constant.WRITE)
	id := utils.NewID()                       // 生成新的 ID
	now := time.Now().Unix()                  // 获取当前时间
	item.CreatedAt, item.UpdatedAt = now, now // 设置创建和更新时间

	if item.Id == "" {
		item.Id = id // 如果 ID 为空，则赋值新生成的 ID
	}

	// 使用 GORM 的 Create 方法插入新记录
	// 这个木落没有忽略错误，所以说这个可以安心使用Create而不用担心出现问题……
	// 这里使用Create可以正确插入byte数组，注意map[string]any里面不可以用byte数组，否则无法入库
	if err := db.Create(item).Error; err != nil {
		return nil, err // 返回错误
	}
	return item, nil // 返回新创建的项
}

func AttrsBindCharacter(operator engine2.DatabaseOperator, charId string, id string) error {
	db := operator.GetDataDB(constant.WRITE)
	// 将新字典值转换为 JSON
	now := time.Now().Unix()
	jsonBytes, err := ds.NewDictVal(nil).V().ToJSON()
	if err != nil {
		return err
	}
	// 原本代码为：
	//	_, _ = db.Exec(`insert into attrs (id, data, is_hidden, binding_sheet_id, created_at, updated_at)
	//					       values ($1, $3, true, '', $2, $2)`, id, time.Now().Unix(), json)
	//
	//	ret, err := db.Exec(`update attrs set binding_sheet_id = $1 where id = $2`, charId, id)
	bytePoint := dbutil.BYTE(jsonBytes)
	result := db.Where("id = ?", id).
		// 按照木落的原版代码，应该是这么个逻辑：查不到的时候能正确执行，查到了就不执行了，所以用Attrs而不是Assign
		Attrs(map[string]any{
			"id": id,
			// 如果想在[]bytes里输入值，注意传参的时候不能给any传[]bytes，否则会无法读取，同时还没有豹错，浪费大量时间。
			// 这里为了兼容，不使用gob的序列化方法处理结构体（同时，也不知道序列化方法是否可用）
			"data":      &bytePoint,
			"is_hidden": true,
			// 如果插入成功，原版代码接下来更新这个值，那么现在就是等价的
			"binding_sheet_id": charId,
			"created_at":       now,
			"updated_at":       now,
		}).
		Assign(map[string]any{
			"binding_sheet_id": charId,
		}).
		FirstOrCreate(&model.AttributesItemModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("群信息不存在或发生更新异常: " + id)
	}
	return nil
}

func AttrsGetCharacterListByUserId(operator engine2.DatabaseOperator, userId string) ([]*model.AttributesItemModel, error) {
	db := operator.GetDataDB(constant.READ)
	// Pinenutn: 在Gorm中，如果gorm:"-"，优先级似乎很高，经过我自己测试：
	// 结构体内若使用gorm="-" ，Scan将无法映射到结果中（GPT胡说八道说可以映射上，我试了半天，被骗。）
	// 如果不带任何标签: GORM对结构体名称进行转换，如BindingGroupNum对应映射:binding_group_num，结果里有binding_group_num自动映射
	// 如果带上标签"column:xxxxx"，则会使用指定的名称映射，如column:xxxxx对应映射xxxxx
	// GPT 说带上JSON标签，可以映射到结果中，但实际上是错误的，无法映射。
	// 所以最终”BindingGroupNum“需要创建这个结构体用来临时存放结果，然后将结果映射到model.AttributesItemModel结构体上。
	// 在gorm="-"这里的配置还有更多可以使用无写入权限，有读权限的标签，但要求必须BindingGroupNum的结构体名称和数据库查询结果一致
	// 且不能指定columns，否则会建表，没找到更好方案。
	type AttrResult struct {
		ID              string `gorm:"column:id"`
		Name            string `gorm:"column:name"`
		SheetType       string `gorm:"column:sheet_type"`
		BindingGroupNum int64  `gorm:"column:binding_group_num"` // 映射 COUNT(a.id)
	}
	var tempResultList []AttrResult
	// 由于是复杂查询，无法直接使用Models，又为了防止以后attrs表名称修改，故不使用Table而是用TableName替换
	modelInstance := model.AttributesItemModel{}
	tableName := modelInstance.TableName()
	// 此处使用了JOIN来避免子查询，数据库一般对JOIN有使用索引的优化，所以有性能提升，但是我没有实际测试过性能差距。
	err := db.Table(fmt.Sprintf("%s AS t1", tableName)).
		Select("t1.id, t1.name, t1.sheet_type, COUNT(a.id) AS binding_group_num").
		Joins(fmt.Sprintf("LEFT JOIN %s AS a ON a.binding_sheet_id = t1.id", tableName)).
		Where("t1.owner_id = ? AND t1.is_hidden IS FALSE", userId).
		Group("t1.id, t1.name, t1.sheet_type").
		// Pinenutn：此处我根据创建时间对创建的卡进行排序，不知道是否有意义？
		Order("t1.created_at ASC").
		Scan(&tempResultList).Error
	if err != nil {
		return nil, err
	}
	items := make([]*model.AttributesItemModel, len(tempResultList))
	for i, tempResult := range tempResultList {
		items[i] = &model.AttributesItemModel{
			Id:               tempResult.ID,
			Name:             tempResult.Name,
			SheetType:        tempResult.SheetType,
			BindingGroupsNum: tempResult.BindingGroupNum,
		}
	}
	return items, nil // 返回角色列表
}
