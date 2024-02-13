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
	"http://127.0.0.1:8787",
}

var BackendUrls = []string{
	"http://127.0.0.1:8787",
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
func uploadFsimage(log *zap.SugaredLogger, data interface{}) string {
	// 逐个尝试所有后端地址
	for _, i := range BackendUrls {
		if i == "" {
			continue
		}
		ret := sendJSONPostRequest(i, log, data)
		if ret != "" {
			return ret
		}
	}
	return ""
}

func sendJSONPostRequest(backendURL string, log *zap.SugaredLogger, data interface{}) string {
	client := &http.Client{}
	// 将数据编码为 JSON 格式
	jsonData, err := json.Marshal(data)
	log.Infof("JSON 数据：%s", string(jsonData))
	if err != nil {
		log.Errorf("上传元数据 JSON 编码失败")
		return ""
	}
	// 构建 POST 请求
	req, err := http.NewRequest(http.MethodPost, backendURL+"/dice/api/log", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Errorf("构建请求错误: %v", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("发送请求错误: %v", err)
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
		log.Error("日志元信息上传的返回结果异常:", string(bodyText))
	}
	return ret.URL
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
		STATUS string `json:"status"`
	}
	_ = json.Unmarshal(bodyText, &ret)
	if ret.STATUS != "success" {
		log.Error("日志分块上传的返回结果异常:", string(bodyText))
	}
	return ret.STATUS
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
