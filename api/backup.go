package api

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	"sealdice-core/logger"
)

func backupGetList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var items []*dice.BackupArchiveInfo
	err := filepath.Walk(dice.BackupDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() || !strings.EqualFold(filepath.Ext(info.Name()), ".zip") {
			return nil
		}
		items = append(items, dice.InspectBackupArchive(path))
		return nil
	})
	if err != nil {
		logger.M().Errorw("[备份列表] 读取备份目录失败", "error", err)
		return Error(&c, "读取备份列表失败: "+err.Error(), Response{})
	}

	slices.Reverse(items)
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
		if dice.BackupInUse(name) {
			return c.JSON(http.StatusConflict, map[string]interface{}{"success": false, "err": "恢复任务正在使用该备份"})
		}
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
			if dice.BackupInUse(name) {
				fails = append(fails, name)
				continue
			}
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

func backupUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	reader, err := c.Request().MultipartReader()
	if err != nil {
		return Error(&c, "无法读取上传内容: "+err.Error(), Response{})
	}
	for {
		part, nextErr := reader.NextPart()
		if nextErr != nil {
			if errors.Is(nextErr, io.EOF) {
				break
			}
			return Error(&c, nextErr.Error(), Response{})
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		originalName := filepath.Base(part.FileName())
		logger.M().Infow("[备份导入] 开始接收并校验备份", "file", originalName)
		item, importErr := dice.ImportBackup(part)
		_ = part.Close()
		if importErr != nil {
			logger.M().Errorw("[备份导入] 导入失败", "file", originalName, "error", importErr)
			return Error(&c, importErr.Error(), Response{})
		}
		if item.Reused {
			logger.M().Infow("[备份导入] 内容已存在，复用已有文件: "+item.Name, "file", originalName, "storedAs", item.Name, "size", item.FileSize)
		} else {
			logger.M().Infow("[备份导入] 新文件已保存: "+item.Name, "file", originalName, "storedAs", item.Name, "size", item.FileSize)
		}
		return Success(&c, Response{"item": item})
	}
	return Error(&c, "请上传备份文件", Response{})
}

func backupRestore(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	var request struct {
		Name string `json:"name"`
	}
	if err := c.Bind(&request); err != nil {
		return Error(&c, err.Error(), Response{})
	}
	logger.M().Infow("[备份恢复] 收到恢复请求", "source", request.Name)
	operation, err := dm.ScheduleRestore(request.Name)
	if err != nil {
		logger.M().Errorw("[备份恢复] 创建恢复任务失败", "source", request.Name, "error", err)
		return Error(&c, err.Error(), Response{})
	}
	logger.M().Infow(
		"[备份恢复] 恢复任务已创建",
		"operationId", operation.OperationID,
		"source", request.Name,
		"safetyBackup", operation.SafetyBackupName,
	)
	responseErr := Success(&c, Response{
		"safetyBackupName": operation.SafetyBackupName,
		"operationId":      operation.OperationID,
		"statusToken":      operation.StatusToken,
		"reloading":        true,
		"switchMode":       "runtime",
	})
	go func() {
		time.Sleep(250 * time.Millisecond)
		if runtimeRestoreFn != nil {
			if restoreErr := runtimeRestoreFn(context.Background()); restoreErr != nil {
				logger.M().Errorw(
					"[备份恢复] Runtime 恢复流程结束，但恢复未成功",
					"operationId", operation.OperationID,
					"error", restoreErr,
				)
			}
		} else {
			logger.M().Errorw("[备份恢复] Runtime 恢复入口未初始化", "operationId", operation.OperationID)
		}
	}()
	return responseErr
}

func backupRestoreStatus(c echo.Context) error {
	if !dice.ValidateRestoreStatusToken(c.Request().Header.Get("X-Seal-Restore-Operation"), c.Request().Header.Get("X-Seal-Restore-Token")) {
		runtimeGate.RLock()
		defer runtimeGate.RUnlock()
		if myDice == nil || !doAuth(c) {
			return c.JSON(http.StatusForbidden, nil)
		}
	}
	return Success(&c, Response{"status": dice.GetRestoreStatus()})
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
