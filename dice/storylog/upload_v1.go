package storylog

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sealdice-core/dice/service"
)

func uploadV1(env UploadEnv) (string, string, error) {
	_ = os.MkdirAll(env.Dir, 0o755)

	url, uploadTS, updateTS, _ := service.LogGetUploadInfo(env.Db, env.GroupID, env.LogName)
	if len(url) > 0 && uploadTS > updateTS {
		// 已有URL且上传时间晚于Log更新时间（最后录入时间），直接返回
		env.Log.Infof(
			"查询到之前上传的URL, 直接使用 Log:%s.%s 上传时间:%s 更新时间:%s URL:%s",
			env.GroupID, env.LogName,
			time.Unix(uploadTS, 0).Format("2006-01-02 15:04:05"),
			time.Unix(updateTS, 0).Format("2006-01-02 15:04:05"),
			url,
		)
		return url, env.Notice, nil
	}
	if len(url) == 0 {
		env.Log.Infof("没有查询到之前上传的URL Log:%s.%s", env.GroupID, env.LogName)
	} else {
		env.Log.Infof(
			"Log上传后又有更新, 重新上传 Log:%s.%s 上传时间:%s 更新时间:%s",
			env.GroupID, env.LogName,
			time.Unix(uploadTS, 0).Format("2006-01-02 15:04:05"),
			time.Unix(updateTS, 0).Format("2006-01-02 15:04:05"),
		)
	}

	lines, err := service.LogGetAllLines(env.Db, env.GroupID, env.LogName)
	if err != nil {
		return "", env.Notice, err
	}
	if len(lines) == 0 {
		return "", env.Notice, errors.New("此log不存在，或条目数为空，名字是否正确？")
	}
	env.lines = lines

	err = formatAndBackup(&env)
	if err != nil {
		return "", env.Notice, err
	}

	var zlibBuffer bytes.Buffer
	w := zlib.NewWriter(&zlibBuffer)
	if _, err = w.Write(*env.data); err != nil {
		_ = w.Close()
		return "", env.Notice, err
	}
	if err = w.Close(); err != nil {
		return "", env.Notice, err
	}

	url = uploadToSealBackends(env, &zlibBuffer)
	if errDB := service.LogSetUploadInfo(env.Db, env.GroupID, env.LogName, url); errDB != nil {
		env.Log.Errorf("记录Log上传信息失败: %v", errDB)
	}
	if len(url) == 0 {
		return "", env.Notice, errors.New("上传 log 到服务器失败，未能获取染色器链接")
	}
	return url, env.Notice, nil
}

// formatAndBackup 将导出的日志序列化到 env.data 并存储为本地 zip
func formatAndBackup(env *UploadEnv) (err error) {
	filename, notice := buildLogBackupFilename(env.GroupID, env.LogName, time.Now())
	env.appendNotice(notice)
	fzip, err := os.OpenFile(
		filepath.Join(env.Dir, filename),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("创建日志备份文件失败: %w", err)
	}
	writer := zip.NewWriter(fzip)
	defer func() {
		if closeErr := writer.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("关闭日志压缩文件失败: %w", closeErr)
		}
		if closeErr := fzip.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("关闭日志备份文件失败: %w", closeErr)
		}
	}()

	var text strings.Builder
	for _, i := range env.lines {
		timeTxt := time.Unix(i.Time, 0).Format("2006-01-02 15:04:05")
		fmt.Fprintf(&text, "%s(%v) %s\n%s\n\n", i.Nickname, i.IMUserID, timeTxt, i.Message)
	}

	{
		readmeWriter, createErr := writer.Create(ExportReadmeFilename)
		if createErr != nil {
			return fmt.Errorf("创建README失败: %w", createErr)
		}
		if _, writeErr := readmeWriter.Write([]byte(ExportReadmeContent)); writeErr != nil {
			return fmt.Errorf("写入README失败: %w", writeErr)
		}
	}
	{
		textWriter, createErr := writer.Create(ExportTxtFilename)
		if createErr != nil {
			return fmt.Errorf("创建文本日志文件失败: %w", createErr)
		}
		if _, writeErr := textWriter.Write([]byte(text.String())); writeErr != nil {
			return fmt.Errorf("写入文本日志文件失败: %w", writeErr)
		}
	}

	data, err := json.Marshal(map[string]interface{}{
		"version": StoryVersionV1,
		"items":   env.lines,
	})
	if err != nil {
		return err
	}
	fileWriter2, err := writer.Create(ExportJsonFilename)
	if err != nil {
		return fmt.Errorf("创建JSON日志文件失败: %w", err)
	}
	if _, err = fileWriter2.Write(data); err != nil {
		return fmt.Errorf("写入JSON日志文件失败: %w", err)
	}
	env.data = &data
	return nil
}

func uploadToSealBackends(env UploadEnv, data io.Reader) string {
	// 逐个尝试所有后端地址
	for _, backend := range env.Backends {
		if backend == "" {
			continue
		}
		ret := uploadToBackend(env, backend, data)
		if ret != "" {
			return ret
		}
	}
	return ""
}
