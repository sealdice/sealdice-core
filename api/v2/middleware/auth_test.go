package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	middleware "sealdice-core/api/v2/middleware"
)

func TestTokenFromHTTPRequestPrefersBearerAuthorization(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.Header.Set("Token", "legacy-header-token")

	token := middleware.TokenFromHTTPRequest(req)
	if token != "header-token" {
		t.Fatalf("token = %q, want header-token", token)
	}
}

func TestTokenFromHTTPRequestFallsBackToTokenHeaderAndQuery(t *testing.T) {
	reqWithHeader := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	reqWithHeader.Header.Set("Token", "legacy-header-token")
	if token := middleware.TokenFromHTTPRequest(reqWithHeader); token != "legacy-header-token" {
		t.Fatalf("header fallback token = %q, want legacy-header-token", token)
	}

	reqWithQuery := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	if token := middleware.TokenFromHTTPRequest(reqWithQuery); token != "query-token" {
		t.Fatalf("query fallback token = %q, want query-token", token)
	}
}

func TestTokenFromFiberCtxPrefersBearerAuthorization(t *testing.T) {
	app := fiber.New()
	var token string
	app.Get("/ws", func(c *fiber.Ctx) error {
		token = middleware.TokenFromFiberCtx(c)
		return c.SendStatus(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.Header.Set("Token", "legacy-header-token")
	if _, err := app.Test(req); err != nil {
		t.Fatalf("fiber test request: %v", err)
	}

	if token != "header-token" {
		t.Fatalf("token = %q, want header-token", token)
	}
}

func TestTokenFromFiberCtxFallsBackToTokenHeaderAndQuery(t *testing.T) {
	app := fiber.New()
	var tokens []string
	app.Get("/ws", func(c *fiber.Ctx) error {
		tokens = append(tokens, middleware.TokenFromFiberCtx(c))
		return c.SendStatus(http.StatusNoContent)
	})

	reqWithHeader := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	reqWithHeader.Header.Set("Token", "legacy-header-token")
	if _, err := app.Test(reqWithHeader); err != nil {
		t.Fatalf("fiber header request: %v", err)
	}

	reqWithQuery := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	if _, err := app.Test(reqWithQuery); err != nil {
		t.Fatalf("fiber query request: %v", err)
	}

	if tokens[0] != "legacy-header-token" {
		t.Fatalf("header fallback token = %q, want legacy-header-token", tokens[0])
	}
	if tokens[1] != "query-token" {
		t.Fatalf("query fallback token = %q, want query-token", tokens[1])
	}
}
