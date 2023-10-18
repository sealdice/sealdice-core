package dice

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/monaco-io/request"
	"go.uber.org/zap"
)

func _tryGetBackendBase(url string) string {
	c := request.Client{
		URL:     url,
		Method:  "GET",
		Timeout: 10 * time.Second,
	}
	resp := c.Send()
	if resp.Code() == 200 {
		return resp.String()
	}
	return ""
}

var backendUrlsRaw = []string{
	"http://dice.weizaima.com",
}

var BackendUrls = []string{
	"http://dice.weizaima.com",
}

func TryGetBackendURL() {
	ret := _tryGetBackendBase("http://sealdice.com/list.txt")
	if ret == "" {
		ret = _tryGetBackendBase("http://test1.sealdice.com/list.txt")
	}
	if ret != "" {
		BackendUrls = append(backendUrlsRaw, strings.Split(ret, "\n")...) //nolint:gocritic
	}
}

func uploadFileToWeizaimaBase(backendURL string, log *zap.SugaredLogger, name string, uniformID string, data io.Reader) string {
	client := &http.Client{}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	field, err := writer.CreateFormField("name")
	if err == nil {
		_, _ = field.Write([]byte(name))
	}

	field, err = writer.CreateFormField("uniform_id")
	if err == nil {
		_, _ = field.Write([]byte(uniformID))
	}

	field, err = writer.CreateFormField("client")
	if err == nil {
		_, _ = field.Write([]byte("SealDice"))
	}

	part, _ := writer.CreateFormFile("file", "log-zlib-compressed")
	_, _ = io.Copy(part, data)
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPut, backendURL+"/dice/api/log", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}

	// req.Header.Set("authority", "transfer.sh")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}

	var ret struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(bodyText, &ret)
	if ret.URL == "" {
		log.Error("日志上传的返回结果异常:", string(bodyText))
	}
	return ret.URL
}

func UploadFileToWeizaima(log *zap.SugaredLogger, name string, uniformID string, data io.Reader) string {
	// 逐个尝试所有后端地址
	for _, i := range BackendUrls {
		if i == "" {
			continue
		}
		ret := uploadFileToWeizaimaBase(i, log, name, uniformID, data)
		if ret != "" {
			return ret
		}
	}
	return ""
}
