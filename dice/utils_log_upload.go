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
	"https://worker.firehomework.top",
}

var BackendUrls = []string{
	"https://worker.firehomework.top",
}

func TryGetBackendURL() {
	ret := _tryGetBackendBase("http://sealdice.com/listA.txt")
	if ret == "" {
		ret = _tryGetBackendBase("http://test1.sealdice.com/listA.txt")
	}
	if ret != "" {
		BackendUrls = append(backendUrlsRaw, strings.Split(ret, "\n")...) //nolint:gocritic
	}
}

// Hadoop里说这是元信息，咱也不知道Go里怎么称呼，反正先这样好了~
func uploadFsimage(name string) {
	// 其实我不喜欢处理这种multipart，但是我不知道怎么写POST传参——
	// 高情商：木落一定有他自己的想法和限制8
	client := &http.Client{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	field, err := writer.CreateFormField("name")
	if err == nil {
		_, _ = field.Write([]byte(name))
	}
}

func uploadFileToPinenutBase(backendURL string, md5 string, log *zap.SugaredLogger, name string, uniformID string, data io.Reader) string {
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
	// 添加md5字段，依靠此存放KV
	field, err = writer.CreateFormField("md5")
	if err == nil {
		_, _ = field.Write([]byte(md5))
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

// 毕竟会和海豹的设计完全不同，改一下名吧反正改回来也很方便（？）
func UploadFileToPinenut(log *zap.SugaredLogger, name string, uniformID string, md5 string, data io.Reader) string {
	// 逐个尝试所有后端地址
	for _, i := range BackendUrls {
		if i == "" {
			continue
		}
		ret := uploadFileToPinenutBase(i, md5, log, name, uniformID, data)
		if ret != "" {
			return ret
		}
	}
	return ""
}
