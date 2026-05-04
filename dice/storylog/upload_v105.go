package storylog

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"

	"sealdice-core/dice/service"
	"sealdice-core/model"
)

// 一个不那么激进的改版
func uploadV105(env UploadEnv) (string, error) {
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

	lines, err := service.LogGetAllParquetLines(env.Db, env.GroupID, env.LogName)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", errors.New("此log不存在，或条目数为空，名字是否正确？")
	}

	txtPath, parquetPath, err := getLogTxtAndParquetFiles(env.LogName, lines)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Remove(txtPath)
		_ = os.Remove(parquetPath)
	}()

	if err := formatAndBackupV105(&env, txtPath, parquetPath); err != nil {
		return "", err
	}

	url = uploadToSealBackends(env, "Parquet", ExportParquetFilename, func(w io.Writer) error {
		parquetFile, err := os.Open(parquetPath)
		if err != nil {
			return err
		}
		defer func() { _ = parquetFile.Close() }()

		_, err = io.Copy(w, parquetFile)
		return err
	})
	if errDB := service.LogSetUploadInfo(env.Db, env.GroupID, env.LogName, url); errDB != nil {
		env.Log.Errorf("记录Log上传信息失败: %v", errDB)
	}
	if len(url) == 0 {
		return "", errors.New("上传 log 到服务器失败，未能获取染色器链接")
	}
	return url, nil
}

func getLogTxtAndParquetFiles(logName string, lines []model.LogOneItemParquet) (txtPath string, parquetPath string, err error) {
	txtPath, err = createTempTXTFile(logName, lines)
	if err != nil {
		return "", "", err
	}

	parquetPath, err = createTempParquetFile(logName, lines)
	if err != nil {
		_ = os.Remove(txtPath)
		return "", "", err
	}
	return txtPath, parquetPath, nil
}

func createTempTXTFile(logName string, lines []model.LogOneItemParquet) (path string, err error) {
	tempLog, err := os.CreateTemp("", exportTempFilePattern(logName, "storylog-v105", ".txt"))
	if err != nil {
		return "", errors.New("log导出出现未知错误")
	}
	path = tempLog.Name()
	removeOnError := true
	defer func() {
		if tempLog != nil {
			_ = tempLog.Close()
		}
		if removeOnError {
			_ = os.Remove(path)
		}
	}()

	if err := writeLogTXTParquet(tempLog, lines); err != nil {
		return "", fmt.Errorf("写入临时 TXT 文件失败: %w", err)
	}
	if err := tempLog.Close(); err != nil {
		return "", fmt.Errorf("关闭临时 TXT 文件失败: %w", err)
	}
	tempLog = nil
	removeOnError = false
	return path, nil
}

func createTempParquetFile(logName string, lines []model.LogOneItemParquet) (path string, err error) {
	tempParquet, err := os.CreateTemp("", exportTempFilePattern(logName, "storylog-v105", ".parquet"))
	if err != nil {
		return "", errors.New("创建临时 Parquet 文件失败")
	}
	path = tempParquet.Name()
	removeOnError := true
	defer func() {
		if tempParquet != nil {
			_ = tempParquet.Close()
		}
		if removeOnError {
			_ = os.Remove(path)
		}
	}()

	writer := parquet.NewGenericWriter[model.LogOneItemParquet](
		tempParquet,
		parquet.Compression(&zstd.Codec{}),
	)
	if _, err := writer.Write(lines); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("写入 Parquet 数据失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("关闭 Parquet writer 失败: %w", err)
	}
	if err := tempParquet.Close(); err != nil {
		return "", fmt.Errorf("关闭临时 Parquet 文件失败: %w", err)
	}
	tempParquet = nil
	removeOnError = false
	return path, nil
}

// formatAndBackupV105 将导出的日志直接写入本地 zip 文件。
func formatAndBackupV105(env *UploadEnv, txtPath string, parquetPath string) error {
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
	if err := copyFileToZip(writer, ExportTxtFilename, txtPath); err != nil {
		return fmt.Errorf("写入文本日志文件失败: %w", err)
	}
	if err := copyFileToZip(writer, ExportParquetFilename, parquetPath); err != nil {
		return fmt.Errorf("写入 Parquet 文件失败: %w", err)
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
