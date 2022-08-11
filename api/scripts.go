package api

import (
	"github.com/labstack/echo/v4"
	"sealdice-core/dice"
)

func scriptReload(c echo.Context) error {
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//myDice.JsLock.Lock()
	//defer myDice.JsLock.Unlock()
	myDice.JsInit()
	myDice.CocExtraRules = map[int]*dice.CocRuleInfo{}
	//myDice.JsLoadScripts()
	return c.JSON(200, nil)
}
