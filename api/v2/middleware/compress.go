package middleware

import (
	"bytes"
	"compress/gzip"
	"strconv"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/gofiber/fiber/v2"
	"github.com/klauspost/compress/zstd"
)

const (
	encodingZstd = "zstd"
	encodingBr   = "br"
	encodingGzip = "gzip"
)

func PreferredCompressionMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if shouldSkipCompressionRequest(c) {
			return c.Next()
		}

		encoding := preferredEncoding(c.Get(fiber.HeaderAcceptEncoding))
		if encoding == "" {
			return c.Next()
		}

		if err := c.Next(); err != nil {
			return err
		}
		if shouldSkipCompressionResponse(c) {
			return nil
		}

		body := c.Response().Body()
		if len(body) == 0 {
			return nil
		}
		compressed, err := compressBody(body, encoding)
		if err != nil {
			return err
		}

		c.Response().SetBodyRaw(compressed)
		c.Set(fiber.HeaderContentEncoding, encoding)
		c.Set(fiber.HeaderVary, appendVary(c.GetRespHeader(fiber.HeaderVary), fiber.HeaderAcceptEncoding))
		c.Response().Header.Del(fiber.HeaderContentLength)
		return nil
	}
}

func shouldSkipCompressionRequest(c *fiber.Ctx) bool {
	if c.Method() == fiber.MethodHead {
		return true
	}
	return websocketUpgradeRequested(c)
}

func shouldSkipCompressionResponse(c *fiber.Ctx) bool {
	if c.GetRespHeader(fiber.HeaderContentEncoding) != "" {
		return true
	}
	status := c.Response().StatusCode()
	if status < fiber.StatusOK || status == fiber.StatusNoContent || status == fiber.StatusNotModified || status >= fiber.StatusMultipleChoices {
		return true
	}
	contentType := strings.ToLower(c.GetRespHeader(fiber.HeaderContentType))
	return strings.HasPrefix(contentType, "text/event-stream")
}

func websocketUpgradeRequested(c *fiber.Ctx) bool {
	connection := strings.ToLower(c.Get(fiber.HeaderConnection))
	upgrade := strings.ToLower(c.Get(fiber.HeaderUpgrade))
	return strings.Contains(connection, "upgrade") && upgrade == "websocket"
}

func preferredEncoding(acceptEncoding string) string {
	for _, encoding := range []string{encodingZstd, encodingBr, encodingGzip} {
		if acceptsEncoding(acceptEncoding, encoding) {
			return encoding
		}
	}
	return ""
}

func acceptsEncoding(header string, encoding string) bool {
	for _, raw := range strings.Split(header, ",") {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		fields := strings.Split(part, ";")
		name := strings.ToLower(strings.TrimSpace(fields[0]))
		if name != encoding && name != "*" {
			continue
		}
		if encodingQualityIsZero(fields[1:]) {
			return false
		}
		return true
	}
	return false
}

func encodingQualityIsZero(params []string) bool {
	for _, param := range params {
		key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
			continue
		}
		q, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return err == nil && q <= 0
	}
	return false
}

func compressBody(body []byte, encoding string) ([]byte, error) {
	var buf bytes.Buffer
	switch encoding {
	case encodingZstd:
		writer, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write(body); err != nil {
			_ = writer.Close()
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
	case encodingBr:
		writer := brotli.NewWriterLevel(&buf, brotli.DefaultCompression)
		if _, err := writer.Write(body); err != nil {
			_ = writer.Close()
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
	case encodingGzip:
		writer, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write(body); err != nil {
			_ = writer.Close()
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
	default:
		return body, nil
	}
	return buf.Bytes(), nil
}

func appendVary(current string, value string) string {
	for _, item := range strings.Split(current, ",") {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return current
		}
	}
	if strings.TrimSpace(current) == "" {
		return value
	}
	return current + ", " + value
}
