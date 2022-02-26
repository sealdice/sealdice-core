package main

import (
	"sealdice-core/core"
	"sealdice-core/model"
	"strings"
)

func VarSetValue(ctx *MsgContext, s string, v *VMValue) {
	name := ctx.player.GetValueNameByAlias(s, nil)

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		ctx.player.ValueMapTemp[s] = *v
		return
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		if ctx.session != nil && ctx.player != nil {
			playerVars := ctx.LoadPlayerVars()
			playerVars.ValueMap[s] = *v
		}
		return
	}

	// 群变量
	if ctx.group != nil && strings.HasPrefix(s, "$g") {
		// 这里不知道原因，但是有时候 ValueMap 不会被创建
		g := ctx.group
		if g.ValueMap == nil {
			g.ValueMap = map[string]VMValue{}
		}

		ctx.group.ValueMap[s] = *v
		return
	}

	ctx.player.ValueMap[name] = *v
}

func VarDelValue(ctx *MsgContext, s string) {
	name := ctx.player.GetValueNameByAlias(s, nil)

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		delete(ctx.player.ValueMapTemp, s)
		return
	}

	// 个人变量
	if strings.HasPrefix(s, "$m") {
		if ctx.session != nil && ctx.player != nil {
			playerVars := ctx.LoadPlayerVars()
			delete(playerVars.ValueMap, s)
		}
	}

	// 群变量
	if ctx.group != nil && strings.HasPrefix(s, "$g") {
		g := ctx.group
		if g.ValueMap == nil {
			g.ValueMap = map[string]VMValue{}
		}

		delete(ctx.group.ValueMap, s)
		return
	}

	delete(ctx.player.ValueMap, name)
}

func VarGetValue(ctx *MsgContext, s string) (*VMValue, bool) {
	name := ctx.player.GetValueNameByAlias(s, nil)

	// 临时变量
	if strings.HasPrefix(s, "$t") {
		v, exists := ctx.player.ValueMapTemp[s]
		return &v, exists
	}

	// 个人全局变量
	if strings.HasPrefix(s, "$m") {
		if ctx.session != nil && ctx.player != nil {
			playerVars := ctx.LoadPlayerVars()
			a, b := playerVars.ValueMap[s]
			return &a, b
		}
	}

	// 群变量
	if ctx.group != nil && strings.HasPrefix(s, "$g") {
		g := ctx.group
		if g.ValueMap == nil {
			g.ValueMap = map[string]VMValue{}
		}

		v, exists := ctx.group.ValueMap[s]
		return &v, exists
	}

	// 个人群变量
	v, e := ctx.player.ValueMap[name]
	return &v, e
}

func (i *PlayerInfo) GetValueNameByAlias(s string, alias map[string][]string) string {
	name := s

	if alias == nil {
		alias = *i.TempValueAlias
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

func (i *PlayerInfo) SetValueInt64(s string, sanNew int64, alias map[string][]string) {
	name := i.GetValueNameByAlias(s, alias)
	VarSetValue(&MsgContext{player: i}, name, &VMValue{VMTypeInt64, sanNew})
}

func (i *PlayerInfo) GetValueInt64(s string, alias map[string][]string) (int64, bool) {
	var ret int64
	name := i.GetValueNameByAlias(s, alias)
	v, exists := VarGetValue(&MsgContext{player: i}, name)

	if exists {
		ret = v.Value.(int64)
	}
	return ret, exists
}

func LoadPlayerVars(s *IMSession, id int64) *PlayerVariablesItem {
	if s.PlayerVarsData == nil {
		s.PlayerVarsData = map[int64]*PlayerVariablesItem{}
	}

	if _, exists := s.PlayerVarsData[id]; !exists {
		s.PlayerVarsData[id] = &PlayerVariablesItem{
			Loaded: false,
		}
	}

	vd, _ := s.PlayerVarsData[id]
	if vd.ValueMap == nil {
		vd.ValueMap = map[string]VMValue{}
	}

	if vd.Loaded == false {
		vd.Loaded = true
		data := model.AttrUserGetAll(id)
		err := JsonValueMapUnmarshal(data, &vd.ValueMap)
		if err != nil {
			core.GetLogger().Error(err)
		}
	}

	return vd
}
