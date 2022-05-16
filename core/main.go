package main

import (
	"context"
	"fmt"
	"github.com/dop251/goja"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cp "github.com/otiai10/copy"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"sealdice-core/api"
	"sealdice-core/dice"
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

func cleanUpCreate(diceManager *dice.DiceManager) func() {
	return func() {
		logger.Info("程序即将退出，进行清理……")
		err := recover()
		if err != nil {
			showWindow()
			logger.Errorf("异常: %v 堆栈: %v", err, string(debug.Stack()))
			exec.Command("pause") // windows专属
		}

		for _, i := range diceManager.Dice {
			i.Save(true)
		}
		for _, i := range diceManager.Dice {
			i.DB.Close()
		}
		// 清理gocqhttp
		for _, i := range diceManager.Dice {
			for _, j := range i.ImSession.EndPoints {
				dice.GoCqHttpServeProcessKill(i, j)
			}
		}

		diceManager.Help.Close()
		diceManager.Save()
		if diceManager.Cron != nil {
			diceManager.Cron.Stop()
		}
	}

}

func main() {
	var opts struct {
		Install                bool   `short:"i" long:"install" description:"安装为系统服务"`
		Uninstall              bool   `long:"uninstall" description:"删除系统服务"`
		ShowConsole            bool   `long:"show-console" description:"Windows上显示控制台界面"`
		HideUIWhenBoot         bool   `long:"hide-ui" description:"启动时不弹出UI"`
		ServiceUser            string `long:"service-user" description:"用于启动服务的用户"`
		ServiceName            string `long:"service-name" description:"自定义服务名，默认为sealdice"`
		MultiInstanceOnWindows bool   `short:"m" long:"multi-instance" description:"允许在Windows上运行多个海豹"`
		Address                string `long:"address" description:"将UI的http服务地址改为此值，例: 0.0.0.0:3211"`
		DoUpdateWin            bool   `long:"do-update-win" description:"windows自动升级用，不要在任何情况下主动调用"`
		DoUpdateOthers         bool   `long:"do-update-others" description:"linux/mac自动升级用，不要在任何情况下主动调用"`
		Delay                  int64  `long:"delay"`
		JustForTest            bool   `long:"just-for-test"`
	}

	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}

	if opts.Delay != 0 {
		time.Sleep(time.Duration(opts.Delay) * time.Second)
	}
	dnsHack()

	os.MkdirAll("./data", 0644)
	MainLoggerInit("./data/main.log", true)

	// 提早初始化是为了读取ServiceName
	diceManager := &dice.DiceManager{}
	diceManager.LoadDice()

	if opts.Address != "" {
		fmt.Println("由参数输入了服务地址:", opts.Address)
		diceManager.ServeAddress = opts.Address
	}

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

	if opts.DoUpdateWin || opts.DoUpdateOthers {
		err := cp.Copy("./update/new", "./")
		if err != nil {
			logger.Warn("升级失败")
			return
		}
		ioutil.WriteFile("./auto_update_ok", []byte(""), 0644)
		logger.Warn("升级完成，即将重启主进程")
		exec.Command("./sealdice-core.exe").Start()
		return
	}

	updateFileName := "./auto_updat3.exe"
	_, err1 := os.Stat("./auto_updat3.exe")
	if err1 != nil {
		_, err1 = os.Stat("./auto_update.exe")
		updateFileName = "./auto_update.exe"
	}
	if err1 == nil {
		_, err = os.Stat("./auto_update_ok")
		if err == nil {
			logger.Warn("检测到 auto_update.exe，进行升级收尾工作")
			os.Remove("./auto_update_ok")
			os.Remove("./auto_update.exe")
			os.Remove("./auto_updat3.exe")
			os.RemoveAll("./update")
		} else {
			logger.Warn("检测到 auto_update.exe，即将进行升级")
			// 这5s延迟是保险，其实并不必要
			name := updateFileName
			err := exec.Command(name, "--delay=5", "--do-update-win").Start()
			if err != nil {
				logger.Warn("升级发生错误: ", err.Error())
				return
			}
			return
		}
	}
	_, err2 := os.Stat("./auto_update")
	if err2 == nil {
		_, err = os.Stat("./auto_update_ok")
		if err == nil {
			logger.Warn("检测到 auto_update.exe，进行升级收尾工作")
			os.Remove("./auto_update_ok")
			os.Remove("./auto_update")
			os.RemoveAll("./update")
		} else {
			logger.Warn("检测到 auto_update.exe，即将进行升级")
			err := cp.Copy("./update/new", "./")
			if err != nil {
				logger.Errorf("更新: 复制文件失败: %s", err.Error())
			}
			_ = os.Chmod("./sealdice-core", 0755)
			_ = os.Chmod("./go-cqhttp/go-cqhttp", 0755)
		}
	}

	if !opts.MultiInstanceOnWindows && TestRunning() {
		return
	}

	if !opts.ShowConsole || opts.MultiInstanceOnWindows {
		hideWindow()
	}

	cwd, _ := os.Getwd()
	fmt.Printf("%s %s\n", dice.APPNAME, dice.VERSION)
	fmt.Println("工作路径: ", cwd)

	if strings.HasPrefix(cwd, os.TempDir()) {
		// C:\Users\XXX\AppData\Local\Temp
		// C:\Users\XXX\AppData\Local\Temp\BNZ.627d774316768935
		tempDirWarn()
		return
	}

	go trayInit()
	go dice.TryGetBackendUrl()

	cleanUp := cleanUpCreate(diceManager)
	defer cleanUp()

	// 初始化核心
	diceManager.TryCreateDefault()
	diceManager.InitDice()

	if opts.JustForTest {
		diceManager.JustForTest = true
	}

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
		go CheckVersion(diceManager)
	})
	go CheckVersion(diceManager)
	go RebootRequestListen(diceManager)
	go UpdateRequestListen(diceManager)
	go UpdateCheckRequestListen(diceManager)

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

	if opts.Address != "" {
		fmt.Println("由参数输入了服务地址:", opts.Address)
	}

	for _, d := range diceManager.Dice {
		go diceServe(d)
	}

	uiServe(diceManager, opts.HideUIWhenBoot)
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

func uiServe(dm *dice.DiceManager, hideUI bool) {
	logger.Info("即将启动webui")
	// Echo instance
	e := echo.New()

	// Middleware
	//e.Use(middleware.Logger())
	//e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "token"},
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	mimePatch()
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline'; img-src 'self' data:;",
	}))
	// X-Content-Type-Options: nosniff

	groupStatic := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().URL.Path == "/" {
				responseWriter := c.Response()
				responseWriter.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				responseWriter.Header().Set("Pragma", "no-cache")
				responseWriter.Header().Set("Expires", "0")
			}
			return next(c)
		}
	}
	e.Use(groupStatic)
	e.Static("/", "./frontend")

	api.Bind(e, dm)
	e.HideBanner = true // 关闭banner，原因是banner图案会改变终端光标位置

	httpServe(e, dm, hideUI)

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

func dnsHack() {
	var (
		dnsResolverIP        = "114.114.114.114:53" // Google DNS resolver.
		dnsResolverProto     = "udp"                // Protocol to use for the DNS resolver
		dnsResolverTimeoutMs = 5000                 // Timeout (ms) for the DNS resolver (optional)
	)

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
				}
				return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	http.DefaultTransport.(*http.Transport).DialContext = dialContext
}

func mimePatch() {
	builtinMimeTypesLower := map[string]string{
		".css":  "text/css; charset=utf-8",
		".gif":  "image/gif",
		".htm":  "text/html; charset=utf-8",
		".html": "text/html; charset=utf-8",
		".jpg":  "image/jpeg",
		".js":   "application/javascript",
		".wasm": "application/wasm",
		".pdf":  "application/pdf",
		".png":  "image/png",
		".svg":  "image/svg+xml",
		".xml":  "text/xml; charset=utf-8",
	}

	for k, v := range builtinMimeTypesLower {
		_ = mime.AddExtensionType(k, v)
	}
}
