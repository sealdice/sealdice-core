package v2

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/api/v2/backup"
	"sealdice-core/api/v2/ban"
	"sealdice-core/api/v2/base"
	"sealdice-core/api/v2/group"
	"sealdice-core/api/v2/imconnection"
	"sealdice-core/api/v2/middleware"
	"sealdice-core/dice"
)

// InitV2Router 初始化v2版本的API路由
// 使用依赖注入模式，将dice实例传递给各个模块
func InitV2Router(api huma.API, dm *dice.DiceManager) {
	baseGroup := huma.NewGroup(api, "/sd-api/v2/base")
	baseGroup.UseSimpleModifier(huma.OperationTags("base"))
	baseService := base.NewBaseService(dm)
	baseService.RegisterRoutes(baseGroup)

	groupPublic := huma.NewGroup(api, "/sd-api/v2/group")
	groupPublic.UseSimpleModifier(huma.OperationTags("group"))
	groupService := group.NewGroupService(dm)
	groupService.RegisterRoutes(groupPublic)

	groupProtected := huma.NewGroup(api, "/sd-api/v2/group")
	groupProtected.UseSimpleModifier(huma.OperationTags("group"))
	groupProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	groupService.RegisterProtectedRoutes(groupProtected)

	backupAuth := huma.NewGroup(api, "/sd-api/v2/backup")
	backupAuth.UseSimpleModifier(huma.OperationTags("backup"))
	backupAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	backupService := backup.NewBackupService(dm)
	backupService.RegisterRoutes(backupAuth)

	backupProtected := huma.NewGroup(api, "/sd-api/v2/backup")
	backupProtected.UseSimpleModifier(huma.OperationTags("backup"))
	backupProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	backupService.RegisterProtectedRoutes(backupProtected)

	banAuth := huma.NewGroup(api, "/sd-api/v2/ban")
	banAuth.UseSimpleModifier(huma.OperationTags("ban"))
	banAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	banService := ban.NewBanService(dm)
	banService.RegisterRoutes(banAuth)

	banProtected := huma.NewGroup(api, "/sd-api/v2/ban")
	banProtected.UseSimpleModifier(huma.OperationTags("ban"))
	banProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	banService.RegisterProtectedRoutes(banProtected)

	imcAuth := huma.NewGroup(api, "/sd-api/v2/imconnection")
	imcAuth.UseSimpleModifier(huma.OperationTags("imconnection"))
	imcAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	imcService := imconnection.NewService(dm)
	imcService.RegisterRoutes(imcAuth)
	imcProtected := huma.NewGroup(api, "/sd-api/v2/imconnection")
	imcProtected.UseSimpleModifier(huma.OperationTags("imconnection"))
	imcProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	imcService.RegisterProtectedRoutes(imcProtected)
	// TODO: 后续可以在这里添加其他模块
	// configService := config.NewConfigService(dice)
	// protected := huma.NewGroup(api, "/sd-api/v2")
	// protected.UseMiddleware(middleware.AuthMiddleware(api, dice))
	// configService.RegisterRoutes(protected)

	// groupService := group.NewGroupService(dice)
	// groupService.RegisterRoutes(protected)
}
