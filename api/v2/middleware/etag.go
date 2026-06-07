package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/etag"
)

const v2DataCacheControl = "private, no-cache"

func V2DataETagMiddleware() fiber.Handler {
	etagHandler := etag.New(etag.Config{
		Next: ShouldSkipV2DataETag,
	})
	return func(c *fiber.Ctx) error {
		if ShouldSkipV2DataETag(c) {
			return c.Next()
		}
		if err := etagHandler(c); err != nil {
			return err
		}
		status := c.Response().StatusCode()
		if status == fiber.StatusOK || status == fiber.StatusNotModified {
			c.Set(fiber.HeaderCacheControl, v2DataCacheControl)
		}
		return nil
	}
}

func ShouldSkipV2DataETag(c *fiber.Ctx) bool {
	method := c.Method()
	if method != fiber.MethodGet && method != fiber.MethodHead {
		return true
	}
	if websocketUpgradeRequested(c) {
		return true
	}
	path := c.Path()
	if path == "/sd-api/v2/realtime/ws" ||
		path == "/sd-api/v2/realtime/sse" ||
		path == "/sd-api/v2/base/logs/ws" ||
		path == "/sd-api/v2/resource/data" {
		return true
	}
	return strings.Contains(path, "/download") ||
		strings.Contains(path, "/export") ||
		strings.Contains(path, "/files/template/")
}
