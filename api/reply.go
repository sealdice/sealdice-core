package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"sealdice-core/dice"
)

func customReplySave(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := dice.ReplyConfig{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	v.Clean()
	myDice.CustomReplyConfig[0] = &v
	v.Save(myDice)
	return c.JSON(http.StatusOK, nil)
}

func customReply(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	rc := dice.CustomReplyConfigRead(myDice)
	return c.JSON(http.StatusOK, rc)
}

type ReplyConfigInfo struct {
	Enable   bool   `yaml:"enable" json:"enable"`
	FileName string `yaml:"-" json:"fileName"`
}

func customReplyFileList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var items []*ReplyConfigInfo
	for _, i := range myDice.CustomReplyConfig {
		items = append(items, &ReplyConfigInfo{
			Enable:   i.Enable,
			FileName: i.FileName,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func customReplyFileNew(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		Filename string `json:"filename"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	rc := dice.CustomReplyConfigNew(myDice, v.Filename)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": rc != nil,
	})
}

func customReplyFileRename(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{})
}

func customReplyFileDelete(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{})
}
