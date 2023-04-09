package main

import (
	"context"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cp "github.com/otiai10/copy"
	"mime"
	"net"
	"net/http"
	"sealdice-core/migrate"
	//_ "net/http/pprof"
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
			logger.Errorf("异常: %v\n堆栈: %v", err, string(debug.Stack()))
			exec.Command("pause") // windows专属
		}

		for _, i := range diceManager.Dice {
			if i.IsAlreadyLoadConfig {
				i.BanList.SaveChanged(i)
				i.Save(true)
				for _, j := range i.ExtList {
					if j.Storage != nil {
						_ = j.Storage.Close()
					}
				}
				i.IsAlreadyLoadConfig = false
			}
		}

		for _, i := range diceManager.Dice {
			d := i
			(func() {
				defer func() {
					_ = recover()
				}()
				var dbData = d.DBData
				if dbData != nil {
					d.DBData = nil
					_ = dbData.Close()
				}
			})()

			(func() {
				defer func() {
					_ = recover()
				}()
				var dbLogs = d.DBLogs
				if dbLogs != nil {
					d.DBLogs = nil
					_ = dbLogs.Close()
				}
			})()
			//if i.DB != nil {
			//	i.DB.Close()
			//}
		}

		// 清理gocqhttp
		for _, i := range diceManager.Dice {
			for _, j := range i.ImSession.EndPoints {
				dice.GoCqHttpServeProcessKill(i, j)
			}
		}

		if diceManager.Help != nil {
			diceManager.Help.Close()
		}
		if diceManager.IsReady {
			diceManager.Save()
		}
		if diceManager.Cron != nil {
			diceManager.Cron.Stop()
		}
	}
}

func deleteOldWrongFile() {
	_ = os.Remove("./data/names/data-logs.db")
	_ = os.Remove("./data/names/names.zip")
	_ = os.Remove("./data/names/serve.yaml")
	_ = os.Remove("./data/names/names/names.xlsx")
	_ = os.Remove("./data/names/names/names-dnd.xlsx")
	_ = os.Remove("./data/names/names")

	// 1.2.5之前版本兼容
	_ = os.RemoveAll("./data/helpdoc/DND/3R")
	_ = os.RemoveAll("./data/helpdoc/DND/核心")
	_ = os.RemoveAll("./data/helpdoc/DND/扩展")
	_ = os.RemoveAll("./data/helpdoc/DND/模组")
	_ = os.RemoveAll("./data/helpdoc/DND/破解奥秘")
	_ = os.Remove("./data/helpdoc/DND/法术列表大全.xlsx")
	_ = os.Remove("./data/helpdoc/DND/名词解释.xlsx")
	_ = os.Remove("./data/helpdoc/DND/子职列表大全.xlsx")
}

func main() {
	var opts struct {
		Version                bool   `long:"version" description:"显示版本号"`
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
		//DBCheck                bool   `long:"db-check" description:"检查数据库是否有问题"`
	}

	//dice.SetDefaultNS([]string{"114.114.114.114:53", "8.8.8.8:53"}, false)
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}

	if opts.Version {
		fmt.Println(dice.VERSION)
		return
	}
	//if opts.DBCheck {
	//	model.DBCheck("data/default")
	//	return
	//}
	deleteOldWrongFile()

	if opts.Delay != 0 {
		time.Sleep(time.Duration(opts.Delay) * time.Second)
	}
	dnsHack()

	_ = os.MkdirAll("./data", 0644)
	MainLoggerInit("./data/main.log", true)

	// 提早初始化是为了读取ServiceName
	diceManager := &dice.DiceManager{}
	diceManager.LoadDice()
	diceManager.IsReady = true

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
		// 为之后留一个接口
		if f, _ := os.Stat("./start.exe"); f != nil {
			// run start.exe
			logger.Warn("检测到启动器，尝试运行")
			_ = exec.Command("./start.exe", "/u-first").Start()
			return
		}

		logger.Warn("准备进行升级程序，先等待10s")
		time.Sleep(10 * time.Second)
		err := cp.Copy("./update/new", "./")
		if err != nil {
			logger.Warn("升级失败")
			return
		}

		// 同样是留接口，如果新版内置了start.exe，就运行它
		if f, _ := os.Stat("./start.exe"); f != nil {
			// run start.exe
			logger.Warn("检测到启动器，尝试运行")
			_ = exec.Command("./start.exe", "/u-second").Start()
			return
		}

		_ = os.WriteFile("./auto_update_ok", []byte(""), 0644)
		logger.Warn("升级完成，即将重启主进程")
		_ = exec.Command("./sealdice-core.exe").Start()
		return
	}

	updateFileName := "./auto_update.exe"
	_, err1 := os.Stat("./auto_update.exe")

	if err1 == nil {
		_, err = os.Stat("./auto_update_ok")
		if err == nil {
			logger.Warn("检测到 auto_update.exe，进行升级收尾工作")
			_ = os.Remove("./auto_update_ok")
			_ = os.Remove("./auto_update.exe")
			_ = os.Remove("./auto_updat3.exe")
			_ = os.RemoveAll("./update")
		} else {
			_ = os.WriteFile("./升级失败指引.txt", []byte("如果升级成功不用理会此文档，直接删除即可。\r\n\r\n如果升级后无法启动，或再次启动后恢复到旧版本，先不要紧张。\r\n你升级前的数据备份在backups目录。\r\n如果无法启动，请删除海豹目录中的\"update\"、\"auto_update.exe\"并手动进行升级。\n如果升级成功但在再次重启后回退版本，同上。\n\n如有其他问题可以加企鹅群询问：524364253 562897832"), 0644)
			logger.Warn("检测到 auto_update.exe，即将进行升级")
			// 这5s延迟是保险，其实并不必要
			// 2023/1/9: 还是必要的，在有些设备上还要更久时间，所以现在改成15s
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
			_ = os.Remove("./auto_update_ok")
			_ = os.Remove("./auto_update")
			_ = os.RemoveAll("./update")
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
	removeUpdateFiles()

	//if !opts.MultiInstanceOnWindows && TestRunning() {
	//	return
	//}

	cwd, _ := os.Getwd()
	fmt.Printf("%s %s\n", dice.APPNAME, dice.VERSION)
	fmt.Println("工作路径: ", cwd)

	if strings.HasPrefix(cwd, os.TempDir()) {
		// C:\Users\XXX\AppData\Local\Temp
		// C:\Users\XXX\AppData\Local\Temp\BNZ.627d774316768935
		tempDirWarn()
		return
	}

	checkFrontendExists := func() bool {
		stat, err := os.Stat("./frontend")
		return err == nil && stat.IsDir()
	}

	// 检查目录是否正确
	//if !checkFrontendExists() {
	// 给一次修正机会吗？
	//exe, err := filepath.Abs(os.Args[0])
	//if err == nil {
	//	ret := filepath.Dir(exe)
	//}
	//}

	if !checkFrontendExists() {
		showWarn("SealDice 文件不完整", "未检查到UI文件目录，程序不完整，将自动退出。\n也可能是当前工作路径错误。")
		logger.Error("因缺少frontend目录而自动退出")
		return
	}

	// 尝试进行升级
	migrate.TryMigrateToV12()

	if !opts.ShowConsole || opts.MultiInstanceOnWindows {
		hideWindow()
	}

	go trayInit(diceManager)
	go dice.TryGetBackendUrl()

	cleanUp := cleanUpCreate(diceManager)
	defer dice.CrashLog()
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
	//vm := goja.New()
	//v, err := vm.RunString("2 + 2")
	//if err != nil {
	//	panic(err)
	//}
	//if num := v.Export().(int64); num != 4 {
	//	panic(num)
	//}

	//_, _ = diceManager.Cron.AddFunc("@every 15min", func() {
	//	go CheckVersion(diceManager)
	//})
	go func() {
		for {
			go CheckVersion(diceManager)
			time.Sleep(5 * time.Minute)
		}
	}()
	go RebootRequestListen(diceManager)
	go UpdateRequestListen(diceManager)
	go UpdateCheckRequestListen(diceManager)

	// 强制清理机制
	go (func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		<-interrupt
		cleanUp()
		time.Sleep(3 * time.Second)
		os.Exit(0)

	})()

	if opts.Address != "" {
		fmt.Println("由参数输入了服务地址:", opts.Address)
	}

	for _, d := range diceManager.Dice {
		go diceServe(d)
	}

	// pprof
	//go func() {
	//	http.ListenAndServe("0.0.0.0:8899", nil)
	//}()

	uiServe(diceManager, opts.HideUIWhenBoot)
	//OOM分析工具
	//err = nil
	//err = http.ListenAndServe(":9090", nil)
	//if err != nil {
	//	fmt.Printf("ListenAndServe: %s", err)
	//}
}

func removeUpdateFiles() {
	// 无论原因，只要走到这里全部删除
	_ = os.Remove("./auto_update_ok")
	_ = os.Remove("./auto_update.exe")
	_ = os.Remove("./auto_updat3.exe")
	_ = os.Remove("./auto_update_ok")
	_ = os.Remove("./auto_update")
	_ = os.RemoveAll("./update")
}

func diceServe(d *dice.Dice) {
	defer dice.CrashLog()
	if len(d.ImSession.EndPoints) == 0 {
		d.Logger.Infof("未检测到任何帐号，请先到“帐号设置”进行添加")
	}

	for _, _conn := range d.ImSession.EndPoints {
		if _conn.Enable {
			go func(conn *dice.EndPointInfo) {
				defer dice.ErrorLogAndContinue(d)

				switch conn.Platform {
				case "QQ":
					if conn.EndPointInfoBase.ProtocolType == "walle-q" {
						pa := conn.Adapter.(*dice.PlatformAdapterWalleQ)
						dice.WalleQServe(d, conn, pa.InPackWalleQPassword, pa.InPackWalleQProtocol, false)
					}
					if conn.EndPointInfoBase.ProtocolType == "onebot" {
						pa := conn.Adapter.(*dice.PlatformAdapterGocq)
						dice.GoCqHttpServe(d, conn, pa.InPackGoCqHttpPassword, pa.InPackGoCqHttpProtocol, true)
					}
					time.Sleep(10 * time.Second) // 稍作等待再连接
					dice.ServeQQ(d, conn)
				case "DISCORD":
					dice.ServeDiscord(d, conn)
				case "KOOK":
					dice.ServeKook(d, conn)
				case "TG":
					dice.ServeTelegram(d, conn)
				case "MC":
					dice.ServeMinecraft(d, conn)
				case "DODO":
					dice.ServeDodo(d, conn)
				}

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
			}(_conn)
		} else {
			_conn.State = 0 // 重置状态
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
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline'; img-src 'self' data: *; style-src  'self' 'unsafe-inline' *; frame-src 'self' *;",
		//XFrameOptions:         "ALLOW-FROM https://captcha.go-cqhttp.org/",
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
		dnsResolverIP    = "114.114.114.114:53" // Google DNS resolver.
		dnsResolverProto = "udp"                // Protocol to use for the DNS resolver
	)
	var dialer net.Dialer
	net.DefaultResolver = &net.Resolver{
		PreferGo: false,
		Dial: func(context context.Context, _, _ string) (net.Conn, error) {
			conn, err := dialer.DialContext(context, dnsResolverProto, dnsResolverIP)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
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
