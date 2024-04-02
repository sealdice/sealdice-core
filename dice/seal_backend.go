package dice

import (
	"strings"
	"time"

	"github.com/monaco-io/request"
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
