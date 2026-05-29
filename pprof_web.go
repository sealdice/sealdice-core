package main

import (
	"net/http/pprof"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	"sealdice-core/dice"
	"sealdice-core/logger"
)

type pprofWebConfig struct {
	Enabled bool
	Token   string
}

func newPprofWebConfig(enabled bool) *pprofWebConfig {
	cfg := &pprofWebConfig{Enabled: enabled}
	if enabled {
		cfg.Token = dice.RandStringBytesMaskImprSrcSB2(32)
	}
	return cfg
}

func registerPprofWebRoutes(router fiber.Router, cfg *pprofWebConfig) {
	if router == nil || cfg == nil || !cfg.Enabled || strings.TrimSpace(cfg.Token) == "" {
		return
	}

	tokenGuard := func(c *fiber.Ctx) error {
		if c.Query("token") != cfg.Token {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Next()
	}

	group := router.Group("/debug/pprof", tokenGuard)
	group.Get("/", adaptor.HTTPHandlerFunc(pprof.Index))
	group.Get("/cmdline", adaptor.HTTPHandlerFunc(pprof.Cmdline))
	group.Get("/profile", adaptor.HTTPHandlerFunc(pprof.Profile))
	group.Post("/symbol", adaptor.HTTPHandlerFunc(pprof.Symbol))
	group.Get("/symbol", adaptor.HTTPHandlerFunc(pprof.Symbol))
	group.Get("/trace", adaptor.HTTPHandlerFunc(pprof.Trace))
	group.Get("/allocs", adaptor.HTTPHandler(pprof.Handler("allocs")))
	group.Get("/block", adaptor.HTTPHandler(pprof.Handler("block")))
	group.Get("/goroutine", adaptor.HTTPHandler(pprof.Handler("goroutine")))
	group.Get("/heap", adaptor.HTTPHandler(pprof.Handler("heap")))
	group.Get("/mutex", adaptor.HTTPHandler(pprof.Handler("mutex")))
	group.Get("/threadcreate", adaptor.HTTPHandler(pprof.Handler("threadcreate")))
}

func logPprofWebEnabled(cfg *pprofWebConfig, serveAddress string) {
	if cfg == nil || !cfg.Enabled || cfg.Token == "" {
		return
	}

	baseURL := pprofBaseURL(serveAddress)
	log := logger.M()
	log.Infof("pprof Web 已启用: %s/debug/pprof/?token=%s", baseURL, cfg.Token)
	log.Infof("pprof heap: go tool pprof %s/debug/pprof/heap?token=%s", baseURL, cfg.Token)
	log.Infof("pprof allocs: go tool pprof %s/debug/pprof/allocs?token=%s", baseURL, cfg.Token)
	log.Infof("pprof cpu: go tool pprof %s/debug/pprof/profile?seconds=30&token=%s", baseURL, cfg.Token)
	log.Infof("pprof goroutine: %s/debug/pprof/goroutine?debug=1&token=%s", baseURL, cfg.Token)
}

func pprofBaseURL(serveAddress string) string {
	address := strings.TrimSpace(serveAddress)
	switch {
	case address == "":
		return "http://127.0.0.1:3211"
	case strings.HasPrefix(address, ":"):
		return "http://127.0.0.1" + address
	case strings.HasPrefix(address, "0.0.0.0:"):
		return "http://127.0.0.1:" + strings.TrimPrefix(address, "0.0.0.0:")
	case strings.HasPrefix(address, "[::]:"):
		return "http://127.0.0.1:" + strings.TrimPrefix(address, "[::]:")
	case strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://"):
		return address
	default:
		return "http://" + address
	}
}
