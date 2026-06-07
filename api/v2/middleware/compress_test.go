package middleware_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/gofiber/fiber/v2"
	"github.com/klauspost/compress/zstd"

	middleware "sealdice-core/api/v2/middleware"
)

func TestPreferredCompressionMiddlewareNegotiatesZstdBrGzip(t *testing.T) {
	for _, tt := range []struct {
		name   string
		accept string
		want   string
	}{
		{name: "zstd first", accept: "gzip, br, zstd", want: "zstd"},
		{name: "brotli fallback", accept: "gzip, br", want: "br"},
		{name: "gzip fallback", accept: "gzip", want: "gzip"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(middleware.PreferredCompressionMiddleware())
			app.Get("/payload", func(c *fiber.Ctx) error {
				c.Type("text")
				return c.SendString(strings.Repeat("sealdice-fiber-compression\n", 32))
			})

			req := httptest.NewRequest(http.MethodGet, "/payload", nil)
			req.Header.Set("Accept-Encoding", tt.accept)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("fiber test request: %v", err)
			}
			defer resp.Body.Close()

			if got := resp.Header.Get("Content-Encoding"); got != tt.want {
				t.Fatalf("Content-Encoding = %q, want %q", got, tt.want)
			}
			if got := resp.Header.Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
				t.Fatalf("Vary = %q, want Accept-Encoding", got)
			}

			body, err := decodeCompressedBody(tt.want, resp.Body)
			if err != nil {
				t.Fatalf("decode %s body: %v", tt.want, err)
			}
			if !strings.Contains(string(body), "sealdice-fiber-compression") {
				t.Fatalf("decoded body missing payload: %q", string(body))
			}
		})
	}
}

func TestPreferredCompressionMiddlewareSkipsEventStreams(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.PreferredCompressionMiddleware())
	app.Get("/events", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/event-stream")
		return c.SendString("event: system/ready\ndata: {}\n\n")
	})

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Accept-Encoding", "zstd, br, gzip")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("fiber test request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty for event stream", got)
	}
}

func decodeCompressedBody(encoding string, body io.Reader) ([]byte, error) {
	switch encoding {
	case "zstd":
		reader, err := zstd.NewReader(body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	case "br":
		return io.ReadAll(brotli.NewReader(body))
	case "gzip":
		reader, err := gzip.NewReader(body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	default:
		return io.ReadAll(body)
	}
}
