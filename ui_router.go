package main

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	removedV2UIPathPrefix      = "/v2ui"
	oldUIPathPrefix            = "/old-ui"
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
	group.Use(redirectRootHashRouteFallback(), rootUICacheHeaders())
	fileServer := http.FileServer(http.FS(root))
	group.Match([]string{http.MethodGet, http.MethodHead}, "/", echo.WrapHandler(fileServer))
	group.Match([]string{http.MethodGet, http.MethodHead}, "/*", echo.WrapHandler(fileServer))
	return nil
}

func registerLegacyUI(router *echo.Echo, filesystem fs.FS) {
	group := router.Group(oldUIPathPrefix)
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
