package api

import (
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	"path/filepath"
)

type backupFileItem struct {
	Name     string `json:"name"`
	FileSize int64  `json:"fileSize"`
}

func backupGetList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	items := []*backupFileItem{}
	filepath.Walk("./backups", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			items = append(items, &backupFileItem{
				Name:     info.Name(),
				FileSize: info.Size(),
			})
		}
		return err
	})

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func backupDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"texts":    myDice.TextMapRaw,
		"helpInfo": myDice.TextMapHelpInfo,
	})
}

func backupDelete(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"texts":    myDice.TextMapRaw,
		"helpInfo": myDice.TextMapHelpInfo,
	})
}

// 快速备份
func backupSimple(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	err := dm.BackupSimple()
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
