package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	"path/filepath"
	"reflect"
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

	ReverseSlice(items)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func backupDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := backupFileItem{}
	err := c.Bind(&v)

	if err == nil {
		return c.Attachment("./backups/"+v.Name, v.Name)
		//f, err := os.Open("./backups/" + v.Name)
		//defer f.Close()
		//if err == nil {
		//	return c.Stream(http.StatusOK, "application/x-zip-compressed", f)
		//}
	}

	return c.JSON(http.StatusOK, nil)
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
		fmt.Println("???", dm.AutoBackupTime)
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}
