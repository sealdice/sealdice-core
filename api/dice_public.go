package api

import (
	"net/http"
	"time"

	"sealdice-core/dice"

	"github.com/labstack/echo/v4"
)

func dicePublicInfo(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"endpoints": myDice.ImSession.EndPoints,
		"config":    myDice.Config.PublicDiceConfig})
}
func dicePublicSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	v := struct {
		Config   dice.PublicDiceConfig    `json:"config"`
		Selected []map[string]interface{} `json:"selected"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	myDice.Config.PublicDiceConfig = v.Config
	for _, i := range myDice.ImSession.EndPoints {
		i.IsPublic = false
		for _, s := range v.Selected {
			if id, ok := s["id"].(string); ok && i.ID == id {
				i.IsPublic = true
			}
		}
	}
	myDice.PublicDiceInfoRegister()
	myDice.PublicDiceEndpointRefresh()
	myDice.PublicDiceSetupTick()
	myDice.LastUpdatedTime = time.Now().Unix()
	myDice.Save(false)
	return Success(&c, Response{})
}
