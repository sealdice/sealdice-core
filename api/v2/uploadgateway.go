package v2

import (
	"github.com/labstack/echo/v4"

	"sealdice-core/api/v2/middleware"
	"sealdice-core/api/v2/uploadgw"
	"sealdice-core/dice"
)

// MountUploadGateway 把基于 tus 协议的统一上传网关挂载到 Echo 上。
// 上传网关不属于 huma/OpenAPI 契约（二进制协议），因此单独挂载，
// 但复用与 huma 相同的 token 鉴权。
func MountUploadGateway(e *echo.Echo, dm *dice.DiceManager) error {
	gw, err := uploadgw.New(dm, "")
	if err != nil {
		return err
	}
	gw.Mount(e, middleware.EchoAuthMiddleware(dm.GetDice()))
	return nil
}
