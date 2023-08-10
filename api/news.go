package api

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/monaco-io/request"
	"github.com/tdewolff/minify/v2"
	hm "github.com/tdewolff/minify/v2/html"
)

func getNews(c echo.Context) error {
	req := request.Client{
		URL:    "https://dice.weizaima.com/dice/api/news",
		Method: "GET",
	}

	resp := req.Send()
	if resp.OK() {
		// 获取新闻后，通过计算压缩空白后的新闻字符串md5来判断是否有更新
		news, mark, err := getNewsMd5(resp.Bytes())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"result": false,
				"err":    "无法解析新闻",
			})
		}

		// 设置缓存时间为3天
		c.Response().Header().Set("Cache-Control", "max-age=120")
		return c.JSON(http.StatusOK, map[string]interface{}{
			"result":   true,
			"checked":  mark == myDice.NewsMark,
			"news":     news,
			"newsMark": mark,
		})
	}
	return c.JSON(http.StatusNotFound, "")
}

func checkNews(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		NewsMark string `json:"newsMark"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		myDice.NewsMark = v.NewsMark
		myDice.MarkModified()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"result":   true,
			"newsMark": v.NewsMark,
		})
	}

	return c.JSON(http.StatusBadRequest, map[string]interface{}{
		"result": false,
	})
}

// getMinifyNewsMd5 获取新闻和md5
func getNewsMd5(news []byte) (string, string, error) {
	m := minify.New()
	m.AddFunc("text/html", hm.Minify)
	miniH, err := m.Bytes("text/html", news)
	if err != nil {
		return "", "", err
	}

	mark := md5.New()
	mark.Write(miniH)
	return string(miniH), hex.EncodeToString(mark.Sum(nil)), nil
}
