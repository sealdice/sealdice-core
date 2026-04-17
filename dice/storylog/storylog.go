package storylog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"sealdice-core/utils/dboperator/engine"
)

type UploadEnv struct {
	Dir      string
	Db       engine.DatabaseOperator
	Log      *zap.SugaredLogger
	Backends []string
	Version  StoryVersion

	LogName   string
	UniformID string
	GroupID   string
	Token     string
}

func Upload(env UploadEnv) (string, error) {
	if env.Version == StoryVersionV1 {
		return uploadV1(env)
	}
	if env.Version == StoryVersionV105 {
		return uploadV105(env)
	}
	return "", errors.New("未指定日志版本")
}

type uploadPartWriter func(io.Writer) error

func uploadToSealBackends(env UploadEnv, clientName string, fileName string, writePart uploadPartWriter) string {
	for _, backend := range env.Backends {
		if backend == "" {
			continue
		}
		ret := uploadToBackend(env, backend, clientName, fileName, writePart)
		if ret != "" {
			return ret
		}
	}
	return ""
}

func uploadToBackend(env UploadEnv, backend string, clientName string, fileName string, writePart uploadPartWriter) string {
	client := &http.Client{}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writeFormField(writer, "name", env.LogName); err != nil {
		env.Log.Errorf("写入上传字段 name 失败: %v", err)
		return ""
	}
	if err := writeFormField(writer, "uniform_id", env.UniformID); err != nil {
		env.Log.Errorf("写入上传字段 uniform_id 失败: %v", err)
		return ""
	}
	if err := writeFormField(writer, "client", clientName); err != nil {
		env.Log.Errorf("写入上传字段 client 失败: %v", err)
		return ""
	}
	if err := writeFormField(writer, "version", strconv.Itoa(int(env.Version))); err != nil {
		env.Log.Errorf("写入上传字段 version 失败: %v", err)
		return ""
	}

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		env.Log.Errorf("创建上传文件字段失败: %v", err)
		return ""
	}
	if err := writePart(part); err != nil {
		env.Log.Errorf("写入上传文件内容失败: %v", err)
		return ""
	}
	if err := writer.Close(); err != nil {
		env.Log.Errorf("关闭 multipart writer 失败: %v", err)
		return ""
	}

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

func writeFormField(writer *multipart.Writer, name string, value string) error {
	field, err := writer.CreateFormField(name)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(field, value); err != nil {
		return fmt.Errorf("write form field %s: %w", name, err)
	}
	return nil
}
