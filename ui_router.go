package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
)

const (
	// 新版 UI 挂在 /v2ui/ 下；旧版 UI 挂在 /old-ui/ 下并作为默认入口。
	v2UIPathPrefix             = "/v2ui"
	oldUIPathPrefix            = "/old-ui"
	oldUIAPIPathPrefix         = oldUIPathPrefix + "/sd-api"
	rootUICacheControlNoCache  = "no-cache"
	rootUICacheControlAssets   = "public, max-age=31536000, immutable"
	rootUIDistIndexPath        = "dist/index.html"
	rootUIPlaceholderIndexPath = "placeholder/index.html"

	// 默认 UI 偏好按浏览器持久化：默认旧版 UI，?ui=new / cookie 可切到新版。
	defaultUICookieName     = "sd_default_ui"
	defaultUICookieValueNew = "new"
	defaultUICookieValueOld = "old"
	defaultUICookieMaxAge   = 31536000 // 1 年
)

// registerRootDefaultUI 处理入口 `/`：默认落旧版 UI，可通过 ?ui=new（并记住）
// 或之前的 cookie 切到新版 UI。旧版 UI 在 /old-ui/，新版 UI 在 /v2ui/。
func registerRootDefaultUI(router *echo.Echo) {
	router.Match([]string{http.MethodGet, http.MethodHead}, "/", redirectDefaultUI)
}

func redirectDefaultUI(c echo.Context) error {
	switch c.QueryParam("ui") {
	case defaultUICookieValueNew:
		setDefaultUICookie(c, defaultUICookieValueNew)
		return c.Redirect(http.StatusFound, v2UIPathPrefix+"/")
	case defaultUICookieValueOld:
		setDefaultUICookie(c, defaultUICookieValueOld)
		return c.Redirect(http.StatusFound, oldUIPathPrefix+"/")
	}
	if cookie, err := c.Cookie(defaultUICookieName); err == nil && cookie.Value == defaultUICookieValueNew {
		return c.Redirect(http.StatusFound, v2UIPathPrefix+"/")
	}
	return c.Redirect(http.StatusFound, oldUIPathPrefix+"/")
}

func setDefaultUICookie(c echo.Context, value string) {
	// #nosec G124 -- This non-sensitive UI preference must also work over local HTTP.
	c.SetCookie(&http.Cookie{
		Name:     defaultUICookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   defaultUICookieMaxAge,
		Secure:   c.Scheme() == "https",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// registerV2UI 在 /v2ui/ 下提供新版 UI。
func registerV2UI(router *echo.Echo, source fs.FS) error {
	root, err := selectRootUIRoot(source)
	if err != nil {
		return err
	}

	group := router.Group(v2UIPathPrefix)
	group.Use(v2UICacheHeaders(), v2UIEtagMiddleware(root))
	handler := echo.WrapHandler(http.StripPrefix(v2UIPathPrefix, http.FileServer(http.FS(root))))
	group.Match([]string{http.MethodGet, http.MethodHead}, "/", handler)
	group.Match([]string{http.MethodGet, http.MethodHead}, "/*", handler)

	router.GET(v2UIPathPrefix, func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, v2UIPathPrefix+"/")
	})
	return nil
}

func registerLegacyUI(router *echo.Echo, filesystem fs.FS) {
	// The legacy bundle derives its API base from the current pathname, so it
	// requests /old-ui/sd-api/* when served below /old-ui/. Route those requests
	// through the existing /sd-api handlers without changing the public UI path.
	router.Pre(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestPath := c.Request().URL.Path
			if requestPath == oldUIAPIPathPrefix || strings.HasPrefix(requestPath, oldUIAPIPathPrefix+"/") {
				c.Request().URL.Path = strings.TrimPrefix(requestPath, oldUIPathPrefix)
				c.Request().URL.RawPath = ""
			}
			return next(c)
		}
	})

	group := router.Group(oldUIPathPrefix)
	group.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cleanPath := strings.TrimSuffix(c.Request().URL.Path, "/")
			if cleanPath == oldUIPathPrefix || cleanPath == oldUIPathPrefix+"/index.html" {
				c.Response().Header().Set(echo.HeaderCacheControl, "no-cache, no-store, must-revalidate")
				c.Response().Header().Set("Pragma", "no-cache")
				c.Response().Header().Set("Expires", "0")
			}
			return next(c)
		}
	})
	group.Use(legacyUIBackLinkMiddleware(filesystem))
	group.StaticFS("/", filesystem)
	router.GET(oldUIPathPrefix, func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, strings.TrimPrefix(oldUIPathPrefix, "/")+"/")
	})
}

// legacyUIBackLinkMarker 标记注入到旧 UI 首页的「切换到新版 UI」入口，避免重复注入。
const legacyUIBackLinkMarker = `id="sd-legacy-switch-to-new-ui"`

// injectLegacyBackLink 在旧 UI 的 index.html 末尾注入一个固定的「切换到新版 UI」入口。
//
// 旧 UI 是构建产物（由 static/gen/download-fe.go 从 sealdice-ui release 下载），
// 仓库没有其源码、无法重新构建，所以在服务端渲染时注入，而不是改源码。
// 目标用 `/?ui=new`：显式 ui=new 会记住偏好并跳到 /v2ui/。
func injectLegacyBackLink(html string) string {
	if strings.Contains(html, legacyUIBackLinkMarker) {
		return html
	}
	snippet := `<a ` + legacyUIBackLinkMarker +
		` href="/?ui=new" style="position:fixed;right:16px;bottom:16px;z-index:2147483647;` +
		`display:inline-flex;align-items:center;gap:6px;padding:8px 14px;border-radius:999px;` +
		`background:#3478f6;color:#fff;font:600 13px/1.4 system-ui,-apple-system,'Segoe UI',sans-serif;` +
		`text-decoration:none;box-shadow:0 6px 18px rgba(0,0,0,.28)">切换到新版 UI</a>`
	if idx := strings.LastIndex(html, "</body>"); idx >= 0 {
		return html[:idx] + snippet + html[idx:]
	}
	return html + snippet
}

// legacyUIBackLinkMiddleware 拦截旧 UI 首页，注入「切换到新版 UI」入口。
// 其它静态资源原样交给 StaticFS。
func legacyUIBackLinkMiddleware(filesystem fs.FS) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodGet && c.Request().Method != http.MethodHead {
				return next(c)
			}
			requestPath := c.Request().URL.Path
			if requestPath != oldUIPathPrefix+"/" && requestPath != oldUIPathPrefix+"/index.html" {
				return next(c)
			}
			raw, err := fs.ReadFile(filesystem, "index.html")
			if err != nil {
				return next(c)
			}
			c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
			if c.Request().Method == http.MethodHead {
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			return c.HTMLBlob(http.StatusOK, []byte(injectLegacyBackLink(string(raw))))
		}
	}
}

func selectRootUIRoot(source fs.FS) (fs.FS, error) {
	if _, err := fs.Stat(source, rootUIDistIndexPath); err == nil {
		return fs.Sub(source, "dist")
	}
	if _, err := fs.Stat(source, rootUIPlaceholderIndexPath); err == nil {
		return fs.Sub(source, "placeholder")
	}
	return nil, fmt.Errorf("embedded root ui: missing %s and %s", rootUIDistIndexPath, rootUIPlaceholderIndexPath)
}

// v2UICacheHeaders 按 /v2ui 前缀剥离后再按路径设置缓存策略。
func v2UICacheHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rel := strings.TrimPrefix(c.Request().URL.Path, v2UIPathPrefix)
			c.Response().Header().Set(echo.HeaderCacheControl, cacheControlForRootUIPath(rel))
			return next(c)
		}
	}
}

// v2UIEtagMiddleware 为 /v2ui 下的可重验证入口文件设置内容派生的弱 ETag。
func v2UIEtagMiddleware(root fs.FS) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				return next(c)
			}
			rel := strings.TrimPrefix(req.URL.Path, v2UIPathPrefix)
			if !isRootUINoCachedRevalidatePath(rel) {
				return next(c)
			}
			filePath := strings.TrimPrefix(rel, "/")
			if filePath == "" {
				filePath = "index.html"
			}
			etag, ok := cachedRootUIEtag(root, filePath)
			if !ok {
				return next(c)
			}
			c.Response().Header().Set("ETag", etag)
			if ifNoneMatch := req.Header.Get("If-None-Match"); ifNoneMatch != "" && strings.Contains(ifNoneMatch, etag) {
				c.Response().WriteHeader(http.StatusNotModified)
				return nil
			}
			return next(c)
		}
	}
}

func cacheControlForRootUIPath(requestPath string) string {
	cleanPath := strings.TrimSuffix(requestPath, "/")
	switch {
	case requestPath == "/" || cleanPath == "/index.html":
		return rootUICacheControlNoCache
	case cleanPath == "/registerSW.js" ||
		cleanPath == "/sw.js" ||
		cleanPath == "/manifest.webmanifest" ||
		strings.HasPrefix(cleanPath, "/workbox-"):
		return rootUICacheControlNoCache
	case strings.HasPrefix(cleanPath, "/assets/"):
		return rootUICacheControlAssets
	default:
		return rootUICacheControlNoCache
	}
}

// isRootUINoCachedRevalidatePath reports whether a path is served with
// "no-cache" (revalidated on every request) and thus needs a content-based ETag.
func isRootUINoCachedRevalidatePath(requestPath string) bool {
	cleanPath := strings.TrimSuffix(requestPath, "/")
	switch {
	case requestPath == "/" || cleanPath == "/index.html":
		return true
	case cleanPath == "/registerSW.js" ||
		cleanPath == "/sw.js" ||
		cleanPath == "/manifest.webmanifest" ||
		strings.HasPrefix(cleanPath, "/workbox-"):
		return true
	default:
		return false
	}
}

var rootUIEtagCache sync.Map // path -> etag

// cachedRootUIEtag sets a weak, content-derived ETag on the revalidated
// entry files and short-circuits unchanged revalidations with 304. Embedded
// files all share an epoch modtime, so Last-Modified alone cannot detect an
// updated fixed-name file (e.g. sw.js); a content hash can.
func cachedRootUIEtag(root fs.FS, relPath string) (string, bool) {
	if value, ok := rootUIEtagCache.Load(relPath); ok {
		return value.(string), true
	}

	file, err := root.Open(relPath)
	if err != nil {
		return "", false
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", false
	}
	etag := `W/"` + hex.EncodeToString(hash.Sum(nil)) + `"`
	rootUIEtagCache.Store(relPath, etag)
	return etag, true
}
