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
	"time"

	"sealdice-core/dice/model"
	"sealdice-core/utils"
)

func uploadV1(env UploadEnv) (string, error) {
	_ = os.MkdirAll(env.Dir, 0o755)

	url, uploadTS, updateTS, _ := model.LogGetUploadInfo(env.Db, env.GroupID, env.LogName)
	if len(url) > 0 && uploadTS > updateTS {
		// 已有URL且上传时间晚于Log更新时间（最后录入时间），直接返回
		env.Log.Infof(
			"查询到之前上传的URL, 直接使用 Log:%s.%s 上传时间:%s 更新时间:%s URL:%s",
			env.GroupID, env.LogName,
			time.Unix(uploadTS, 0).Format("2006-01-02 15:04:05"),
			time.Unix(updateTS, 0).Format("2006-01-02 15:04:05"),
			url,
		)
		return url, nil
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

	lines, err := model.LogGetAllLines(env.Db, env.GroupID, env.LogName)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", errors.New("此log不存在，或条目数为空，名字是否正确？")
	}
	env.lines = lines

	err = formatAndBackup(&env)
	if err != nil {
		return "", err
	}

	var zlibBuffer bytes.Buffer
	w := zlib.NewWriter(&zlibBuffer)
	_, _ = w.Write(*env.data)
	_ = w.Close()

	url = uploadToSealBackends(env, &zlibBuffer)
	if errDB := model.LogSetUploadInfo(env.Db, env.GroupID, env.LogName, url); errDB != nil {
		env.Log.Errorf("记录Log上传信息失败: %v", errDB)
	}
	if len(url) == 0 {
		return "", errors.New("上传 log 到服务器失败，未能获取染色器链接")
	}
	return url, nil
}

// formatAndBackup 将导出的日志序列化到 env.data 并存储为本地 zip
func formatAndBackup(env *UploadEnv) error {
	fzip, _ := os.OpenFile(
		filepath.Join(env.Dir, utils.FilenameClean(fmt.Sprintf(
			"%s_%s.%s.zip",
			env.GroupID, env.LogName, time.Now().Format("060102150405"),
		))),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	writer := zip.NewWriter(fzip)

	text := ""
	for _, i := range env.lines {
		timeTxt := time.Unix(i.Time, 0).Format("2006-01-02 15:04:05")
		text += fmt.Sprintf("%s(%v) %s\n%s\n\n", i.Nickname, i.IMUserID, timeTxt, i.Message)
	}

	{
		wr, _ := writer.Create(ExportReadmeFilename)
		_, _ = wr.Write([]byte(ExportReadmeContent))
	}
	{
		fileWriter, _ := writer.Create(ExportTxtFilename)
		_, _ = fileWriter.Write([]byte(text))
	}

	data, err := json.Marshal(map[string]interface{}{
		"version": StoryVersionV1,
		"items":   env.lines,
	})
	if err == nil {
		fileWriter2, _ := writer.Create(ExportJsonFilename)
		_, _ = fileWriter2.Write(data)
		env.data = &data
	}

	_ = writer.Close()
	_ = fzip.Close()

	return err
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
