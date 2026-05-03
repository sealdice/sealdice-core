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

const customReplyPackageWarning = "此文件来自扩展包运行时缓存，重装、刷新缓存或升级扩展包时可能被覆盖"

type customReplySaveResponse struct {
	Success bool   `json:"success"`
	Warning string `json:"warning,omitempty"`
}

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
	if v.PackageID != "" {
		replyFile, ok := findCustomReplyPackageFile(v.PackageID, v.Filename)
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "扩展包自定义回复不存在",
			})
		}
		v.Filename = filepath.Base(replyFile.Path)
		v.CacheBacked = true
		v.Warning = customReplyPackageWarning
		if err := v.SaveToPath(replyFile.Path); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		upsertCustomReplyConfig(v)
		return c.JSON(http.StatusOK, customReplySaveResponse{
			Success: true,
			Warning: customReplyPackageWarning,
		})
	}

	for index, i := range myDice.CustomReplyConfig {
		if i != nil && i.PackageID == "" && strings.EqualFold(i.Filename, v.Filename) {
			myDice.CustomReplyConfig[index].Enable = v.Enable
			myDice.CustomReplyConfig[index].Conditions = v.Conditions
			myDice.CustomReplyConfig[index].Items = v.Items
			myDice.CustomReplyConfig[index].Warning = ""
			myDice.CustomReplyConfig[index].CacheBacked = false
			break
		}
	}
	v.Save(myDice)
	return c.JSON(http.StatusOK, customReplySaveResponse{Success: true})
}

func customReplyGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	filename := c.QueryParam("filename")
	packageID := c.QueryParam("packageId")
	if packageID == "" {
		packageID = c.QueryParam("packageID")
	}
	if packageID != "" {
		if replyFile, ok := findCustomReplyPackageFile(packageID, filename); ok {
			rc, err := dice.CustomReplyConfigReadFromPath(myDice, replyFile.Path, filepath.Base(replyFile.Path))
			if err != nil {
				return c.String(http.StatusInternalServerError, err.Error())
			}
			rc.PackageID = replyFile.PackageID
			rc.CacheBacked = true
			rc.Warning = customReplyPackageWarning
			return c.JSON(http.StatusOK, rc)
		}
		return c.JSON(http.StatusNotFound, nil)
	}

	if dice.CustomReplyConfigCheckExists(myDice, filename) {
		rc, _ := dice.CustomReplyConfigRead(myDice, filename)
		return c.JSON(http.StatusOK, rc)
	}

	if replyFile, ok := findCustomReplyPackageFile("", filename); ok {
		rc, err := dice.CustomReplyConfigReadFromPath(myDice, replyFile.Path, filepath.Base(replyFile.Path))
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		rc.PackageID = replyFile.PackageID
		rc.CacheBacked = true
		rc.Warning = customReplyPackageWarning
		return c.JSON(http.StatusOK, rc)
	}

	rc, _ := dice.CustomReplyConfigRead(myDice, filename)
	return c.JSON(http.StatusOK, rc)
}

type ReplyConfigInfo struct {
	Enable      bool   `json:"enable"   yaml:"enable"`
	Filename    string `json:"filename" yaml:"-"`
	PackageID   string `json:"packageId,omitempty" yaml:"-"`
	CacheBacked bool   `json:"cacheBacked,omitempty" yaml:"-"`
	Warning     string `json:"warning,omitempty" yaml:"-"`
}

func customReplyFileList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var items []*ReplyConfigInfo
	seen := map[string]struct{}{}
	for _, i := range myDice.CustomReplyConfig {
		if i == nil {
			continue
		}
		key := customReplyListKey(i.PackageID, i.Filename)
		if _, exists := seen[key]; exists {
			continue
		}
		cacheBacked := i.PackageID != ""
		warning := ""
		if cacheBacked {
			warning = customReplyPackageWarning
		}

		items = append(items, &ReplyConfigInfo{
			Enable:      i.Enable,
			Filename:    i.Filename,
			PackageID:   i.PackageID,
			CacheBacked: cacheBacked,
			Warning:     warning,
		})
		seen[key] = struct{}{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func customReplyListKey(packageID, filename string) string {
	return strings.ToLower(packageID) + "\x00" + strings.ToLower(filename)
}

func upsertCustomReplyConfig(rc dice.ReplyConfig) {
	for index, item := range myDice.CustomReplyConfig {
		if item == nil {
			continue
		}
		if strings.EqualFold(item.PackageID, rc.PackageID) && strings.EqualFold(item.Filename, rc.Filename) {
			next := rc
			myDice.CustomReplyConfig[index] = &next
			return
		}
	}
	next := rc
	myDice.CustomReplyConfig = append(myDice.CustomReplyConfig, &next)
}

func getCustomReplyPackageFiles() []dice.PackageContentFile {
	if myDice == nil || myDice.PackageManager == nil {
		return nil
	}
	files := myDice.PackageManager.GetEnabledContentFiles("reply")
	result := make([]dice.PackageContentFile, 0, len(files))
	for _, replyFile := range files {
		ext := strings.ToLower(filepath.Ext(replyFile.Path))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		result = append(result, replyFile)
	}
	return result
}

func findCustomReplyPackageFile(packageID, filename string) (dice.PackageContentFile, bool) {
	if filename == "" {
		return dice.PackageContentFile{}, false
	}
	for _, replyFile := range getCustomReplyPackageFiles() {
		if packageID != "" && !strings.EqualFold(replyFile.PackageID, packageID) {
			continue
		}
		if strings.EqualFold(filepath.Base(replyFile.Path), filename) {
			return replyFile, true
		}
	}
	return dice.PackageContentFile{}, false
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
		Filename  string `json:"filename"`
		PackageID string `json:"packageId"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	var warning string
	success := false
	if v.PackageID != "" {
		replyFile, ok := findCustomReplyPackageFile(v.PackageID, v.Filename)
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "扩展包自定义回复不存在",
			})
		}
		success = os.Remove(replyFile.Path) == nil
		warning = customReplyPackageWarning
	} else {
		success = dice.CustomReplyConfigDelete(myDice, v.Filename)
	}
	if success {
		dice.ReplyReload(myDice)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": success,
		"warning": warning,
	})
}

func customReplyFileDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	name := c.QueryParam("name")
	packageID := c.QueryParam("packageId")
	if packageID == "" {
		packageID = c.QueryParam("packageID")
	}
	if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
		if packageID != "" {
			if replyFile, ok := findCustomReplyPackageFile(packageID, name); ok {
				myDice.Logger.Infof("下载扩展包自定义回复文件: %s", replyFile.Path)
				return c.Attachment(replyFile.Path, filepath.Base(replyFile.Path))
			}
			return c.JSON(http.StatusNotFound, nil)
		}
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
