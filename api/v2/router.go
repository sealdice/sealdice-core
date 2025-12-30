package v2

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/api/v2/base"
	"sealdice-core/dice"
)

// InitV2Router 初始化v2版本的API路由
// 使用依赖注入模式，将dice实例传递给各个模块
func InitV2Router(api huma.API, dm *dice.DiceManager) {
	baseGroup := huma.NewGroup(api, "/sd-api/v2/base")
	baseGroup.UseSimpleModifier(huma.OperationTags("base"))
	baseService := base.NewBaseService(dm)
	baseService.RegisterRoutes(baseGroup)

	// TODO: 后续可以在这里添加其他模块
	// configService := config.NewConfigService(dice)
	// protected := huma.NewGroup(api, "/sd-api/v2")
	// protected.UseMiddleware(middleware.AuthMiddleware(api, dice))
	// configService.RegisterRoutes(protected)

	// groupService := group.NewGroupService(dice)
	// groupService.RegisterRoutes(protected)
}
