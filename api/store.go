package api

import (
	"net/http"

	"github.com/Masterminds/semver/v3"
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
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}

	if err := myDice.StoreManager.StoreAddBackend(params.Url); err != nil {
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
		ID string `json:"id"`
	}
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}

	if err := myDice.StoreManager.StoreRemoveBackend(params.ID); err != nil {
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
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}

	page, err := myDice.StoreManager.StoreQueryPage(params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	myDice.StoreManager.RefreshInstalled(page.Data)
	return Success(&c, Response{
		"data":     page.Data,
		"pageNum":  page.PageNum,
		"pageSize": page.PageSize,
		"next":     page.Next,
	})
}

func storeDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	var params struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}

	target, ok := myDice.StoreManager.FindPackage(params.ID, params.Version)
	if !ok {
		return Error(&c, "未找到已缓存的商店包，请先刷新商店列表后重试", Response{})
	}

	if installedPkg, exists := myDice.PackageManager.Get(target.ID); exists && installedPkg != nil && installedPkg.Manifest != nil {
		existingVer, existingErr := semver.NewVersion(installedPkg.Manifest.Package.Version)
		targetVer, targetErr := semver.NewVersion(target.Version)
		if existingErr == nil && targetErr == nil && !targetVer.GreaterThan(existingVer) {
			return Error(&c, "当前已安装相同或更高版本的扩展包", Response{})
		}
	}

	if err := myDice.PackageManager.InstallFromURL(target.Download.URL, target.Download.Hash); err != nil {
		return Error(&c, err.Error(), Response{})
	}

	myDice.StoreManager.RefreshInstalled([]*dice.StorePackage{target})
	return Success(&c, Response{})
}

func storeRating(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	return Error(&c, "not implemented", Response{})
}
