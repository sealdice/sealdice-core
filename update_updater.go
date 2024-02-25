package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"sealdice-core/dice"

	"go.uber.org/zap"
)

const updaterVersion = "0.1.1"

func checkURLOne(url string, wg *sync.WaitGroup, resultChan chan string) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		// URL 请求异常
		return
	}
	defer resp.Body.Close()

	<-ctx.Done()
	// URL 可用，但已经超过 7 秒，强制中断
	resultChan <- url
}

// 检查一组URL是否可用，返回可用的URL
func checkURLs(urls []string) []string {
	var wg sync.WaitGroup
	resultChan := make(chan string, len(urls))

	for _, url := range urls {
		wg.Add(1)
		go checkURLOne(url, &wg, resultChan)
	}

	wg.Wait()
	close(resultChan)

	var availableURLs []string
	for result := range resultChan {
		availableURLs = append(availableURLs, result)
	}

	return availableURLs
}

func getUpdaterFn() string {
	fn := "./seal-updater.exe"
	if runtime.GOOS != "windows" {
		fn = "./seal-updater"
	}
	return fn
}

func CheckUpdater(dm *dice.DiceManager) error {
	// 检查updater是否存在
	exists := false
	fn := getUpdaterFn()
	if _, err := os.Stat(fn); err == nil {
		logger.Info("检测到海豹更新程序")
		exists = true
	}

	// 获取updater版本
	isUpdaterOk := false
	if exists {
		err := os.Chmod(fn, 0o755)
		if err != nil {
			logger.Error("设置升级程序执行权限失败", err.Error())
		}
		cmd := exec.Command(fn, "--version")
		out, err := cmd.Output()
		if err != nil {
			logger.Error("获取升级程序版本失败")
		} else {
			ver := strings.TrimSpace(string(out))
			logger.Info("升级程序版本：", ver)
			if ver == "seal-updater "+updaterVersion {
				isUpdaterOk = true
			}
		}
	}

	// 如果升级程序不可用，那么下载一个
	if !isUpdaterOk {
		logger.Info("未检测到可用更新程序，开始下载")
		err := downloadUpdater(dm)
		if err != nil {
			logger.Error("下载更新程序失败")
			return errors.New("下载更新程序失败，无可用更新程序")
		} else {
			logger.Info("下载更新程序成功")
			err := os.Chmod(fn, 0o755)
			if err != nil {
				logger.Error("设置升级程序执行权限失败", err.Error())
			}
		}
	}

	return nil
}

func downloadUpdater(dm *dice.DiceManager) error {
	ver := dm.AppVersionOnline

	platform := runtime.GOOS
	arch := runtime.GOARCH

	prefix := "http://dice.weizaima.com/u/v" + updaterVersion
	if ver != nil {
		prefix = ver.UpdaterURLPrefix
	}
	link := prefix + "/" + "seal-updater-" + platform + "-" + arch

	// 如无法访问，尝试使用备用地址，但此地址不保证可用
	if len(checkURLs([]string{link})) == 0 {
		prefix := "https://d1.sealdice.com/u/v" + updaterVersion
		link = prefix + "/" + "seal-updater-" + platform + "-" + arch
	}
	fn := "./seal-updater"
	if platform == "windows" {
		fn += ".exe"
		link += ".exe"
	}
	err := DownloadFile(fn, link)
	if err != nil {
		return err
	}
	return nil
}

func UpdateByFile(dm *dice.DiceManager, log *zap.SugaredLogger, packName string, syncMode bool) bool {
	// 注意: 当执行完就立即退进程的情况下，需要使用 syncMode 为true
	if log == nil {
		log = logger
	}
	fn := getUpdaterFn()
	err := os.Chmod(fn, 0o755)
	if err != nil {
		log.Error("设置升级程序执行权限失败", err.Error())
	}

	log.Infof("升级程序: 预计使用 %s 进行升级", packName)

	// 创建一个具有5秒超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 确保在函数结束时取消上下文

	args := []string{"--upgrade", packName, "--pid", strconv.Itoa(os.Getpid())}
	cmd := exec.CommandContext(ctx, fn, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Info("升级程序: 验证成功，进入下一阶段，即将退出进程")

			updateFunc := func() {
				if runtime.GOOS == "windows" {
					log.Info("升级程序: 参数 ", args)
					cmd := executeWin(fn, args...)
					errStart := cmd.Start()
					if errStart != nil {
						log.Error("升级程序: 执行失败 ", errStart.Error())
						return
					}
				} else {
					args = append([]string{fn}, args...)
					log.Info("升级程序: 参数 ", args)
					errStart := syscall.Exec(fn, args, os.Environ())
					if errStart != nil {
						log.Error("升级程序: 执行失败 ", errStart.Error())
						return
					}
				}

				time.Sleep(5 * time.Second)
				cleanUpCreate(dm)()
				os.Exit(0)
			}
			if syncMode {
				updateFunc()
			} else {
				go updateFunc()
			}
			return true
		} else {
			log.Info("升级程序: 命令执行失败 ", err.Error())
			log.Info("升级程序: 详情 ", string(out))
		}
	}

	return false
}
