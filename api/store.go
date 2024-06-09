package api

import (
	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

var storeCache = make(map[string]*dice.StoreExt)

func checkInstalled(exts []*dice.StoreExt) {
	for _, ext := range exts {
		switch ext.Type {
		case dice.StoreExtTypeDeck:
			ext.Installed = myDice.InstalledDecks[ext.ID]
		case dice.StoreExtTypePlugin:
			ext.Installed = myDice.InstalledPlugins[ext.ID]
		default:
			// pass
		}
		if len(ext.ID) > 0 {
			storeCache[ext.ID] = ext
		}
	}
}

func storeRecommend(c echo.Context) error {
	data, err := myDice.StoreQueryRecommend()
	if err != nil {
		return Error(&c, err.Error(), Response{})
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
	params := dice.StoreQueryPageParams{}
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	page, err := myDice.StoreQueryPage(params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	data := page.Data
	checkInstalled(data)
	for _, elem := range data {
		storeCache[elem.ID] = elem
	}
	return Success(&c, Response{
		"data":     data,
		"pageNum":  page.PageNum,
		"pageSize": page.PageSize,
		"next":     page.Next,
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
	case dice.StoreExtTypeDeck:
		err = myDice.DeckDownload(target.Name, target.Ext, target.DownloadUrl, target.Hash)
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}
		target.Installed = true
		return Success(&c, Response{})
	case dice.StoreExtTypePlugin:
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
