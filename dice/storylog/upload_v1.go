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
	"regexp"
	"time"

	"sealdice-core/dice/model"
)

func uploadV1(ctx UploadContext) (string, error) {
	_ = os.MkdirAll(ctx.Dir, 0o755)

	url, uploadTS, updateTS, _ := model.LogGetUploadInfo(ctx.Db, ctx.GroupID, ctx.LogName)
	if len(url) > 0 && uploadTS > updateTS {
		// 已有URL且上传时间晚于Log更新时间（最后录入时间），直接返回
		ctx.Log.Infof(
			"查询到之前上传的URL, 直接使用 Log:%s.%s 上传时间:%s 更新时间:%s URL:%s",
			ctx.GroupID, ctx.LogName,
			time.Unix(uploadTS, 0).Format("2006-01-02 15:04:05"),
			time.Unix(updateTS, 0).Format("2006-01-02 15:04:05"),
			url,
		)
		return url, nil
	}
	if len(url) == 0 {
		ctx.Log.Infof("没有查询到之前上传的URL Log:%s.%s", ctx.GroupID, ctx.LogName)
	} else {
		ctx.Log.Infof(
			"Log上传后又有更新, 重新上传 Log:%s.%s 上传时间:%s 更新时间:%s",
			ctx.GroupID, ctx.LogName,
			time.Unix(uploadTS, 0).Format("2006-01-02 15:04:05"),
			time.Unix(updateTS, 0).Format("2006-01-02 15:04:05"),
		)
	}

	lines, err := model.LogGetAllLines(ctx.Db, ctx.GroupID, ctx.LogName)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", errors.New("此log不存在，或条目数为空，名字是否正确？")
	}
	ctx.lines = lines

	err = backupBeforeUpload(&ctx)
	if err != nil {
		return "", err
	}

	var zlibBuffer bytes.Buffer
	w := zlib.NewWriter(&zlibBuffer)
	_, _ = w.Write(*ctx.data)
	_ = w.Close()

	url = uploadToSealBackends(ctx, &zlibBuffer)
	if errDB := model.LogSetUploadInfo(ctx.Db, ctx.GroupID, ctx.LogName, url); errDB != nil {
		ctx.Log.Errorf("记录Log上传信息失败: %v", errDB)
	}
	if len(url) == 0 {
		return "", errors.New("上传 log 到服务器失败，未能获取染色器链接")
	}
	return url, nil
}

// backupBeforeUpload 将导出的日志留档 zip，会修改 ctx.data
func backupBeforeUpload(ctx *UploadContext) error {
	fzip, _ := os.OpenFile(
		filepath.Join(ctx.Dir, filenameReplace(fmt.Sprintf(
			"%s_%s.%s.zip",
			ctx.GroupID, ctx.LogName, time.Now().Format("060102150405"),
		))),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	writer := zip.NewWriter(fzip)

	text := ""
	for _, i := range ctx.lines {
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
		"items":   ctx.lines,
	})
	if err == nil {
		fileWriter2, _ := writer.Create(ExportJsonFilename)
		_, _ = fileWriter2.Write(data)
		ctx.data = &data //nolint:govet
	}

	_ = writer.Close()
	_ = fzip.Close()

	return err
}

func filenameReplace(name string) string {
	re := regexp.MustCompile(`[/:\*\?"<>\|\\]`)
	return re.ReplaceAllString(name, "")
}

func uploadToSealBackends(ctx UploadContext, data io.Reader) string {
	// 逐个尝试所有后端地址
	for _, sealBackend := range ctx.Backends {
		if sealBackend == "" {
			continue
		}
		ret := uploadToBackend(ctx, sealBackend+"/dice/api/log", data)
		if ret != "" {
			return ret
		}
	}
	return ""
}
