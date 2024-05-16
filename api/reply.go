package api

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"

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
	for index, i := range myDice.Config.CustomReplyConfig {
		if i.Filename == v.Filename {
			myDice.Config.CustomReplyConfig[index].Enable = v.Enable
			myDice.Config.CustomReplyConfig[index].Conditions = v.Conditions
			myDice.Config.CustomReplyConfig[index].Items = v.Items
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

	rc, _ := dice.CustomReplyConfigRead(myDice, c.QueryParam("filename"))
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
	for _, i := range myDice.Config.CustomReplyConfig {
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

func customReplyFileRename(c echo.Context) error { //nolint
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

	success := dice.CustomReplyConfigDelete(myDice, v.Filename)
	if success {
		dice.ReplyReload(myDice)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": success,
	})
}

func customReplyFileDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	name := c.QueryParam("name")
	if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
		myDice.Logger.Infof("下载自定义文件: %s", myDice.GetExtConfigFilePath("reply", name))
		return c.Attachment(myDice.GetExtConfigFilePath("reply", name), name)
	}
	return c.JSON(http.StatusOK, nil)
}

func customReplyFileUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//-----------
	// Read file
	//-----------

	// Source
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func(src multipart.File) {
		_ = src.Close()
	}(src)

	file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
	file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")
	thePath := myDice.GetExtConfigFilePath("reply", file.Filename)

	if !dice.CustomReplyConfigCheckExists(myDice, file.Filename) {
		myDice.Logger.Infof("上传自定义文件: %s", thePath)
		var dst *os.File
		dst, err = os.Create(thePath)
		if err != nil {
			return err
		}
		defer func(dst *os.File) {
			_ = dst.Close()
		}(dst)

		// Copy
		if _, err = io.Copy(dst, src); err != nil {
			return err
		}
		dice.ReplyReload(myDice)
	} else {
		return c.JSON(http.StatusConflict, nil)
	}

	return c.JSON(http.StatusOK, nil)
}

func customReplyDebugModeGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"value": myDice.Config.ReplyDebugMode,
	})
}

func customReplyDebugModeSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		Value bool `json:"value"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	myDice.Config.ReplyDebugMode = v.Value
	myDice.MarkModified()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"value": myDice.Config.ReplyDebugMode,
	})
}
