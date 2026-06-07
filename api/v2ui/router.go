package v2ui

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

const (
	PathPrefix             = "/v2ui"
	cacheControlNoCache    = "no-cache"
	cacheControlViteAssets = "public, max-age=31536000, immutable"
	distIndexPath          = "dist/index.html"
	placeholderIndexPath   = "placeholder/index.html"
)

func Register(router fiber.Router, source fs.FS) error {
	root, err := selectRoot(source)
	if err != nil {
		return err
	}

	router.Use(PathPrefix, redirectBarePrefix())
	router.Use(PathPrefix, redirectHashRouteFallback())
	router.Use(PathPrefix, v2UICacheHeaders())
	router.Use(PathPrefix, filesystem.New(filesystem.Config{
		Root: http.FS(root),
	}))
	router.Use(PathPrefix, v2UINotFound())
	return nil
}

func redirectBarePrefix() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodGet && c.Path() == PathPrefix {
			return c.Redirect(strings.TrimPrefix(PathPrefix, "/")+"/", fiber.StatusPermanentRedirect)
		}
		return c.Next()
	}
}

func redirectHashRouteFallback() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method != fiber.MethodGet && method != fiber.MethodHead {
			return c.Next()
		}

		relativePath := strings.TrimPrefix(c.Path(), PathPrefix)
		if relativePath == "" || relativePath == "/" || path.Ext(relativePath) != "" {
			return c.Next()
		}

		location := hashRouteFallbackLocation(relativePath, string(c.Request().URI().QueryString()))
		return c.Redirect(location, fiber.StatusPermanentRedirect)
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

func v2UINotFound() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return fiber.ErrNotFound
	}
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

func v2UICacheHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}
		if c.Response().StatusCode() != fiber.StatusOK {
			return nil
		}
		c.Set(fiber.HeaderCacheControl, cacheControlForPath(c.Path()))
		return nil
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
