package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
)

const (
	removedV2UIPathPrefix      = "/v2ui"
	oldUIPathPrefix            = "/old-ui"
	oldUIAPIPathPrefix         = oldUIPathPrefix + "/sd-api"
	rootUICacheControlNoCache  = "no-cache"
	rootUICacheControlAssets   = "public, max-age=31536000, immutable"
	rootUIDistIndexPath        = "dist/index.html"
	rootUIPlaceholderIndexPath = "placeholder/index.html"
)

var rootUIReservedPrefixes = []string{
	removedV2UIPathPrefix,
	oldUIPathPrefix,
	"/api",
	"/sd-api",
	"/debug/pprof",
	"/docs",
	"/schemas",
}

func registerRemovedV2UIRoutes(router *echo.Echo) error {
	notFound := func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}

	router.Any(removedV2UIPathPrefix, notFound)
	router.Any(removedV2UIPathPrefix+"/*", notFound)
	return nil
}

func registerRootUI(router *echo.Echo, source fs.FS) error {
	root, err := selectRootUIRoot(source)
	if err != nil {
		return err
	}

	group := router.Group("")
	group.Use(redirectRootHashRouteFallback(), rootUICacheHeaders(), rootUIEtagMiddleware(root))
	fileServer := http.FileServer(http.FS(root))
	group.Match([]string{http.MethodGet, http.MethodHead}, "/", echo.WrapHandler(fileServer))
	group.Match([]string{http.MethodGet, http.MethodHead}, "/*", echo.WrapHandler(fileServer))
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
	group.StaticFS("/", filesystem)
	router.GET(oldUIPathPrefix, func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, strings.TrimPrefix(oldUIPathPrefix, "/")+"/")
	})
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

func redirectRootHashRouteFallback() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method
			if method != http.MethodGet && method != http.MethodHead {
				return next(c)
			}

			requestPath := c.Request().URL.Path
			if shouldBypassRootUI(requestPath) {
				return echo.NewHTTPError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
			}
			if requestPath == "/" || path.Ext(requestPath) != "" {
				return next(c)
			}

			return c.Redirect(http.StatusPermanentRedirect, rootHashRouteFallbackLocation(requestPath, c.Request().URL.RawQuery))
		}
	}
}

func rootHashRouteFallbackLocation(requestPath string, query string) string {
	cleanPath := strings.Trim(requestPath, "/")
	location := "/"
	if cleanPath != "" {
		location = "/#/" + cleanPath
		if strings.HasSuffix(requestPath, "/") {
			location += "/"
		}
	}
	if query != "" {
		location += "?" + query
	}
	return location
}

func shouldBypassRootUI(requestPath string) bool {
	for _, prefix := range rootUIReservedPrefixes {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}

func rootUICacheHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderCacheControl, cacheControlForRootUIPath(c.Request().URL.Path))
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

// rootUIEtagMiddleware sets a weak, content-derived ETag on the revalidated
// entry files and short-circuits unchanged revalidations with 304. Embedded
// files all share an epoch modtime, so Last-Modified alone cannot detect an
// updated fixed-name file (e.g. sw.js); a content hash can.
func rootUIEtagMiddleware(root fs.FS) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				return next(c)
			}
			if !isRootUINoCachedRevalidatePath(req.URL.Path) {
				return next(c)
			}

			relPath := strings.TrimPrefix(req.URL.Path, "/")
			if relPath == "" {
				relPath = "index.html"
			}
			etag, ok := cachedRootUIEtag(root, relPath)
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
