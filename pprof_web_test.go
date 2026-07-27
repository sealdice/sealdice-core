package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRegisterPprofWebRoutesSkipsDisabledConfig(t *testing.T) {
	app := echo.New()
	registerPprofWebRoutes(app, &pprofWebConfig{})

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestRegisterPprofWebRoutesRequiresToken(t *testing.T) {
	app := echo.New()
	registerPprofWebRoutes(app, &pprofWebConfig{
		Enabled: true,
		Token:   "test-token",
	})

	unauthorizedResp := performRequest(app, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if unauthorizedResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorizedResp.StatusCode, http.StatusUnauthorized)
	}

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/?token=test-token", nil)
	authorizedResp := performRequest(app, req)
	if authorizedResp.StatusCode != http.StatusOK {
		t.Fatalf("authorized status = %d, want %d", authorizedResp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(authorizedResp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), "/debug/pprof/") {
		t.Fatalf("authorized body = %q, want pprof index content", string(body))
	}

	heapReq := httptest.NewRequest(http.MethodGet, "/debug/pprof/heap?token=test-token", nil)
	heapResp := performRequest(app, heapReq)
	if heapResp.StatusCode != http.StatusOK {
		t.Fatalf("heap status = %d, want %d", heapResp.StatusCode, http.StatusOK)
	}
}

func performRequest(app *echo.Echo, req *http.Request) *http.Response {
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec.Result()
}
