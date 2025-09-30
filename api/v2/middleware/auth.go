package middleware

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
)

func AuthMiddleware(d *dice.Dice) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		token := ctx.Header("token")
		if token == "" {
			token = ctx.Query("token")
		}
		if d.Parent.AccessTokens.Exists(token) {
			next(ctx)
		}
		// 失败登录 400 错误
	}
}
