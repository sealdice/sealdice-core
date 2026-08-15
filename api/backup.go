package api

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/utils/constant"
)

var restoreRequestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

func applyRuntimeRestoreCapability(item *dice.BackupArchiveInfo) {
	if item == nil {
		return
	}
	if dm == nil || dm.Operator == nil {
		item.Restorable = false
		item.RestoreError = "Runtime 数据库尚未就绪"
		return
	}
	if dm.Operator.Type() == constant.SQLITE {
		return
	}
	item.Restorable = false
	item.RestoreError = "当前实例使用外部数据库，仅支持导入、查看和下载备份"
}

func backupGetList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	entries, err := os.ReadDir(dice.BackupDir)
	if errors.Is(err, os.ErrNotExist) {
		return c.JSON(http.StatusOK, map[string]interface{}{"items": []*dice.BackupArchiveInfo{}})
	}
	if err != nil {
		logger.M().Errorw("[备份列表] 读取备份目录失败", "error", err)
		return Error(&c, "读取备份列表失败: "+err.Error(), Response{})
	}
	type listedBackup struct {
		info    *dice.BackupArchiveInfo
		modTime time.Time
	}
	listed := make([]listedBackup, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".zip") {
			continue
		}
		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			logger.M().Warnw("[备份列表] 跳过无法读取的文件", "file", entry.Name(), "error", infoErr)
			continue
		}
		archiveInfo := dice.InspectBackupArchive(filepath.Join(dice.BackupDir, entry.Name()))
		applyRuntimeRestoreCapability(archiveInfo)
		listed = append(listed, listedBackup{
			info:    archiveInfo,
			modTime: entryInfo.ModTime(),
		})
	}
	sort.Slice(listed, func(i, j int) bool { return listed[i].modTime.After(listed[j].modTime) })
	items := make([]*dice.BackupArchiveInfo, 0, len(listed))
	for _, item := range listed {
		items = append(items, item.info)
	}
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
	archive, archiveInfo, err := dice.OpenBackupArchive(name)
	if err != nil {
		// Preserve the legacy response shape for invalid names.
		return c.JSON(http.StatusOK, nil)
	}
	defer archive.Close()
	c.Response().Header().Set(
		echo.HeaderContentDisposition,
		mime.FormatMediaType("attachment", map[string]string{"filename": name}),
	)
	http.ServeContent(c.Response(), c.Request(), name, archiveInfo.ModTime(), archive)
	return nil
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

	name := c.QueryParam("name")
	err := dice.DeleteBackup(name)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success": false,
			"result":  false,
			"err":     err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
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
		// Preserve the legacy skip behavior for empty or path-shaped names.
		if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
			continue
		}
		if deleteErr := dice.DeleteBackup(name); deleteErr != nil {
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
	partCount := 0
	fileParts := 0
	for {
		part, nextErr := reader.NextPart()
		if nextErr != nil {
			if errors.Is(nextErr, io.EOF) {
				break
			}
			return Error(&c, nextErr.Error(), Response{})
		}
		partCount++
		if partCount > 64 {
			_ = part.Close()
			return Error(&c, "上传包含过多表单字段", Response{})
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		fileParts++
		if fileParts > 1 {
			_ = part.Close()
			return Error(&c, "一次只能上传一个备份文件", Response{})
		}
		originalName := filepath.Base(part.FileName())
		logger.M().Infow("[备份导入] 开始接收并校验备份", "file", originalName)
		item, importErr := dice.ImportBackup(part)
		_ = part.Close()
		if importErr != nil {
			logger.M().Errorw("[备份导入] 导入失败", "file", originalName, "error", importErr)
			return Error(&c, importErr.Error(), Response{})
		}
		applyRuntimeRestoreCapability(item)
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
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	var request struct {
		Name      string `json:"name"`
		RequestID string `json:"requestId"`
	}
	if err := c.Bind(&request); err != nil {
		return Error(&c, err.Error(), Response{})
	}
	if !restoreRequestIDPattern.MatchString(request.RequestID) {
		return Error(&c, "requestId 格式非法", Response{})
	}
	if runtimeRestoreFn == nil {
		return Error(&c, "Runtime 恢复入口尚未初始化", Response{})
	}
	logger.M().Infow("[备份恢复] 收到恢复请求", "source", request.Name)
	operation, err := dm.ScheduleRestore(request.Name, request.RequestID)
	if err != nil {
		logger.M().Errorw("[备份恢复] 创建恢复任务失败", "source", request.Name, "error", err)
		return Error(&c, err.Error(), Response{})
	}
	logger.M().Infow(
		"[备份恢复] 恢复任务已创建",
		"operationId", operation.OperationID,
		"source", request.Name,
		"reused", operation.Reused,
	)
	if operation.Reused {
		status := dice.GetRestoreStatus()
		if status.State == "succeeded" && status.OperationID == operation.OperationID {
			return Success(&c, Response{
				"safetyBackupName": operation.SafetyBackupName,
				"operationId":      operation.OperationID,
				"statusToken":      operation.StatusToken,
				"expiresAt":        operation.ExpiresAt,
				"reloading":        false,
				"switchMode":       "runtime",
				"status":           status,
			})
		}
	}
	if !runtimeRestoreFn(operation.OperationID) {
		logger.M().Errorw(
			"[备份恢复] Runtime 恢复任务未被调度器接收",
			"operationId", operation.OperationID,
		)
		return Error(&c, "恢复任务已安全排队，但 Runtime 调度器暂不可用；请使用同一请求重试", Response{
			"operationId": operation.OperationID,
			"queued":      true,
		})
	}
	return Success(&c, Response{
		"safetyBackupName": operation.SafetyBackupName,
		"operationId":      operation.OperationID,
		"statusToken":      operation.StatusToken,
		"expiresAt":        operation.ExpiresAt,
		"reloading":        true,
		"switchMode":       "runtime",
	})
}

func backupRestoreStatus(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	operationID := c.Request().Header.Get("X-Seal-Restore-Operation")
	statusToken := c.Request().Header.Get("X-Seal-Restore-Token")
	if status, ok := dice.GetRestoreStatusAuthorized(operationID, statusToken); ok {
		return Success(&c, Response{"status": status})
	}
	if runtimeState.Load() != runtimeStateRunning {
		return c.JSON(http.StatusForbidden, nil)
	}
	runtimeGate.RLock()
	defer runtimeGate.RUnlock()
	if myDice == nil || !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
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
