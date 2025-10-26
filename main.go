package main

// _ "net/http/pprof"
import (
	"errors"
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

	"github.com/gofrs/flock"
	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap/zapcore"

	"sealdice-core/api"
	"sealdice-core/dice"
	"sealdice-core/dice/service"
	"sealdice-core/logger"
	v2 "sealdice-core/migrate/v2"
	"sealdice-core/static"
	"sealdice-core/utils/crypto"
	"sealdice-core/utils/dboperator"
	"sealdice-core/utils/oschecker"
	"sealdice-core/utils/paniclog"
)

/*
*
二进制目录结构:
data/configs
data/extensions
data/logs
extensions/
*/

var sealLock = flock.New("sealdice-lock.lock")

func cleanupCreate(diceManager *dice.DiceManager) func() {
	return func() {
		log := logger.M()
		log.Info("程序即将退出，进行清理……")
		err := recover()
		if err != nil {
			showWindow()
			log.Errorf("异常: %v\n堆栈: %v", err, string(debug.Stack()))
			// 顺便修正一下上面这个，应该是木落忘了。
			if runtime.GOOS == "windows" {
				exec.Command("pause") // windows专属
			}
		}
		err = sealLock.Unlock()
		if err != nil {
			log.Errorf("文件锁归还出现异常 %v", err)
		}

		if !diceManager.CleanupFlag.CompareAndSwap(0, 1) {
			// 尝试更新cleanup标记，如果已经为1则退出
			return
		}

		for _, i := range diceManager.Dice {
			if i.IsAlreadyLoadConfig {
				i.Config.BanList.SaveChanged(i)
				i.Save(true)
				i.AttrsManager.Stop()
				for _, j := range i.ExtList {
					if j.Storage != nil {
						// 关闭
						err := j.StorageClose()
						if err != nil {
							showWindow()
							log.Errorf("异常: %v\n堆栈: %v", err, string(debug.Stack()))
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
			d.DBOperator.Close()
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
	time.Local = z //nolint:reassign // old code
}

func main() {
	var opts struct {
		Version                bool   `description:"显示版本号"                                                           long:"version"`
		Install                bool   `description:"安装为系统服务"                                                         long:"install"          short:"i"`
		Uninstall              bool   `description:"删除系统服务"                                                          long:"uninstall"`
		ShowConsole            bool   `description:"Windows上显示控制台界面"                                                 long:"show-console"`
		HideUIWhenBoot         bool   `description:"启动时不弹出UI"                                                        long:"hide-ui"`
		ServiceUser            string `description:"用于启动服务的用户"                                                       long:"service-user"`
		ServiceName            string `description:"自定义服务名，默认为sealdice"                                              long:"service-name"`
		MultiInstanceOnWindows bool   `description:"允许在Windows上运行多个海豹"                                               long:"multi-instance"   short:"m"`
		Address                string `description:"将UI的http服务地址改为此值，例: 0.0.0.0:3211"                                long:"address"`
		DoUpdateWin            bool   `description:"windows自动升级用，不要在任何情况下主动调用"                                       long:"do-update-win"`
		DoUpdateOthers         bool   `description:"linux/mac自动升级用，不要在任何情况下主动调用"                                     long:"do-update-others"`
		Delay                  int64  `long:"delay"`
		JustForTest            bool   `long:"just-for-test"`
		DBCheck                bool   `description:"检查数据库是否有问题"                                                      long:"db-check"`
		ShowEnv                bool   `description:"显示环境变量"                                                          long:"show-env"`
		VacuumDB               bool   `description:"对数据库进行整理, 使其收缩到最小尺寸"                                             long:"vacuum"`
		UpdateTest             bool   `description:"更新测试"                                                            long:"update-test"`
		LogLevel               int8   `choice:"-1"                                                                   choice:"0"              choice:"1" choice:"2" choice:"3" choice:"4" choice:"5" default:"0" description:"设置日志等级"             long:"log-level"`
		ContainerMode          bool   `description:"容器模式，该模式下禁用内置客户端"                                                long:"container-mode"`
	}
	// pprof
	// go func() {
	//	http.ListenAndServe("0.0.0.0:8899", nil)
	// }()
	// 读取命令行传参
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}
	// 防止输出
	if opts.Version {
		fmt.Fprintln(os.Stdout, dice.VERSION.String())
		return
	}
	if opts.ShowEnv {
		for i, e := range os.Environ() {
			fmt.Fprintln(os.Stdout, i, e)
		}
		return
	}
	// 提前到最开始初始化所有日志
	uiWriter := logger.NewUIWriter()
	log := logger.InitLogger(zapcore.Level(opts.LogLevel), uiWriter).Named(logger.LogKeyMain)
	// 初始化PanicLog
	paniclog.InitPanicLog()

	// 3. 提示日志打印
	log.Info("运行日志开始记录，海豹出现故障时可查看 data/main.log 与 data/panic.log 获取更多信息")
	// 加载env相关
	err = godotenv.Load()
	if err != nil {
		log.Errorf("未读取到.env参数，若您未使用docker或第三方数据库，可安全忽略。")
	}
	// 初始化文件加锁系统
	locked, err := sealLock.TryLock()
	// 如果有错误，或者未能取到锁
	if err != nil || !locked {
		// 打日志的时候防止打出nil
		if err == nil {
			err = errors.New("海豹正在运行中")
		}
		log.Errorf("获取锁文件失败，原因为: %v", err)
		showMsgBox("获取锁文件失败", "为避免数据损坏，拒绝继续启动。请检查是否启动多份海豹程序！")
		return
	}
	judge, osr := oschecker.OldVersionCheck()
	// 预留收集信息的接口，如果有需要可以考虑从这里拿数据。不从这里做提示的原因是Windows和Linux的展示方式不同。
	if judge {
		log.Info(fmt.Sprintf("您当前使用的系统为: %s", osr))
	}
	if opts.DBCheck {
		dboperator.DBCheck()
		return
	}
	if opts.VacuumDB {
		service.DBVacuum()
		return
	}
	deleteOldWrongFile()

	if opts.Delay != 0 {
		log.Infof("延迟启动 %d 秒", opts.Delay)
		time.Sleep(time.Duration(opts.Delay) * time.Second)
	}

	if runtime.GOOS == "android" {
		fixTimezone()
	}

	_ = os.MkdirAll("./data", 0o755)

	// 提早初始化是为了读取ServiceName

	// diceManager初始化数据库
	operator, err := dboperator.GetDatabaseOperator()
	if err != nil {
		log.Errorf("Failed to init database: %v", err)
		return
	}
	diceManager := &dice.DiceManager{
		Operator: operator,
	}

	if opts.ContainerMode {
		log.Info("当前为容器模式，内置适配器与更新功能已被禁用")
		diceManager.ContainerMode = true
	}

	diceManager.LoadDice()
	diceManager.IsReady = true

	if opts.Address != "" {
		log.Infof("由参数输入了服务地址: %s", opts.Address)
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
			log.Warn("检测到 auto_update.exe，即将自动退出当前程序并进行升级")
			log.Warn("程序目录下会出现“升级日志.log”，这代表升级正在进行中，如果失败了请检查此文件。")

			err = CheckUpdater(diceManager)
			if err != nil {
				log.Error("升级程序检查失败: ", err.Error())
			} else {
				_ = os.Remove("./auto_update.exe")
				// ui资源已经内置，删除旧的ui文件，这里有点风险，但是此时已经不考虑升级失败的情况
				_ = os.RemoveAll("./frontend")
				UpdateByFile(diceManager, "./update/update.zip", true)
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
			err = CheckUpdater(diceManager)
			if err != nil {
				log.Error("升级程序检查失败: ", err.Error())
			} else {
				_ = os.Remove("./auto_update")
				// ui资源已经内置，删除旧的ui文件，这里有点风险，但是此时已经不考虑升级失败的情况
				_ = os.RemoveAll("./frontend")
				UpdateByFile(diceManager, "./update/update.tar.gz", true)
			}
			return
		}
	}
	removeUpdateFiles()

	if opts.UpdateTest {
		err = CheckUpdater(diceManager)
		if err != nil {
			log.Error("升级程序检查失败: ", err.Error())
		} else {
			UpdateByFile(diceManager, "./xx.zip", true)
		}
	}

	// 先临时放这里，后面再整理一下升级模块
	diceManager.UpdateSealdiceByFile = func(packName string) bool {
		err = CheckUpdater(diceManager)
		if err != nil {
			log.Error("升级程序检查失败: ", err.Error())
			return false
		} else {
			return UpdateByFile(diceManager, packName, false)
		}
	}

	cwd, _ := os.Getwd()
	log.Info(dice.APPNAME, dice.VERSION.String(), "当前工作路径: ", cwd)

	if strings.HasPrefix(cwd, os.TempDir()) {
		// C:\Users\XXX\AppData\Local\Temp
		// C:\Users\XXX\AppData\Local\Temp\BNZ.627d774316768935
		tempDirWarn()
		return
	}

	useBuiltinUI := false
	checkFrontendExists := func() bool {
		var stat os.FileInfo
		stat, err = os.Stat("./frontend_overwrite")
		return err == nil && stat.IsDir()
	}
	if !checkFrontendExists() {
		log.Info("未检测到外置的UI资源文件，将使用内置资源启动UI")
		useBuiltinUI = true
	} else {
		log.Info("检测到外置的UI资源文件，将使用frontend_overwrite文件夹内的资源启动UI")
	}

	// // 尝试进行升级
	// migrate.TryMigrateToV12()
	// // 尝试修正log_items表的message字段类型
	// if migrateErr := migrate.LogItemFixDatatype(); migrateErr != nil {
	//	log.Fatalf("修正log_items表时出错，%s", migrateErr.Error())
	//	return
	// }
	// // v131迁移历史设置项到自定义文案
	// if migrateErr := migrate.V131DeprecatedConfig2CustomText(); migrateErr != nil {
	//	log.Fatalf("迁移历史设置项时出错，%s", migrateErr.Error())
	//	return
	// }
	// // v141重命名刷屏警告字段
	// if migrateErr := migrate.V141DeprecatedConfigRename(); migrateErr != nil {
	//	log.Fatalf("迁移历史设置项时出错，%s", migrateErr.Error())
	//	return
	// }
	// // v144删除旧的帮助文档
	// if migrateErr := migrate.V144RemoveOldHelpdoc(); migrateErr != nil {
	//	log.Fatalf("移除旧帮助文档时出错，%v", migrateErr)
	// }
	// // v150升级
	// err = migrate.V150Upgrade()
	// if err != nil {
	//	// Fatalf将会退出程序...或许应该用Errorf一类的吗？
	//	log.Fatalf("您的146数据库可能存在问题，为保护数据，已经停止执行150升级命令。请尝试联系开发者，并提供你的日志。\n"+
	//		"数据已回滚，您可暂时使用旧版本等待进一步的修复和更新。您的报错内容为: %v", err)
	// }
	err = v2.InitUpgrader(operator)
	if err != nil {
		log.Warnf("升级流程出现问题，请检查，问题为: %v", err)
	}

	if !opts.ShowConsole || opts.MultiInstanceOnWindows {
		hideWindow()
	}

	go dice.TryGetBackendURL()

	cleanUp := cleanupCreate(diceManager)
	defer dice.CrashLog()
	defer cleanUp()

	// 初始化核心
	diceManager.TryCreateDefault()
	diceManager.InitDice(uiWriter)

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
		log.Infof("由参数输入了服务地址: %s", opts.Address)
	}

	for _, d := range diceManager.Dice {
		go diceServe(d)
	}

	go uiServe(diceManager, opts.HideUIWhenBoot, useBuiltinUI)
	// OOM分析工具
	// err = nil
	// err = http.ListenAndServe(":9090", nil)
	// if err != nil {
	// 	fmt.Fprintf(os.Stdout, "ListenAndServe: %s", err)
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
	log := d.Logger
	defer dice.CrashLog()
	if len(d.ImSession.EndPoints) == 0 {
		log.Infof("未检测到任何帐号，请先到“帐号设置”进行添加")
	}

	d.UIEndpoint = new(dice.EndPointInfo)
	d.UIEndpoint.Enable = true
	d.UIEndpoint.Platform = "UI"
	d.UIEndpoint.ID = "1"
	d.UIEndpoint.State = 1
	d.UIEndpoint.UserID = "UI:1000"
	d.UIEndpoint.Adapter = &dice.PlatformAdapterHTTP{Session: d.ImSession, EndPoint: d.UIEndpoint}
	d.UIEndpoint.Session = d.ImSession

	dice.TextMapCompatibleCheckAll(d)

	for _, _conn := range d.ImSession.EndPoints {
		if _conn.Enable {
			go func(conn *dice.EndPointInfo) {
				defer dice.ErrorLogAndContinue(d)

				switch conn.Platform {
				case "QQ":
					if conn.ProtocolType == "walle-q" {
						pa := conn.Adapter.(*dice.PlatformAdapterWalleQ)
						dice.WalleQServe(d, conn, pa.InPackWalleQPassword, pa.InPackWalleQProtocol, false)
					}
					if conn.ProtocolType == "onebot" {
						pa := conn.Adapter.(*dice.PlatformAdapterGocq)
						if pa.BuiltinMode == "lagrange" || pa.BuiltinMode == "lagrange-gocq" {
							dice.LagrangeServe(d, conn, dice.LagrangeLoginInfo{
								IsAsyncRun: true,
							})
							return
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
					if conn.ProtocolType == "red" {
						dice.ServeRed(d, conn)
					}
					if conn.ProtocolType == "official" {
						dice.ServerOfficialQQ(d, conn)
					}
					if conn.ProtocolType == "satori" {
						dice.ServeSatori(d, conn)
					}
					if conn.ProtocolType == "LagrangeGo" {
						// dice.ServeLagrangeGo(d, conn)
						return
					}
					if conn.ProtocolType == "milky" {
						dice.ServeMilky(d, conn)
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
	log := logger.M()
	log.Info("即将启动webui")
	// Echo instance
	e := echo.New()

	// 为UI添加日志，以echo方式输出
	e.Use(logger.EchoLogMiddleware())
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
