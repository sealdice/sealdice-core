package storylog

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"

	"sealdice-core/dice/service"
	"sealdice-core/model"
	"sealdice-core/utils"
)

func GetLogTxtAndParquetFile(env UploadEnv) (*os.File, *bytes.Buffer, error) {
	// 创建临时文件
	parquetBuffer := parquet.NewGenericBuffer[model.LogOneItemParquet](
		parquet.SortingRowGroupConfig(
			parquet.SortingColumns(
				parquet.Ascending("id"),
				parquet.Ascending("time"),
			),
		))
	buf := new(bytes.Buffer)
	tempLog, err := os.CreateTemp("", fmt.Sprintf(
		"%s(*).txt",
		utils.FilenameClean("sealdice_v105_prefix_"),
	))
	if err != nil {
		return nil, nil, errors.New("log导出出现未知错误")
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tempLog.Name()) //nolint:gosec
		}
	}()

	counter := 0
	currentCursor := paginator.Cursor{} // 初始游标为空

	// 腾讯元宝: 创建带10秒超时的 context
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // 确保 context 被正确取消

	// 腾讯元宝: 使用 channel 接收 goroutine 的结果
	resultCh := make(chan error, 1)
	go func() {
		defer close(resultCh) // 确保 channel 被关闭

		for {
			select {
			case <-ctxWithTimeout.Done(): // 检查是否超时或被取消
				resultCh <- errors.New("日志导出超时（10秒限制），请尝试减少数据量或联系管理员")
				return
			default:
				// 获取当前游标对应的数据
				cursorLines, cursor, err0 := service.LogGetExportCursorLines(env.Db, env.GroupID, env.LogName, currentCursor)
				if err0 != nil {
					resultCh <- err0
					return
				}

				// 写入当前批次的数据
				for _, line := range cursorLines {
					timeTxt := time.Unix(line.Time, 0).Format("2006-01-02 15:04:05")
					text := fmt.Sprintf("%s(%v) %s\n%s\n\n", line.Nickname, line.IMUserID, timeTxt, line.Message)
					_, _ = tempLog.WriteString(text)
					counter++
				}
				// ========== 新增：每批写入后强制同步 ==========
				if err = tempLog.Sync(); err != nil { // 确保批次数据落盘
					resultCh <- fmt.Errorf("批次同步失败: %w", err)
				}

				_, err0 = parquetBuffer.Write(cursorLines)
				if err0 != nil {
					resultCh <- err0
					return
				}

				// 如果没有下一页，则成功完成
				if cursor.After == nil {
					resultCh <- nil
					return
				}

				// 更新游标，继续获取下一页
				currentCursor.After = cursor.After
			}
		}
	}()

	// 等待 goroutine 完成或超时
	if err = <-resultCh; err != nil {
		return nil, nil, err
	}

	// 如果没有任何数据，返回错误
	if counter == 0 {
		return nil, nil, errors.New("此log不存在，或条目数为空，名字是否正确？")
	}
	// 对其进行压缩
	compressOption := parquet.Compression(&zstd.Codec{})
	writer := parquet.NewGenericWriter[model.LogOneItemParquet](buf, compressOption)
	// 2. 确保文件指针回到开头
	if _, err = tempLog.Seek(0, 0); err != nil {
		return nil, nil, fmt.Errorf("重置文件指针失败: %w", err)
	}
	// 写入到writer中
	_, err = parquet.CopyRows(writer, parquetBuffer.Rows())
	if err != nil {
		return nil, nil, err
	}
	// 关闭writer让他写入到内存
	err = writer.Close()
	if err != nil {
		return nil, nil, err
	}
	return tempLog, buf, nil
}

// 一个不那么激进的改版
func uploadV105(env UploadEnv) (string, error) {
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

	file, b, err := GetLogTxtAndParquetFile(env)
	if err != nil {
		return "", err
	}
	err = formatAndBackupV105(&env, file, b)
	if err != nil {
		return "", err
	}

	url = parquetUploadToSealBackends(env, b)
	if errDB := service.LogSetUploadInfo(env.Db, env.GroupID, env.LogName, url); errDB != nil {
		env.Log.Errorf("记录Log上传信息失败: %v", errDB)
	}
	if len(url) == 0 {
		return "", errors.New("上传 log 到服务器失败，未能获取染色器链接")
	}
	return url, nil
}

// formatAndBackup 将导出的日志序列化到 env.data 并存储为本地 zip
func formatAndBackupV105(env *UploadEnv, tempLog *os.File, parquetFile *bytes.Buffer) error {
	fzip, _ := os.OpenFile(
		filepath.Join(env.Dir, utils.FilenameClean(fmt.Sprintf(
			"%s_%s.%s.zip",
			env.GroupID, env.LogName, time.Now().Format("060102150405"),
		))),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o600,
	)
	writer := zip.NewWriter(fzip)
	var err error
	// 写入README文件
	{
		wr, _ := writer.Create(ExportReadmeFilename)
		_, _ = wr.Write([]byte(ExportReadmeContent))
	}
	// 写入文本TXT文件

	fileWriter, _ := writer.Create(ExportTxtFilename)
	if _, err = io.Copy(fileWriter, tempLog); err != nil {
		return fmt.Errorf("写入文本日志文件失败: %w", err)
	}

	// 写入parquet文件
	parquetWriter, err := writer.Create(ExportParquetFilename)
	if err != nil {
		return fmt.Errorf("创建Parquet文件失败: %w", err)
	}
	if _, err = parquetWriter.Write(parquetFile.Bytes()); err != nil {
		return fmt.Errorf("写入Parquet文件失败: %w", err)
	}

	_ = writer.Close()
	_ = fzip.Close()

	return err
}

func parquetUploadToSealBackends(env UploadEnv, data io.Reader) string {
	// 逐个尝试所有后端地址
	for _, backend := range env.Backends {
		if backend == "" {
			continue
		}
		ret := uploadToBackendParquet(env, backend, data)
		if ret != "" {
			return ret
		}
	}
	return ""
}

func uploadToBackendParquet(env UploadEnv, backend string, data io.Reader) string {
	client := &http.Client{}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	field, err := writer.CreateFormField("name")
	if err == nil {
		_, _ = field.Write([]byte(env.LogName))
	}

	field, err = writer.CreateFormField("uniform_id")
	if err == nil {
		_, _ = field.Write([]byte(env.UniformID))
	}

	field, err = writer.CreateFormField("client")
	if err == nil {
		_, _ = field.Write([]byte("Parquet"))
	}

	field, err = writer.CreateFormField("version")
	if err == nil {
		_, _ = field.Write([]byte(strconv.Itoa(int(env.Version))))
	}

	part, _ := writer.CreateFormFile("file", "log-zlib-compressed")
	_, _ = io.Copy(part, data)
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPut, backend, body)
	if err != nil {
		env.Log.Errorf(err.Error())
		return ""
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if len(env.Token) > 0 {
		req.Header.Set("Authorization", "Bearer "+env.Token)
	}

	resp, err := client.Do(req) //nolint:gosec
	if err != nil {
		env.Log.Errorf(err.Error())
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		env.Log.Errorf(err.Error())
		return ""
	}

	var ret struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(bodyText, &ret)
	if ret.URL == "" {
		env.Log.Error("日志上传的返回结果异常:", string(bodyText))
	}
	return ret.URL
}
