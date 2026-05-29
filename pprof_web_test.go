package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
)

func TestRegisterPprofWebRoutesSkipsDisabledConfig(t *testing.T) {
	app := fiber.New()
	registerPprofWebRoutes(app, &pprofWebConfig{})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestRegisterPprofWebRoutesRequiresToken(t *testing.T) {
	app := fiber.New()
	registerPprofWebRoutes(app, &pprofWebConfig{
		Enabled: true,
		Token:   "test-token",
	})

	unauthorizedResp, err := app.Test(httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if err != nil {
		t.Fatalf("unauthorized app.Test() error = %v", err)
	}
	if unauthorizedResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorizedResp.StatusCode, http.StatusUnauthorized)
	}

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/?token=test-token", nil)
	authorizedResp, err := app.Test(req)
	if err != nil {
		t.Fatalf("authorized app.Test() error = %v", err)
	}
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
	heapResp, err := app.Test(heapReq)
	if err != nil {
		t.Fatalf("heap app.Test() error = %v", err)
	}
	if heapResp.StatusCode != http.StatusOK {
		t.Fatalf("heap status = %d, want %d", heapResp.StatusCode, http.StatusOK)
	}
}

func TestRegisterPprofWebRoutesBypassesGlobalCompression(t *testing.T) {
	app := fiber.New()
	registerPprofWebRoutes(app, &pprofWebConfig{
		Enabled: true,
		Token:   "test-token",
	})
	app.Use(compress.New())

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/?token=test-token", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}
