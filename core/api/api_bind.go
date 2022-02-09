package api

import (
	"encoding/json"
	"net/http"
	"sealdice-core/core/dice"

	"github.com/labstack/echo/v4"
)

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func hello2(c echo.Context) error {
	dice.CmdRegister("aaa", "bb");
	a := dice.CmdList();
	b, _ := json.Marshal(a)
	return c.String(http.StatusOK, string(b));
}

func Bind(e *echo.Echo) {
	e.GET("/", hello)
	e.GET("/cmd/register", hello2)
}
