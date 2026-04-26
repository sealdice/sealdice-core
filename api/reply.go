package api

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	for index, i := range myDice.CustomReplyConfig {
		if i.Filename == v.Filename {
			myDice.CustomReplyConfig[index].Enable = v.Enable
			myDice.CustomReplyConfig[index].Conditions = v.Conditions
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

	filename := c.QueryParam("filename")
	rc, _ := dice.CustomReplyConfigRead(myDice, filename)
	if rc != nil {
		rc.PackageID = getCustomReplyPackageID(filename)
	}
	return c.JSON(http.StatusOK, rc)
}

type ReplyConfigInfo struct {
	Enable    bool   `json:"enable"   yaml:"enable"`
	Filename  string `json:"filename" yaml:"-"`
	PackageID string `json:"packageId,omitempty" yaml:"-"`
}

func customReplyFileList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	packageIDs := getCustomReplyPackageIDs()
	var items []*ReplyConfigInfo
	seen := map[string]int{}
	for _, i := range myDice.CustomReplyConfig {
		packageID := i.PackageID
		if packageID == "" {
			packageID = packageIDs[strings.ToLower(i.Filename)]
		}

		key := strings.ToLower(i.Filename)
		if idx, exists := seen[key]; exists {
			if items[idx].PackageID == "" && packageID != "" {
				items[idx].PackageID = packageID
				items[idx].Enable = i.Enable
			}
			continue
		}

		items = append(items, &ReplyConfigInfo{
			Enable:    i.Enable,
			Filename:  i.Filename,
			PackageID: packageID,
		})
		seen[key] = len(items) - 1
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func getCustomReplyPackageID(filename string) string {
	if filename == "" {
		return ""
	}
	for _, item := range myDice.CustomReplyConfig {
		if strings.EqualFold(item.Filename, filename) && item.PackageID != "" {
			return item.PackageID
		}
	}
	return getCustomReplyPackageIDs()[strings.ToLower(filename)]
}

func getCustomReplyPackageIDs() map[string]string {
	result := map[string]string{}
	if myDice == nil || myDice.PackageManager == nil {
		return result
	}
	for _, replyFile := range myDice.PackageManager.GetEnabledContentFiles("reply") {
		ext := strings.ToLower(filepath.Ext(replyFile.Path))
		if ext != ".yaml" && ext != ".yml" && ext != "" {
			continue
		}
		filename := strings.ToLower(filepath.Base(replyFile.Path))
		if _, exists := result[filename]; !exists {
			result[filename] = replyFile.PackageID
		}
	}
	return result
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
