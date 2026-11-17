package api

import (
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

var backupFileNamePattern = regexp.MustCompile(`^(bak_\d{6}_\d{6}(?:_auto)?_r([0-9a-f]{1,16}))_([0-9a-f]{8})\.zip$`)

func ReverseSlice(s interface{}) {
	size := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func resolveBackupFilePath(name string) (string, bool) {
	matches := backupFileNamePattern.FindStringSubmatch(name)
	if len(matches) != 4 ||
		strings.ContainsAny(name, `/\`) ||
		filepath.IsAbs(name) || filepath.VolumeName(name) != "" ||
		filepath.Base(name) != name {
		return "", false
	}
	if _, err := strconv.ParseUint(matches[2], 16, 64); err != nil ||
		crypto.CalculateSHA512Str([]byte(matches[1]))[:8] != matches[3] {
		return "", false
	}
	root, err := filepath.Abs(dice.BackupDir)
	if err != nil {
		return "", false
	}
	target := filepath.Join(root, name)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative != name {
		return "", false
	}
	return target, true
}

func isRegularBackupFile(path string) bool {
	info, err := os.Lstat(path) // #nosec G703 -- path comes from resolveBackupFilePath.
	return err == nil && info.Mode().IsRegular()
}

func backupGetList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var items []*backupFileItem
	entries, _ := os.ReadDir(dice.BackupDir)
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		fn := entry.Name()
		if !backupFileNamePattern.MatchString(fn) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		matches := backupFileNamePattern.FindStringSubmatch(fn)
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
	if path, ok := resolveBackupFilePath(name); ok {
		if !isRegularBackupFile(path) {
			return c.NoContent(http.StatusNotFound)
		}
		return c.Attachment(path, name)
	}
	return c.NoContent(http.StatusNotFound)
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

	var err = os.ErrInvalid
	name := c.QueryParam("name")
	if path, ok := resolveBackupFilePath(name); ok {
		if isRegularBackupFile(path) {
			err = os.Remove(path) // #nosec G703 -- the filename is restricted to generated backup names.
		}
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
		path, ok := resolveBackupFilePath(name)
		if !ok || !isRegularBackupFile(path) {
			fails = append(fails, name)
			continue
		}
		err = os.Remove(path) // #nosec G703 -- the filename is restricted to generated backup names.
		if err != nil {
			fails = append(fails, name)
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
