package logger

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func FiberLogMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := zap.S().Named(LogKeyWeb)
		start := time.Now()

		err := c.Next()
		stop := time.Now()

		id := c.Get(fiber.HeaderXRequestID)
		if id == "" {
			id = c.GetRespHeader(fiber.HeaderXRequestID)
		}
		reqSize := c.Get(fiber.HeaderContentLength)
		if reqSize == "" {
			reqSize = "0"
		}

		log.Debugf("%s %s [%v] %s %-7s %s %3d %s %s %13v %s %s",
			id,
			c.IP(),
			stop.Format(time.RFC3339),
			c.Hostname(),
			c.Method(),
			c.OriginalURL(),
			c.Response().StatusCode(),
			reqSize,
			strconv.Itoa(len(c.Response().Body())),
			stop.Sub(start).String(),
			c.Get(fiber.HeaderReferer),
			c.Get(fiber.HeaderUserAgent),
		)
		return err
	}
}
