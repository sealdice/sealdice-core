package api

import (
	"github.com/labstack/echo/v4"
)

type storeElem struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`
	Version   string `json:"version"`
	Key       string `json:"key"` // @<namespace>/<id>@<version>, e.g. @seal/example@1.0.0
	Installed bool   `json:"installed"`

	Source string `json:"source"` // official
	Type   string `json:"type"`   // plugin | deck | reply
	Ext    string `json:"ext"`    // .js | .json |...

	Name         string            `json:"name"`
	Authors      []string          `json:"authors"`
	Desc         string            `json:"desc"`
	License      string            `json:"license"`
	ReleaseTime  uint64            `json:"releaseTime"`
	UpdateTime   uint64            `json:"updateTime"`
	Tags         []string          `json:"tags"`
	Rate         float64           `json:"rate"`
	Extra        map[string]string `json:"extra"`
	DownloadNum  uint64            `json:"downloadNum"`
	DownloadUrl  string            `json:"downloadUrl"`
	Hash         map[string]string `json:"hash"`
	HomePage     string            `json:"homePage"`
	SealVersion  string            `json:"sealVersion"`
	Dependencies map[string]string `json:"dependencies"`
}

var demos = []storeElem{
	{
		Source:    "official",
		Type:      "plugin",
		Ext:       ".js",
		Key:       "js-1",
		Namespace: "JustAnotherID",
		Name:      "野兽插件",
		Authors:   []string{"JustAnotherID"},
		Version:   "1.0.0",
		License:   "MIT",
		Desc:      "测试用占位插件",
		Tags: []string{
			"下北泽",
			"会员制插件",
			"逸一时误一世",
			"逸久逸久罢已龄",
		},
		Rate:        5,
		ReleaseTime: 1716867914,
		UpdateTime:  1716867914,
		Extra:       nil,
		DownloadNum: 114514,
		DownloadUrl: "",
	},
}

func storeRecommend(c echo.Context) error {
	return Success(&c, Response{
		"data": demos,
	})
}

func storeGetPage(c echo.Context) error {
	return Success(&c, Response{
		"data":     demos,
		"total":    2,
		"pageNum":  1,
		"pageSize": 2,
	})
}

func storeDownload(c echo.Context) error {
	return Error(&c, "not implemented", Response{})
}

func storeRating(c echo.Context) error {
	return Error(&c, "not implemented", Response{})
}
