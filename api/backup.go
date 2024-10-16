package api

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	"sealdice-core/utils/crypto"
)

type backupFileItem struct {
	Name      string `json:"name"`
	FileSize  int64  `json:"fileSize"`
	Selection int64  `json:"selection"`
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

	reFn := regexp.MustCompile(`^(bak_\d{6}_\d{6}(?:_auto)?_r([0-9a-f]+))_([0-9a-f]{8})\.zip$`)

	var items []*backupFileItem
	_ = filepath.Walk(dice.BackupDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			fn := info.Name()
			matches := reFn.FindStringSubmatch(fn)
			selection := int64(0)
			if len(matches) == 4 {
				hashed := crypto.CalculateSHA512Str([]byte(matches[1]))
				if hashed[:8] == matches[3] {
					selection, _ = strconv.ParseInt(matches[2], 16, 64)
				} else {
					selection = -1
				}
			}

			items = append(items, &backupFileItem{
				Name:      fn,
				FileSize:  info.Size(),
				Selection: selection,
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
func backupExec(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Selection uint64 `json:"selection"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	_, err = dm.Backup(dice.BackupSelection(v.Selection), false)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": err == nil,
	})
}

type backupConfig struct {
	AutoBackupEnable    bool   `json:"autoBackupEnable"`
	AutoBackupTime      string `json:"autoBackupTime"`
	AutoBackupSelection uint64 `json:"autoBackupSelection"`

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
	bc.AutoBackupSelection = uint64(dm.AutoBackupSelection)
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
	dm.AutoBackupSelection = dice.BackupSelection(v.AutoBackupSelection)

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
	dm.Save()
	return c.String(http.StatusOK, "")
}
