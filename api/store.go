package api

import (
	"github.com/labstack/echo/v4"
)

var storeCache = make(map[string]*storeElem)

type storeElemType string

const (
	storeElemTypePlugin storeElemType = "plugin"
	storeElemTypeDeck   storeElemType = "deck"
	storeElemTypeReply  storeElemType = "reply"
)

type storeElem struct {
	ID        string `json:"id"` // @<namespace>/<key>@<version>, e.g. @seal/example@1.0.0
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`

	Source string        `json:"source"` // official
	Type   storeElemType `json:"type"`   // plugin | deck | reply
	Ext    string        `json:"ext"`    // .js | .json |...

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

var demos = []storeElem{{
	Source:      "JustAnotherID",
	Type:        storeElemTypeDeck,
	Ext:         ".toml",
	ID:          "@id/toml牌堆样例@1.0.0",
	Name:        "toml牌堆样例",
	Version:     "1.0.0",
	Authors:     []string{"JustAnotherID"},
	License:     "MIT",
	Desc:        "这是一个toml牌堆的样例",
	ReleaseTime: 1693670400,
	UpdateTime:  1693670400,
	DownloadUrl: "https://ghproxy.com/https://raw.githubusercontent.com/JustAnotherID/just-another-seal-mod-repo/master/deck/toml%E7%89%8C%E5%A0%86%E6%A0%B7%E4%BE%8B.toml",
}}

func checkInstalled(exts []*storeElem) {
	for _, ext := range exts {
		switch ext.Type {
		case storeElemTypeDeck:
			ext.Installed = myDice.InstalledDecks[ext.ID]
		case storeElemTypePlugin:
			ext.Installed = myDice.InstalledPlugins[ext.ID]
		}
		if len(ext.ID) > 0 {
			storeCache[ext.ID] = ext
		}
	}
}

func storeRecommend(c echo.Context) error {
	var data []*storeElem
	for _, ext := range demos {
		temp := ext
		data = append(data, &temp)
	}
	checkInstalled(data)
	for _, elem := range data {
		storeCache[elem.ID] = elem
	}
	return Success(&c, Response{
		"data": data,
	})
}

func storeGetPage(c echo.Context) error {
	query := struct {
		PageNum  int    `query:"pageNum"`
		PageSize int    `query:"pageSize"`
		Author   string `query:"author"`
		Name     string `query:"name"`
		SortBy   string `query:"sortBy"`
		Order    string `query:"order"`
	}{}
	_ = c.Bind(&query)

	var data []*storeElem
	for _, ext := range demos {
		temp := ext
		data = append(data, &temp)
	}
	checkInstalled(data)
	for _, elem := range data {
		storeCache[elem.ID] = elem
	}

	return Success(&c, Response{
		"data":     data,
		"pageNum":  query.PageNum,
		"pageSize": 5,
		"next":     query.PageNum < 10,
	})
}

func storeDownload(c echo.Context) error {
	var params struct {
		ID string `json:"id"`
	}
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	target := storeCache[params.ID]
	if target.Installed {
		return Error(&c, "请勿重复安装", Response{})
	}
	switch target.Type {
	case storeElemTypeDeck:
		err = myDice.DeckDownload(target.Name, target.Ext, target.DownloadUrl, target.Hash)
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}
		target.Installed = true
		return Success(&c, Response{})
	case storeElemTypePlugin:
		err = myDice.JsDownload(target.Name, target.DownloadUrl, target.Hash)
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}
		target.Installed = true
		return Success(&c, Response{})
	default:
		return Error(&c, "该类型的扩展目前不支持下载", Response{})
	}
}

func storeRating(c echo.Context) error {
	return Error(&c, "not implemented", Response{})
}
