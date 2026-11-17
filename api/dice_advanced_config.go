package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func DiceAdvancedConfigGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	return c.JSON(http.StatusOK, myDice.AdvancedConfig)
}

func DiceAdvancedConfigSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	advancedConfig := myDice.AdvancedConfig
	err := c.Bind(&advancedConfig)
	if err != nil {
		return Error(&c, err.Error(), nil)
	}

	myDice.UpdateAdvancedConfig(advancedConfig)
	return Success(&c, Response{})
}
