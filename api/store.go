package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func storeBackendList(c echo.Context) error {
	backends := myDice.StoreManager.StoreBackendList()
	return Success(&c, Response{
		"data": backends,
	})
}

func storeAddBackend(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}
	var params struct {
		Url string `json:"url"`
	}
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	err = myDice.StoreManager.StoreAddBackend(params.Url)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{})
}

func storeRemoveBackend(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}
	var params struct {
		ID string `query:"id"`
	}
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	err = myDice.StoreManager.StoreRemoveBackend(params.ID)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{})
}

func storeRecommend(c echo.Context) error {
	data, err := myDice.StoreManager.StoreQueryRecommend()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	myDice.StoreManager.RefreshInstalled(data)
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

	page, err := myDice.StoreManager.StoreQueryPage(params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	data := page.Data
	myDice.StoreManager.RefreshInstalled(data)
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

	target, ok := myDice.StoreManager.FindExt(params.ID)
	if ok && target.Installed {
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
