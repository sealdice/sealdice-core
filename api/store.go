package api

import (
	"github.com/labstack/echo/v4"
)

type storeElem struct {
	Source       string            `json:"source"` // official
	Type         string            `json:"type"`   // plugin | deck | reply
	Ext          string            `json:"ext"`    // .js | .json
	Key          string            `json:"key"`    // [<author>]<name>(<version>), e.g. [JustAnotherID]Demo(1.0.0)
	Namespace    string            `json:"namespace"`
	Name         string            `json:"name"`
	Authors      []string          `json:"authors"`
	Version      string            `json:"version"`
	License      string            `json:"license"`
	Desc         string            `json:"desc"`
	ReleaseTime  uint64            `json:"releaseTime"`
	UpdateTime   uint64            `json:"updateTime"`
	Tags         []string          `json:"tags"`
	Extra        map[string]string `json:"extra"`
	DownloadNum  uint64            `json:"downloadNum"`
	DownloadUrl  string            `json:"downloadUrl"`
	Hash         map[string]string `json:"hash"`
	HomePage     string            `json:"homePage"`
	SealVersion  string            `json:"sealVersion"`
	Dependencies map[string]string `json:"dependencies"`
}

func storeGetPage(c echo.Context) error {
	return Success(&c, Response{
		"data": []storeElem{
			{
				Source:      "official",
				Type:        "plugin",
				Ext:         ".js",
				Key:         "js-1",
				Name:        "测试插件1",
				Authors:     []string{"JustAnotherID"},
				Version:     "1.0.0",
				License:     "MIT",
				Desc:        "测试插件1",
				ReleaseTime: 1714318458,
				UpdateTime:  1714318458,
				Extra:       nil,
				DownloadNum: 114514,
				DownloadUrl: "",
			},
			{
				Source:      "official",
				Type:        "plugin",
				Ext:         ".js",
				Key:         "js-2",
				Name:        "测试插件2",
				Authors:     []string{"JustAnotherID"},
				Version:     "1.0.0",
				License:     "MIT",
				Desc:        "测试插件2-1\n测试插件2-2\n测试插件2-3",
				ReleaseTime: 1714318458,
				UpdateTime:  1714318458,
				Extra:       nil,
				DownloadNum: 1919810,
				DownloadUrl: "",
			},
		},
		"total":    2,
		"pageNum":  1,
		"pageSize": 2,
	})
}

func storeDownload(c echo.Context) error {
	return Error(&c, "not implemented", Response{})
}
