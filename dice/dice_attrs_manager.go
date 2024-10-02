package dice

import (
	"errors"
	"fmt"
	"time"

	ds "github.com/sealdice/dicescript"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/dice/model"
)

type AttrsManager struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
	m      SyncMap[string, *AttributesItem]
}

// LoadByCtx 获取当前角色，如有绑定，则获取绑定的角色，若无绑定，获取群内默认卡
func (am *AttrsManager) LoadByCtx(ctx *MsgContext) (*AttributesItem, error) {
	return am.Load(ctx.Group.GroupID, ctx.Player.UserID)
	// if ctx.AttrsCurCache == nil {
	// 	var err error
	// 	ctx.AttrsCurC
	//	ache, err = am.Load(ctx.Group.GroupID, ctx.Player.UserID)
	// 	return ctx.AttrsCurCache, err
	// }
	// return ctx.AttrsCurCache, nil
}

func (am *AttrsManager) Load(groupId string, userId string) (*AttributesItem, error) {
	userId = am.UIDConvert(userId)

	// 组装当前群-用户的id
	gid := fmt.Sprintf("%s-%s", groupId, userId)

	//	1. 首先获取当前群+用户所绑定的卡
	// 绑定卡的id是nanoid
	id, err := model.AttrsGetBindingSheetIdByGroupId(am.db, gid)
	if err != nil {
		return nil, err
	}

	// 2. 如果取不到，那么取用户在当前群的默认卡
	if id == "" {
		id = gid
	}

	return am.LoadById(id)
}

func (am *AttrsManager) UIDConvert(userId string) string {
	// 如果存在一个虚拟id，那么返回虚拟id，不存在原样返回
	return userId
}

func (am *AttrsManager) GetCharacterList(userId string) ([]*model.AttributesItemModel, error) {
	userId = am.UIDConvert(userId)
	lst, err := model.AttrsGetCharacterListByUserId(am.db, userId)
	if err != nil {
		return nil, err
	}
	return lst, err
}

func (am *AttrsManager) CharNew(userId string, name string, sheetType string) (*model.AttributesItemModel, error) {
	userId = am.UIDConvert(userId)
	dict := &ds.ValueMap{}
	// dict.Store("$sheetType", ds.NewStrVal(sheetType))
	json, err := ds.NewDictVal(dict).V().ToJSON()
	if err != nil {
		return nil, err
	}

	return model.AttrsNewItem(am.db, &model.AttributesItemModel{
		Name:      name,
		OwnerId:   userId,
		AttrsType: "character",
		SheetType: sheetType,
		Data:      json,
	})
}

func (am *AttrsManager) CharDelete(id string) error {
	if err := model.AttrsDeleteById(am.db, id); err != nil {
		return err
	}
	// 从缓存中删除
	am.m.Delete(id)
	return nil
}

// LoadById 数据加载，负责以下数据
// 1. 群内用户的默认卡(id格式为：群id:用户id)
// 2. 用户创建出的角色卡（指定id）
// 3. 群属性(id为群id)
// 4. 用户全局属性
func (am *AttrsManager) LoadById(id string) (*AttributesItem, error) {
	// 1. 如果当前有缓存，那么从缓存中返回。
	// 但是。。如果有人把这个对象一直持有呢？
	i, exists := am.m.Load(id)
	if exists {
		return i, nil
	}

	// 2. 从新数据库加载
	data, err := model.AttrsGetById(am.db, id)
	if err == nil {
		if data.IsDataExists() {
			var v *ds.VMValue
			v, err = ds.VMValueFromJSON(data.Data)
			if err != nil {
				return nil, err
			}
			if dd, ok := v.ReadDictData(); ok {
				i = &AttributesItem{
					ID:           id,
					valueMap:     dd.Dict,
					Name:         data.Name,
					SheetType:    data.SheetType,
					LastUsedTime: time.Now().Unix(),
				}
				am.m.Store(id, i)
				return i, nil
			} else {
				return nil, errors.New("角色数据类型不正确")
			}
		}
	}
	// 之前 else 是读不出时返回报错
	// return nil, err
	// 改为创建新数据集。因为遇到一个特别案例：因为clr前会读取当前角色数据，因为读不出来所以无法st clr
	// 从而永久卡死

	// 3. 创建一个新的
	// 注: 缺 created_at、updated_at、sheet_type、owner_id、is_hidden、nickname等各项
	// 可能需要ctx了
	i = &AttributesItem{
		ID:       id,
		valueMap: &ds.ValueMap{},
	}
	am.m.Store(id, i)
	return i, nil
}

func (am *AttrsManager) Init(d *Dice) {
	am.db = d.DBData
	am.logger = d.Logger
	go func() {
		// NOTE(Xiangze Li): 这种不退出的goroutine不利于平稳结束程序
		for {
			am.CheckForSave()
			am.CheckAndFreeUnused()
			time.Sleep(15 * time.Second)
		}
	}()
}

func (am *AttrsManager) CheckForSave() (int, int) {
	times := 0
	saved := 0

	db := am.db
	if db == nil {
		// 尚未初始化
		return 0, 0
	}

	tx := db.Begin()

	am.m.Range(func(key string, value *AttributesItem) bool {
		if !value.IsSaved {
			saved += 1
			value.SaveToDB(db)
		}
		times += 1
		return true
	})

	err := tx.Commit().Error
	if err != nil {
		if am.logger != nil {
			am.logger.Errorf("定期写入用户数据出错(提交事务): %v", err)
		}
		_ = tx.Rollback()
		return times, 0
	}
	return times, saved
}

// CheckAndFreeUnused 此函数会被定期调用，释放最近不用的对象
func (am *AttrsManager) CheckAndFreeUnused() {
	db := am.db
	if db == nil {
		// 尚未初始化
		return
	}

	prepareToFree := map[string]int{}
	currentTime := time.Now().Unix()
	am.m.Range(func(key string, value *AttributesItem) bool {
		if value.LastUsedTime-currentTime > 60*10 {
			prepareToFree[key] = 1
			// 直接保存
			value.SaveToDB(am.db)
		}
		return true
	})

	for key := range prepareToFree {
		am.m.Delete(key)
	}
}

func (am *AttrsManager) CharBind(charId string, groupId string, userId string) error {
	userId = am.UIDConvert(userId)
	id := fmt.Sprintf("%s-%s", groupId, userId)
	return model.AttrsBindCharacter(am.db, charId, id)
}

// CharGetBindingId 获取当前群绑定的角色ID
func (am *AttrsManager) CharGetBindingId(groupId string, userId string) (string, error) {
	userId = am.UIDConvert(userId)
	id := fmt.Sprintf("%s-%s", groupId, userId)
	return model.AttrsGetBindingSheetIdByGroupId(am.db, id)
}

func (am *AttrsManager) CharIdGetByName(userId string, name string) (string, error) {
	return model.AttrsGetIdByUidAndName(am.db, userId, name)
}

func (am *AttrsManager) CharCheckExists(userId string, name string) bool {
	id, _ := am.CharIdGetByName(userId, name)
	return id != ""
}

func (am *AttrsManager) CharGetBindingGroupIdList(id string) []string {
	all, err := model.AttrsCharGetBindingList(am.db, id)
	if err != nil {
		return []string{}
	}
	// 只要群号
	for i, v := range all {
		a, b, _ := UnpackGroupUserId(v)
		if b != "" {
			all[i] = a
		} else {
			all[i] = b
		}
	}
	return all
}

func (am *AttrsManager) CharUnbindAll(id string) []string {
	all := am.CharGetBindingGroupIdList(id)
	_, err := model.AttrsCharUnbindAll(am.db, id)
	if err != nil {
		return []string{}
	}
	return all
}

// AttributesItem 这是一个人物卡对象
type AttributesItem struct {
	ID               string
	valueMap         *ds.ValueMap // SyncMap[string, *ds.VMValue] 这种类型更好吗？我得确认一下js兼容性
	LastModifiedTime int64        // 上次修改时间
	LastUsedTime     int64        // 上次使用时间
	IsSaved          bool
	Name             string
	SheetType        string
}

func (i *AttributesItem) SaveToDB(db *gorm.DB) {
	// 使用事务写入
	rawData, err := ds.NewDictVal(i.valueMap).V().ToJSON()
	if err != nil {
		return
	}
	err = model.AttrsPutById(db, i.ID, rawData, i.Name, i.SheetType)
	if err != nil {
		fmt.Println("保存数据失败", err.Error())
		return
	}
	i.IsSaved = true
}

func (i *AttributesItem) Load(name string) *ds.VMValue {
	v, _ := i.valueMap.Load(name)
	i.LastUsedTime = time.Now().Unix()
	return v
}

func (i *AttributesItem) LoadX(name string) (*ds.VMValue, bool) {
	v, exists := i.valueMap.Load(name)
	i.LastUsedTime = time.Now().Unix()
	return v, exists
}

func (i *AttributesItem) Delete(name string) {
	i.valueMap.Delete(name)
	i.LastModifiedTime = time.Now().Unix()
}

func (i *AttributesItem) SetModified() {
	i.LastModifiedTime = time.Now().Unix()
	i.IsSaved = false
}

func (i *AttributesItem) Store(name string, value *ds.VMValue) {
	now := time.Now().Unix()
	i.valueMap.Store(name, value)
	i.LastModifiedTime = now
	i.LastUsedTime = now
	i.IsSaved = false
}

func (i *AttributesItem) Clear() int {
	size := i.valueMap.Length()
	i.valueMap.Clear()
	i.LastModifiedTime = time.Now().Unix()
	i.IsSaved = false
	return size
}

func (i *AttributesItem) ToArrayKeys() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(items, ds.NewStrVal(key))
		return true
	})
	return items
}

func (i *AttributesItem) ToArrayValues() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(items, value)
		return true
	})
	return items
}

func (i *AttributesItem) ToArrayItems() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(
			items,
			ds.NewArrayVal(ds.NewStrVal(key), value),
		)
		return true
	})
	return items
}

func (i *AttributesItem) Range(f func(key string, value *ds.VMValue) bool) {
	i.valueMap.Range(f)
}

func (i *AttributesItem) SetSheetType(system string) {
	i.SheetType = system
	i.LastModifiedTime = time.Now().Unix()
}

func (i *AttributesItem) Len() int {
	return i.valueMap.Length()
}
