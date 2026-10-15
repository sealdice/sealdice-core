package api

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

const customReplyPackageWarning = dice.ReplyPackageWarning

type customReplySaveResponse struct {
	Success bool   `json:"success"`
	Warning string `json:"warning,omitempty"`
}

func customReplySave(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	req := struct {
		dice.ReplyConfig
		VMVersion *string `json:"vmVersion"`
	}{}
	if err := c.Bind(&req); err != nil {
		return c.String(430, err.Error())
	}
	if err := myDice.SaveReplyConfig(&req.ReplyConfig, req.VMVersion); err != nil {
		if errors.Is(err, dice.ErrInvalidConfig) {
			return c.String(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, dice.ErrReplyPackageNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{"success": false, "message": err.Error()})
		}
		myDice.Logger.Errorf("保存自定义回复文件失败: %v", err)
		return c.String(http.StatusInternalServerError, "保存自定义回复文件失败")
	}
	return c.JSON(http.StatusOK, customReplySaveResponse{Success: true, Warning: req.Warning})
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
		if replyFile, ok := myDice.FindReplyPackageFile(packageID, filename); ok {
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

	if replyFile, ok := myDice.FindReplyPackageFile("", filename); ok {
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
	VMVersion   string `json:"vmVersion,omitempty" yaml:"-"`
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
			VMVersion:   i.VMVersion,
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

func customReplyFileNew(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		Filename  string `json:"filename"`
		VMVersion string `json:"vmVersion"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	if v.VMVersion == "" {
		v.VMVersion = dice.ReplyVMVersionV2
	}
	vmVersion, validVMVersion := dice.ValidateReplyVMVersion(v.VMVersion)
	if !validVMVersion {
		return c.String(http.StatusBadRequest, "不支持的自定义回复 VM 版本")
	}

	if v.Filename != "" && !dice.CustomReplyConfigCheckExists(myDice, v.Filename) {
		rc := dice.CustomReplyConfigNewWithVMVersion(myDice, v.Filename, vmVersion)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":   rc != nil,
			"vmVersion": v.VMVersion,
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
	var success bool
	if v.PackageID != "" {
		replyFile, ok := myDice.FindReplyPackageFile(v.PackageID, v.Filename)
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
			if replyFile, ok := myDice.FindReplyPackageFile(packageID, name); ok {
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
		dst, err = os.Create(thePath) // #nosec G703 -- the upload filename has path separators replaced.
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
