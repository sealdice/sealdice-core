package api

import (
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type backupFileItem struct {
	Name     string `json:"name"`
	FileSize int64  `json:"fileSize"`
}

func ReverseSlice(s interface{}) {
	size := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func backupGetList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var items []*backupFileItem
	_ = filepath.Walk("./backups", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			items = append(items, &backupFileItem{
				Name:     info.Name(),
				FileSize: info.Size(),
			})
		}
		return err
	})

	ReverseSlice(items)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func backupDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	name := c.QueryParam("name")
	if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
		return c.Attachment("./backups/"+name, name)
	}
	return c.JSON(http.StatusOK, nil)
}

func backupDelete(c echo.Context) error { //nolint
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	var err error
	name := c.QueryParam("name")
	if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
		err = os.Remove("./backups/" + name)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": err == nil,
	})
}

// 快速备份
func backupSimple(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	_, err := dm.BackupSimple()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": err == nil,
	})
}

type backupConfig struct {
	AutoBackupEnable bool   `json:"autoBackupEnable"`
	AutoBackupTime   string `json:"autoBackupTime"`
}

func backupConfigGet(c echo.Context) error {
	bc := backupConfig{}
	bc.AutoBackupEnable = dm.AutoBackupEnable
	bc.AutoBackupTime = dm.AutoBackupTime
	return c.JSON(http.StatusOK, bc)
}

func backupConfigSave(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := backupConfig{}
	err := c.Bind(&v)
	if err == nil {
		dm.AutoBackupEnable = v.AutoBackupEnable
		dm.AutoBackupTime = v.AutoBackupTime
		dm.ResetAutoBackup()
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}
