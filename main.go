package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/fy0/procs"
	"github.com/jessevdk/go-flags"
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
	var opts struct {
		GoCqhttpClear bool `long:"gclr" description:"清除go-cqhttp登录信息"`
		Install       bool `short:"i" long:"install" description:"安装为系统服务"`
		Uninstall     bool `long:"uninstall" description:"删除系统服务"`
	}

	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}

	if opts.GoCqhttpClear {
		fmt.Println("清除go-cqhttp登录信息……")
		os.Remove("./go-cqhttp/config.yml")
		os.Remove("./go-cqhttp/device.json")
		os.Remove("./go-cqhttp/session.token")
		return
	}

	if opts.Install {
		serviceInstall(true)
		return
	}

	if opts.Uninstall {
		serviceInstall(false)
		return
	}

	model.DBInit()
	cleanUp := func() {
		fmt.Println("程序即将退出，进行清理……")
		model.GetDB().Close()
	}
	defer cleanUp()

	// 初始化核心
	myDice := &dice.Dice{}
	myDice.Init()

	cwd, _ := os.Getwd()
	fmt.Println("工作路径: ", cwd)

	a, d, err := myDice.ExprEval("7d12k4", nil)
	if err == nil {
		fmt.Println(a.Parser.GetAsmText())
		fmt.Println(d)
		fmt.Println("DDD"+"#{a}", a.TypeId, a.Value, d, err)
	} else {
		fmt.Println("DDD2", err)
	}

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
	if len(myDice.ImSession.Conns) == 0 {
		myDice.ImSession.Conns = append(myDice.ImSession.Conns, &dice.ConnectInfoItem{
			ConnectUrl:        "ws://127.0.0.1:6700",
			Platform:          "qq",
			UseInPackGoCqhttp: myDice.InPackGoCqHttpExists,
		})
	}

	for {
		// 骰子开始连接
		fmt.Println("开始连接 onebot 服务")
		ret := myDice.ImSession.Serve(0)

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
	api.Bind(e, myDice)
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

	if _, err := os.Stat("./go-cqhttp/session.token"); errors.Is(err, os.ErrNotExist) {
		// 并未登录成功，删除记录文件
		os.Remove("./go-cqhttp/config.yml")
		os.Remove("./go-cqhttp/device.json")
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
			fmt.Printf("onebot | %s\n", line)
		}
		return line
	}

	go func() {
		<-chQrCode
		if _, err := os.Stat(qrcodeFile); err == nil {
			fmt.Println("二维码已经就绪")
			fmt.Println("如控制台二维码不好扫描，可以手动打开go-cqhttp目录下qrcode.png")
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
