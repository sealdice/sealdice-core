//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"net"
	"regexp"
	"runtime"
	"sealdice-core/dice"

	"github.com/labstack/echo/v4"
)

func trayInit(dm *dice.DiceManager) {

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

func httpServe(e *echo.Echo, dm *dice.DiceManager, hideUI bool) {
	portStr := "3211"
	rePort := regexp.MustCompile(`:(\d+)$`)
	m := rePort.FindStringSubmatch(dm.ServeAddress)
	if len(m) > 0 {
		portStr = m[1]
	}

	ln, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		logger.Errorf("端口已被占用，即将自动退出: %s", dm.ServeAddress)
		runtime.Goexit()
	}
	_ = ln.Close()

	//exec.Command(`cmd`, `/c`, `start`, fmt.Sprintf(`http://localhost:%s`, portStr)).Start()
	fmt.Println("如果浏览器没有自动打开，请手动访问:")
	fmt.Printf(`http://localhost:%s`, portStr) // 默认:3211
	err = e.Start(dm.ServeAddress)
	if err != nil {
		logger.Errorf("端口已被占用，即将自动退出: %s", dm.ServeAddress)
		return
	}
}

type Cmd struct {
}

func (cmd *Cmd) Start() error {
	return nil
}

func executeWin(name string, arg ...string) *Cmd {
	return nil
}
