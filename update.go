package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

func UpdateRequestListen(dm *dice.DiceManager) {
	curDice := <-dm.UpdateRequestChan
	log := curDice.Logger
	updatePackFn, err := downloadUpdate(dm, log)
	if err == nil {
		dm.UpdateDownloadedChan <- ""
		time.Sleep(2 * time.Second)
		log.Info("进行升级准备工作")

		dm.UpdateSealdiceByFile(updatePackFn)
		// 旧版本行为: 将新升级包里的主程序复制到当前目录，命名为 auto_update.exe 或 auto_update
		// 然后重启主程序
	} else {
		dm.UpdateDownloadedChan <- err.Error()
	}
}

func doReboot(dm *dice.DiceManager) {
	log := logger.M()
	executablePath, err := os.Executable()
	if err != nil {
		log.Errorf("Restart Error: %s", err)
		return
	}
	binary, err := filepath.Abs(executablePath)
	if err != nil {
		log.Errorf("Restart Error: %s", err)
		return
	}
	probe, err := os.Open(binary)
	if err != nil {
		log.Errorf("Restart Error: %s", err)
		return
	}
	_ = probe.Close()

	restartArgs := buildRestartArgs(os.Args[1:], "1")
	platform := runtime.GOOS
	if platform == "windows" {
		restartArgs = buildRestartArgs(os.Args[1:], "3")
	}
	if platform == "windows" {
		cleanupCreate(dm)()
		cmd := executeWin(binary, restartArgs...)
		err := cmd.Start()
		if err != nil {
			log.Errorf("Restart error: %s %v", binary, err)
		}
	} else {
		cleanupCreate(dm)()
		execArgs := append([]string{os.Args[0]}, restartArgs...)
		execErr := syscall.Exec(binary, execArgs, os.Environ()) //nolint:gosec
		if execErr != nil {
			log.Errorf("Restart error: %s %v", binary, execErr)
		}
	}
	os.Exit(0)
}

func buildRestartArgs(args []string, delay string) []string {
	restartArgs := make([]string, 0, len(args)+1)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--delay" {
			i++
			continue
		}
		if strings.HasPrefix(arg, "--delay=") {
			continue
		}
		restartArgs = append(restartArgs, arg)
	}
	return append(restartArgs, "--delay="+delay)
}

func checkVersionBase(ctx context.Context, backendUrl string, dm *dice.DiceManager) *dice.VersionInfo {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, backendUrl+"/dice/api/version?versionCode="+strconv.FormatInt(dm.AppVersionCode, 10)+"&v="+strconv.FormatInt(rand.Int63(), 10), nil)
	if err != nil {
		return nil
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(request)
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
	return CheckVersionContext(context.Background(), dm)
}

func CheckVersionContext(ctx context.Context, dm *dice.DiceManager) *dice.VersionInfo {
	if runtime.GOOS == "android" {
		return nil
	}
	// 逐个尝试所有后端地址
	for _, i := range dice.BackendUrls {
		ret := checkVersionBase(ctx, i, dm)
		if ret != nil {
			return ret
		}
	}
	return nil
}
