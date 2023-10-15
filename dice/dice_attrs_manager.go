package dice

import (
	"errors"
	"fmt"
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

	//	1. 首先获取当前群+用户所绑定的卡
	// 绑定卡的id是nanoid
	id, err := model.AttrsGetBindingSheetId(am.parent.DBData, userId)
	if err != nil {
		return nil, err
	}

	// 2. 如果取不到，那么取用户在当前群的默认卡
	if id == "" {
		id = fmt.Sprintf("%s-%s", groupId, userId)
	}
	return am.LoadBase(id)
}

func (am *AttrsManager) UIDConvert(userId string) string {
	// 如果存在一个虚拟id，那么返回虚拟id，不存在原样返回
	return userId
}

// LoadBase 数据加载，负责以下数据
// 1. 群内用户的默认卡(id格式为：群id:用户id)
// 2. 用户创建出的角色卡（指定id）
// 3. 群属性(id为群id)
// 4. 用户全局属性
func (am *AttrsManager) LoadBase(id string) (*AttributesItem, error) {
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
		if data != nil {
			v, err := ds.VMValueFromJSON([]byte(data.Data))
			if err != nil {
				return nil, err
			}
			if dd, ok := v.ReadDictData(); ok {
				i = &AttributesItem{
					ID:           id,
					ValueMap:     dd.Dict,
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
		return nil, errors.New("数据库异常，无法读取")
	}

	// 3. 从老数据库读取 - 群用户数据
	// （其实还有一种，但是读不了，就是玩家的卡数据，因为没有id）
	dataOld := model.AttrGroupUserGetAllBase(d.DBData, id)
	if dataOld != nil {
		mapData := make(map[string]*VMValue)
		err := JsonValueMapUnmarshal(dataOld, &mapData)
		if err != nil {
			d.Logger.Errorf("读取玩家数据失败！错误 %v 原数据 %v", err, data)
		}

		m := &ds.ValueMap{}
		for k, v := range mapData {
			m.Store(k, v.ConvertToDiceScriptValue())
		}

		now := time.Now().Unix()
		i = &AttributesItem{
			ID:               id,
			ValueMap:         m,
			LastUsedTime:     now,
			LastModifiedTime: now,
		}
		am.m.Store(id, i)
		return i, nil
	}

	// 4. 创建一个新的
	i = &AttributesItem{
		ID:       id,
		ValueMap: &ds.ValueMap{},
	}
	am.m.Store(id, i)
	return i, nil
}

func (am *AttrsManager) CheckForSave() (int, int) {
	times := 0
	saved := 0
	am.m.Range(func(key string, value *AttributesItem) bool {
		if !value.IsSaved {
			saved += 1
			value.SaveToDB()
		}
		times += 1
		return true
	})
	return times, saved
}

// CheckAndFreeUnused 此函数会被定期调用，释放最近不用的对象
func (am *AttrsManager) CheckAndFreeUnused() {
	prepareToFree := map[string]int{}
	currentTime := time.Now().Unix()
	am.m.Range(func(key string, value *AttributesItem) bool {
		if value.LastUsedTime-currentTime > 60*10 {
			prepareToFree[key] = 1
			value.SaveToDB()
		}
		return true
	})

	for key := range prepareToFree {
		am.m.Delete(key)
	}
}

// AttributesItem 这是一个人物卡对象
type AttributesItem struct {
	ID               string
	ValueMap         *ds.ValueMap // SyncMap[string, *ds.VMValue] 这种类型更好吗？我得确认一下js兼容性
	LastModifiedTime int64        // 上次修改时间
	LastUsedTime     int64        // 上次使用时间
	IsSaved          bool
}

func (i *AttributesItem) SaveToDB() {
	// 可能还需要一点别的，例如db对象
	i.IsSaved = true
}
