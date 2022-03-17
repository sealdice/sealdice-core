//go:build windows
// +build windows

package main

import (
	"github.com/fy0/systray"
	"github.com/gen2brain/beeep"
	"github.com/lxn/win"
	"os"
	"os/exec"
	"sealdice-core/icon"
	"syscall"
	"time"
	"unsafe"
)

func hideWindow() {
	win.ShowWindow(win.GetConsoleWindow(), win.SW_HIDE)
}

func aaa() {
	go systray.Run(onReady, onExit)
}

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

func CreateMutex(name string) (uintptr, error) {
	ret, _, err := procCreateMutex.Call(
		0,
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
	)
	switch int(err.(syscall.Errno)) {
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
	s2, _ := syscall.UTF16PtrFromString("你看到这个是因为SealDice应该已经在运行了，如果你是想打开UI界面，请在任务栏右下角的系统托盘区域找到SealDice图标并右键。")
	win.MessageBox(0, s2, s1, win.MB_OK)
	return false
}

func onReady() {
	systray.SetIcon(icon.Data)

	mOpen := systray.AddMenuItem("打开界面", "开启WebUI")
	mShowHide := systray.AddMenuItemCheckbox("显示终端窗口", "显示终端窗口", false)
	mQuit := systray.AddMenuItem("退出", "退出程序")
	mOpen.SetIcon(icon.Data)

	go beeep.Notify("SealDice", "我藏在托盘区域了，点我的小图标可以快速打开UI", "assets/information.png")

	for {
		select {
		case <-mOpen.ClickedCh:
			exec.Command(`cmd`, `/c`, `start`, `http://localhost:3211`).Start()
		case <-mQuit.ClickedCh:
			systray.Quit()
			time.Sleep(1 * time.Second)
			os.Exit(0)
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
