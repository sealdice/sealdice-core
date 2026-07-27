package v2ui

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

const (
	PathPrefix             = "/v2ui"
	cacheControlNoCache    = "no-cache"
	cacheControlViteAssets = "public, max-age=31536000, immutable"
	distIndexPath          = "dist/index.html"
	placeholderIndexPath   = "placeholder/index.html"
)

func Register(router *echo.Echo, source fs.FS) error {
	root, err := selectRoot(source)
	if err != nil {
		return err
	}

	group := router.Group(PathPrefix)
	group.Use(redirectHashRouteFallback(), v2UICacheHeaders())
	group.StaticFS("/", root)
	router.GET(PathPrefix, redirectBarePrefix())
	return nil
}

func redirectBarePrefix() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, strings.TrimPrefix(PathPrefix, "/")+"/")
	}
}

func redirectHashRouteFallback() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method
			if method != http.MethodGet && method != http.MethodHead {
				return next(c)
			}

			requestPath := c.Request().URL.Path
			relativePath := strings.TrimPrefix(requestPath, PathPrefix)
			if relativePath == "" || relativePath == "/" || path.Ext(relativePath) != "" {
				return next(c)
			}

			location := hashRouteFallbackLocation(relativePath, c.Request().URL.RawQuery)
			return c.Redirect(http.StatusPermanentRedirect, location)
		}
	}
}

func hashRouteFallbackLocation(relativePath string, query string) string {
	cleanPath := strings.Trim(relativePath, "/")
	if cleanPath == "" {
		return "./#/"
	}

	segments := strings.Split(cleanPath, "/")
	depth := len(segments) - 1
	if strings.HasSuffix(relativePath, "/") {
		depth = len(segments)
	}
	if depth < 0 {
		depth = 0
	}

	prefix := "./"
	if depth > 0 {
		prefix = strings.Repeat("../", depth)
	}

	location := prefix + "#/" + cleanPath
	if strings.HasSuffix(relativePath, "/") {
		location += "/"
	}
	if query != "" {
		location += "?" + query
	}
	return location
}

func selectRoot(source fs.FS) (fs.FS, error) {
	if _, err := fs.Stat(source, distIndexPath); err == nil {
		return fs.Sub(source, "dist")
	}
	if _, err := fs.Stat(source, placeholderIndexPath); err == nil {
		return fs.Sub(source, "placeholder")
	}
	return nil, fmt.Errorf("static v2ui: missing %s and %s", distIndexPath, placeholderIndexPath)
}

func v2UICacheHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderCacheControl, cacheControlForPath(c.Request().URL.Path))
			return next(c)
		}
	}
}

func cacheControlForPath(path string) string {
	cleanPath := strings.TrimSuffix(path, "/")
	switch {
	case path == PathPrefix+"/" || cleanPath == PathPrefix || cleanPath == PathPrefix+"/index.html":
		return cacheControlNoCache
	case cleanPath == PathPrefix+"/registerSW.js" ||
		cleanPath == PathPrefix+"/sw.js" ||
		cleanPath == PathPrefix+"/manifest.webmanifest" ||
		strings.HasPrefix(cleanPath, PathPrefix+"/workbox-"):
		return cacheControlNoCache
	case strings.HasPrefix(cleanPath, PathPrefix+"/assets/"):
		return cacheControlViteAssets
	default:
		return cacheControlNoCache
	}
}
