package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice/sealpkg"
)

// ================== 扩展包管理 API ==================
//
// 扩展包 (sealpkg) 是 SealDice 的模块化扩展机制，支持：
// - JS 脚本扩展
// - 牌堆 (decks)
// - 自定义回复 (reply)
// - 帮助文档 (helpdoc)
// - 游戏模板 (template)
//
// 包 ID 格式：作者/包名（如 "test/my-package"）
// 由于 Echo 路由不支持路径参数包含 /，所有需要包 ID 的接口
// 都支持通过查询参数 ?id=xxx 传递，优先级高于路径参数。
//
// 状态说明：
// - enabled: 已启用，资源会被加载
// - disabled: 已禁用，资源不会被加载
//
// 重载机制：
// 启用/禁用包后，部分资源需要重载才能生效：
// - JS 脚本：调用 reload 或 js/reload 接口
// - 牌堆：调用 reload 或 deck/reload 接口
// - 其他资源：可能需要重启

// packageList 获取所有已安装的扩展包列表
// GET /package/list
// 返回: { data: []PackageInstance, result: true }
func packageList(c echo.Context) error {
	packages := myDice.PackageManager.List()
	return Success(&c, Response{
		"data": packages,
	})
}

// getPackageIDFromRequest 从请求中获取包ID
// 优先使用查询参数 ?id=xxx（支持包含 / 的ID）
// 其次使用路径参数 :id
func getPackageIDFromRequest(c echo.Context) string {
	if id := c.QueryParam("id"); id != "" {
		return id
	}
	return c.Param("id")
}

// packageGet 获取指定扩展包的详细信息
// GET /package/:id 或 GET /package/_?id=xxx
// 返回: { data: PackageInstance, result: true }
func packageGet(c echo.Context) error {
	pkgID := getPackageIDFromRequest(c)
	pkg, exists := myDice.PackageManager.Get(pkgID)
	if !exists {
		return Error(&c, "扩展包不存在", Response{})
	}
	return Success(&c, Response{
		"data": pkg,
	})
}

// packageInstall 从本地文件安装扩展包
// POST /package/install
// 参数: { path: string } - 本地 .sealpkg 文件的绝对路径
// 返回: { message: string, result: true }
// 注意: 安装后包默认为 disabled 状态，需要调用 enable 启用
func packageInstall(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}

	var params struct {
		Path string `json:"path"` // 本地 zip 文件路径
	}
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	err = myDice.PackageManager.Install(params.Path)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"message": "扩展包安装成功",
	})
}

// packageInstallFromURL 从远程 URL 安装扩展包
// POST /package/install-from-url
// 参数: { url: string } - 远程 .sealpkg 文件的 URL
// 返回: { message: string, result: true }
func packageInstallFromURL(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}

	var params struct {
		URL string `json:"url"` // 远程 zip 文件 URL
	}
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	err = myDice.PackageManager.InstallFromURL(params.URL)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"message": "扩展包安装成功",
	})
}

// packageUninstall 卸载扩展包
// POST /package/uninstall
// 参数:
//   - id: string - 包 ID
//   - mode: string - 卸载模式
//   - "full": 完全删除（默认）
//   - "keep_data": 保留用户数据
//   - "disable": 仅禁用，不删除文件
//
// 返回: { message: string, result: true }
// 注意: 卸载后需要调用 js/reload 清理内存中的 JS 扩展
func packageUninstall(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}

	var params struct {
		ID   string                `json:"id"`
		Mode sealpkg.UninstallMode `json:"mode"` // full, keep_data, disable
	}
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	// 默认模式为 full
	if params.Mode == "" {
		params.Mode = sealpkg.UninstallModeFull
	}

	err = myDice.PackageManager.Uninstall(params.ID, params.Mode)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"message": "扩展包卸载成功",
	})
}

// packageEnable 启用扩展包
// POST /package/enable
// 参数: { id: string }
// 返回: { data: EnableResult, result: true }
// EnableResult 包含:
//   - success: bool - 是否成功
//   - message: string - 提示信息
//   - reloadNeeded: bool - 是否需要重载
//   - reloadHints: []string - 需要重载的资源类型提示
func packageEnable(c echo.Context) error {
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
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	result, err := myDice.PackageManager.Enable(params.ID)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"data": result,
	})
}

// packageDisable 禁用扩展包
// POST /package/disable
// 参数: { id: string }
// 返回: { data: DisableResult, result: true }
// 注意: 禁用后需要重载才能从内存中移除资源
func packageDisable(c echo.Context) error {
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
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	result, err := myDice.PackageManager.Disable(params.ID)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"data": result,
	})
}

// packageReload 重载指定扩展包的资源
// POST /package/reload
// 参数: { id: string }
// 返回: { data: ReloadResult, result: true }
// ReloadResult 包含:
//   - success: bool - 是否成功
//   - message: string - 提示信息
//   - reloadedItems: map[string]string - 已重载的资源类型及说明
//   - needRestart: bool - 是否需要重启才能完全生效
//   - restartHints: []string - 需要重启的原因
func packageReload(c echo.Context) error {
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
	err := c.Bind(&params)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	result, err := myDice.PackageManager.Reload(params.ID)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"data": result,
	})
}

// packageReloadAll 重载所有已启用扩展包的资源
// POST /package/reload-all
// 返回: { data: ReloadResult, result: true }
// 该接口会重载所有已启用包的 JS 脚本和牌堆
func packageReloadAll(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}

	result, err := myDice.PackageManager.ReloadAll()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"data": result,
	})
}

// packageGetConfig 获取扩展包的用户配置
// GET /package/:id/config 或 GET /package/_/config?id=xxx
// 返回: { data: map[string]interface{}, result: true }
// 配置值由用户通过 UI 或 API 设置，JS 扩展可通过 ext.getPackageConfig() 读取
func packageGetConfig(c echo.Context) error {
	pkgID := getPackageIDFromRequest(c)

	config, err := myDice.PackageManager.GetConfig(pkgID)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"data": config,
	})
}

// packageSetConfig 设置扩展包的用户配置
// POST /package/:id/config 或 POST /package/_/config?id=xxx
// 参数: 配置对象 map[string]interface{}
// 返回: { message: string, result: true }
// 注意: 配置会立即生效，JS 扩展下次调用 ext.getPackageConfig() 时会获取新值
// 配置会根据 manifest.toml 中的 [config] 定义进行类型验证
func packageSetConfig(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return Success(&c, map[string]interface{}{
			"testMode": true,
		})
	}

	pkgID := getPackageIDFromRequest(c)

	var config map[string]interface{}
	err := c.Bind(&config)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	err = myDice.PackageManager.SetConfig(pkgID, config)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"message": "配置更新成功",
	})
}

// packageGetConfigSchema 获取扩展包的配置模式定义
// GET /package/:id/config-schema 或 GET /package/_/config-schema?id=xxx
// 返回: { data: map[string]ConfigItem, result: true }
// ConfigItem 包含 type, title, description, default 等字段
// UI 可根据此 schema 动态渲染配置表单
func packageGetConfigSchema(c echo.Context) error {
	pkgID := getPackageIDFromRequest(c)

	pkg, exists := myDice.PackageManager.Get(pkgID)
	if !exists {
		return Error(&c, "扩展包不存在", Response{})
	}

	if pkg.Manifest == nil {
		return Error(&c, "扩展包 manifest 不存在", Response{})
	}

	return Success(&c, Response{
		"data": pkg.Manifest.Config,
	})
}
