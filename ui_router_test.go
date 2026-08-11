package main

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
)

func TestRegisterRootUIServesEmbeddedAssetsAnd404sRemovedV2UI(t *testing.T) {
	app := echo.New()
	source := fstest.MapFS{
		"dist/index.html": {
			Data: []byte("<!doctype html><title>v2 ui</title>"),
		},
		"dist/assets/app-abc123.js": {
			Data: []byte("console.log('v2-root')"),
		},
		"placeholder/index.html": {
			Data: []byte("<!doctype html><title>placeholder</title>"),
		},
	}

	if err := registerRemovedV2UIRoutes(app); err != nil {
		t.Fatalf("registerRemovedV2UIRoutes() error = %v", err)
	}
	if err := registerRootUI(app, source); err != nil {
		t.Fatalf("registerRootUI() error = %v", err)
	}

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/", nil))
	assertBodyContains(t, resp, "v2 ui")
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("index Cache-Control = %q, want no-cache", got)
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/assets/app-abc123.js", nil))
	assertBodyContains(t, resp, "v2-root")
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset Cache-Control = %q, want immutable asset cache", got)
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/mod/story?tab=detail", nil))
	if resp.StatusCode != http.StatusPermanentRedirect {
		t.Fatalf("history route status = %d, want %d", resp.StatusCode, http.StatusPermanentRedirect)
	}
	if got := resp.Header.Get("Location"); got != "/#/mod/story?tab=detail" {
		t.Fatalf("history route Location = %q, want /#/mod/story?tab=detail", got)
	}

	for _, path := range []string{"/v2ui", "/v2ui/", "/v2ui/assets/app-abc123.js", "/v2ui/mod/story"} {
		resp = performRequest(app, httptest.NewRequest(http.MethodGet, path, nil))
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", path, resp.StatusCode, http.StatusNotFound)
		}
	}
}

func TestRegisterRootUIFallsBackToPlaceholderWhenDistMissing(t *testing.T) {
	app := echo.New()
	source := fstest.MapFS{
		"placeholder/index.html": {
			Data: []byte("<!doctype html><title>placeholder</title>"),
		},
	}

	if err := registerRootUI(app, source); err != nil {
		t.Fatalf("registerRootUI() error = %v", err)
	}

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/", nil))
	assertBodyContains(t, resp, "placeholder")
}

func TestRegisterRootUIReturnsErrorWithoutDistOrPlaceholder(t *testing.T) {
	app := echo.New()
	err := registerRootUI(app, fstest.MapFS{})
	if err == nil {
		t.Fatal("registerRootUI() error = nil, want missing asset error")
	}
	if !strings.Contains(err.Error(), "embedded root ui") {
		t.Fatalf("error = %q, want embedded root ui context", err.Error())
	}
}

func TestRegisterLegacyUIServesFilesUnderOldUI(t *testing.T) {
	app := echo.New()
	source := fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><title>legacy ui</title>"),
		},
		"assets/legacy.js": {
			Data: []byte("console.log('legacy')"),
		},
	}

	registerLegacyUI(app, source)

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/old-ui", nil))
	if resp.StatusCode != http.StatusPermanentRedirect {
		t.Fatalf("bare old ui status = %d, want %d", resp.StatusCode, http.StatusPermanentRedirect)
	}
	if got := resp.Header.Get("Location"); got != "old-ui/" {
		t.Fatalf("bare old ui Location = %q, want old-ui/", got)
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/old-ui/", nil))
	assertBodyContains(t, resp, "legacy ui")

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/old-ui/assets/legacy.js", nil))
	assertBodyContains(t, resp, "legacy")
}

func assertBodyContains(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), want) {
		t.Fatalf("body = %q, want contains %q", string(body), want)
	}
}

var _ fs.FS = fstest.MapFS{}
