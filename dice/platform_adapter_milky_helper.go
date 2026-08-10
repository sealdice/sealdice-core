package dice

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"sealdice-core/logger"
	"sealdice-core/utils/procs"
)

var defaultLagrangeV2Config = `{
    "$schema": "https://raw.githubusercontent.com/LagrangeDev/LagrangeV2/refs/heads/main/Lagrange.Milky/Resources/appsettings_schema.json",
    "Logging": {
        "LogLevel": {
            "Default": "Information",
        },
    },
    "Core": {
        // "Server": {
        //     Whether to automatically reconnect to the server
        //     "AutoReconnect": true,

        //     Whether to use IPv6 to connect to the server
        //     "UseIPv6Network": false,

        //     Whether to automatically select the fastest server
        //     "GetOptimumServer": true,
        // },
        "Signer": {
            // Signer URL
            "Url": "{NTSignServer地址}",

            // Signer token
            // "Token": null

            // Proxy for connect signer
            // only supports Http proxy
            // "ProxyUrl": null,
        },
        "Login": {
            // Account uin
            // If the Uin is inconsistent with the actual login account, quick login will not be possible
            "Uin": {账号UIN},
            
            // Account password
            // Set to null to login via QrCode
            // "Password": null,

            // Device Name
            // Only valid when logging in without Keystore
            "DeviceName": "Ubuntu 22.04",

            // Whether to try to log in automatically after disconnection
            // "AutoReLogin": true,

            // Whether to use ASCII compatible QrCode
            // "CompatibleQrCode": false,

            // Whether to use the online validating parser provided by the mysterious person
            // "UseOnlineCaptchaResolver": true,
        },
    },
    "Milky": {
        // The host that Milky service listens on
        // Look https://learn.microsoft.com/zh-cn/dotnet/fundamentals/runtime-libraries/system-net-httplistener
        // If you use * to expose your data to all networks, please ensure proper security settings
        // e.g. setting a access token, configuring a firewall
        "Host": "127.0.0.1",

        // The port that the Milky service listens on
        "Port": {WS端口},

        // The path prefix that Milky service listens on
        // "Prefix": "/",

        // Token for verification, Set to null to disable
        "AccessToken": "{访问Token}",

        // Whether to enable WebSocket service
        // "EnabledWebSocket": true,

        // Set to null to disable the WebHook service
        // "WebHook": null, // Default
        // "WebHook": {
        //     // WebHook Target URL
        //     "Url": "http://127.0.0.1:3001/webhook"
        // }

        // "Message": {
        //     // Whether to ignore messages sent by Bot
        //     "IgnoreBotMessage": false,
        //     "Cache": {
        //         "Policy": "LRU",
        //         // Maximum cache capacity
        //         "Capacity": 1000,
        //     },
        // },
    },
}
`

var defaultYogurtConfig = `{
    "configVersion": 3,
    "protocol": {
        "uin": {账号UIN},
        "password": "",
        "os": "Linux",
        "version": "46494",
        "signApiUrl": "{NTSignServer地址}",
        "pcLagrangeSignToken": "placeholder",
        "androidUseLegacySign": false
    },
    "milky": {
        "http": {
            "host": "127.0.0.1",
            "port": {WS端口},
            "prefix": "",
            "accessToken": "{访问Token}",
            "corsOrigins": []
        },
        "webhook": {
            "endpoints": []
        },
        "reportSelfMessage": true,
        "preloadContacts": false,
        "ffmpegPath": ""
    },
    "logging": {
        "ansiLevel": "ANSI256",
        "coreLogLevel": "DEBUG"
    },
    "security": {
        "skipOnLaunchListenAddressCheck": false
    },
    "debug": {
        "enableFaceDetailsApi": false
    }
}`

var SealSignV3Url = ``

type AddMilkyEcho struct {
	Token       string
	WsGateway   string
	RestGateway string
	BuiltInMode string
}

func NewMilkyConnItem(v AddMilkyEcho) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "milky"
	conn.Enable = false
	conn.RelWorkDir = "extra/milky-" + conn.ID
	conn.Adapter = &PlatformAdapterMilky{
		EndPoint:    conn,
		Token:       v.Token,
		WsGateway:   v.WsGateway,
		RestGateway: v.RestGateway,
		BuiltInMode: v.BuiltInMode,
	}
	return conn
}

func ServeMilky(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "QQ" {
		ep.BindRuntime(d.ImSession)
		d.Logger.Infof("Milky 尝试连接")
		_ = StartEndpointLifecycle(d, ep)
	}
}

func BuiltinMilkyClientKill(dice *Dice, conn *EndPointInfo) {
	defer func() {
		if r := recover(); r != nil {
			dice.Logger.Error("内置 Milky 客户端清理报错: ", r)
		}
	}()
	pa, ok := conn.Adapter.(*PlatformAdapterMilky)
	if !ok || pa.BuiltInMode == "" {
		return
	}
	pa.builtInProcessMu.Lock()
	defer pa.builtInProcessMu.Unlock()
	if err := stopMilkyBuiltInLocked(pa); err != nil {
		dice.Logger.Error("停止 Milky 进程失败: ", err)
	}
}

func ServeMilkyBuiltIn(d *Dice, ep *EndPointInfo) {
	ep.BindRuntime(d.ImSession)
	_ = StartEndpointLifecycle(d, ep)
}

func serveMilkyBuiltIn(ctx context.Context, d *Dice, ep *EndPointInfo, reporter EndpointRunReporter) error {
	defer CrashLog()
	pa, ok := ep.Adapter.(*PlatformAdapterMilky)
	if !ok || pa.BuiltInMode == "" {
		return errors.New("milky built-in adapter is unavailable")
	}
	pa.builtInProcessMu.Lock()
	defer pa.builtInProcessMu.Unlock()

	if d.ContainerMode {
		d.Logger.Warnf("当前处于容器模式，Milky 内置版本不可用")
		ep.State = StateConnectionFailed
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return errors.New("milky built-in is unavailable in container mode")
	}
	uin, err := strconv.ParseInt(ExtractQQUserID(ep.UserID), 10, 64)
	if err != nil {
		d.Logger.Errorf("解析QQ号失败: %s", ep.UserID)
		ep.State = StateConnectionFailed
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return err
	}
	doServe := func() {
		if ep.Platform == "QQ" {
			d.Logger.Infof("Milky 尝试连接")
			if pa.Serve() != 0 {
				d.Logger.Errorf("连接Milky失败")
				ep.State = StateConnectionFailed
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				BuiltinMilkyClientKill(d, ep)
				if reporter != nil {
					reporter.Failed(errors.New("milky built-in websocket connect failed"))
				}
				return
			}
			if reporter != nil {
				reporter.Started()
			}
		}
	}
	ep.BindRuntime(d.ImSession)
	log := zap.S().Named(logger.LogKeyAdapter)

	workDir := filepath.Join(d.BaseConfig.DataDir, ep.RelWorkDir)
	diceWorkdir, _ := os.Getwd()
	milkyExePath, _ := filepath.Abs(filepath.Join(diceWorkdir, fmt.Sprintf("milky/%s", pa.BuiltInMode)))
	var configFilePath string
	switch pa.BuiltInMode {
	case "lagrangeV2":
		configFilePath = filepath.Join(workDir, "appsettings.jsonc")
	case "yogurt":
		configFilePath = filepath.Join(workDir, "config.json")
	}
	qrcodeFilePath := filepath.Join(workDir, "qrcode.png")
	milkyExePath = filepath.ToSlash(milkyExePath) // windows平台需要这个替换
	if runtime.GOOS == "windows" {
		milkyExePath += ".exe" //nolint:ineffassign
	}
	_ = os.MkdirAll(workDir, 0o755)
	_ = os.Chmod(milkyExePath, 0o755)
	if pa.IntentSession != nil {
		_ = pa.IntentSession.Close()
		pa.IntentSession = nil
	}
	if err := stopMilkyBuiltInLocked(pa); err != nil {
		log.Errorf("停止已有 Milky 进程失败: %s", err)
		ep.State = StateConnectionFailed
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return err
	}
	if err := prepareMilkyBuiltInConfig(ep, configFilePath); err != nil {
		log.Errorf("生成 Milky 启动配置失败: %s", err)
		ep.State = StateConnectionFailed
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return err
	}
	command := fmt.Sprintf(`"%s"`, milkyExePath)
	p := procs.NewProcess(command)
	p.Dir = workDir
	p.Env = []string{
		fmt.Sprintf("APP_LAUNCHER_SIG=%s", BuildSignature(uint64(uin))),
	}
	chQrCode := make(chan int, 1)
	qrSignalCalled := atomic.Bool{}
	qrSignalCalled.Store(false)
	pa.BuiltInLoginState = MilkyLoginStateInit
	p.OutputHandler = func(line string, _type string) string {
		// 登录中
		if pa.BuiltInLoginState < MilkyLoginStateConnecting {
			var qrcodeSignal string
			var onlineSignal string
			var qrcodeExpiredSignal string
			switch pa.BuiltInMode {
			case "lagrangeV2":
				qrcodeSignal = "Fetch QrCode Success"
				onlineSignal = "successfully logged in"
				qrcodeExpiredSignal = "QrCode State: CodeExpired"
			case "yogurt":
				qrcodeSignal = "二维码文件已保存"
				onlineSignal = "已上线"
				qrcodeExpiredSignal = "二维码已过期"
			}

			// 读取二维码
			if strings.Contains(line, qrcodeSignal) && !qrSignalCalled.Load() {
				qrSignalCalled.Store(true)
				chQrCode <- 1
			}

			// 登录成功
			if strings.Contains(line, onlineSignal) {
				pa.BuiltInLoginState = MilkyLoginStateQRConnected
				log.Infof("Milky 登录成功，账号：<%s>(%s)", ep.Nickname, ep.UserID)
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)

				// 经测试，若不延时，登录成功的同一时刻进行ws正向连接有几率导致第一次连接失败
				time.Sleep(1 * time.Second)
				go doServe()
			}

			if strings.Contains(line, qrcodeExpiredSignal) {
				// 二维码过期，登录失败，杀掉进程
				pa.BuiltInLoginState = MilkyLoginStateFailed
				log.Infof("Milky 二维码过期，登录失败，账号：%s", ep.UserID)
				BuiltinMilkyClientKill(d, ep)
				if reporter != nil {
					reporter.Failed(errors.New("milky qrcode expired"))
				}
			}
		}

		if _type == "stderr" {
			log.Error("Milky Internal: ", strings.TrimSpace(line))
		} else {
			if ep.State != 1 {
				log.Info("Milky Internal: ", strings.TrimSpace(line))
			} else {
				log.Debug("Milky Internal: ", strings.TrimSpace(line))
			}
		}

		return ""
	}

	go func() {
		<-chQrCode
		time.Sleep(3 * time.Second)
		if _, err := os.Stat(qrcodeFilePath); err == nil {
			log.Info("Milky 二维码已就绪")
			qrdata, err := os.ReadFile(qrcodeFilePath)
			if err == nil {
				pa.BuiltInLoginState = MilkyLoginStateQRWaitingForScan
				pa.QrCodeData = qrdata
				log.Info("Milky 读取二维码成功")
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
			} else {
				pa.BuiltInLoginState = MilkyLoginStateFailed
				pa.QrCodeData = nil
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				log.Infof("Milky 读取二维码失败：%s", err)
			}
		}
	}()

	if err := p.Start(); err != nil {
		log.Info("Milky 进程启动失败: ", err)
		ep.State = StateConnectionFailed
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return err
	}
	done := make(chan struct{})
	pa.MilkyProcess = p
	pa.builtInProcessDone = done
	go waitMilkyBuiltIn(d, pa, p, done, reporter, ctx)

	if d.Parent.progressExitGroupWin != 0 && p.Cmd != nil {
		if err := d.Parent.progressExitGroupWin.AddProcess(p.Cmd.Process); err != nil {
			log.Warn("添加到进程组失败，若主进程崩溃，Milky 进程可能需要手动结束")
		}
	}

	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return nil
}

func prepareMilkyBuiltInConfig(ep *EndPointInfo, configFilePath string) error {
	port, err := GetRandomFreePort()
	if err != nil {
		return fmt.Errorf("获取随机端口失败: %w", err)
	}
	accessToken := uuid.NewString()
	config := GenerateMilkyConfig(port, SealSignV3Url, accessToken, ep)
	if len(config) == 0 {
		return errors.New("不支持的内置 Milky 模式")
	}
	if err := os.WriteFile(configFilePath, config, 0o644); err != nil {
		return fmt.Errorf("写入 Milky 配置文件失败: %w", err)
	}

	pa := ep.Adapter.(*PlatformAdapterMilky)
	pa.WsGateway = fmt.Sprintf("ws://127.0.0.1:%d/event", port)
	pa.RestGateway = fmt.Sprintf("http://127.0.0.1:%d/api", port)
	pa.Token = accessToken
	return nil
}

func stopMilkyBuiltInLocked(pa *PlatformAdapterMilky) error {
	process := pa.MilkyProcess
	if process == nil {
		return nil
	}
	done := pa.builtInProcessDone
	stopErr := process.Stop()
	if done == nil {
		return errors.New("milky 进程缺少退出通知")
	}

	select {
	case <-done:
		if pa.MilkyProcess == process {
			pa.MilkyProcess = nil
			pa.builtInProcessDone = nil
		}
		return nil
	case <-time.After(5 * time.Second):
		if stopErr != nil {
			return fmt.Errorf("结束进程失败: %w", stopErr)
		}
		return errors.New("milky 进程未能在 5 秒内退出，可能需要手动结束")
	}
}

func waitMilkyBuiltIn(d *Dice, pa *PlatformAdapterMilky, process *procs.Process, done chan struct{}, reporter EndpointRunReporter, ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			d.Logger.Errorf("MilkyInternal 异常: %v 堆栈: %v", r, string(debug.Stack()))
		}
		close(done)

		pa.builtInProcessMu.Lock()
		defer pa.builtInProcessMu.Unlock()
		if pa.MilkyProcess == process {
			pa.MilkyProcess = nil
			pa.builtInProcessDone = nil
		}
	}()

	err := process.Wait()
	if err != nil {
		d.Logger.Info("Milky 进程异常退出: ", err)
		if reporter != nil && ctx.Err() == nil {
			reporter.Closed(err)
		}
	} else {
		d.Logger.Info("Milky 进程退出")
		if reporter != nil && ctx.Err() == nil {
			reporter.Closed(nil)
		}
	}
}

// GenerateMilkyConfig 似乎暂时不需要 APPInfo, 如果以后需要了再改成双返回值
func GenerateMilkyConfig(port int, signServerUrl string, accessToken string, info *EndPointInfo) []byte {
	pa := info.Adapter.(*PlatformAdapterMilky)
	switch pa.BuiltInMode {
	case "lagrangeV2":
		conf := strings.ReplaceAll(defaultLagrangeV2Config, "{WS端口}", strconv.Itoa(port))
		conf = strings.ReplaceAll(conf, "{NTSignServer地址}", signServerUrl)
		conf = strings.ReplaceAll(conf, "{账号UIN}", info.UserID[3:])
		conf = strings.ReplaceAll(conf, "{访问Token}", accessToken)
		return []byte(conf)
	case "yogurt":
		conf := strings.ReplaceAll(defaultYogurtConfig, "{WS端口}", strconv.Itoa(port))
		conf = strings.ReplaceAll(conf, "{NTSignServer地址}", signServerUrl)
		conf = strings.ReplaceAll(conf, "{账号UIN}", info.UserID[3:])
		conf = strings.ReplaceAll(conf, "{访问Token}", accessToken)
		return []byte(conf)
	default:
		return nil
	}
}

func findKeystoreFiles(root string) ([]string, error) {
	var matches []string
	re := regexp.MustCompile(`^\d+\.keystore$`)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && re.MatchString(d.Name()) {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

func MilkyRemoveSession(dice *Dice, conn *EndPointInfo) {
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	keyStores, err := findKeystoreFiles(workDir)
	if err != nil {
		dice.Logger.Errorf("查找 keystore 文件失败: %v", err)
	}
	for _, file := range keyStores {
		if _, err := os.Stat(file); err == nil {
			_ = os.Remove(file)
		}
	}
}
