package main

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
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
		Install                bool   `short:"i" long:"install" description:"安装为系统服务"`
		Uninstall              bool   `long:"uninstall" description:"删除系统服务"`
		ShowConsole            bool   `long:"show-console" description:"Windows上显示控制台界面"`
		ServiceUser            string `long:"service-user" description:"用于启动服务的用户"`
		ServiceName            string `long:"service-name" description:"自定义服务名，默认为sealdice"`
		MultiInstanceOnWindows bool   `short:"m" long:"multi-instance" description:"允许在Windows上运行多个海豹"`
		Address                string `long:"address" description:"将UI的http服务地址改为此值，例: 0.0.0.0:3211"`
	}

	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}

	// 提早初始化是为了读取ServiceName
	diceManager := &dice.DiceManager{}
	diceManager.LoadDice()

	if opts.Install {
		serviceName := opts.ServiceName
		if serviceName == "" {
			serviceName = diceManager.ServiceName
		}
		if serviceName == "" {
			serviceName = "sealdice"
		}
		if serviceName != diceManager.ServiceName {
			diceManager.ServiceName = serviceName
			diceManager.Save()
		}
		serviceInstall(true, serviceName, opts.ServiceUser)
		return
	}

	if opts.Uninstall {
		serviceName := diceManager.ServiceName
		serviceInstall(false, serviceName, "")
		return
	}

	if !opts.ShowConsole || opts.MultiInstanceOnWindows {
		hideWindow()
	}

	if !opts.MultiInstanceOnWindows && TestRunning() {
		return
	}

	cwd, _ := os.Getwd()
	fmt.Printf("%s %s\n", dice.APPNAME, dice.VERSION)
	fmt.Println("工作路径: ", cwd)

	go trayInit()

	os.MkdirAll("./data", 0644)
	MainLoggerInit("./data/main.log", true)

	cleanUp := func() {
		logger.Info("程序即将退出，进行清理……")
		err := recover()
		if err != nil {
			logger.Errorf("异常: %v 堆栈: %v", err, string(debug.Stack()))
		}

		for _, i := range diceManager.Dice {
			i.Save(true)
		}
		for _, i := range diceManager.Dice {
			i.DB.Close()
		}
		diceManager.Help.Close()
		diceManager.Save()
		if diceManager.Cron != nil {
			diceManager.Cron.Stop()
		}
	}
	defer cleanUp()

	// 初始化核心
	diceManager.TryCreateDefault()
	diceManager.InitDice()

	// goja 大概占据5MB空间，压缩后1MB，还行
	// 按tengo和他自己的benchmark来看，还是比较出色的（当然和v8啥的不能比）
	// 一些想法:
	// 1. 脚本调用独立加锁，因为他线程不安全
	// 2. 将部分函数注册进去，如SetVar等
	// 3. 模拟一个LocalStorage给js用
	// 4. 提供一个自定义条件(js脚本)，返回true即为成功
	// 5. 所有条目(如helpdoc、牌堆、自定义回复)都带上一个mod字段，以mod名字为标记，可以一键装卸
	// 6. 可以向骰子注册varname solver，以指定的正则去实现自定义语法（例如我定义一个算符c，匹配c5e2这样的变量名）
	// 7. 可以向骰子注册自定义指令，指令必须存在模块归属，以便于关闭
	// 8. 存在一个tick()或update()函数，每隔一段时间必定会调用一次
	vm := goja.New()
	v, err := vm.RunString("2 + 2")
	if err != nil {
		panic(err)
	}
	if num := v.Export().(int64); num != 4 {
		panic(num)
	}

	diceManager.Cron.AddFunc("@every 15min", func() {
		go checkVersion(diceManager)
	})
	go checkVersion(diceManager)

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
			logger.Info("5s仍未关闭，稍后强制退出……")
			cleanUp()
			time.Sleep(time.Duration(3 * time.Second))
			os.Exit(0)
		}
	})()

	for _, d := range diceManager.Dice {
		go diceServe(d)
	}

	if opts.Address != "" {
		fmt.Println("由参数输入了服务地址:", opts.Address)
		diceManager.ServeAddress = opts.Address
	}

	uiServe(diceManager)
}

func diceServe(d *dice.Dice) {
	if len(d.ImSession.EndPoints) == 0 {
		d.Logger.Infof("未检测到任何帐号，请先到“帐号设置”进行添加")
	}

	for _, conn := range d.ImSession.EndPoints {
		if conn.Enable {
			if conn.Platform == "QQ" {
				pa := conn.Adapter.(*dice.PlatformAdapterQQOnebot)
				dice.GoCqHttpServe(d, conn, pa.InPackGoCqHttpPassword, pa.InPackGoCqHttpProtocol, true)
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

func uiServe(myDice *dice.DiceManager) {
	logger.Info("即将启动webui")
	// Echo instance
	e := echo.New()

	// Middleware
	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "token"},
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline'; img-src 'self' data:;",
	}))
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

//
//func checkCqHttpExists() bool {
//	if _, err := os.Stat("./go-cqhttp"); err == nil {
//		return true
//	}
//	return false
//}
