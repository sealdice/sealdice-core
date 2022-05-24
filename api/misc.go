package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func upgrade(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, "auth")
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	dm.UpdateCheckRequestChan <- 1
	time.Sleep(3 * time.Second) // 等待1s，应该能够取得新版本了。如果获取失败也不至于卡住

	if dm.AppVersionOnline != nil {
		if dm.AppVersionOnline.VersionLatestCode != dm.AppVersionCode {
			dm.UpdateRequestChan <- 1
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
