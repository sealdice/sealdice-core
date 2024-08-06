package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func customText(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"texts":       myDice.TextMapRaw,
		"helpInfo":    myDice.TextMapHelpInfo,
		"previewInfo": &myDice.TextMapCompatible,
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

func customTextPreviewRefresh(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	// 按理说不该全部更新吧，但是 customTextSave 里没做区分，先保持一致，后续再看看
	v := struct {
		Category string `form:"category" json:"category"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for k, v2 := range myDice.TextMapRaw[v.Category] {
			dice.TextMapCompatibleCheck(myDice, v.Category, k, v2)
		}
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}
