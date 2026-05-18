package v2

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/labstack/echo/v4"

	"sealdice-core/api/v2/backup"
	"sealdice-core/api/v2/ban"
	"sealdice-core/api/v2/base"
	"sealdice-core/api/v2/config"
	"sealdice-core/api/v2/customreply"
	"sealdice-core/api/v2/customtext"
	"sealdice-core/api/v2/deck"
	"sealdice-core/api/v2/group"
	"sealdice-core/api/v2/imconnection"
	"sealdice-core/api/v2/js"
	"sealdice-core/api/v2/middleware"
	"sealdice-core/api/v2/realtime"
	"sealdice-core/api/v2/story"
	"sealdice-core/dice"
)

// InitV2Router 初始化v2版本的API路由
// 使用依赖注入模式，将dice实例传递给各个模块
func InitV2Router(api huma.API, e *echo.Echo, dm *dice.DiceManager) {
	baseGroup := huma.NewGroup(api, "/sd-api/v2/base")
	baseGroup.UseSimpleModifier(huma.OperationTags("base"))
	baseService := base.NewBaseService(dm)
	baseService.RegisterRoutes(baseGroup)
	realtime.RegisterRoutes(e, dm)

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

	customTextAuth := huma.NewGroup(api, "/sd-api/v2/custom-text")
	customTextAuth.UseSimpleModifier(huma.OperationTags("custom-text"))
	customTextAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	customTextService := customtext.NewService(dm)
	customTextService.RegisterRoutes(customTextAuth)

	customTextProtected := huma.NewGroup(api, "/sd-api/v2/custom-text")
	customTextProtected.UseSimpleModifier(huma.OperationTags("custom-text"))
	customTextProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	customTextService.RegisterProtectedRoutes(customTextProtected)

	configAuth := huma.NewGroup(api, "/sd-api/v2/config")
	configAuth.UseSimpleModifier(huma.OperationTags("config"))
	configAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	configService := config.NewService(dm)
	configService.RegisterRoutes(configAuth)

	configProtected := huma.NewGroup(api, "/sd-api/v2/config")
	configProtected.UseSimpleModifier(huma.OperationTags("config"))
	configProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	configService.RegisterProtectedRoutes(configProtected)

	customReplyAuth := huma.NewGroup(api, "/sd-api/v2/custom-reply")
	customReplyAuth.UseSimpleModifier(huma.OperationTags("custom-reply"))
	customReplyAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	customReplyService := customreply.NewService(dm)
	customReplyService.RegisterRoutes(customReplyAuth)

	customReplyProtected := huma.NewGroup(api, "/sd-api/v2/custom-reply")
	customReplyProtected.UseSimpleModifier(huma.OperationTags("custom-reply"))
	customReplyProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	customReplyService.RegisterProtectedRoutes(customReplyProtected)

	deckAuth := huma.NewGroup(api, "/sd-api/v2/deck")
	deckAuth.UseSimpleModifier(huma.OperationTags("deck"))
	deckAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	deckService := deck.NewService(dm)
	deckService.RegisterRoutes(deckAuth)

	deckProtected := huma.NewGroup(api, "/sd-api/v2/deck")
	deckProtected.UseSimpleModifier(huma.OperationTags("deck"))
	deckProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	deckService.RegisterProtectedRoutes(deckProtected)

	storyAuth := huma.NewGroup(api, "/sd-api/v2/story")
	storyAuth.UseSimpleModifier(huma.OperationTags("story"))
	storyAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	storyService := story.NewService(dm)
	storyService.RegisterRoutes(storyAuth)

	storyProtected := huma.NewGroup(api, "/sd-api/v2/story")
	storyProtected.UseSimpleModifier(huma.OperationTags("story"))
	storyProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	storyService.RegisterProtectedRoutes(storyProtected)

	jsAuth := huma.NewGroup(api, "/sd-api/v2/js")
	jsAuth.UseSimpleModifier(huma.OperationTags("js"))
	jsAuth.UseMiddleware(middleware.AuthMiddleware(api, dm.GetDice()))
	jsService := js.NewService(dm)
	jsService.RegisterRoutes(jsAuth)

	jsProtected := huma.NewGroup(api, "/sd-api/v2/js")
	jsProtected.UseSimpleModifier(huma.OperationTags("js"))
	jsProtected.UseMiddleware(middleware.WriteProtectedMiddleware(api, dm.GetDice()))
	jsService.RegisterProtectedRoutes(jsProtected)

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
