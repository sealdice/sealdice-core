package api

import (
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"os"
	"sealdice-core/dice"
	"strings"
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
	defer src.Close()

	// Destination
	//fmt.Println("????", filepath.Join("./data/decks", file.Filename))
	file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
	file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")
	thePath := myDice.GetExtConfigFilePath("reply", file.Filename)

	if !dice.CustomReplyConfigCheckExists(myDice, file.Filename) {
		myDice.Logger.Infof("上传自定义文件: %s", thePath)

		dst, err := os.Create(thePath)
		if err != nil {
			return err
		}
		defer dst.Close()

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
		"value": myDice.ReplyDebugMode,
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

	myDice.ReplyDebugMode = v.Value
	return c.JSON(http.StatusOK, map[string]interface{}{
		"value": myDice.ReplyDebugMode,
	})
}
