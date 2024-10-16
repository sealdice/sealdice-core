//go:build !windows && !darwin
// +build !windows,!darwin

package main

import (
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"syscall"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	log "sealdice-core/utils/kratos"
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
	log.Warn("当前工作路径为临时目录，因此拒绝继续执行。")
}

func showMsgBox(title string, message string) {
	log.Info(title, message)
}

func httpServe(e *echo.Echo, dm *dice.DiceManager, hideUI bool) {
	portStr := "3211"
	rePort := regexp.MustCompile(`:(\d+)$`)
	m := rePort.FindStringSubmatch(dm.ServeAddress)
	if len(m) > 0 {
		portStr = m[1]
	}

	ln, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		log.Errorf("端口已被占用，即将自动退出: %s", dm.ServeAddress)
		runtime.Goexit()
	}
	_ = ln.Close()

	log.Infof("如果浏览器没有自动打开，请手动访问:\nhttp://localhost:%s", portStr)
	err = e.Start(dm.ServeAddress)
	if err != nil {
		log.Errorf("端口已被占用，即将自动退出: %s", dm.ServeAddress)
		return
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
