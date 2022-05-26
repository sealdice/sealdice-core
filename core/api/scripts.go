package api

import "github.com/labstack/echo/v4"

func scriptReload(c echo.Context) error {
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	myDice.JsLock.Lock()
	myDice.JsInit()
	myDice.JsLoadScripts()
	myDice.JsLock.Unlock()
	return c.JSON(200, nil)
}
