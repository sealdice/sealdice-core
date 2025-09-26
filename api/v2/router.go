package v2

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/api/v2/base"
	"sealdice-core/dice"
)

// InitV2Router 初始化v2版本的API路由
// 使用依赖注入模式，将dice实例传递给各个模块
func InitV2Router(api huma.API, dice *dice.Dice) {
	// 注册base模块
	baseService := base.NewBaseService(dice)
	baseService.RegisterRoutes(api)

	// TODO: 后续可以在这里添加其他模块
	// configService := config.NewConfigService(dice)
	// configService.RegisterRoutes(api)

	// groupService := group.NewGroupService(dice)
	// groupService.RegisterRoutes(api)
}
