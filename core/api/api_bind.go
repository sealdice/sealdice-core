package api

import (
	"net/http"
	"sealdice-core/dice"

	"github.com/labstack/echo/v4"
)

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func hello2(c echo.Context) error {
	//dice.CmdRegister("aaa", "bb");
	//a := dice.CmdList();
	//b, _ := json.Marshal(a)
	return c.String(http.StatusOK, string(""))
}

var myDice *dice.Dice

func customText(c echo.Context) error {
	return c.JSON(http.StatusOK, myDice.TextMapRaw)
}

func Bind(e *echo.Echo, _myDice *dice.Dice) {
	myDice = _myDice
	e.GET("/", hello)
	e.GET("/cmd/register", hello2)
	e.GET("/configs/customText", customText)
}
