package dice

// 用户变量相关

import (
	"encoding/json"
	"fmt"
	"github.com/fy0/lockfree"
	"reflect"
	"sealdice-core/dice/model"
	"strings"
	"time"
)

// LoadPlayerGlobalVars 加载个人全局数据
func (ctx *MsgContext) LoadPlayerGlobalVars() *PlayerVariablesItem {
	if ctx.Player != nil {
		return LoadPlayerGlobalVars(ctx.Session, ctx.Player.UserId)
	}
	return nil
}

// LoadPlayerGroupVars 加载个人群内数据
func (ctx *MsgContext) LoadPlayerGroupVars(group *GroupInfo, player *GroupPlayerInfo) *PlayerVariablesItem {
	if ctx.Dice != nil {
		return LoadPlayerGroupVars(ctx.Dice, group, player)
	}
	return nil
}

func (ctx *MsgContext) LoadGroupVars() {
	g := ctx.Group
	if g.ValueMap == nil {
		g.ValueMap = lockfree.NewHashMap()
	}
	data := model.AttrGroupGetAll(ctx.Dice.DB, g.GroupId)
	rawData := map[string]*VMValue{}
	err := json.Unmarshal(data, &rawData)
	if err != nil {
		return
	}
	for k, v := range rawData {
		g.ValueMap.Set(k, v)
	}
}

func VarSetValueStr(ctx *MsgContext, s string, v string) {
	VarSetValue(ctx, s, &VMValue{VMTypeString, v})
}

func VarSetValueDNDComputed(ctx *MsgContext, s string, val int64, expr string) {
	vd := &VMComputedValueData{
		BaseValue: VMValue{
			VMTypeInt64,
			val,
		},
		Expr: expr,
	}
	VarSetValue(ctx, s, &VMValue{VMTypeComputedValue, vd})
}

func VarSetValueInt64(ctx *MsgContext, s string, v int64) {
	VarSetValue(ctx, s, &VMValue{VMTypeInt64, v})
}

func VarSetValueAuto(ctx *MsgContext, s string, v interface{}) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(int))})
	case reflect.Int8:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(int8))})
	case reflect.Int16:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(int16))})
	case reflect.Int32:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(int32))})
	case reflect.Int64:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(int64))})
	case reflect.Uint:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(uint))})
	case reflect.Uint8:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(uint8))})
	case reflect.Uint16:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(uint16))})
	case reflect.Uint32:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(uint32))})
	case reflect.Uint64:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(uint64))})
	case reflect.Float32:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(uint64))})
	case reflect.Float64:
		VarSetValue(ctx, s, &VMValue{VMTypeInt64, int64(v.(float64))})
	case reflect.String:
		VarSetValue(ctx, s, &VMValue{VMTypeString, v.(string)})
	}
}

func VarSetValue(ctx *MsgContext, s string, v *VMValue) {
	name := ctx.Player.GetValueNameByAlias(s, nil)
	vClone := *v

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		if ctx.Player.ValueMapTemp == nil {
			ctx.Player.ValueMapTemp = lockfree.NewHashMap()
		}
		ctx.Player.ValueMapTemp.Set(s, &vClone)
		return
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		if ctx.Session != nil && ctx.Player != nil {
			playerVars := ctx.LoadPlayerGlobalVars()
			playerVars.ValueMap.Set(s, &vClone)
			playerVars.LastWriteTime = time.Now().Unix()
		}
		return
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		ctx.LoadGroupVars()
		ctx.Group.ValueMap.Set(s, &vClone)
		return
	}

	// 个人属性
	if ctx.Player.Vars.ValueMap != nil && ctx.Player.Vars.Loaded {
		ctx.Player.Vars.ValueMap.Set(name, &vClone)
		ctx.Player.Vars.LastWriteTime = time.Now().Unix()
	}
}

func VarDelValue(ctx *MsgContext, s string) {
	name := ctx.Player.GetValueNameByAlias(s, nil)

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		ctx.Player.ValueMapTemp.Del(s)
		return
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		if ctx.Session != nil && ctx.Player != nil {
			playerVars := ctx.LoadPlayerGlobalVars()
			playerVars.ValueMap.Del(s)
			playerVars.LastWriteTime = time.Now().Unix()
		}
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		g := ctx.Group
		if g.ValueMap == nil {
			g.ValueMap = lockfree.NewHashMap()
		}

		g.ValueMap.Del(s)
		return
	}

	if ctx.Player.Vars.ValueMap != nil && ctx.Player.Vars.Loaded {
		ctx.Player.Vars.ValueMap.Del(name)
		ctx.Player.Vars.LastWriteTime = time.Now().Unix()
	}
}

func VarGetValueInt64(ctx *MsgContext, s string) (int64, bool) {
	v, exists := VarGetValue(ctx, s)
	if exists && v.TypeId == VMTypeInt64 {
		return v.Value.(int64), true
	}
	return 0, false
}

func VarGetValue(ctx *MsgContext, s string) (*VMValue, bool) {
	name := ctx.Player.GetValueNameByAlias(s, nil)

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		var v *VMValue
		_v, exists := ctx.Player.ValueMapTemp.Get(s)
		//v, exists := ctx.Player.ValueMapTemp[s]
		if exists {
			v = _v.(*VMValue)
		}
		return v, exists
	}

	// 个人全局变量
	if strings.HasPrefix(s, "$m") {
		if ctx.Session != nil && ctx.Player != nil {
			var v *VMValue
			playerVars := ctx.LoadPlayerGlobalVars()
			_v, e := playerVars.ValueMap.Get(s)
			if e {
				v = _v.(*VMValue)
			}

			return v, e
		}
	}

	// 群变量
	if ctx.Group != nil && strings.HasPrefix(s, "$g") {
		g := ctx.Group
		if g.ValueMap == nil {
			g.ValueMap = lockfree.NewHashMap()
		}

		var v *VMValue
		_v, exists := ctx.Group.ValueMap.Get(s)
		if exists {
			v = _v.(*VMValue)
		}
		return v, exists
	}

	// 个人群变量
	if ctx.Player != nil {
		if ctx.Player.Vars != nil && ctx.Player.Vars.Loaded {
			var v *VMValue
			_v, e := ctx.Player.Vars.ValueMap.Get(name)
			if e {
				v = _v.(*VMValue)
			}
			return v, e
		}
	}
	return nil, false
}

func (i *GroupPlayerInfo) GetValueNameByAlias(s string, alias map[string][]string) string {
	name := s

	if alias == nil {
		// 当私聊的时候，i就会是nil
		if i != nil && i.TempValueAlias != nil {
			alias = *i.TempValueAlias
		}
	}

	for k, v := range alias {
		if strings.EqualFold(s, k) {
			break // 名字本身就是确定值，不用修改
		}
		for _, i := range v {
			if strings.EqualFold(s, i) {
				name = k
				break
			}
		}
	}

	return name
}

func (i *GroupPlayerInfo) SetValueInt64(s string, value int64, alias map[string][]string) {
	name := i.GetValueNameByAlias(s, alias)
	VarSetValue(&MsgContext{Player: i}, name, &VMValue{VMTypeInt64, value})
}

func (i *GroupPlayerInfo) GetValueInt64(s string, alias map[string][]string) (int64, bool) {
	var ret int64
	name := i.GetValueNameByAlias(s, alias)
	v, exists := VarGetValue(&MsgContext{Player: i}, name)

	if exists {
		ret = v.Value.(int64)
	}
	return ret, exists
}

func LoadPlayerGlobalVars(s *IMSession, id string) *PlayerVariablesItem {
	if s.PlayerVarsData == nil {
		s.PlayerVarsData = map[string]*PlayerVariablesItem{}
	}

	if _, exists := s.PlayerVarsData[id]; !exists {
		s.PlayerVarsData[id] = &PlayerVariablesItem{
			Loaded: false,
		}
	}

	vd, _ := s.PlayerVarsData[id]
	if vd.ValueMap == nil {
		vd.ValueMap = lockfree.NewHashMap()
	}

	if vd.Loaded == false {
		vd.ValueMap = lockfree.NewHashMap()
		vd.Loaded = true
		data := model.AttrUserGetAll(s.Parent.DB, id)

		mapData := make(map[string]*VMValue)
		err := JsonValueMapUnmarshal(data, &mapData)

		for k, v := range mapData {
			vd.ValueMap.Set(k, v)
		}
		// 保险起见？应该不用
		//vd.LastWriteTime = time.Now().Unix()

		if err != nil {
			s.Parent.Logger.Error(err)
		}
	}

	return vd
}

func LoadPlayerGroupVars(dice *Dice, group *GroupInfo, player *GroupPlayerInfo) *PlayerVariablesItem {
	if player.Vars == nil {
		player.Vars = &PlayerVariablesItem{
			Loaded: false,
		}
	}

	vd := player.Vars
	if vd.Loaded == false {
		vd.ValueMap = lockfree.NewHashMap()
		vd.Loaded = true

		// QQ-Group:131687852-QQ:303451945
		data := model.AttrGroupUserGetAll(dice.DB, group.GroupId, player.UserId)
		mapData := make(map[string]*VMValue)
		err := JsonValueMapUnmarshal(data, &mapData)

		for k, v := range mapData {
			vd.ValueMap.Set(k, v)
		}
		if err != nil {
			dice.Logger.Error(err)
		}
	}

	return vd
}

func SetTempVars(ctx *MsgContext, qqNickname string) {
	// 设置临时变量
	if ctx.Player != nil {
		VarSetValue(ctx, "$t玩家", &VMValue{VMTypeString, fmt.Sprintf("<%s>", ctx.Player.Name)})
		VarSetValue(ctx, "$tQQ昵称", &VMValue{VMTypeString, fmt.Sprintf("<%s>", qqNickname)})
		VarSetValue(ctx, "$t帐号昵称", &VMValue{VMTypeString, fmt.Sprintf("<%s>", qqNickname)})
		VarSetValue(ctx, "$t个人骰子面数", &VMValue{VMTypeInt64, int64(ctx.Player.DiceSideNum)})
		//VarSetValue(ctx, "$tQQ", &VMValue{VMTypeInt64, ctx.Player.UserId})
		VarSetValue(ctx, "$tQQ", &VMValue{VMTypeString, ctx.EndPoint.UserId})
		VarSetValue(ctx, "$t骰子帐号", &VMValue{VMTypeString, ctx.EndPoint.UserId})
		VarSetValue(ctx, "$t骰子昵称", &VMValue{VMTypeString, ctx.EndPoint.Nickname})
	}
	if ctx.Group != nil {
		if ctx.MessageType == "group" {
			VarSetValue(ctx, "$t群号", &VMValue{VMTypeString, ctx.Group.GroupId})
			VarSetValue(ctx, "$t群名", &VMValue{VMTypeString, ctx.Group.GroupName})
		}
		VarSetValue(ctx, "$t群组骰子面数", &VMValue{VMTypeInt64, ctx.Group.DiceSideNum})
		VarSetValue(ctx, "$t当前骰子面数", &VMValue{VMTypeInt64, getDefaultDicePoints(ctx)})
	}
}
