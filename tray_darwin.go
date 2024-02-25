//go:build darwin
// +build darwin

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"syscall"
	"time"

	"sealdice-core/dice"
	"sealdice-core/icon"

	"github.com/fy0/systray"
	"github.com/gen2brain/beeep"
	"github.com/labstack/echo/v4"
)

var theDm *dice.DiceManager

func trayInit(dm *dice.DiceManager) {
	theDm = dm
	runtime.LockOSThread()
	systray.Run(onReady, onExit)
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

var _trayPortStr = "3211"

var systrayQuited bool = false

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("海豹核心")
	systray.SetTooltip("海豹TRPG骰点核心")

	mOpen := systray.AddMenuItem("打开界面", "开启WebUI")
	mOpen.SetIcon(icon.Data)
	mOpenExeDir := systray.AddMenuItem("打开海豹目录", "访达访问程序所在目录")
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		_ = beeep.Notify("SealDice", "我藏在托盘区域了，点我的小图标可以快速打开UI", "icon/icon.ico")
	}()

	for {
		select {
		case <-mOpen.ClickedCh:
			_ = exec.Command(`open`, `http://localhost:`+_trayPortStr).Start()
		case <-mOpenExeDir.ClickedCh:
			_ = exec.Command(`open`, filepath.Dir(os.Args[0])).Start()
		case <-mQuit.ClickedCh:
			systrayQuited = true
			cleanUpCreate(theDm)()
			systray.Quit()
			time.Sleep(3 * time.Second)
			os.Exit(0)
		}
	}
}

func onExit() {
	// clean up hear
}

func httpServe(e *echo.Echo, dm *dice.DiceManager, hideUI bool) {
	portStr := "3211"

	go func() {
		for {
			time.Sleep(5 * time.Second)
			if systrayQuited {
				break
			}
			runtime.LockOSThread()
			systray.SetTooltip("海豹TRPG骰点核心 #" + portStr)
			runtime.UnlockOSThread()
		}
	}()

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

	// exec.Command(`cmd`, `/c`, `start`, fmt.Sprintf(`http://localhost:%s`, portStr)).Start()
	fmt.Println("如果浏览器没有自动打开，请手动访问:")
	fmt.Printf(`http://localhost:%s`, portStr) // 默认:3211
	err = e.Start(dm.ServeAddress)
	if err != nil {
		logger.Errorf("端口已被占用，即将自动退出: %s", dm.ServeAddress)
		return
	}
}
