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
		conn := ep.Adapter.(*PlatformAdapterMilky)
		conn.EndPoint = ep
		conn.Session = d.ImSession
		ep.Session = d.ImSession
		d.Logger.Infof("Milky 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接Milky失败")
			ep.State = 3
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}

func BuiltinMilkyClientKill(dice *Dice, conn *EndPointInfo) {
	defer func() {
		if r := recover(); r != nil {
			dice.Logger.Error("内置 Milky 客户端清理报错: ", r)
		}
	}()
	pa, ok := conn.Adapter.(*PlatformAdapterMilky)
	if !ok {
		return
	}
	if pa.BuiltInMode == "" {
		return
	}
	defer func() {
		pa.MilkyProcess = nil
	}()
	if pa.MilkyProcess != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		go func() {
			<-ctx.Done()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				dice.Logger.Error("Milky 进程未能在 5 秒内退出，可能需要手动结束")
			}
		}()
		err := pa.MilkyProcess.Stop()
		if err != nil {
			dice.Logger.Error("停止 Milky 进程失败: ", err)
		}
		_ = pa.MilkyProcess.Wait()
	}
}

func ServeMilkyBuiltIn(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if d.ContainerMode {
		d.Logger.Warnf("当前处于容器模式，Milky 内置版本不可用")
		ep.State = 3
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return
	}
	uin, err := strconv.ParseInt(ExtractQQUserID(ep.UserID), 10, 64)
	if err != nil {
		d.Logger.Errorf("解析QQ号失败: %s", ep.UserID)
		ep.State = 3
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return
	}
	conn := ep.Adapter.(*PlatformAdapterMilky)
	doServe := func() {
		if ep.Platform == "QQ" {
			d.Logger.Infof("Milky 尝试连接")
			if conn.Serve() != 0 {
				d.Logger.Errorf("连接Milky失败")
				ep.State = 3
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				BuiltinMilkyClientKill(d, ep)
			}
		}
	}
	pa := conn
	conn.EndPoint = ep
	conn.Session = d.ImSession
	ep.Session = d.ImSession
	log := zap.S().Named(logger.LogKeyAdapter)

	workDir := filepath.Join(d.BaseConfig.DataDir, ep.RelWorkDir)
	diceWorkdir, _ := os.Getwd()
	milkyExePath, _ := filepath.Abs(filepath.Join(diceWorkdir, fmt.Sprintf("milky/%s", pa.BuiltInMode)))
	configFilePath := filepath.Join(workDir, "appsettings.jsonc")
	qrcodeFilePath := filepath.Join(workDir, "qrcode.png")
	milkyExePath = filepath.ToSlash(milkyExePath) // windows平台需要这个替换
	if runtime.GOOS == "windows" {
		milkyExePath += ".exe" //nolint:ineffassign
	}
	_ = os.MkdirAll(workDir, 0o755)
	_ = os.Chmod(milkyExePath, 0o755)
	if pa.MilkyProcess != nil {
		BuiltinMilkyClientKill(d, ep)
	}
	if pa.WsGateway == "" {
		p, err := GetRandomFreePort()
		if err != nil {
			log.Errorf("获取随机端口失败: %s", err)
			ep.State = 3
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
			return
		}
		pa.WsGateway = fmt.Sprintf("ws://127.0.0.1:%d/event", p)
		pa.RestGateway = fmt.Sprintf("http://127.0.0.1:%d/api", p)
		// 生成配置写入文件
		accessToken := uuid.NewString()
		pa.Token = accessToken
		c := GenerateMilkyConfig(p, SealSignV3Url, accessToken, ep)
		_ = os.WriteFile(configFilePath, c, 0o644)
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
			qrcodeSignal := "Fetch QrCode Success"
			onlineSignal := "successfully logged in"
			qrcodeExpiredSignal := "QrCode State: 17"
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

	run := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("MilkyInternal 异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		conn.MilkyProcess = p
		// processStartTime := time.Now().Unix()
		errRun := p.Start()

		if errRun == nil {
			if d.Parent.progressExitGroupWin != 0 && p.Cmd != nil {
				errAdd := d.Parent.progressExitGroupWin.AddProcess(p.Cmd.Process)
				if errAdd != nil {
					log.Warn("添加到进程组失败，若主进程崩溃，Milky 进程可能需要手动结束")
				}
			}
			errRun = p.Wait() //nolint:ineffassign
		}

		if errRun != nil {
			log.Info("Milky 进程异常退出: ", errRun)
			// Maybe some state change here
		} else {
			log.Info("Milky 进程退出")
		}
	}

	go run()
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
