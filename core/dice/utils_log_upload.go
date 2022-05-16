package dice

import (
	"bytes"
	"encoding/json"
	"github.com/monaco-io/request"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
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
	"https://dice.weizaima.com",
}

var BackendUrls = []string{
	"https://dice.weizaima.com",
}

func TryGetBackendUrl() {
	ret := _tryGetBackendBase("https://sealdice.com/list.txt")
	if ret == "" {
		ret = _tryGetBackendBase("https://test1.sealdice.com/list.txt")
	}
	if ret != "" {
		BackendUrls = append(backendUrlsRaw, strings.Split(ret, "\n")...)
	}
}

func uploadFileToWeizaimaBase(backendUrl string, log *zap.SugaredLogger, name string, uniformId string, data io.Reader) string {
	client := &http.Client{}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	field, err := writer.CreateFormField("name")
	if err == nil {
		_, _ = field.Write([]byte(name))
	}

	field, err = writer.CreateFormField("uniform_id")
	if err == nil {
		_, _ = field.Write([]byte(uniformId))
	}

	field, err = writer.CreateFormField("client")
	if err == nil {
		_, _ = field.Write([]byte("SealDice"))
	}

	part, _ := writer.CreateFormFile("file", "log-zlib-compressed")
	io.Copy(part, data)
	writer.Close()

	req, err := http.NewRequest("PUT", backendUrl+"/dice/api/log", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}

	//req.Header.Set("authority", "transfer.sh")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}

	defer resp.Body.Close()
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf(err.Error())
		return ""
	}

	var ret struct {
		Url string `json:"url"`
	}
	_ = json.Unmarshal(bodyText, &ret)
	return ret.Url
}

func UploadFileToWeizaima(log *zap.SugaredLogger, name string, uniformId string, data io.Reader) string {
	// 逐个尝试所有后端地址
	for _, i := range BackendUrls {
		ret := uploadFileToWeizaimaBase(i, log, name, uniformId, data)
		if ret != "" {
			return ret
		}
	}
	return ""
}
