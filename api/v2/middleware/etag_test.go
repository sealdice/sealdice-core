package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	middleware "sealdice-core/api/v2/middleware"
)

func TestV2DataETagMiddlewareCachesJsonData(t *testing.T) {
	app := fiber.New()
	app.Use("/sd-api/v2", middleware.V2DataETagMiddleware())
	app.Get("/sd-api/v2/base/health", func(c *fiber.Ctx) error {
		c.Type("json")
		return c.SendString(`{"status":"ok"}`)
	})

	first := requestFiber(t, app, http.MethodGet, "/sd-api/v2/base/health")
	etag := first.Header.Get("Etag")
	if etag == "" {
		t.Fatal("Etag is empty")
	}
	if got := first.Header.Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("Cache-Control = %q, want private, no-cache", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/sd-api/v2/base/health", nil)
	req.Header.Set("If-None-Match", etag)
	second, err := app.Test(req)
	if err != nil {
		t.Fatalf("conditional request: %v", err)
	}
	defer second.Body.Close()

	if second.StatusCode != fiber.StatusNotModified {
		t.Fatalf("conditional status = %d, want %d", second.StatusCode, fiber.StatusNotModified)
	}
	if got := second.Header.Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("304 Cache-Control = %q, want private, no-cache", got)
	}
}

func TestV2DataETagMiddlewareSkipsStreamsDownloadsAndUnsafeMethods(t *testing.T) {
	for _, tt := range []struct {
		name   string
		method string
		path   string
		setup  func(*fiber.Ctx) error
	}{
		{
			name:   "post data mutation",
			method: http.MethodPost,
			path:   "/sd-api/v2/base/login",
			setup:  func(c *fiber.Ctx) error { return c.SendString(`{"ok":true}`) },
		},
		{
			name:   "realtime websocket",
			method: http.MethodGet,
			path:   "/sd-api/v2/realtime/ws",
			setup:  func(c *fiber.Ctx) error { return c.SendString("ws") },
		},
		{
			name:   "event stream",
			method: http.MethodGet,
			path:   "/sd-api/v2/realtime/sse",
			setup: func(c *fiber.Ctx) error {
				c.Set(fiber.HeaderContentType, "text/event-stream")
				return c.SendString("event: system/ready\n\n")
			},
		},
		{
			name:   "parquet export",
			method: http.MethodGet,
			path:   "/sd-api/v2/story/log/export-parquet",
			setup: func(c *fiber.Ctx) error {
				c.Set(fiber.HeaderContentDisposition, `attachment; filename="log.parquet"`)
				return c.SendString("parquet")
			},
		},
		{
			name:   "resource download",
			method: http.MethodGet,
			path:   "/sd-api/v2/resource/download",
			setup: func(c *fiber.Ctx) error {
				c.Set(fiber.HeaderContentDisposition, `attachment; filename="resource.bin"`)
				return c.SendString("resource")
			},
		},
		{
			name:   "resource data stream",
			method: http.MethodGet,
			path:   "/sd-api/v2/resource/data",
			setup:  func(c *fiber.Ctx) error { return c.SendString("resource-data") },
		},
		{
			name:   "template file",
			method: http.MethodGet,
			path:   "/sd-api/v2/censor/files/template/toml",
			setup:  func(c *fiber.Ctx) error { return c.SendString("template") },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use("/sd-api/v2", middleware.V2DataETagMiddleware())
			app.Add(tt.method, tt.path, tt.setup)

			resp := requestFiber(t, app, tt.method, tt.path)
			if got := resp.Header.Get("Etag"); got != "" {
				t.Fatalf("Etag = %q, want empty", got)
			}
			if got := resp.Header.Get("Cache-Control"); got != "" {
				t.Fatalf("Cache-Control = %q, want empty", got)
			}
		})
	}
}

func requestFiber(t *testing.T, app *fiber.App, method string, path string) *http.Response {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(method, path, nil))
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})
	return resp
}
