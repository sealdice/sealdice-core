//go:build windows
// +build windows

package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/fy0/go-autostart"
	"github.com/fy0/systray"
	"github.com/gen2brain/beeep"
	"github.com/labstack/echo/v4"
	win "github.com/lxn/win"
	"github.com/monaco-io/request"
	"golang.org/x/sys/windows"

	"sealdice-core/dice"
	"sealdice-core/icon"
)

func hideWindow() {
	win.ShowWindow(win.GetConsoleWindow(), win.SW_HIDE)
}

func showWindow() {
	win.ShowWindow(win.GetConsoleWindow(), win.SW_SHOW)
}

var theDM *dice.DiceManager

func trayInit(dm *dice.DiceManager) {
	theDM = dm
	// 确保能收到系统消息，从而避免不能弹出菜单
	runtime.LockOSThread()
	systray.Run(onReady, onExit)
}

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

func CreateMutex(name string) (uintptr, error) {
	s, _ := syscall.UTF16PtrFromString(name)
	ret, _, err := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(s)),
	)
	switch int(err.(syscall.Errno)) { //nolint:errorlint
	case 0:
		return ret, nil
	default:
		return ret, err
	}
}

func TestRunning() bool {
	_, err := CreateMutex("SealDice")
	if err == nil {
		return false
	}

	s1, _ := syscall.UTF16PtrFromString("SealDice 海豹已经在运作")
	s2, _ := syscall.UTF16PtrFromString("如果你想在Windows上打开多个海豹，请点“确定”，或加参数-m启动。\n如果只是打开UI界面，请在任务栏右下角的系统托盘区域找到海豹图标并右键，点“取消")
	ret := win.MessageBox(0, s2, s1, win.MB_YESNO|win.MB_ICONWARNING|win.MB_DEFBUTTON2)
	return ret != win.IDYES
}

func PortExistsWarn() {
	s1, _ := syscall.UTF16PtrFromString("SealDice 启动失败")
	s2, _ := syscall.UTF16PtrFromString("端口已被占用，建议换用其他端口")
	win.MessageBox(0, s2, s1, win.MB_OK)
}

func getAutoStart() *autostart.App {
	exePath, err := filepath.Abs(os.Args[0])
	if err == nil {
		pathName := filepath.Dir(exePath)
		pathName = filepath.Base(pathName)
		autostartName := fmt.Sprintf("SealDice_%s", pathName)

		appStart := &autostart.App{
			Name:        autostartName,
			DisplayName: "海豹骰点核心 - 目录: " + pathName,
			Exec:        []string{exePath, "-m --hide-ui"}, // 分开写会有问题
		}
		return appStart
	}
	return nil
}

var systrayQuited bool = false

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("海豹TRPG骰点核心")
	systray.SetTooltip("海豹TRPG骰点核心")

	mOpen := systray.AddMenuItem("打开界面", "开启WebUI")
	mOpenExeDir := systray.AddMenuItem("打开海豹目录", "资源管理器访问程序所在目录")
	mShowHide := systray.AddMenuItemCheckbox("显示终端窗口", "显示终端窗口", false)
	mAutoBoot := systray.AddMenuItemCheckbox("开机自启动", "开机自启动", false)
	mQuit := systray.AddMenuItem("退出", "退出程序")
	mOpen.SetIcon(icon.Data)

	go func() {
		_ = beeep.Notify("SealDice", "我藏在托盘区域了，点我的小图标可以快速打开UI", "assets/information.png")
	}()

	// 自启动检查
	go func() {
		runtime.LockOSThread()
		for {
			time.Sleep(10 * time.Second)
			if getAutoStart().IsEnabled() {
				mAutoBoot.Check()
			} else {
				mAutoBoot.Uncheck()
			}
		}
	}()

	if getAutoStart().IsEnabled() {
		mAutoBoot.Check()
	}

	for {
		select {
		case <-mOpen.ClickedCh:
			_ = exec.Command(`cmd`, `/c`, `start`, `http://localhost:`+_trayPortStr).Start()
		case <-mOpenExeDir.ClickedCh:
			_ = exec.Command(`cmd`, `/c`, `explorer`, filepath.Dir(os.Args[0])).Start()
		case <-mQuit.ClickedCh:
			systray.Quit()
			systrayQuited = true
			cleanUpCreate(theDM)()
			time.Sleep(3 * time.Second)
			os.Exit(0)
		case <-mAutoBoot.ClickedCh:
			if mAutoBoot.Checked() {
				err := getAutoStart().Disable()
				if err != nil {
					s1, _ := syscall.UTF16PtrFromString("SealDice 临时目录错误")
					s2, _ := syscall.UTF16PtrFromString("自启动失败设置失败，原因: " + err.Error())
					win.MessageBox(0, s2, s1, win.MB_OK|win.MB_ICONERROR)
					fmt.Println("自启动设置失败: ", err.Error())
				}
				mAutoBoot.Uncheck()
			} else {
				err := getAutoStart().Enable()
				if err != nil {
					s1, _ := syscall.UTF16PtrFromString("SealDice 临时目录错误")
					s2, _ := syscall.UTF16PtrFromString("自启动失败设置失败，原因: " + err.Error())
					win.MessageBox(0, s2, s1, win.MB_OK|win.MB_ICONERROR)
					fmt.Println("自启动设置失败: ", err.Error())
				}
				mAutoBoot.Check()
			}
		case <-mShowHide.ClickedCh:
			if mShowHide.Checked() {
				win.ShowWindow(win.GetConsoleWindow(), win.SW_HIDE)
				mShowHide.Uncheck()
			} else {
				win.ShowWindow(win.GetConsoleWindow(), win.SW_SHOW)
				mShowHide.Check()
			}
		}
	}
}

func onExit() {
	// clean up here
}

var _trayPortStr = "3211"

func httpServe(e *echo.Echo, dm *dice.DiceManager, hideUI bool) {
	portStr := "3211"
	// runtime.LockOSThread()

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

	showUI := func() {
		if !hideUI {
			time.Sleep(2 * time.Second)
			url := fmt.Sprintf(`http://localhost:%s`, portStr)
			url2 := fmt.Sprintf(`http://127.0.0.1:%s`, portStr) // 因为dns被换了，localhost不能解析
			c := request.Client{
				URL:     url2,
				Method:  "GET",
				Timeout: 1,
			}
			resp := c.Send()
			if resp.OK() {
				time.Sleep(1 * time.Second)
				_ = exec.Command(`cmd`, `/c`, `start`, url).Start()
			}
		}
	}

	for {
		rePort := regexp.MustCompile(`:(\d+)$`)
		m := rePort.FindStringSubmatch(dm.ServeAddress)
		if len(m) > 0 {
			portStr = m[1]
			_trayPortStr = portStr
		}

		err := e.Start(dm.ServeAddress)

		if err != nil {
			s1, _ := syscall.UTF16PtrFromString("海豹TRPG骰点核心")
			s2, _ := syscall.UTF16PtrFromString(fmt.Sprintf("端口 %s 已被占用，点“是”随机换一个端口，点“否”退出\n注意，此端口将被自动写入配置，后续可用启动参数改回", portStr))
			ret := win.MessageBox(0, s2, s1, win.MB_YESNO|win.MB_ICONWARNING|win.MB_DEFBUTTON2)
			if ret == win.IDYES {
				newPort := 3000 + rand.Int()%4000
				dm.ServeAddress = fmt.Sprintf("0.0.0.0:%d", newPort)
				continue
			} else {
				logger.Errorf("端口已被占用，即将自动退出: %s", dm.ServeAddress)
				os.Exit(0)
			}
		} else {
			fmt.Println("如果浏览器没有自动打开，请手动访问:")
			fmt.Printf("http://localhost:%s\n", portStr) // 默认:3211
			go showUI()
			break
		}
	}
}

func tempDirWarn() {
	s1, _ := syscall.UTF16PtrFromString("SealDice 临时目录错误")
	s2, _ := syscall.UTF16PtrFromString("你正在临时文件目录运行海豹，最可能的情况是没有解压而是直接双击运行！\n请先完整解压后再进行运行操作！\n按确定后将自动退出")
	win.MessageBox(0, s2, s1, win.MB_OK|win.MB_ICONERROR)
	fmt.Println("当前工作路径为临时目录，因此拒绝继续执行。")
}

func showMsgBox(title string, message string) {
	s1, _ := syscall.UTF16PtrFromString(title)
	s2, _ := syscall.UTF16PtrFromString(message)
	win.MessageBox(0, s2, s1, win.MB_OK|win.MB_ICONERROR)
}

func executeWin(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
		CreationFlags:    windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NEW_CONSOLE,
		NoInheritHandles: true,
	}

	// cmd.Dir, _ = os.Getwd()
	// path, err := os.Executable()
	// if err != nil {
	//	 cmd.Dir, _ = filepath.Abs(filepath.Dir(path))
	// }
	return cmd
}
