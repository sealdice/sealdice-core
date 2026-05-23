package v2ui_test

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v2"

	"sealdice-core/api/v2ui"
)

func TestRegisterServesV2UIWithCachePolicy(t *testing.T) {
	app := fiber.New()
	source := fstest.MapFS{
		"dist/index.html": {
			Data: []byte("<!doctype html><title>v2 ui</title>"),
		},
		"dist/assets/app-abc123.js": {
			Data: []byte("console.log('v2ui')"),
		},
		"placeholder/index.html": {
			Data: []byte("<!doctype html><title>placeholder</title>"),
		},
	}

	if err := v2ui.Register(app, source); err != nil {
		t.Fatalf("register v2ui: %v", err)
	}

	resp := request(t, app, http.MethodGet, "/v2ui")
	if resp.StatusCode != fiber.StatusPermanentRedirect {
		t.Fatalf("/v2ui status = %d, want %d", resp.StatusCode, fiber.StatusPermanentRedirect)
	}
	if got := resp.Header.Get("Location"); got != "v2ui/" {
		t.Fatalf("Location = %q, want v2ui/", got)
	}

	resp = request(t, app, http.MethodGet, "/v2ui/")
	assertBodyContains(t, resp, "v2 ui")
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("index Cache-Control = %q, want no-cache", got)
	}

	resp = request(t, app, http.MethodGet, "/v2ui/assets/app-abc123.js")
	assertBodyContains(t, resp, "v2ui")
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset Cache-Control = %q, want immutable asset cache", got)
	}

	resp = request(t, app, http.MethodGet, "/v2ui/mod/story?tab=detail")
	if resp.StatusCode != fiber.StatusPermanentRedirect {
		t.Fatalf("history route status = %d, want %d", resp.StatusCode, fiber.StatusPermanentRedirect)
	}
	if got := resp.Header.Get("Location"); got != "../#/mod/story?tab=detail" {
		t.Fatalf("history route Location = %q, want ../#/mod/story?tab=detail", got)
	}

	resp = request(t, app, http.MethodGet, "/v2ui/mod")
	if resp.StatusCode != fiber.StatusPermanentRedirect {
		t.Fatalf("single segment history route status = %d, want %d", resp.StatusCode, fiber.StatusPermanentRedirect)
	}
	if got := resp.Header.Get("Location"); got != "./#/mod" {
		t.Fatalf("single segment history route Location = %q, want ./#/mod", got)
	}

	resp = request(t, app, http.MethodGet, "/v2ui/assets/missing.js")
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("missing asset status = %d, want %d", resp.StatusCode, fiber.StatusNotFound)
	}
}

func TestRegisterFallsBackToPlaceholderWhenDistMissing(t *testing.T) {
	app := fiber.New()
	source := fstest.MapFS{
		"placeholder/index.html": {
			Data: []byte("<!doctype html><title>placeholder</title>"),
		},
	}

	if err := v2ui.Register(app, source); err != nil {
		t.Fatalf("register v2ui: %v", err)
	}

	resp := request(t, app, http.MethodGet, "/v2ui/")
	assertBodyContains(t, resp, "placeholder")
}

func TestRegisterReturnsErrorWithoutDistOrPlaceholder(t *testing.T) {
	app := fiber.New()
	err := v2ui.Register(app, fstest.MapFS{})
	if err == nil {
		t.Fatal("Register returned nil, want error")
	}
	if !strings.Contains(err.Error(), "static v2ui") {
		t.Fatalf("error = %q, want static v2ui context", err.Error())
	}
}

func request(t *testing.T, app *fiber.App, method string, path string) *http.Response {
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

func assertBodyContains(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), want) {
		t.Fatalf("body = %q, want contains %q", string(body), want)
	}
}

var _ fs.FS = fstest.MapFS{}
