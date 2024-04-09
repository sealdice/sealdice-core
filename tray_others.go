//go:build !windows && !darwin
// +build !windows,!darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func trayInit(dm *dice.DiceManager) {
	select {}
}

func hideWindow() {
}

func showWindow() {
}

func TestRunning() bool {
	return false
}

func tempDirWarn() {
	fmt.Println("当前工作路径为临时目录，因此拒绝继续执行。")
}

func showMsgBox(title string, message string) {
	fmt.Println(title, message)
}

func httpServe(e *echo.Echo, dm *dice.DiceManager, hideUI bool) {
	portStr := "3211"
	rePort := regexp.MustCompile(`:(\d+)$`)
	m := rePort.FindStringSubmatch(dm.ServeAddress)
	if len(m) > 0 {
		portStr = m[1]
	}

	for {
		ln, err := net.Listen("tcp", ":"+portStr)
		_ = ln.Close()
		if err != nil {
			logger.Errorf("端口已被占用: %s", dm.ServeAddress)
			goto _nextRound
		}

		fmt.Println("如果浏览器没有自动打开，请手动访问:")
		fmt.Printf(`http://localhost:%s`, portStr) // 默认:3211
		err = e.Start(dm.ServeAddress)
		if err != nil {
			logger.Errorf("端口已被占用: %s", dm.ServeAddress)
			goto _nextRound
		} else {
			break
		}

	_nextRound:
		time.Sleep(7 * time.Second)
		continue
	}
}

func executeWin(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    os.Getppid(),
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	return cmd
}
