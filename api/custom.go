package api

import (
	"net/http"
	"sealdice-core/dice"
	"strings"

	"github.com/labstack/echo/v4"
)

func customText(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"texts":    myDice.TextMapRaw,
		"helpInfo": myDice.TextMapHelpInfo,
	})
}

func customTextSave(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		Category string                      `form:"category" json:"category"`
		Data     dice.TextTemplateWithWeight `form:"data" json:"data"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, v1 := range v.Data {
			for _, v2 := range v1 {
				v2[0] = strings.TrimSpace(v2[0].(string))
				v2[1] = int(v2[1].(float64))
			}
		}
		myDice.TextMapRaw[v.Category] = v.Data
		dice.SetupTextHelpInfo(myDice, myDice.TextMapHelpInfo, myDice.TextMapRaw, "configs/text-template.yaml")
		myDice.GenerateTextMap()
		myDice.SaveText()
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}
