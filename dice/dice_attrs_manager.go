package dice

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"sealdice-core/dice/model"
	"time"

	ds "github.com/sealdice/dicescript"
)

type AttrsManager struct {
	parent *Dice
	m      SyncMap[string, *AttributesItem]
}

func (am *AttrsManager) Load(groupId string, userId string) (*AttributesItem, error) {
	userId = am.UIDConvert(userId)

	// 组装当前群-用户的id
	gid := fmt.Sprintf("%s-%s", groupId, userId)

	//	1. 首先获取当前群+用户所绑定的卡
	// 绑定卡的id是nanoid
	id, err := model.AttrsGetBindingSheetIdByGroupId(am.parent.DBData, gid)
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
	lst, err := model.AttrsGetCharacterListByUserId(am.parent.DBData, userId)
	if err != nil {
		return nil, err
	}
	return lst, err
}

func (am *AttrsManager) CharNew(userId string, name string, sheetType string) (*model.AttributesItemModel, error) {
	userId = am.UIDConvert(userId)
	dict := &ds.ValueMap{}
	dict.Store("$sheetType", ds.VMValueNewStr(sheetType))
	json, err := ds.VMValueNewDict(dict).V().ToJSON()
	if err != nil {
		return nil, err
	}

	return model.AttrsNewItem(am.parent.DBData, &model.AttributesItemModel{
		Nickname:  name,
		OwnerId:   userId,
		SheetType: sheetType,
		Data:      json,
	})
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
	d := am.parent
	data, err := model.AttrsGetById(d.DBData, id)
	if err == nil {
		if data.IsDataExists() {
			v, err := ds.VMValueFromJSON(data.Data)
			if err != nil {
				return nil, err
			}
			if dd, ok := v.ReadDictData(); ok {
				i = &AttributesItem{
					ID:           id,
					valueMap:     dd.Dict,
					NickName:     data.Nickname,
					LastUsedTime: time.Now().Unix(),
				}
				am.m.Store(id, i)
				return i, nil
			} else {
				return nil, errors.New("角色数据类型不正确")
			}
		}
	} else {
		// 啊？表读不了？
		return nil, err
	}

	// 3. 从老数据库读取 - 群用户数据
	// （其实还有一种，但是读不了，就是玩家的卡数据，因为没有id）
	// 暂时先不弄这种了，太容易出问题
	//dataOld := model.AttrGroupUserGetAllBase(d.DBData, id)
	//if dataOld != nil {
	//	mapData := make(map[string]*VMValue)
	//	err := JsonValueMapUnmarshal(dataOld, &mapData)
	//	if err != nil {
	//		d.Logger.Errorf("读取玩家数据失败！错误 %v 原数据 %v", err, data)
	//	}
	//
	//	m := &ds.ValueMap{}
	//	for k, v := range mapData {
	//		m.Store(k, v.ConvertToDiceScriptValue())
	//	}
	//
	//	now := time.Now().Unix()
	//	i = &AttributesItem{
	//		ID:               id,
	//		valueMap:         m,
	//		LastUsedTime:     now,
	//		LastModifiedTime: now,
	//	}
	//	am.m.Store(id, i)
	//	return i, nil
	//}

	// 4. 创建一个新的
	// 注: 缺 created_at、updated_at、sheet_type、owner_id、is_hidden、nickname等各项
	// 可能需要ctx了
	i = &AttributesItem{
		ID:       id,
		valueMap: &ds.ValueMap{},
	}
	am.m.Store(id, i)
	return i, nil
}

func (am *AttrsManager) Init() {
	go func() {
		for {
			am.CheckForSave()
			am.CheckAndFreeUnused()
			time.Sleep(30 * time.Second)
		}
	}()
}

func (am *AttrsManager) CheckForSave() (int, int) {
	times := 0
	saved := 0

	dice := am.parent
	db := am.parent.DBData
	if db == nil {
		// 尚未初始化
		return 0, 0
	}

	tx, err := db.Begin()
	if err != nil {
		dice.Logger.Errorf("定期写入用户数据出错(创建事务): %v", err)
		return 0, 0
	}

	am.m.Range(func(key string, value *AttributesItem) bool {
		if !value.IsSaved {
			saved += 1
			value.SaveToDB(db, tx)
		}
		times += 1
		return true
	})

	err = tx.Commit()
	if err != nil {
		dice.Logger.Errorf("定期写入用户数据出错(提交事务): %v", err)
		_ = tx.Rollback()
		return times, 0
	}
	return times, saved
}

// CheckAndFreeUnused 此函数会被定期调用，释放最近不用的对象
func (am *AttrsManager) CheckAndFreeUnused() {
	db := am.parent.DBData
	if db == nil {
		// 尚未初始化
		return
	}

	prepareToFree := map[string]int{}
	currentTime := time.Now().Unix()
	am.m.Range(func(key string, value *AttributesItem) bool {
		if value.LastUsedTime-currentTime > 60*10 {
			prepareToFree[key] = 1
			value.SaveToDB(am.parent.DBData, nil)
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
	return model.AttrsBindCharacter(am.parent.DBData, charId, id)
}

// CharGetBindingId 获取当前群绑定的角色ID
func (am *AttrsManager) CharGetBindingId(groupId string, userId string) (string, error) {
	userId = am.UIDConvert(userId)
	id := fmt.Sprintf("%s-%s", groupId, userId)
	return model.AttrsGetBindingSheetIdByGroupId(am.parent.DBData, id)
}

func (am *AttrsManager) CharIdGetByName(userId string, name string) (string, error) {
	return model.AttrsGetIdByUidAndName(am.parent.DBData, userId, name)
}

func (am *AttrsManager) CharCheckExists(name string, groupId string) bool {
	// TODO: xxxx
	//model.AttrsCharCheckExists(am.parent.DBData, name, id)
	return false
}

func (am *AttrsManager) CharGetBindingGroupIdList(id string) []string {
	all, err := model.AttrsCharGetBindingList(am.parent.DBData, id)
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
	_, err := model.AttrsCharUnbindAll(am.parent.DBData, id)
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
	NickName         string
}

func (i *AttributesItem) SaveToDB(db *sqlx.DB, tx *sql.Tx) {
	// 使用事务写入
	rawData, err := i.toDict().V().ToJSON()
	if err != nil {
		return
	}
	err = model.AttrsPutById(db, tx, i.ID, rawData, i.NickName)
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

func (i *AttributesItem) Store(name string, value *ds.VMValue) {
	now := time.Now().Unix()
	i.valueMap.Store(name, value)
	i.LastModifiedTime = now
	i.LastUsedTime = now
}

func (i *AttributesItem) toDict() *ds.VMDictValue {
	// 这里有一个风险，就是对dict的改动可能不会影响修改时间和使用时间，从而被丢弃
	return ds.VMValueNewDict(i.valueMap)
}

func (i *AttributesItem) Clear() {
	// TODO: 塞进函数里
	var keys []string
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		keys = append(keys, key)
		return true
	})

	for _, key := range keys {
		i.valueMap.Delete(key)
	}
}

func (i *AttributesItem) toArrayKeys() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(items, ds.VMValueNewStr(key))
		return true
	})
	return items
}

func (i *AttributesItem) toArrayValues() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(items, value)
		return true
	})
	return items
}

func (i *AttributesItem) toArrayItems() []*ds.VMValue {
	var items []*ds.VMValue
	i.valueMap.Range(func(key string, value *ds.VMValue) bool {
		items = append(
			items,
			ds.VMValueNewArray(ds.VMValueNewStr(key), value),
		)
		return true
	})
	return items
}

func (i *AttributesItem) Range(f func(key string, value *ds.VMValue) bool) {
	i.valueMap.Range(f)
}
