package api

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/monaco-io/request"
)

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func doAuth(c echo.Context) bool {
	token := c.Request().Header.Get("token")
	if token == "" {
		token = c.QueryParam("token")
	}
	if myDice.Parent.AccessTokens[token] {
		return true
	}
	return false
}

func GetHexData(c echo.Context, method string, name string) (value []byte, finished bool) {
	var err error
	var strValue string
	// var exists bool

	if method == "GET" {
		strValue = c.Param(name)
	} else if method == "POST" {
		strValue = c.FormValue(name)
	}

	// if !exists {
	// 	c.String(http.StatusNotAcceptable, "")
	// 	return nil, true
	// }

	value, err = hex.DecodeString(strValue)
	if err != nil {
		_ = c.String(http.StatusBadRequest, "")
		return nil, true
	}

	return value, false
}

var times = 0

func getGithubAvatar(c echo.Context) error {
	times += 1
	if times > 500 {
		// 请求次数过多
		return c.JSON(http.StatusNotFound, "")
	}

	uid := c.Param("uid")
	req := request.Client{
		URL:    fmt.Sprintf("https://avatars.githubusercontent.com/%s?s=200", uid),
		Method: "GET",
	}

	resp := req.Send()
	if resp.OK() {
		// 设置缓存时间为3天
		c.Response().Header().Set("Cache-Control", "max-age=259200")

		return c.Blob(http.StatusOK, resp.ContentType(), resp.Bytes())
	}
	return c.JSON(http.StatusNotFound, "")
}
