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

// nolint:unused // historical code
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
		splits := strings.Split(ret, "\n")
		for _, s := range splits {
			if s != "" {
				BackendUrls = append(BackendUrls, s)
			}
		}
	}
}
