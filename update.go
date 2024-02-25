package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"sealdice-core/dice"

	"go.uber.org/zap"
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
			err = DownloadFile(fn2, fileUrl)
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

		dm.UpdateSealdiceByFile(updatePackFn, log)
		// 旧版本行为: 将新升级包里的主程序复制到当前目录，命名为 auto_update.exe 或 auto_update
		// 然后重启主程序
	} else {
		dm.UpdateDownloadedChan <- err.Error()
	}
}

func doReboot(dm *dice.DiceManager) {
	executablePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return
	}

	binary, err := exec.LookPath(executablePath)
	if err != nil {
		logger.Errorf("Restart Error: %s", err)
		return
	}
	platform := runtime.GOOS
	if platform == "windows" {
		cleanUpCreate(dm)()

		// name, _ := filepath.Abs(binary)
		// err = exec.Command(`cmd`, `/C`, "start", name, "--delay=15").Start()
		cmd := executeWin(binary, "--delay=15")
		err := cmd.Start()
		if err != nil {
			logger.Errorf("Restart error: %s %v", binary, err)
		}
	} else {
		// 手动cleanup
		cleanUpCreate(dm)()
		// os.Args[1:]...
		execErr := syscall.Exec(binary, []string{os.Args[0], "--delay=25"}, os.Environ())
		if execErr != nil {
			logger.Errorf("Restart error: %s %v", binary, execErr)
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

func DownloadFile(filepath string, url string) error {
	// Get the data
	// resp, err := http.Get(url)
	client := new(http.Client)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	request.Header.Add("Accept-Encoding", "gzip")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	if resp.StatusCode == http.StatusOK {
		// Write the body to file
		if resp.Header.Get("Content-Encoding") == "gzip" {
			// 如果响应使用了GZIP压缩，需要解压缩
			var reader io.ReadCloser
			reader, err = gzip.NewReader(resp.Body)
			if err != nil {
				fmt.Println("GZIP解压出错:", err)
				return err
			}
			defer reader.Close()
			_, err = io.Copy(out, reader)
		} else {
			_, err = io.Copy(out, resp.Body)
		}

		return err
	}

	return errors.New("http status:" + resp.Status)
}

func sha256Checksum(fn string) string {
	f, err := os.Open(fn)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	// bytes 比较需要使用 bytes.Equal 这里直接转文本了
	return fmt.Sprintf("%x", h.Sum(nil))
}
