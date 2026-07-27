package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/labstack/echo/v4"

	middleware "sealdice-core/api/v2/middleware"
	"sealdice-core/dice"
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

func TestTokenFromEchoContextPrefersBearerAuthorization(t *testing.T) {
	app := echo.New()
	var token string
	app.GET("/ws", func(c echo.Context) error {
		token = middleware.TokenFromEchoContext(c)
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.Header.Set("Token", "legacy-header-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if token != "header-token" {
		t.Fatalf("token = %q, want header-token", token)
	}
}

func TestTokenFromEchoContextFallsBackToTokenHeaderAndQuery(t *testing.T) {
	app := echo.New()
	var tokens []string
	app.GET("/ws", func(c echo.Context) error {
		tokens = append(tokens, middleware.TokenFromEchoContext(c))
		return c.NoContent(http.StatusNoContent)
	})

	reqWithHeader := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	reqWithHeader.Header.Set("Token", "legacy-header-token")
	recHeader := httptest.NewRecorder()
	app.ServeHTTP(recHeader, reqWithHeader)

	reqWithQuery := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	recQuery := httptest.NewRecorder()
	app.ServeHTTP(recQuery, reqWithQuery)

	if tokens[0] != "legacy-header-token" {
		t.Fatalf("header fallback token = %q, want legacy-header-token", tokens[0])
	}
	if tokens[1] != "query-token" {
		t.Fatalf("query fallback token = %q, want query-token", tokens[1])
	}
}

func TestWriteProtectedMiddlewareReturnsTestModePayload(t *testing.T) {
	dm := &dice.DiceManager{JustForTest: true}
	d := &dice.Dice{Parent: dm}
	dm.Dice = []*dice.Dice{d}
	d.Parent.AccessTokens.Store("token-1", true)
	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token-1")
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, req, rec)

	middleware.WriteProtectedMiddleware(nil, d)(ctx, func(huma.Context) {
		t.Fatal("next should not be called in test mode")
	})

	resp := rec.Result()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	var payload struct {
		Code     string `json:"code"`
		Detail   string `json:"detail"`
		Message  string `json:"message"`
		TestMode bool   `json:"testMode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "TEST_MODE_BLOCKED" {
		t.Fatalf("code = %q, want TEST_MODE_BLOCKED", payload.Code)
	}
	if payload.Detail != "展示模式不支持该操作" {
		t.Fatalf("detail = %q, want 展示模式不支持该操作", payload.Detail)
	}
	if !payload.TestMode {
		t.Fatalf("testMode = false, want true")
	}
}
