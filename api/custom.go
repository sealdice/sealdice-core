package api

import (
	"errors"
	"net/http"

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
		Data     dice.TextTemplateWithWeight `form:"data"     json:"data"`
	}{}
	if err := c.Bind(&v); err != nil {
		return c.String(430, "")
	}
	if err := myDice.SaveTextCategory(v.Category, v.Data); err != nil {
		if errors.Is(err, dice.ErrInvalidConfig) {
			return c.String(http.StatusBadRequest, err.Error())
		}
		myDice.Logger.Errorf("保存文案失败: %v", err)
		return c.String(http.StatusInternalServerError, "保存文案失败")
	}
	return c.String(http.StatusOK, "")
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
