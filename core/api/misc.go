package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func upgrade(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	if dm.AppVersionOnline != nil {
		if dm.AppVersionOnline.VersionLatestCode != dm.AppVersionCode {
			ret := <-dm.UpdateDownloadedChan
			if ret == "" {
				dm.UpdateRequestChan <- 1
				return c.JSON(200, map[string]interface{}{
					"text": "准备开始升级，服务即将离线",
				})
			} else {
				return c.JSON(200, map[string]interface{}{
					"text": "升级失败，原因: " + ret,
				})
			}
		}
	}

	return c.JSON(http.StatusForbidden, nil)
}
