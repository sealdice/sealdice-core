package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/acarl005/stripansi"
	"github.com/fy0/procs"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io/ioutil"
	"os"
	"os/signal"
	"sealdice-core/api"
	"sealdice-core/core"
	"sealdice-core/dice"
	"sealdice-core/model"
	"strconv"
	"strings"
	"syscall"
	"time"
)

/**
二进制目录结构:
data/configs
data/extensions
data/logs

extensions/
*/

func main() {
	core.LoggerInit()
	model.DBInit()
	cleanUp := func() {
		fmt.Println("程序即将退出，进行清理……")
		model.GetDB().Close()
	}
	defer cleanUp()

	// 初始化核心
	myDice := &dice.Dice{}
	myDice.Init()

	// 强制清理机制
	go (func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-interrupt:
			time.Sleep(time.Duration(5 * time.Second))
			fmt.Println("仍未关闭，强制退出")
			cleanUp()
			os.Exit(0)
		}
	})()

	if checkCqHttpExists() {
		myDice.InPackGoCqHttpExists = true
		go goCqHttpServe(myDice)
		// 等待登录成功
		for {
			if myDice.InPackGoCqHttpLoginSuccess {
				diceIMServe(myDice)
				break
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		myDice.InPackGoCqHttpExists = false
		// 假设已经有一个onebot服务存在
		diceIMServe(myDice)
	}

	//uiServe(myDice)
}

func diceIMServe(myDice *dice.Dice) {
	for {
		// 骰子开始连接
		fmt.Println("开始连接 onebot 服务")
		ret := myDice.ImSession.Serve()

		if ret == 0 {
			break
		}

		fmt.Println("onebot 连接中断，将在15秒后重新连接")
		time.Sleep(time.Duration(15 * time.Second))
	}
}

func uiServe(myDice *dice.Dice) {
	fmt.Println("即将启动webui")
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	api.Bind(e)
	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

func checkCqHttpExists() bool {
	if _, err := os.Stat("./go-cqhttp"); err == nil {
		return true
	}
	return false
}

func goCqHttpServe(myDice *dice.Dice) {
	// 注：给他另一份log
	qrcodeFile := "./go-cqhttp/qrcode.png"
	if _, err := os.Stat(qrcodeFile); err == nil {
		// 如果已经存在二维码文件，将其删除
		os.Remove(qrcodeFile)
		fmt.Println("删除已存在的二维码文件")
	}

	// 创建设备配置文件
	if _, err := os.Stat("./go-cqhttp/device.json"); errors.Is(err, os.ErrNotExist) {
		deviceInfo, err := dice.GenerateDeviceJson()
		if err == nil {
			ioutil.WriteFile("./go-cqhttp/device.json", deviceInfo, 0644)
		}
	} else {
		fmt.Println("设备文件已存在，跳过")
	}

	// 创建配置文件
	if _, err := os.Stat("./go-cqhttp/config.yml"); errors.Is(err, os.ErrNotExist) {
		// 如果不存在 config.yml 那么启动一次，让它自动生成
		// 改为：如果不存在，帮他创建
		input := bufio.NewScanner(os.Stdin)
		fmt.Println("请输入你的QQ号:")
		input.Scan()
		qq, err := strconv.ParseInt(input.Text(), 10, 64)
		if err != nil {
			panic(err)
		}
		fmt.Println("请输入密码(可以不填，直接扫二维码登录):")
		input.Scan()
		pw := input.Text()
		if err != nil {
			panic(err)
		}

		c := dice.GenerateConfig(qq, pw, 6700)
		ioutil.WriteFile("./go-cqhttp/config.yml", []byte(c), 0644)
	}

	// 启动客户端
	p := procs.NewProcess("./go-cqhttp faststart")
	p.Dir = "./go-cqhttp"

	chQrCode := make(chan int, 1)
	p.OutputHandler = func(line string) string {
		// 请使用手机QQ扫描二维码 (qrcode.png) :
		if strings.Contains(line, "qrcode.png") {
			chQrCode <- 1
		}
		if strings.Contains(line, "CQ WebSocket 服务器已启动") {
			// CQ WebSocket 服务器已启动
			// 登录成功 欢迎使用
			myDice.InPackGoCqHttpLoginSuccess = true
		}

		if myDice.InPackGoCqHttpLoginSuccess == false || strings.Contains(line, "风控") || strings.Contains(line, "WARNING") || strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") {
			fmt.Printf("onebot | %s\n", stripansi.Strip(line))
		}
		return line
	}

	go func() {
		<-chQrCode
		if _, err := os.Stat(qrcodeFile); err == nil {
			fmt.Println("二维码已经就绪")
			//qrdata, err := ioutil.ReadFile(qrcodeFile)
		}
	}()

	myDice.InPackGoCqHttpRunning = true
	err := p.Run()
	defer func() {
		p.Stop()
	}()
	myDice.InPackGoCqHttpRunning = false
	if err != nil {
		fmt.Println("go-cqhttp 进程退出: ", err)
	} else {
		fmt.Println("go-cqhttp 进程退出")
	}
}
