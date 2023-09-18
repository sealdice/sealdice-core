package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"sealdice-core/dice"
	"strconv"
	"strings"
	"syscall"
	"time"
)

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
		_ = os.Chmod(fn, 0755)
		cmd := exec.Command(fn, "--version")
		out, err := cmd.Output()
		if err != nil {
			logger.Error("获取升级程序版本失败")
		} else {
			ver := strings.TrimSpace(string(out))
			logger.Info("升级程序版本：", ver)
			if ver == "seal-updater 0.1.0" {
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
		}
	}

	return nil
}

func downloadUpdater(dm *dice.DiceManager) error {
	ver := dm.AppVersionOnline

	platform := runtime.GOOS
	arch := runtime.GOARCH

	prefix := "http://dice.weizaima.com/u/v0.1.0"
	if ver != nil {
		prefix = ver.UpdaterUrlPrefix
	}
	link := prefix + "/" + "seal-updater-" + platform + "-" + arch
	fn := "./seal-updater"
	if platform == "windows" {
		fn += ".exe"
	}
	err := DownloadFile(fn, link)
	if err != nil {
		return err
	}
	return nil
}

func UpdateByFile(dm *dice.DiceManager, packName string) bool {
	fn := getUpdaterFn()
	logger.Infof("升级程序: 预计使用 %s 进行升级", packName)

	// 创建一个具有5秒超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // 确保在函数结束时取消上下文

	args := []string{"--upgrade", packName, "--pid", strconv.Itoa(os.Getpid())}
	cmd := exec.CommandContext(ctx, fn, args...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Info("升级程序: 验证成功，进入下一阶段，即将退出进程")

			if runtime.GOOS == "windows" {
				logger.Info("升级程序: 参数 ", args)
				cmd := executeWin(fn, args...)
				err := cmd.Start()
				if err != nil {
					logger.Error("升级程序: 执行失败 ", err.Error())
					return false
				}
			} else {
				err := syscall.Exec(fn, args, os.Environ())
				if err != nil {
					logger.Error("升级程序: 执行失败 ", err.Error())
					return false
				}
			}

			time.Sleep(5 * time.Second)
			cleanUpCreate(dm)()
			os.Exit(0)
			return true
		} else {
			logger.Info("升级程序: 命令执行失败 ", err.Error())
			logger.Info("升级程序: 详情 ", string(out))
		}
	}

	return false
}
