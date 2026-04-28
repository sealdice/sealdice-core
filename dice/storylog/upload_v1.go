package storylog

import (
	"archive/zip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"sealdice-core/dice/service"
	"sealdice-core/model"
)

func uploadV1(env UploadEnv) (string, error) {
	if err := os.MkdirAll(env.Dir, 0o755); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}

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

	lines, err := service.LogGetAllLines(env.Db, env.GroupID, env.LogName)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", errors.New("此log不存在，或条目数为空，名字是否正确？")
	}

	if err := formatAndBackup(&env, lines); err != nil {
		return "", err
	}

	url = uploadToSealBackends(env, "SealDice", "log-zlib-compressed", func(w io.Writer) error {
		zlibWriter := zlib.NewWriter(w)
		if err := writeStoryV1JSON(zlibWriter, lines); err != nil {
			_ = zlibWriter.Close()
			return err
		}
		return zlibWriter.Close()
	})
	if errDB := service.LogSetUploadInfo(env.Db, env.GroupID, env.LogName, url); errDB != nil {
		env.Log.Errorf("记录Log上传信息失败: %v", errDB)
	}
	if len(url) == 0 {
		return "", errors.New("上传 log 到服务器失败，未能获取染色器链接")
	}
	return url, nil
}

// formatAndBackup 将导出的日志直接写入本地 zip 文件。
func formatAndBackup(env *UploadEnv, lines []*model.LogOneItem) (err error) {
	zipPath := filepath.Join(env.Dir, exportZipFilename(env.GroupID, env.LogName, time.Now()))
	fzip, err := os.OpenFile(zipPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建本地备份 zip 失败: %w", err)
	}
	removeOnError := true
	defer func() {
		if fzip != nil {
			_ = fzip.Close()
		}
		if removeOnError {
			_ = os.Remove(zipPath)
		}
	}()

	writer := zip.NewWriter(fzip)
	defer func() {
		if writer != nil {
			_ = writer.Close()
		}
	}()

	if err := writeReadmeToZip(writer); err != nil {
		return fmt.Errorf("写入 README 失败: %w", err)
	}

	txtWriter, err := writer.Create(ExportTxtFilename)
	if err != nil {
		return fmt.Errorf("创建 TXT 导出文件失败: %w", err)
	}
	if err := WriteLogTXT(txtWriter, lines); err != nil {
		return fmt.Errorf("写入 TXT 导出文件失败: %w", err)
	}

	jsonWriter, err := writer.Create(ExportJsonFilename)
	if err != nil {
		return fmt.Errorf("创建 JSON 导出文件失败: %w", err)
	}
	if err := writeStoryV1JSON(jsonWriter, lines); err != nil {
		return fmt.Errorf("写入 JSON 导出文件失败: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭本地备份 zip 失败: %w", err)
	}
	writer = nil
	if err := fzip.Close(); err != nil {
		return fmt.Errorf("关闭本地备份文件失败: %w", err)
	}
	fzip = nil
	removeOnError = false
	return nil
}
