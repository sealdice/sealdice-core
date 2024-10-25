package v2

import (
	"sealdice-core/api/v2/ui"
	"sealdice-core/dice"
)

// 有概率以后支持WEBHOOK等等和UI无关代码
type ApiGroup struct {
	SystemApiGroup ui.WebUIDiceGroup
}

// InitSealdiceAPIV2 初始化SealdiceAPI V2
// 先初始化，然后再执行router绑定的代码，这样的好处是任何一块代码都可插拔。
func InitSealdiceAPIV2(d *dice.Dice) *ApiGroup {
	a := new(ApiGroup)
	a.SystemApiGroup.Init(d)
	return a
}
