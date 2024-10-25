package middleware

import (
	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	"sealdice-core/utils/web/response"
)

// AuthMiddleware 鉴权中间件，接收Dice参数，仅允许可用Token.
func AuthMiddleware(d *dice.Dice) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authToken := c.Request().Header.Get("Authorization")
			// 这里统一使用Dice的目的是，以后考虑直接把DiceManager扬了，改单例模式
			if d.Parent.AccessTokens[authToken] {
				return next(c)
			}
			return response.NoAuth(c)
		}
	}
}

func TestModeMiddleware(d *dice.Dice) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if d.Parent.JustForTest {
				// TODO：补充更多细节，比如”展示模式不能进行xxx操作“
				return response.FailWithMessage("展示模式，无法进行……", c)
			}
			return next(c)
		}
	}
}
