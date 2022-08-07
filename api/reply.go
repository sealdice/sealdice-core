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
	for index, i := range myDice.CustomReplyConfig {
		if i.Filename == v.Filename {
			myDice.CustomReplyConfig[index].Enable = v.Enable
			myDice.CustomReplyConfig[index].Items = v.Items
			break
		}
	}
	v.Save(myDice)
	return c.JSON(http.StatusOK, nil)
}

func customReplyGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	//v := struct {
	//	Filename string `json:"filename" form:"filename"`
	//}{}
	//
	//err := c.Bind(&v)
	//if err != nil {
	//	return c.String(430, err.Error())
	//}

	rc := dice.CustomReplyConfigRead(myDice, c.QueryParam("filename"))
	return c.JSON(http.StatusOK, rc)
}

type ReplyConfigInfo struct {
	Enable   bool   `yaml:"enable" json:"enable"`
	Filename string `yaml:"-" json:"filename"`
}

func customReplyFileList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var items []*ReplyConfigInfo
	for _, i := range myDice.CustomReplyConfig {
		items = append(items, &ReplyConfigInfo{
			Enable:   i.Enable,
			Filename: i.Filename,
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

	if v.Filename != "" && !dice.CustomReplyConfigCheckExists(myDice, v.Filename) {
		rc := dice.CustomReplyConfigNew(myDice, v.Filename)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": rc != nil,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": false,
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

	v := struct {
		Filename string `json:"filename"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": dice.CustomReplyConfigDelete(myDice, v.Filename),
	})
}
