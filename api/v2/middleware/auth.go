package middleware

import (
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
)

func TokenFromHTTPRequest(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if token != "" && strings.HasPrefix(token, "Bearer ") {
		token = token[7:]
	}
	if token == "" {
		token = r.Header.Get("Token")
	}
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	return token
}

func TokenFromHumaContext(ctx huma.Context) string {
	token := ctx.Header("Authorization")
	if token != "" && strings.HasPrefix(token, "Bearer ") {
		token = token[7:]
	}
	if token == "" {
		token = ctx.Header("token")
	}
	if token == "" {
		token = ctx.Query("token")
	}
	return token
}

func IsAuthorized(d *dice.Dice, token string) bool {
	return d != nil && d.Parent != nil && d.Parent.AccessTokens.Exists(token)
}

func AuthMiddleware(api huma.API, d *dice.Dice) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if IsAuthorized(d, TokenFromHumaContext(ctx)) {
			next(ctx)
			return
		}
		_ = huma.WriteErr(api, ctx, 401, "unauthorized")
	}
}

func WriteProtectedMiddleware(api huma.API, d *dice.Dice) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if !IsAuthorized(d, TokenFromHumaContext(ctx)) {
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
