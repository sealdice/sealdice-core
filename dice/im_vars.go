package dice

// 用户变量相关

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sealdice-core/dice/model"
	"strconv"
	"strings"
	"time"

	"github.com/fy0/lockfree"
)

// LoadPlayerGlobalVars 加载个人全局数据
func (ctx *MsgContext) LoadPlayerGlobalVars() *PlayerVariablesItem {
	if ctx.Player != nil {
		return LoadPlayerGlobalVars(ctx.Session, ctx.Player.UserID)
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

		data := model.AttrGroupGetAll(ctx.Dice.DBData, g.GroupID)
		rawData := map[string]*VMValue{}
		err := json.Unmarshal(data, &rawData)
		if err != nil {
			return
		}
		for k, v := range rawData {
			g.ValueMap.Set(k, v)
		}
	}
}

func VarSetValueStr(ctx *MsgContext, s string, v string) {
	VarSetValue(ctx, s, &VMValue{TypeID: VMTypeString, Value: v})
}

func VarSetValueDNDComputed(ctx *MsgContext, s string, val int64, expr string) {
	vd := &VMDndComputedValueData{
		BaseValue: VMValue{
			TypeID: VMTypeInt64,
			Value:  val,
		},
		Expr: expr,
	}
	VarSetValue(ctx, s, &VMValue{TypeID: VMTypeDNDComputedValue, Value: vd})
}

func VarSetValueInt64(ctx *MsgContext, s string, v int64) {
	VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: v})
}

func VarSetValueAuto(ctx *MsgContext, s string, v interface{}) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(int))})
	case reflect.Int8:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(int8))})
	case reflect.Int16:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(int16))})
	case reflect.Int32:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(int32))})
	case reflect.Int64:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: v.(int64)})
	case reflect.Uint:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(uint))})
	case reflect.Uint8:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(uint8))})
	case reflect.Uint16:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(uint16))})
	case reflect.Uint32:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(uint32))})
	case reflect.Uint64:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(uint64))})
	case reflect.Float32:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(uint64))})
	case reflect.Float64:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeInt64, Value: int64(v.(float64))})
	case reflect.String:
		VarSetValue(ctx, s, &VMValue{TypeID: VMTypeString, Value: v.(string)})
	default: /*no-op*/
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
	if ctx.Session != nil && ctx.Player != nil {
		vars, _ := ctx.ChVarsGet()
		vars.Set(name, &vClone)
		ctx.ChVarsUpdateTime()
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
		vars, _ := ctx.ChVarsGet()
		vars.Del(name)
		ctx.ChVarsUpdateTime()
	}
}

func VarGetValueInt64(ctx *MsgContext, s string) (int64, bool) {
	v, exists := VarGetValue(ctx, s)
	if exists && v.TypeID == VMTypeInt64 {
		return v.Value.(int64), true
	}
	return 0, false
}

func VarGetValueStr(ctx *MsgContext, s string) (string, bool) {
	v, exists := VarGetValue(ctx, s)
	if exists && v.TypeID == VMTypeString {
		return v.Value.(string), true
	}
	return "", false
}

func VarGetValue(ctx *MsgContext, s string) (*VMValue, bool) {
	name := ctx.Player.GetValueNameByAlias(s, nil)

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		var v *VMValue
		// 跟入群致辞闪退的一个bug有关，当时是报 _v, exists := ctx.Player.ValueMapTemp.Get(s) 这一行 nil pointer
		if ctx.Player.ValueMapTemp == nil {
			ctx.Player.ValueMapTemp = lockfree.NewHashMap()
			return nil, false
		}
		_v, exists := ctx.Player.ValueMapTemp.Get(s)
		// v, exists := ctx.Player.ValueMapTemp[s]
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
			vars, _ := ctx.ChVarsGet()
			var v *VMValue
			_v, e := vars.Get(name)
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
			name = k // 防止一手大小写不一致
			break    // 名字本身就是确定值，不用修改
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

func GetValueNameByAlias(s string, alias map[string][]string) string {
	name := s

	for k, v := range alias {
		if strings.EqualFold(s, k) {
			name = k // 防止一手大小写不一致
			break    // 名字本身就是确定值，不用修改
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

func LoadPlayerGlobalVars(s *IMSession, id string) *PlayerVariablesItem {
	if s.PlayerVarsData == nil {
		s.PlayerVarsData = map[string]*PlayerVariablesItem{}
	}

	if _, exists := s.PlayerVarsData[id]; !exists {
		s.PlayerVarsData[id] = &PlayerVariablesItem{
			Loaded: false,
		}
	}

	vd := s.PlayerVarsData[id]
	if vd.ValueMap == nil {
		vd.ValueMap = lockfree.NewHashMap()
	}

	if !vd.Loaded {
		vd.ValueMap = lockfree.NewHashMap()
		data := model.AttrUserGetAll(s.Parent.DBData, id)

		mapData := make(map[string]*VMValue)
		err := JSONValueMapUnmarshal(data, &mapData)
		if err != nil {
			s.Parent.Logger.Errorf("读取玩家数据失败！错误 %v 原数据 %v", err, data)
		}

		needToLoad := map[string]bool{}
		for k, v := range mapData {
			vd.ValueMap.Set(k, v)
			if strings.HasPrefix(k, "$:group-bind:") {
				// needToLoad[k[len("$:group-bind:"):]] = true
				name, _ := v.ReadString()
				// fmt.Println("@@@@@@@@", k, name, v)
				if name != "" {
					needToLoad[name] = true
				}
			}
		}
		// 保险起见？应该不用
		// vd.LastWriteTime = time.Now().Unix()

		// 进行绑定角色的设置
		for name := range needToLoad {
			_data := mapData["$ch:"+name]
			if _data != nil {
				chData := make(map[string]*VMValue)
				err := JSONValueMapUnmarshal([]byte(_data.Value.(string)), &chData)

				if err == nil {
					m := lockfree.NewHashMap()
					for k, v := range chData {
						m.Set(k, v)
					}

					// $:ch-bind-data:角色
					m.Set("$:cardName", &VMValue{TypeID: VMTypeString, Value: name}) // 防止出事，覆盖一次
					vd.ValueMap.Set("$:ch-bind-data:"+name, m)
				}
			}
		}

		vd.Loaded = true
	}

	return vd
}

func LoadPlayerGroupVars(dice *Dice, group *GroupInfo, player *GroupPlayerInfo) *PlayerVariablesItem {
	if player.Vars == nil {
		player.Vars = &model.PlayerVariablesItem{
			Loaded: false,
		}
	}

	vd := player.Vars
	if vd.Loaded {
		return (*PlayerVariablesItem)(vd)
	}

	vd.ValueMap = lockfree.NewHashMap()
	vd.Loaded = true

	// QQ-Group:131687852-QQ:303451945
	data := model.AttrGroupUserGetAll(dice.DBData, group.GroupID, player.UserID)
	// fmt.Println("???", group.GroupId, string(data))
	if len(data) > 0 {
		mapData := make(map[string]*VMValue)
		err := JSONValueMapUnmarshal(data, &mapData)

		for k, v := range mapData {
			vd.ValueMap.Set(k, v)
		}

		if _, exists := mapData["$:cardBindMark"]; exists {
			vars := LoadPlayerGlobalVars(dice.ImSession, player.UserID)

			if _data, exists := vars.ValueMap.Get("$:group-bind:" + group.GroupID); exists {
				if data, ok := _data.(*VMValue); ok {
					name, ok := data.ReadString()

					if ok {
						_m, ok := vars.ValueMap.Get("$:ch-bind-data:" + name)
						if ok {
							m := _m.(lockfree.HashMap)
							m.Set("$:cardName", &VMValue{TypeID: VMTypeString, Value: name}) // 防止出事，覆盖一次
							player.Vars.ValueMap.Set("$:card", m)
						}
					}
				}
			}
		}

		if err != nil {
			dice.Logger.Errorf("加载玩家数据失败%s-%s: %s", group.GroupID, player.UserID, err.Error())
		}
	}
	return (*PlayerVariablesItem)(vd)
}

func SetTempVars(ctx *MsgContext, qqNickname string) {
	// 设置临时变量
	if ctx.Player != nil {
		pcName := ctx.Player.Name
		pcName = strings.ReplaceAll(pcName, "\n", "")
		pcName = strings.ReplaceAll(pcName, "\r", "")
		pcName = strings.ReplaceAll(pcName, `\n`, "")
		pcName = strings.ReplaceAll(pcName, `\r`, "")
		pcName = strings.ReplaceAll(pcName, `\f`, "")

		VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", pcName))
		if ctx.Dice != nil && !ctx.Dice.PlayerNameWrapEnable {
			VarSetValueStr(ctx, "$t玩家", pcName)
		}
		VarSetValueStr(ctx, "$t玩家_RAW", pcName)
		VarSetValueStr(ctx, "$tQQ昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t帐号昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t账号昵称", fmt.Sprintf("<%s>", qqNickname))
		VarSetValueStr(ctx, "$t帐号ID", ctx.Player.UserID)
		VarSetValueStr(ctx, "$t账号ID", ctx.Player.UserID)
		VarSetValueInt64(ctx, "$t个人骰子面数", int64(ctx.Player.DiceSideNum))
		VarSetValueStr(ctx, "$tQQ", ctx.Player.UserID)
		VarSetValueStr(ctx, "$t骰子帐号", ctx.EndPoint.UserID)
		VarSetValueStr(ctx, "$t骰子账号", ctx.EndPoint.UserID)
		VarSetValueStr(ctx, "$t骰子昵称", ctx.EndPoint.Nickname)
		VarSetValueStr(ctx, "$t帐号ID_RAW", UserIDExtract(ctx.Player.UserID))
		VarSetValueStr(ctx, "$t账号ID_RAW", UserIDExtract(ctx.Player.UserID))
		VarSetValueStr(ctx, "$t平台", ctx.EndPoint.Platform)

		rpSeed := (time.Now().Unix() + (8 * 60 * 60)) / (24 * 60 * 60)
		rpSeed += int64(fingerprint(ctx.EndPoint.UserID))
		rpSeed += int64(fingerprint(ctx.Player.UserID))
		randItem := rand.NewSource(rpSeed)
		rp := randItem.Int63()%100 + 1
		VarSetValueInt64(ctx, "$t人品", rp)

		now := time.Now()
		t, _ := strconv.ParseInt(now.Format("20060102"), 10, 64)
		VarSetValueInt64(ctx, "$tDate", t)
		wd := int64(now.Weekday())
		if wd == 0 {
			wd = 7
		}
		VarSetValueInt64(ctx, "$tWeekday", wd)

		t, _ = strconv.ParseInt(now.Format("2006"), 10, 64)
		VarSetValueInt64(ctx, "$tYear", t)
		t, _ = strconv.ParseInt(now.Format("01"), 10, 64)
		VarSetValueInt64(ctx, "$tMonth", t)
		t, _ = strconv.ParseInt(now.Format("02"), 10, 64)
		VarSetValueInt64(ctx, "$tDay", t)
		t, _ = strconv.ParseInt(now.Format("15"), 10, 64)
		VarSetValueInt64(ctx, "$tHour", t)
		t, _ = strconv.ParseInt(now.Format("04"), 10, 64)
		VarSetValueInt64(ctx, "$tMinute", t)
		t, _ = strconv.ParseInt(now.Format("05"), 10, 64)
		VarSetValueInt64(ctx, "$tSecond", t)
		VarSetValueInt64(ctx, "$tTimestamp", now.Unix())
	}

	if ctx.Group != nil {
		if ctx.MessageType == "group" {
			VarSetValueStr(ctx, "$t群号", ctx.Group.GroupID)
			VarSetValueStr(ctx, "$t群名", ctx.Group.GroupName)
			VarSetValueStr(ctx, "$t群号_RAW", UserIDExtract(ctx.Group.GroupID))
		}
		VarSetValueInt64(ctx, "$t群组骰子面数", ctx.Group.DiceSideNum)
		VarSetValueInt64(ctx, "$t当前骰子面数", getDefaultDicePoints(ctx))
		if ctx.Group.System == "" {
			ctx.Group.System = "coc7"
			ctx.Group.UpdatedAtTime = time.Now().Unix()
		}
		VarSetValueStr(ctx, "$t游戏模式", ctx.Group.System)
		VarSetValueStr(ctx, "$t规则模板", ctx.Group.System)
		VarSetValueStr(ctx, "$tSystem", ctx.Group.System)
		VarSetValueStr(ctx, "$t当前记录", ctx.Group.LogCurName)
	}

	if ctx.MessageType == "group" {
		VarSetValueStr(ctx, "$t消息类型", "group")
	} else {
		VarSetValueStr(ctx, "$t消息类型", "private")
	}
}
