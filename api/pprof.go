package api

import (
	"net/http"
	"net/http/pprof"

	"github.com/labstack/echo/v4"
)

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !doAuth(c) {
			return c.JSON(http.StatusForbidden, nil)
		}
		return next(c)
	}
}

func bindPProfAPIs(e *echo.Echo, prefix string) {
	g := e.Group(prefix+"/debug/pprof", AuthMiddleware)

	g.GET("", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	g.GET("/", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	g.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	g.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	g.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	g.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	g.GET("/allocs", echo.WrapHandler(pprof.Handler("allocs")))
	g.GET("/block", echo.WrapHandler(pprof.Handler("block")))
	g.GET("/goroutine", echo.WrapHandler(pprof.Handler("goroutine")))
	g.GET("/heap", echo.WrapHandler(pprof.Handler("heap")))
	g.GET("/mutex", echo.WrapHandler(pprof.Handler("mutex")))
	g.GET("/threadcreate", echo.WrapHandler(pprof.Handler("threadcreate")))
}
