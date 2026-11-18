package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"

	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/utils"
)

var binPrefix = "https://sealdice.coding.net/p/sealdice/d/sealdice-binaries/git/raw/master"

func downloadUpdate(dm *dice.DiceManager, log *zap.SugaredLogger) (string, error) {
	var packFn string
	if dm.AppVersionOnline != nil {
		ver := dm.AppVersionOnline
		if ver.VersionLatestCode != dm.AppVersionCode {
			platform := runtime.GOOS
			arch := runtime.GOARCH
			version := ver.VersionLatest
			var ext string

			// 如果线上版本小于当前版本，那么拒绝更新
			if ver.VersionLatestCode < dm.AppVersionCode {
				return "", errors.New("获取到的线上版本旧于当前版本，停止更新")
			}

			switch platform {
			case "windows":
				ext = "zip"
			default:
				// 其他各种平台似乎都是 .tar.gz
				ext = "tar.gz"
			}

			if arch == "386" {
				arch = "i386"
			}

			fn := fmt.Sprintf("sealdice-core_%s_%s_%s.%s", version, platform, arch, ext)
			fileUrl := binPrefix + "/" + fn

			if ver.NewVersionURLPrefix != "" {
				fileUrl = ver.NewVersionURLPrefix + "/" + fn
			}

			log.Infof("准备下载更新: %s", fn)
			err := os.RemoveAll("./update")
			if err != nil {
				return "", errors.New("更新: 删除缓存目录(update)失败")
			}

			_ = os.MkdirAll("./update", 0o755)
			fn2 := "./update/update." + ext
			err = utils.DownloadFile(fn2, fileUrl)
			if err != nil {
				return "", errors.New("更新: 下载更新文件失败")
			}
			log.Infof("更新下载完成，保存于: %s", fn2)
			packFn = fn2
		}
	}
	return packFn, nil
}

func RebootRequestListen(dm *dice.DiceManager) {
	<-dm.RebootRequestChan
	doReboot(dm)
}

func UpdateCheckRequestListen(dm *dice.DiceManager) {
	for {
		<-dm.UpdateCheckRequestChan
		CheckVersion(dm)
	}
}

// UpdateRequestListen 处理升级请求。下载或升级程序失败时继续监听，允许再次请求；
// 更新包已交给升级程序后进程即将被替换，监听随之结束（旧 API 在成功路径上会多发一次请求，
// 继续监听会在升级程序校验期间删除并重新下载更新包）。
func UpdateRequestListen(dm *dice.DiceManager) {
	for {
		<-dm.UpdateRequestChan
		if handleUpdateRequest(dm) {
			return
		}
	}
}

// handleUpdateRequest 处理一次升级请求，并通过 UpdateDownloadedChan 回复下载结果。
// 返回 true 表示更新包已交给升级程序。
func handleUpdateRequest(dm *dice.DiceManager) bool {
	curDice := dm.GetDice()
	if curDice == nil {
		dm.UpdateDownloadedChan <- "Dice实例不存在"
		return false
	}
	log := curDice.Logger
	updatePackFn, err := downloadUpdate(dm, log)
	if err != nil {
		dm.UpdateDownloadedChan <- err.Error()
		return false
	}
	dm.UpdateDownloadedChan <- ""
	time.Sleep(2 * time.Second)
	log.Info("进行升级准备工作")

	// 旧版本行为: 将新升级包里的主程序复制到当前目录，命名为 auto_update.exe 或 auto_update
	// 然后重启主程序
	return dm.UpdateSealdiceByFile(updatePackFn)
}

func doReboot(dm *dice.DiceManager) {
	log := logger.M()
	executablePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return
	}

	binary, err := exec.LookPath(executablePath)
	if err != nil {
		log.Errorf("Restart Error: %s", err)
		return
	}
	platform := runtime.GOOS
	if platform == "windows" {
		cleanupCreate(dm)()

		// name, _ := filepath.Abs(binary)
		// err = exec.Command(`cmd`, `/C`, "start", name, "--delay=15").Start()
		cmd := executeWin(binary, "--delay=15")
		err := cmd.Start()
		if err != nil {
			log.Errorf("Restart error: %s %v", binary, err)
		}
	} else {
		// 手动cleanup
		cleanupCreate(dm)()
		// os.Args[1:]...
		execErr := syscall.Exec(binary, []string{os.Args[0], "--delay=25"}, os.Environ()) //nolint:gosec
		if execErr != nil {
			log.Errorf("Restart error: %s %v", binary, execErr)
		}
	}
	os.Exit(0)
}

func checkVersionBase(backendUrl string, dm *dice.DiceManager) *dice.VersionInfo {
	resp, err := http.Get(backendUrl + "/dice/api/version?versionCode=" + strconv.FormatInt(dm.AppVersionCode, 10) + "&v=" + strconv.FormatInt(rand.Int63(), 10))
	if err != nil {
		// logger.Errorf("获取新版本失败: %s", err.Error())
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	var ver dice.VersionInfo
	err = json.NewDecoder(resp.Body).Decode(&ver)
	if err != nil {
		return nil
	}

	dm.AppVersionOnline = &ver
	// downloadUpdate(dm)
	return &ver
}

func CheckVersion(dm *dice.DiceManager) *dice.VersionInfo {
	if runtime.GOOS == "android" {
		return nil
	}
	// 逐个尝试所有后端地址
	for _, i := range dice.BackendUrls {
		ret := checkVersionBase(i, dm)
		if ret != nil {
			return ret
		}
	}
	return nil
}
