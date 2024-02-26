package api

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
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
	_ = filepath.Walk(dice.BackupDir, func(path string, info fs.FileInfo, err error) error {
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
		return c.Attachment(dice.BackupDir+"/"+name, name)
	}
	return c.JSON(http.StatusOK, nil)
}

func backupDelete(c echo.Context) error {
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
		err = os.Remove(dice.BackupDir + "/" + name)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": err == nil,
	})
}

func backupBatchDelete(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	v := struct {
		Names []string `json:"names"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	fails := make([]string, 0, len(v.Names))
	for _, name := range v.Names {
		if name != "" && (!strings.Contains(name, "/")) && (!strings.Contains(name, "\\")) {
			err = os.Remove(dice.BackupDir + "/" + name)
			if err != nil {
				fails = append(fails, name)
			}
		}
	}

	if len(fails) == 0 {
		return Success(&c, Response{})
	}
	return Error(&c, "失败列表", Response{
		"fails": fails,
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

	BackupCleanStrategy  int    `json:"backupCleanStrategy"`
	BackupCleanKeepCount int    `json:"backupCleanKeepCount"`
	BackupCleanKeepDur   string `json:"backupCleanKeepDur"`
	BackupCleanTrigger   int    `json:"backupCleanTrigger"`
	BackupCleanCron      string `json:"backupCleanCron"`
}

func backupConfigGet(c echo.Context) error {
	bc := backupConfig{}
	bc.AutoBackupEnable = dm.AutoBackupEnable
	bc.AutoBackupTime = dm.AutoBackupTime
	bc.BackupCleanStrategy = int(dm.BackupCleanStrategy)
	bc.BackupCleanKeepCount = dm.BackupCleanKeepCount
	bc.BackupCleanKeepDur = dm.BackupCleanKeepDur.String()
	bc.BackupCleanTrigger = int(dm.BackupCleanTrigger)
	bc.BackupCleanCron = dm.BackupCleanCron
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
	if err != nil {
		return c.String(430, "")
	}

	dm.AutoBackupEnable = v.AutoBackupEnable
	dm.AutoBackupTime = v.AutoBackupTime

	if int(dice.BackupCleanStrategyDisabled) <= v.BackupCleanStrategy && v.BackupCleanStrategy <= int(dice.BackupCleanStrategyByTime) {
		dm.BackupCleanStrategy = dice.BackupCleanStrategy(v.BackupCleanStrategy)
		if dm.BackupCleanStrategy == dice.BackupCleanStrategyByCount && v.BackupCleanKeepCount > 0 {
			dm.BackupCleanKeepCount = v.BackupCleanKeepCount
		}
		if dm.BackupCleanStrategy == dice.BackupCleanStrategyByTime && len(v.BackupCleanKeepDur) > 0 {
			if dur, err := time.ParseDuration(v.BackupCleanKeepDur); err == nil {
				dm.BackupCleanKeepDur = dur
			} else {
				myDice.Logger.Errorf("设定的自动清理保留时间有误: %q %v", v.BackupCleanKeepDur, err)
			}
		}
		if v.BackupCleanTrigger > 0 {
			dm.BackupCleanTrigger = dice.BackupCleanTrigger(v.BackupCleanTrigger)
			if dm.BackupCleanTrigger&dice.BackupCleanTriggerCron > 0 {
				dm.BackupCleanCron = v.BackupCleanCron
			}
		}
	}

	dm.ResetAutoBackup()
	dm.ResetBackupClean()
	return c.String(http.StatusOK, "")
}
