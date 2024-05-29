package api

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"sealdice-core/dice"

	"github.com/labstack/echo/v4"
	cp "github.com/otiai10/copy"
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

	if dm.ContainerMode {
		return c.JSON(200, map[string]interface{}{
			"text": "容器模式下禁止更新，请手动拉取最新镜像",
		})
	}

	dm.UpdateCheckRequestChan <- 1
	time.Sleep(3 * time.Second) // 等待1s，应该能够取得新版本了。如果获取失败也不至于卡住

	if dm.AppVersionOnline != nil {
		if dm.AppVersionOnline.VersionLatestCode != dm.AppVersionCode {
			dm.UpdateRequestChan <- myDice
			ret := <-dm.UpdateDownloadedChan
			if ret == "" {
				myDice.Save(true)
				bakFn, _ := myDice.Parent.Backup(dice.BackupSelectionAll, false)
				tmpParent := os.TempDir()
				tmpPath := path.Join(tmpParent, bakFn)
				_ = os.MkdirAll(filepath.Join(tmpParent, "backups"), 0644)
				myDice.Logger.Infof("将备份文件复制到此路径: %s", tmpPath)
				_ = cp.Copy(bakFn, tmpPath)

				dm.UpdateRequestChan <- myDice
				return c.JSON(200, map[string]interface{}{
					"text": "准备开始升级，服务即将离线",
				})
			}
			return c.JSON(200, map[string]interface{}{
				"text": "升级失败，原因: " + ret,
			})
		}
	}

	return c.JSON(http.StatusForbidden, nil)
}
