package storylog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"sealdice-core/dice/model"
)

type UploadContext struct {
	Dir      string
	Db       *sqlx.DB
	Log      *zap.SugaredLogger
	Backends []string
	Version  StoryVersion

	LogName   string
	UniformID string
	GroupID   string
	Token     string

	lines []*model.LogOneItem
	data  *[]byte
}

func Upload(ctx UploadContext) (string, error) {
	if ctx.Version == StoryVersionV1 {
		return uploadV1(ctx)
	}
	return "", errors.New("未指定日志版本")
}

func uploadToBackend(ctx UploadContext, backend string, data io.Reader) string {
	client := &http.Client{}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	field, err := writer.CreateFormField("name")
	if err == nil {
		_, _ = field.Write([]byte(ctx.LogName))
	}

	field, err = writer.CreateFormField("uniform_id")
	if err == nil {
		_, _ = field.Write([]byte(ctx.UniformID))
	}

	field, err = writer.CreateFormField("client")
	if err == nil {
		_, _ = field.Write([]byte("SealDice"))
	}

	field, err = writer.CreateFormField("version")
	if err == nil {
		_, _ = field.Write([]byte(strconv.Itoa(int(ctx.Version))))
	}

	part, _ := writer.CreateFormFile("file", "log-zlib-compressed")
	_, _ = io.Copy(part, data)
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPut, backend, body)
	if err != nil {
		ctx.Log.Errorf(err.Error())
		return ""
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if len(ctx.Token) > 0 {
		req.Header.Set("Authorization", "Bearer "+ctx.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		ctx.Log.Errorf(err.Error())
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.Log.Errorf(err.Error())
		return ""
	}

	var ret struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(bodyText, &ret)
	if ret.URL == "" {
		ctx.Log.Error("日志上传的返回结果异常:", string(bodyText))
	}
	return ret.URL
}
