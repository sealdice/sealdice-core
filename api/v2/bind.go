package v2

import (
	"runtime"

	"github.com/labstack/echo/v4"

	"sealdice-core/api/v2/middleware"
	"sealdice-core/dice"
)

var ApiGroupApp *ApiGroup

// Dice 由于现在还不能将配置拆出，导致middleware中间件没法正经做，只能在这里保存一下Dice进行控制。
var diceInstance *dice.Dice

// 统一格式：前带/后不带/。

// InitBaseRouter 初始化基础路由
func InitBaseRouter(router *echo.Group) {
	publicRouter := router.Group("/base")
	baseApi := ApiGroupApp.SystemApiGroup.BaseApi
	{
		publicRouter.GET("/preinfo", baseApi.PreInfo)
		publicRouter.GET("/baseInfo", baseApi.BaseInfo, middleware.AuthMiddleware(diceInstance))
		publicRouter.GET("/heartbeat", baseApi.HeartBeat, middleware.AuthMiddleware(diceInstance))
		publicRouter.GET("/checkSecurity", baseApi.CheckSecurity, middleware.AuthMiddleware(diceInstance))
		// 安卓专属停止代码
		if runtime.GOOS == "android" {
			publicRouter.GET("/force_stop", baseApi.ForceStop, middleware.AuthMiddleware(diceInstance))
		}
	}
}

func InitLoginRouter(router *echo.Group) {
	publicRouter := router.Group("/login")
	loginApi := ApiGroupApp.SystemApiGroup.LoginApi
	{
		publicRouter.POST("/signin", loginApi.DoSignIn)
		publicRouter.GET("/salt", loginApi.DoSignInGetSalt)
	}
}

func InitRouter(router *echo.Echo, dice *dice.Dice) {
	diceInstance = dice
	ApiGroupApp = InitSealdiceAPIV2(dice)
	group := router.Group("/v2/sd-api")
	InitBaseRouter(group)
	InitLoginRouter(group)
}
