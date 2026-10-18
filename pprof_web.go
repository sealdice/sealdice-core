package main

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/labstack/echo/v4"

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

func registerPprofWebRoutes(router *echo.Echo, cfg *pprofWebConfig) {
	if router == nil || cfg == nil || !cfg.Enabled || strings.TrimSpace(cfg.Token) == "" {
		return
	}

	group := router.Group("/debug/pprof", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.QueryParam("token") != cfg.Token {
				return c.NoContent(http.StatusUnauthorized)
			}
			return next(c)
		}
	})

	group.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	group.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	group.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	group.POST("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	group.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	group.GET("/allocs", echo.WrapHandler(pprof.Handler("allocs")))
	group.GET("/block", echo.WrapHandler(pprof.Handler("block")))
	group.GET("/goroutine", echo.WrapHandler(pprof.Handler("goroutine")))
	group.GET("/heap", echo.WrapHandler(pprof.Handler("heap")))
	group.GET("/mutex", echo.WrapHandler(pprof.Handler("mutex")))
	group.GET("/threadcreate", echo.WrapHandler(pprof.Handler("threadcreate")))
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
