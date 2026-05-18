package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenFromHTTPRequestPrefersBearerAuthorization(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.Header.Set("token", "legacy-header-token")

	token := TokenFromHTTPRequest(req)
	if token != "header-token" {
		t.Fatalf("token = %q, want header-token", token)
	}
}

func TestTokenFromHTTPRequestFallsBackToTokenHeaderAndQuery(t *testing.T) {
	reqWithHeader := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	reqWithHeader.Header.Set("token", "legacy-header-token")
	if token := TokenFromHTTPRequest(reqWithHeader); token != "legacy-header-token" {
		t.Fatalf("header fallback token = %q, want legacy-header-token", token)
	}

	reqWithQuery := httptest.NewRequest(http.MethodGet, "/ws?token=query-token", nil)
	if token := TokenFromHTTPRequest(reqWithQuery); token != "query-token" {
		t.Fatalf("query fallback token = %q, want query-token", token)
	}
}
