package main

import (
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	// _ "net/http/pprof"

	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"sealdice-core/api"
	"sealdice-core/dice"
	diceLogger "sealdice-core/dice/logger"
	"sealdice-core/dice/model"
	"sealdice-core/migrate"
	"sealdice-core/static"
	"sealdice-core/utils/crypto"
)

/**
二进制目录结构:
data/configs
data/extensions
data/logs

extensions/
*/

func cleanupCreate(diceManager *dice.DiceManager) func() {
	return func() {
		logger.Info("程序即将退出，进行清理……")
		err := recover()
		if err != nil {
			showWindow()
			logger.Errorf("异常: %v\n堆栈: %v", err, string(debug.Stack()))
			// 顺便修正一下上面这个，应该是木落忘了。
			if runtime.GOOS == "windows" {
				exec.Command("pause") // windows专属
			}
		}

		if !diceManager.CleanupFlag.CompareAndSwap(0, 1) {
			// 尝试更新cleanup标记，如果已经为1则退出
			return
		}

		for _, i := range diceManager.Dice {
			if i.IsAlreadyLoadConfig {
				i.BanList.SaveChanged(i)
				i.Save(true)
				for _, j := range i.ExtList {
					if j.Storage != nil {
						// 关闭
						err := j.StorageClose()
						if err != nil {
							showWindow()
							logger.Errorf("异常: %v\n堆栈: %v", err, string(debug.Stack()))
							// 木落没有加该检查 补充上
							if runtime.GOOS == "windows" {
								exec.Command("pause") // windows专属
							}
						}
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
				dbData := d.DBData
				if dbData != nil {
					d.DBData = nil
					_ = dbData.Close()
				}
			})()

			(func() {
				defer func() {
					_ = recover()
				}()
				dbLogs := d.DBLogs
				if dbLogs != nil {
					d.DBLogs = nil
					_ = dbLogs.Close()
				}
			})()

			(func() {
				defer func() {
					_ = recover()
				}()
				cm := d.CensorManager
				if cm != nil && cm.DB != nil {
					dbCensor := cm.DB
					cm.DB = nil
					_ = dbCensor.Close()
				}
			})()
		}

		// 清理gocqhttp
		for _, i := range diceManager.Dice {
			if i.ImSession != nil && i.ImSession.EndPoints != nil {
				for _, j := range i.ImSession.EndPoints {
					dice.BuiltinQQServeProcessKill(i, j)
				}
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

func fixTimezone() {
	out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return
	}
	z, err := time.LoadLocation(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	time.Local = z
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
		DBCheck                bool   `long:"db-check" description:"检查数据库是否有问题"`
		ShowEnv                bool   `long:"show-env" description:"显示环境变量"`
		VacuumDB               bool   `long:"vacuum" description:"对数据库进行整理, 使其收缩到最小尺寸"`
		UpdateTest             bool   `long:"update-test" description:"更新测试"`
		LogLevel               int8   `long:"log-level" description:"设置日志等级" default:"0" choice:"-1" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4" choice:"5"`
		ContainerMode          bool   `long:"container-mode" description:"容器模式，该模式下禁用内置客户端"`
	}

	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}

	if opts.Version {
		fmt.Println(dice.VERSION.String())
		return
	}
	if opts.DBCheck {
		model.DBCheck("data/default")
		return
	}
	if opts.VacuumDB {
		model.DBVacuum()
		return
	}
	if opts.ShowEnv {
		for i, e := range os.Environ() {
			println(i, e)
		}
		return
	}
	deleteOldWrongFile()

	if opts.Delay != 0 {
		fmt.Println("延迟启动", opts.Delay, "秒")
		time.Sleep(time.Duration(opts.Delay) * time.Second)
	}

	if runtime.GOOS == "android" {
		fixTimezone()
	}

	_ = os.MkdirAll("./data", 0o755)
	MainLoggerInit("./data/main.log", true)

	diceLogger.SetEnableLevel(zapcore.Level(opts.LogLevel))

	// 提早初始化是为了读取ServiceName
	diceManager := &dice.DiceManager{}

	if opts.ContainerMode {
		logger.Info("当前为容器模式，内置适配器与更新功能已被禁用")
		diceManager.ContainerMode = true
	}

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

	if _, err1 := os.Stat("./auto_update.exe"); err1 == nil {
		doNext := true
		if filepath.Base(os.Args[0]) != "auto_update.exe" {
			a := crypto.Sha256Checksum("./auto_update.exe")
			b := crypto.Sha256Checksum(os.Args[0])
			doNext = a != b
		}
		if doNext {
			// 只有不同文件才进行校验
			// windows平台旧版本到1.4.0流程
			_ = os.WriteFile("./升级失败指引.txt", []byte("如果升级成功不用理会此文档，直接删除即可。\r\n\r\n如果升级后无法启动，或再次启动后恢复到旧版本，先不要紧张。\r\n你升级前的数据备份在backups目录。\r\n如果无法启动，请删除海豹目录中的\"update\"、\"auto_update.exe\"并手动进行升级。\n如果升级成功但在再次重启后回退版本，同上。\n\n如有其他问题可以加企鹅群询问：524364253 562897832"), 0o644)
			logger.Warn("检测到 auto_update.exe，即将自动退出当前程序并进行升级")
			logger.Warn("程序目录下会出现“升级日志.log”，这代表升级正在进行中，如果失败了请检查此文件。")

			err := CheckUpdater(diceManager)
			if err != nil {
				logger.Error("升级程序检查失败: ", err.Error())
			} else {
				_ = os.Remove("./auto_update.exe")
				// ui资源已经内置，删除旧的ui文件，这里有点风险，但是此时已经不考虑升级失败的情况
				_ = os.RemoveAll("./frontend")
				UpdateByFile(diceManager, nil, "./update/update.zip", true)
			}
			return
		}
	}

	if _, err2 := os.Stat("./auto_update"); err2 == nil {
		doNext := true
		if filepath.Base(os.Args[0]) != "auto_update" {
			a := crypto.Sha256Checksum("./auto_update")
			b := crypto.Sha256Checksum(os.Args[0])
			doNext = a != b
		}

		if doNext {
			err := CheckUpdater(diceManager)
			if err != nil {
				logger.Error("升级程序检查失败: ", err.Error())
			} else {
				_ = os.Remove("./auto_update")
				// ui资源已经内置，删除旧的ui文件，这里有点风险，但是此时已经不考虑升级失败的情况
				_ = os.RemoveAll("./frontend")
				UpdateByFile(diceManager, nil, "./update/update.tar.gz", true)
			}
			return
		}
	}
	removeUpdateFiles()

	if opts.UpdateTest {
		err := CheckUpdater(diceManager)
		if err != nil {
			logger.Error("升级程序检查失败: ", err.Error())
		} else {
			UpdateByFile(diceManager, nil, "./xx.zip", true)
		}
	}

	// 先临时放这里，后面再整理一下升级模块
	diceManager.UpdateSealdiceByFile = func(packName string, log *zap.SugaredLogger) bool {
		err := CheckUpdater(diceManager)
		if err != nil {
			logger.Error("升级程序检查失败: ", err.Error())
			return false
		} else {
			return UpdateByFile(diceManager, log, packName, false)
		}
	}

	cwd, _ := os.Getwd()
	fmt.Printf("%s %s\n", dice.APPNAME, dice.VERSION.String())
	fmt.Println("工作路径: ", cwd)

	if strings.HasPrefix(cwd, os.TempDir()) {
		// C:\Users\XXX\AppData\Local\Temp
		// C:\Users\XXX\AppData\Local\Temp\BNZ.627d774316768935
		tempDirWarn()
		return
	}

	useBuiltinUI := false
	checkFrontendExists := func() bool {
		stat, err := os.Stat("./frontend_overwrite")
		return err == nil && stat.IsDir()
	}
	if !checkFrontendExists() {
		logger.Info("未检测到外置的UI资源文件，将使用内置资源启动UI")
		useBuiltinUI = true
	} else {
		logger.Info("检测到外置的UI资源文件，将使用frontend_overwrite文件夹内的资源启动UI")
	}

	// 删除遗留的shm和wal文件
	if !model.DBCacheDelete() {
		logger.Error("数据库缓存文件删除失败")
		showMsgBox("数据库缓存文件删除失败", "为避免数据损坏，拒绝继续启动。请检查是否启动多份程序，或有其他程序正在使用数据库文件！")
		return
	}

	// 尝试进行升级
	migrate.TryMigrateToV12()
	// 尝试修正log_items表的message字段类型
	if migrateErr := migrate.LogItemFixDatatype(); migrateErr != nil {
		logger.Errorf("修正log_items表时出错，%s", migrateErr.Error())
		return
	}
	// v131迁移历史设置项到自定义文案
	if migrateErr := migrate.V131DeprecatedConfig2CustomText(); migrateErr != nil {
		logger.Errorf("迁移历史设置项时出错，%s", migrateErr.Error())
		return
	}
	// v141重命名刷屏警告字段
	if migrateErr := migrate.V141DeprecatedConfigRename(); migrateErr != nil {
		logger.Errorf("迁移历史设置项时出错，%s", migrateErr.Error())
		return
	}
	// v144删除旧的帮助文档
	if migrateErr := migrate.V144RemoveOldHelpdoc(); migrateErr != nil {
		logger.Errorf("移除旧帮助文档时出错，%v", migrateErr)
	}
	// v150升级
	migrate.V150Upgrade()

	if !opts.ShowConsole || opts.MultiInstanceOnWindows {
		hideWindow()
	}

	go dice.TryGetBackendURL()

	cleanUp := cleanupCreate(diceManager)
	defer dice.CrashLog()
	defer cleanUp()

	// 初始化核心
	diceManager.TryCreateDefault()
	diceManager.InitDice()

	if opts.JustForTest {
		diceManager.JustForTest = true
	}

	go func() {
		// 每5分钟做一次新版本检查
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
	// go func() {
	//	http.ListenAndServe("0.0.0.0:8899", nil)
	// }()

	go uiServe(diceManager, opts.HideUIWhenBoot, useBuiltinUI)
	// OOM分析工具
	// err = nil
	// err = http.ListenAndServe(":9090", nil)
	// if err != nil {
	// 	fmt.Printf("ListenAndServe: %s", err)
	// }

	// darwin 的托盘菜单似乎需要在主线程启动才能工作，调整到这里
	trayInit(diceManager)
}

func removeUpdateFiles() {
	// 无论原因，只要走到这里全部删除
	_ = os.Remove("./auto_update_ok")
	_ = os.Remove("./auto_update.exe")
	_ = os.Remove("./auto_updat3.exe")
	_ = os.Remove("./auto_update_ok")
	_ = os.Remove("./auto_update")
	_ = os.Remove("./_delete_me.exe")
	_ = os.RemoveAll("./update")
}

func diceServe(d *dice.Dice) {
	defer dice.CrashLog()
	if len(d.ImSession.EndPoints) == 0 {
		d.Logger.Infof("未检测到任何帐号，请先到“帐号设置”进行添加")
	}

	d.UIEndpoint = new(dice.EndPointInfo)
	d.UIEndpoint.Enable = true
	d.UIEndpoint.Platform = "UI"
	d.UIEndpoint.ID = "1"
	d.UIEndpoint.State = 1
	d.UIEndpoint.UserID = "UI:1000"
	d.UIEndpoint.Adapter = &dice.PlatformAdapterHTTP{Session: d.ImSession, EndPoint: d.UIEndpoint}

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
						if pa.Implementation == "lagrange" {
							dice.LagrangeServe(d, conn, dice.GoCqhttpLoginInfo{
								IsAsyncRun: true,
							})
						} else {
							dice.GoCqhttpServe(d, conn, dice.GoCqhttpLoginInfo{
								Password:         pa.InPackGoCqhttpPassword,
								Protocol:         pa.InPackGoCqhttpProtocol,
								AppVersion:       pa.InPackGoCqhttpAppVersion,
								IsAsyncRun:       true,
								UseSignServer:    pa.UseSignServer,
								SignServerConfig: pa.SignServerConfig,
							})
						}
					}
					if conn.EndPointInfoBase.ProtocolType == "red" {
						dice.ServeRed(d, conn)
					}
					if conn.EndPointInfoBase.ProtocolType == "official" {
						dice.ServerOfficialQQ(d, conn)
					}
					if conn.EndPointInfoBase.ProtocolType == "satori" {
						dice.ServeSatori(d, conn)
					}
					if conn.EndPointInfoBase.ProtocolType == "LagrangeGo" {
						// dice.ServeLagrangeGo(d, conn)
						return
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
				case "SLACK":
					dice.ServeSlack(d, conn)
				case "DINGTALK":
					dice.ServeDingTalk(d, conn)
				case "SEALCHAT":
					dice.ServeSealChat(d, conn)
				}
			}(_conn)
		} else {
			_conn.State = 0 // 重置状态
		}
	}
}

func uiServe(dm *dice.DiceManager, hideUI bool, useBuiltin bool) {
	logger.Info("即将启动webui")
	// Echo instance
	e := echo.New()

	// Middleware
	// e.Use(middleware.Logger())
	// e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		Skipper:      middleware.DefaultSkipper,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "token"},
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))

	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	mimePatch()
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline'; img-src 'self' data: blob: *; style-src  'self' 'unsafe-inline' *; frame-src 'self' *;",
		// XFrameOptions:         "ALLOW-FROM https://captcha.go-cqhttp.org/",
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
	if useBuiltin {
		frontend, _ := fs.Sub(static.Frontend, "frontend")
		e.StaticFS("/", frontend)
	} else {
		e.Static("/", "./frontend_overwrite")
	}

	api.Bind(e, dm)
	e.HideBanner = true // 关闭banner，原因是banner图案会改变终端光标位置

	httpServe(e, dm, hideUI)
}

//
// func checkCqHttpExists() bool {
//	if _, err := os.Stat("./go-cqhttp"); err == nil {
//		return true
//	}
//	return false
// }

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
