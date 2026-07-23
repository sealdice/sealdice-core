package api

import (
	"errors"
	"net/http"

	"github.com/Masterminds/semver/v3"
	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	dicesealpack "sealdice-core/dice/sealpack"
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
		ID        string `json:"id"`
		BackendID string `json:"backendID"`
		Url       string `json:"url"`
	}
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}
	id := params.ID
	if id == "" {
		id = params.BackendID
	}

	if err := myDice.StoreManager.StoreRemoveBackend(id, params.Url); err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{})
}

func storeEnableBackend(c echo.Context) error {
	return storeSetBackendEnabled(c, true)
}

func storeDisableBackend(c echo.Context) error {
	return storeSetBackendEnabled(c, false)
}

func storeSetBackendEnabled(c echo.Context, enabled bool) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}
	var params struct {
		ID        string `json:"id"`
		BackendID string `json:"backendID"`
		Url       string `json:"url"`
	}
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}
	id := params.ID
	if id == "" {
		id = params.BackendID
	}

	if err := myDice.StoreManager.StoreSetBackendEnabled(id, params.Url, enabled); err != nil {
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

func storePackageFiles(c echo.Context) error {
	files, err := myDice.StoreManager.StoreQueryPackageFiles(c.Param("namespace"), c.Param("package"), c.Param("version"))
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{
		"data": files,
	})
}

func storePackageFilePreview(c echo.Context) error {
	resp, err := myDice.StoreManager.StorePreviewPackageFile(c.Request().Context(), c.Param("namespace"), c.Param("package"), c.Param("version"), c.QueryParam("path"))
	if err != nil {
		return c.JSON(http.StatusBadGateway, Response{
			"result": false,
			"err":    err.Error(),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errText := http.StatusText(resp.StatusCode)
		if errText == "" {
			errText = "商店文件预览失败"
		}
		return c.JSON(resp.StatusCode, Response{
			"result": false,
			"err":    errText,
		})
	}

	headers := c.Response().Header()
	headers.Set("Cache-Control", "public, max-age=31536000, immutable")
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("Content-Security-Policy", filePreviewContentSecurityPolicy)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return c.Stream(http.StatusOK, contentType, resp.Body)
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

	target, err := myDice.StoreManager.ResolvePackage(params.ID, params.Version)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	if _, err := installStorePackage(target, true); err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{})
}

const maxStoreInstallListPackages = 200

type storeInstallListPackage struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type storeInstallListItemResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type storePackageInfoItemResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name,omitempty"`
	Error   string `json:"error,omitempty"`
}

type pendingStoreInstall struct {
	target      *dice.StorePackage
	resultIndex int
	lastError   error
}

func storePackageInfoList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	var params struct {
		Packages []storeInstallListPackage `json:"packages"`
	}
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}
	if len(params.Packages) == 0 {
		return Error(&c, "清单中没有可查询的扩展包", Response{})
	}
	if len(params.Packages) > maxStoreInstallListPackages {
		return Error(&c, "清单中的扩展包不能超过 200 个", Response{})
	}

	results := make([]storePackageInfoItemResult, 0, len(params.Packages))
	seen := make(map[string]struct{}, len(params.Packages))
	for _, item := range params.Packages {
		result := storePackageInfoItemResult{ID: item.ID, Version: item.Version}
		if _, exists := seen[item.ID]; exists {
			return Error(&c, "清单中存在重复的扩展包: "+item.ID, Response{})
		}
		seen[item.ID] = struct{}{}

		if installed, exists := myDice.PackageManager.Get(item.ID); exists && installed != nil && installed.Manifest != nil {
			result.Name = installed.Manifest.Package.Name
			results = append(results, result)
			continue
		}

		manifest, err := myDice.StoreManager.StoreQueryPackageManifest(c.Request().Context(), item.ID, item.Version)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.ID = manifest.Package.ID
			result.Version = manifest.Package.Version
			result.Name = manifest.Package.Name
		}
		results = append(results, result)
	}

	return Success(&c, Response{"data": results})
}

func storeInstallList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	var params struct {
		Packages []storeInstallListPackage `json:"packages"`
	}
	if err := c.Bind(&params); err != nil {
		return Error(&c, err.Error(), Response{})
	}
	if len(params.Packages) == 0 {
		return Error(&c, "清单中没有可安装的扩展包", Response{})
	}
	if len(params.Packages) > maxStoreInstallListPackages {
		return Error(&c, "清单中的扩展包不能超过 200 个", Response{})
	}

	results := make([]storeInstallListItemResult, len(params.Packages))
	pending := make([]pendingStoreInstall, 0, len(params.Packages))
	seen := make(map[string]struct{}, len(params.Packages))
	for index, item := range params.Packages {
		target, err := myDice.StoreManager.ResolvePackage(item.ID, item.Version)
		if err != nil {
			return Error(&c, err.Error(), Response{})
		}
		if _, exists := seen[target.ID]; exists {
			return Error(&c, "清单中存在重复的扩展包: "+target.ID, Response{})
		}
		seen[target.ID] = struct{}{}
		results[index] = storeInstallListItemResult{ID: target.ID, Version: target.Version}
		pending = append(pending, pendingStoreInstall{target: target, resultIndex: index})
	}

	installStorePackageBatch(results, pending, installStorePackage)

	installedCount := 0
	skippedCount := 0
	failedCount := 0
	for _, item := range results {
		switch item.Status {
		case "installed":
			installedCount++
		case "skipped":
			skippedCount++
		case "failed":
			failedCount++
		}
	}

	return Success(&c, Response{
		"data": Response{
			"items":     results,
			"installed": installedCount,
			"skipped":   skippedCount,
			"failed":    failedCount,
		},
	})
}

func installStorePackageBatch(
	results []storeInstallListItemResult,
	pending []pendingStoreInstall,
	installer func(*dice.StorePackage, bool) (string, error),
) {
	for len(pending) > 0 {
		nextPending := make([]pendingStoreInstall, 0, len(pending))
		installedThisPass := 0
		for _, item := range pending {
			status, err := installer(item.target, false)
			if err == nil {
				results[item.resultIndex].Status = status
				if status == "skipped" {
					results[item.resultIndex].Message = "已安装目标版本"
				} else {
					installedThisPass++
				}
				continue
			}

			var dependencyErr *dice.DependencyError
			if errors.As(err, &dependencyErr) {
				item.lastError = err
				nextPending = append(nextPending, item)
				continue
			}
			results[item.resultIndex].Status = "failed"
			results[item.resultIndex].Message = err.Error()
		}

		if len(nextPending) == 0 {
			break
		}
		if installedThisPass == 0 {
			for _, item := range nextPending {
				results[item.resultIndex].Status = "failed"
				results[item.resultIndex].Message = item.lastError.Error()
			}
			break
		}
		pending = nextPending
	}
}

func installStorePackage(target *dice.StorePackage, reinstallExactVersion bool) (string, error) {
	if installedPkg, exists := myDice.PackageManager.Get(target.ID); exists && installedPkg != nil && installedPkg.Manifest != nil {
		existingVer, existingErr := semver.NewVersion(installedPkg.Manifest.Package.Version)
		targetVer, targetErr := semver.NewVersion(target.Version)
		if existingErr == nil && targetErr == nil && targetVer.LessThan(existingVer) {
			return "", errors.New("当前已安装更高版本的扩展包")
		}
		if existingErr == nil && targetErr == nil && targetVer.Equal(existingVer) {
			if !reinstallExactVersion {
				return "skipped", nil
			}
			if err := myDice.PackageManager.Uninstall(target.ID, dicesealpack.UninstallModeKeepData); err != nil {
				return "", err
			}
		}
	}

	if err := myDice.PackageManager.InstallFromURL(target.Download.URL, target.Download.Hash); err != nil {
		return "", err
	}
	myDice.StoreManager.RefreshInstalled([]*dice.StorePackage{target})
	return "installed", nil
}

func storePreviewDownload(c echo.Context) error {
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

	preview, err := myDice.PackageManager.PreviewFromURL(target.Download.URL, target.Download.Hash)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"data": preview,
	})
}

func storeRating(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	return Error(&c, "not implemented", Response{})
}
