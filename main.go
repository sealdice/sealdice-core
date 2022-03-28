package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sealdice-core/api"
	"sealdice-core/dice"
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
	var opts struct {
		GoCqhttpClear bool `long:"gclr" description:"清除go-cqhttp登录信息"`
		Install       bool `short:"i" long:"install" description:"安装为系统服务"`
		Uninstall     bool `long:"uninstall" description:"删除系统服务"`
		ShowConsole   bool `long:"show-console" description:"Windows上显示控制台界面"`
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

	if !opts.ShowConsole {
		hideWindow()
	}

	if TestRunning() {
		return
	}

	aaa()

	cwd, _ := os.Getwd()
	fmt.Printf("%s %s\n", dice.APPNAME, dice.VERSION)
	fmt.Println("工作路径: ", cwd)

	diceManager := &dice.DiceManager{}

	cleanUp := func() {
		fmt.Println("程序即将退出，进行清理……")
		for _, i := range diceManager.Dice {
			i.DB.Close()
		}
		diceManager.Save()
	}
	defer cleanUp()

	// 初始化核心
	//myDice := &dice.Dice{}
	//myDice.Init()
	diceManager.LoadDice()
	diceManager.TryCreateDefault()
	diceManager.InitDice()

	//a, d, err := myDice.ExprEval("7d12k4", nil)
	//if err == nil {
	//	fmt.Println(a.Parser.GetAsmText())
	//	fmt.Println(d)
	//	fmt.Println("DDD"+"#{a}", a.TypeId, a.Value, d, err)
	//} else {
	//	fmt.Println("DDD2", err)
	//}

	//runtime := quickjs.NewRuntime()
	//defer runtime.Free()
	//
	//context := runtime.NewContext()
	//defer context.Free()

	//globals := context.Globals()

	// Test evaluating template strings.

	//result, err := context.Eval("`Hello world! 2 ** 8 = ${2 ** 8}.`")
	//fmt.Println("XXXXXXX", result, err)

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

	//for _, v := range myDice.ImSession.Conns {
	//	if v.UseInPackGoCqhttp {
	//		go dice.goCqHttpServe(myDice)
	//	}
	//}

	for _, d := range diceManager.Dice {
		go diceServe(d)
	}

	uiServe(diceManager)

	//if checkCqHttpExists() {
	//	myDice.InPackGoCqHttpExists = true
	//	go goCqHttpServe(myDice)
	//	// 等待登录成功
	//	for {
	//		if myDice.InPackGoCqHttpLoginSuccess {
	//			diceIMServe(myDice)
	//			break
	//		}
	//		time.Sleep(1 * time.Second)
	//	}
	//} else {
	//	myDice.InPackGoCqHttpExists = false
	//	// 假设已经有一个onebot服务存在
	//	diceIMServe(myDice)
	//}
}

func diceServe(d *dice.Dice) {
	if len(d.ImSession.Conns) == 0 {
		d.Logger.Infof("未检测到任何帐号，请先到“帐号设置”进行添加")
	}

	for _, conn := range d.ImSession.Conns {
		if conn.Enable {
			if conn.UseInPackGoCqhttp {
				dice.GoCqHttpServe(d, conn, "", 1, true)
				time.Sleep(10 * time.Second) // 稍作等待再连接
			}

			go dice.DiceServe(d, conn)
			//for {
			//	conn.DiceServing = true
			//	// 骰子开始连接
			//	d.Logger.Infof("开始连接 onebot 服务，帐号 <%s>(%d)", conn.Nickname, conn.UserId)
			//	ret := d.ImSession.Serve(index)
			//
			//	if ret == 0 {
			//		break
			//	}
			//
			//	d.Logger.Infof("onebot 连接中断，将在15秒后重新连接，帐号 <%s>(%d)", conn.Nickname, conn.UserId)
			//	time.Sleep(time.Duration(15 * time.Second))
			//}
		}
	}
}

func diceIMServe(myDice *dice.Dice) {
	if len(myDice.ImSession.Conns) == 0 {
		myDice.ImSession.Conns = append(myDice.ImSession.Conns, &dice.ConnectInfoItem{
			ConnectUrl:        "ws://127.0.0.1:6700",
			Platform:          "qq",
			Type:              "onebot",
			UseInPackGoCqhttp: myDice.InPackGoCqHttpExists,
			Enable:            true,
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

func uiServe(myDice *dice.DiceManager) {
	fmt.Println("即将启动webui")
	// Echo instance
	e := echo.New()

	// Middleware
	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	//e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
	//	XSSProtection:      "1; mode=block",
	//	ContentTypeNosniff: "nosniff",
	//	XFrameOptions:      "SAMEORIGIN",
	//	HSTSMaxAge:         3600,
	//	//ContentSecurityPolicy: "default-src 'self'",
	//}))
	// X-Content-Type-Options: nosniff
	e.Static("/", "./frontend")

	api.Bind(e, myDice)
	e.HideBanner = true // 关闭banner，原因是banner图案会改变终端光标位置

	exec.Command(`cmd`, `/c`, `start`, `http://localhost:3211`).Start()
	fmt.Println("如果浏览器没有自动打开，请手动访问:")
	fmt.Println("http://localhost:3211")
	e.Start(myDice.ServeAddress) // 默认:3211

	//interrupt := make(chan os.Signal, 1)
	//signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	//
	//for {
	//	select {
	//	case <-interrupt:
	//		fmt.Println("主动关闭")
	//		return
	//	}
	//}
}

func checkCqHttpExists() bool {
	if _, err := os.Stat("./go-cqhttp"); err == nil {
		return true
	}
	return false
}
