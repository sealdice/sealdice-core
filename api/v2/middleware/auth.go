package middleware

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
)

func AuthMiddleware(api huma.API, d *dice.Dice) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		token := ctx.Header("Authorization")
		if token != "" {
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
		}
		if token == "" {
			token = ctx.Header("token")
		}
		if token == "" {
			token = ctx.Query("token")
		}
		if d.Parent.AccessTokens.Exists(token) {
			next(ctx)
			return
		}
		_ = huma.WriteErr(api, ctx, 401, "unauthorized")
	}
}

func WriteProtectedMiddleware(api huma.API, d *dice.Dice) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		token := ctx.Header("Authorization")
		if token != "" {
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
		}
		if token == "" {
			token = ctx.Header("token")
		}
		if token == "" {
			token = ctx.Query("token")
		}
		if !d.Parent.AccessTokens.Exists(token) {
			_ = huma.WriteErr(api, ctx, 401, "unauthorized")
			return
		}
		if d.Parent.JustForTest {
			_ = huma.WriteErr(api, ctx, 403, "展示模式不支持该操作")
			return
		}
		next(ctx)
	}
}
