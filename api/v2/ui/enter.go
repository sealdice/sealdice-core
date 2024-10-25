package ui

import (
	"sealdice-core/dice"
)

// 管理所有WEBUI
type WebUIDiceGroup struct {
	BaseApi  *BaseApi
	LoginApi *LoginApi
}

func (a *WebUIDiceGroup) Init(d *dice.Dice) {
	// 初始化API DICE引用，如果不是操作DICE的，到时候我们再单独开一栏，不传dice进去
	a.BaseApi = &BaseApi{dice: d}
	a.LoginApi = &LoginApi{dice: d}
}
