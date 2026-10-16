package main

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
)

func TestRootRedirectsToOldUIByDefault(t *testing.T) {
	app := echo.New()
	registerRootDefaultUI(app)

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("root status = %d, want %d", resp.StatusCode, http.StatusFound)
	}
	if got := resp.Header.Get("Location"); got != oldUIPathPrefix+"/" {
		t.Fatalf("root Location = %q, want %q", got, oldUIPathPrefix+"/")
	}
}

func TestRootDefaultUISwitchByQueryAndCookie(t *testing.T) {
	newApp := func() *echo.Echo {
		app := echo.New()
		registerRootDefaultUI(app)
		return app
	}

	// ?ui=new：记住新版并跳 /v2ui/。
	resp := performRequest(newApp(), httptest.NewRequest(http.MethodGet, "/?ui=new", nil))
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != v2UIPathPrefix+"/" {
		t.Fatalf("?ui=new => %d %q, want 302 %q", resp.StatusCode, resp.Header.Get("Location"), v2UIPathPrefix+"/")
	}
	if got := cookieValue(resp, defaultUICookieName); got != defaultUICookieValueNew {
		t.Fatalf("?ui=new cookie = %q, want %q", got, defaultUICookieValueNew)
	}

	// ?ui=old：记住旧版并跳 /old-ui/。
	resp = performRequest(newApp(), httptest.NewRequest(http.MethodGet, "/?ui=old", nil))
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != oldUIPathPrefix+"/" {
		t.Fatalf("?ui=old => %d %q, want 302 %q", resp.StatusCode, resp.Header.Get("Location"), oldUIPathPrefix+"/")
	}
	if got := cookieValue(resp, defaultUICookieName); got != defaultUICookieValueOld {
		t.Fatalf("?ui=old cookie = %q, want %q", got, defaultUICookieValueOld)
	}

	// cookie=new：默认跳到新版。
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: defaultUICookieName, Value: defaultUICookieValueNew})
	resp = performRequest(newApp(), req)
	if resp.Header.Get("Location") != v2UIPathPrefix+"/" {
		t.Fatalf("cookie=new Location = %q, want %q", resp.Header.Get("Location"), v2UIPathPrefix+"/")
	}
}

func TestRegisterV2UIServesIndexAndAssetsUnderPrefix(t *testing.T) {
	app := echo.New()
	source := fstest.MapFS{
		"dist/index.html": {
			Data: []byte("<!doctype html><title>v2 ui</title>"),
		},
		"dist/assets/app-abc123.js": {
			Data: []byte("console.log('v2')"),
		},
	}
	if err := registerV2UI(app, source); err != nil {
		t.Fatalf("registerV2UI() error = %v", err)
	}

	// 裸 /v2ui 跳带斜杠形式。
	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/v2ui", nil))
	if resp.StatusCode != http.StatusPermanentRedirect || resp.Header.Get("Location") != v2UIPathPrefix+"/" {
		t.Fatalf("/v2ui => %d %q, want 308 %q", resp.StatusCode, resp.Header.Get("Location"), v2UIPathPrefix+"/")
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/v2ui/", nil))
	assertBodyContains(t, resp, "v2 ui")
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("index Cache-Control = %q, want no-cache", got)
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/v2ui/assets/app-abc123.js", nil))
	assertBodyContains(t, resp, "console.log('v2')")
	if got := resp.Header.Get("Cache-Control"); got != rootUICacheControlAssets {
		t.Fatalf("asset Cache-Control = %q, want immutable asset cache", got)
	}
}

func TestRegisterV2UIEtagForRevalidatedEntries(t *testing.T) {
	rootUIEtagCache = sync.Map{}

	app := echo.New()
	source := fstest.MapFS{
		"dist/index.html":        {Data: []byte("<!doctype html>v2 ui")},
		"dist/sw.js":             {Data: []byte("/* service worker */")},
		"dist/assets/app-abc.js": {Data: []byte("console.log(1)")},
	}
	if err := registerV2UI(app, source); err != nil {
		t.Fatalf("registerV2UI() error = %v", err)
	}

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/v2ui/", nil))
	etag := resp.Header.Get("ETag")
	if etag == "" || !strings.HasPrefix(etag, `W/"`) {
		t.Fatalf("index ETag = %q, want weak ETag", etag)
	}

	req := httptest.NewRequest(http.MethodGet, "/v2ui/", nil)
	req.Header.Set("If-None-Match", etag)
	resp = performRequest(app, req)
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidation status = %d, want %d", resp.StatusCode, http.StatusNotModified)
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/v2ui/assets/app-abc.js", nil))
	if got := resp.Header.Get("ETag"); got != "" {
		t.Fatalf("asset ETag = %q, want empty", got)
	}
}

func TestRegisterV2UIFallsBackToPlaceholderWhenDistMissing(t *testing.T) {
	app := echo.New()
	if err := registerV2UI(app, fstest.MapFS{
		"placeholder/index.html": {Data: []byte("<!doctype html><title>placeholder</title>")},
	}); err != nil {
		t.Fatalf("registerV2UI() error = %v", err)
	}

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/v2ui/", nil))
	assertBodyContains(t, resp, "placeholder")
}

func TestRegisterV2UIReturnsErrorWithoutDistOrPlaceholder(t *testing.T) {
	err := registerV2UI(echo.New(), fstest.MapFS{})
	if err == nil {
		t.Fatal("registerV2UI() error = nil, want missing asset error")
	}
	if !strings.Contains(err.Error(), "embedded root ui") {
		t.Fatalf("error = %q, want embedded root ui context", err.Error())
	}
}

func TestRegisterLegacyUIServesFilesUnderOldUI(t *testing.T) {
	app := echo.New()
	source := fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><html><body><title>legacy ui</title></body></html>"),
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
	body := readBody(t, resp)
	if !strings.Contains(body, legacyUIBackLinkMarker) || !strings.Contains(body, "切换到新版 UI") {
		t.Fatalf("old ui index missing injected switch entry: %q", body)
	}
	if !strings.Contains(body, "legacy ui") {
		t.Fatalf("old ui index lost original content: %q", body)
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/old-ui/index.html", nil))
	if got := readBody(t, resp); !strings.Contains(got, legacyUIBackLinkMarker) {
		t.Fatalf("old ui index.html missing injected switch entry: %q", got)
	}

	resp = performRequest(app, httptest.NewRequest(http.MethodGet, "/old-ui/assets/legacy.js", nil))
	if body := readBody(t, resp); strings.Contains(body, legacyUIBackLinkMarker) {
		t.Fatal("static asset should not be injected with the switch entry")
	}
}

func TestRegisterLegacyUIRewritesLegacyAPIPath(t *testing.T) {
	app := echo.New()
	app.GET("/sd-api/signin/salt", func(c echo.Context) error {
		return c.String(http.StatusOK, "salt")
	})

	registerLegacyUI(app, fstest.MapFS{
		"index.html": {Data: []byte("legacy ui")},
	})

	resp := performRequest(app, httptest.NewRequest(http.MethodGet, "/old-ui/sd-api/signin/salt", nil))
	assertBodyContains(t, resp, "salt")
}

func TestInjectLegacyBackLinkIsIdempotentAndPreservesContent(t *testing.T) {
	html := "<!doctype html><html><body><div>legacy</div></body></html>"
	once := injectLegacyBackLink(html)
	if !strings.Contains(once, legacyUIBackLinkMarker) {
		t.Fatal("injected html missing switch marker")
	}
	if !strings.Contains(once, `href="/?ui=new"`) {
		t.Fatal("switch entry should target /?ui=new")
	}
	if !strings.Contains(once, "<div>legacy</div>") {
		t.Fatal("injected html dropped original content")
	}
	if strings.Count(once, legacyUIBackLinkMarker) != 1 {
		t.Fatalf("marker count = %d, want 1", strings.Count(once, legacyUIBackLinkMarker))
	}
	twice := injectLegacyBackLink(once)
	if strings.Count(twice, legacyUIBackLinkMarker) != 1 {
		t.Fatalf("second injection duplicated marker: count = %d", strings.Count(twice, legacyUIBackLinkMarker))
	}
}

func TestInjectLegacyBackLinkAppendsWithoutBodyTag(t *testing.T) {
	out := injectLegacyBackLink("<div>legacy</div>")
	if !strings.HasSuffix(out, "切换到新版 UI</a>") {
		t.Fatalf("html without </body> should append snippet, got %q", out)
	}
}

func cookieValue(resp *http.Response, name string) string {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	return string(body)
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
