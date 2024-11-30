package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	log "sealdice-core/utils/kratos"
	"sealdice-core/utils/procs"
)

type LagrangeLoginInfo struct {
	UIN               int64
	SignServerUrl     string
	SignServerVersion string
	IsAsyncRun        bool
}

func lagrangeGetWorkDir(dice *Dice, conn *EndPointInfo) string {
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	return workDir
}

func NewLagrangeConnectInfoItem(account string, isGocq bool) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "onebot"
	conn.Enable = false
	conn.RelWorkDir = "extra/lagrange-qq" + account
	conn.Adapter = &PlatformAdapterGocq{
		EndPoint:        conn,
		UseInPackClient: true,
		BuiltinMode:     "lagrange",
	}

	if isGocq {
		conn.RelWorkDir = "extra/lagrange-gocq-qq" + account
		conn.Adapter.(*PlatformAdapterGocq).BuiltinMode = "lagrange-gocq"
	}
	return conn
}

func LagrangeServe(dice *Dice, conn *EndPointInfo, loginInfo LagrangeLoginInfo) {
	pa := conn.Adapter.(*PlatformAdapterGocq)

	pa.CurLoginIndex++
	loginIndex := pa.CurLoginIndex
	pa.GoCqhttpState = StateCodeInLogin

	if pa.UseInPackClient && (pa.BuiltinMode == "lagrange" || pa.BuiltinMode == "lagrange-gocq") { //nolint:nestif
		helper := log.NewCustomHelper(log.LOG_LAGR, false, nil)

		if dice.ContainerMode {
			if pa.BuiltinMode == "lagrange" {
				helper.Warn("onebot: 尝试启动内置客户端，但内置客户端在容器模式下被禁用")
			} else {
				helper.Warn("onebot: 尝试启动内置gocq，但内置gocq在容器模式下被禁用")
			}
			conn.State = 3
			pa.GoCqhttpState = StateCodeLoginFailed
			dice.Save(false)
			return
		}

		workDir := lagrangeGetWorkDir(dice, conn)
		_ = os.MkdirAll(workDir, 0o755)
		wd, _ := os.Getwd()
		exeFilePath, _ := filepath.Abs(filepath.Join(wd, "lagrange/Lagrange.OneBot"))
		qrcodeFilePath := filepath.Join(workDir, fmt.Sprintf("qr-%s.png", conn.UserID[3:]))
		configFilePath := filepath.Join(workDir, "appsettings.json")

		if pa.BuiltinMode == "lagrange-gocq" {
			exeFilePath, _ = filepath.Abs(filepath.Join(wd, "lagrange/go-cqhttp"))
			qrcodeFilePath = filepath.Join(workDir, "qrcode.png")
			configFilePath = filepath.Join(workDir, "config.yml")
		}

		exeFilePath = filepath.ToSlash(exeFilePath) // windows平台需要这个替换
		if runtime.GOOS == "windows" {
			exeFilePath += ".exe"
		}

		if _, err := os.Stat(qrcodeFilePath); err == nil {
			// 如果已经存在二维码文件，将其删除
			_ = os.Remove(qrcodeFilePath)
		}
		helper.Info("onebot: 删除已存在的二维码文件")

		// 创建配置文件
		pa.ConnectURL = ""
		if file, err := os.ReadFile(configFilePath); err == nil {
			var result map[string]interface{}
			if pa.BuiltinMode == "lagrange" {
				if err := json.Unmarshal(file, &result); err == nil {
					if val, ok := result["Implementations"].([]interface{})[0].(map[string]interface{})["Port"].(float64); ok {
						pa.ConnectURL = fmt.Sprintf("ws://127.0.0.1:%d", int(val))
					}
				}
			} else {
				if err := yaml.Unmarshal(file, &result); err == nil {
					if val, ok := result["servers"].([]interface{})[0].(map[string]interface{})["ws"].(map[string]interface{})["address"].(string); ok {
						pa.ConnectURL = fmt.Sprintf("ws://%s", val)
					}
				}
			}
		}
		if pa.ConnectURL == "" {
			p, _ := GetRandomFreePort()
			pa.ConnectURL = fmt.Sprintf("ws://127.0.0.1:%d", p)
			c := GenerateLagrangeConfig(p, loginInfo.SignServerUrl, loginInfo.SignServerVersion, conn)
			_ = os.WriteFile(configFilePath, []byte(c), 0o644)
		}

		if pa.GoCqhttpProcess != nil {
			// 如果有正在运行的lagrange，先将其杀掉
			BuiltinQQServeProcessKill(dice, conn)
		}

		// 启动客户端
		_ = os.Chmod(exeFilePath, 0o755)

		command := ""
		if runtime.GOOS == "android" {
			for i, s := range os.Environ() {
				if strings.HasPrefix(s, "RUNNER_PATH=") {
					helper.Infof("RUNNER_PATH: %s", os.Environ()[i][12:])
					command = os.Environ()[i][12:]
					break
				}
			}
			command, _ = filepath.Abs(filepath.Join(command, "start.sh"))
			_ = os.Chmod(command, 0o755)
			command += " " + exeFilePath
		} else {
			command = fmt.Sprintf(`"%s"`, exeFilePath)
		}
		helper.Info("onebot: 正在启动 onebot 客户端…… ", command)
		conn.State = 2
		conn.Enable = true
		p := procs.NewProcess(command)
		p.Dir = workDir

		chQrCode := make(chan int, 1)
		isSelfKilling := false
		isPrintLog := true

		regFatal := regexp.MustCompile(`\s*\[\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\]\s+\[[^]]+\]\s+\[(FATAL)\]:`)

		p.OutputHandler = func(line string, _type string) string {
			if loginIndex != pa.CurLoginIndex {
				// 当前连接已经无用，进程自杀
				if !isSelfKilling {
					helper.Infof("检测到新的连接序号 %d，当前连接 %d 将自动退出", pa.CurLoginIndex, loginIndex)
					// 注: 这里不要调用kill
					isSelfKilling = true
					_ = p.Stop()
				}
				return ""
			}

			// 登录中
			if pa.IsInLogin() {
				qrcodeSignal := "QrCode Fetched"
				onlineSignal := "Bot Online: "
				qrcodeExpiredSignal := "QrCode Expired, Please Fetch QrCode Again"
				if pa.BuiltinMode == "lagrange-gocq" {
					qrcodeSignal = "请使用手机QQ扫描二维码"
					onlineSignal = "登录成功"
					qrcodeExpiredSignal = "二维码过期"
				}
				// 读取二维码
				if strings.Contains(line, qrcodeSignal) {
					chQrCode <- 1
				}

				// 登录成功
				if strings.Contains(line, "Success") || strings.Contains(line, onlineSignal) {
					pa.GoCqhttpState = StateCodeLoginSuccessed
					pa.GoCqhttpLoginSucceeded = true
					helper.Infof("onebot: 登录成功，账号：<%s>(%s)", conn.Nickname, conn.UserID)
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)
					isPrintLog = false

					// 经测试，若不延时，登录成功的同一时刻进行ws正向连接有几率导致第一次连接失败
					time.Sleep(1 * time.Second)
					go ServeQQ(dice, conn)
				}

				if strings.Contains(line, qrcodeExpiredSignal) {
					// 二维码过期，登录失败，杀掉进程
					pa.GoCqhttpState = StateCodeLoginFailed
					helper.Infof("onebot: 二维码过期，登录失败，账号：%s", conn.UserID)
					BuiltinQQServeProcessKill(dice, conn)
				}
			}

			if _type == "stderr" {
				helper.Error("onebot | ", line)
			} else {
				isPrint := isPrintLog || pa.ForcePrintLog || strings.HasPrefix(line, "warn:")
				if isPrint {
					helper.Warn("onebot | ", line)
				}
				if regFatal.MatchString(line) {
					helper.Error("onebot | ", line)
				}
			}

			return ""
		}

		go func() {
			<-chQrCode
			time.Sleep(3 * time.Second)
			if _, err := os.Stat(qrcodeFilePath); err == nil {
				helper.Info("onebot: 二维码已就绪")
				qrdata, err := os.ReadFile(qrcodeFilePath)
				if err == nil {
					pa.GoCqhttpState = StateCodeInLoginQrCode
					pa.GoCqhttpQrcodeData = qrdata
					helper.Info("onebot: 读取二维码成功")
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)
				} else {
					pa.GoCqhttpState = StateCodeLoginFailed
					pa.GoCqhttpQrcodeData = nil
					pa.GocqhttpLoginFailedReason = "读取二维码失败"
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)
					helper.Infof("onebot: 读取二维码失败：%s", err)
				}
			}
		}()

		run := func() {
			defer func() {
				if r := recover(); r != nil {
					helper.Errorf("onebot: 异常: %v 堆栈: %v", r, string(debug.Stack()))
				}
			}()

			pa.GoCqhttpProcess = p
			processStartTime := time.Now().Unix()
			err := p.Start()

			if err == nil {
				if dice.Parent.progressExitGroupWin != 0 && p.Cmd != nil {
					errAdd := dice.Parent.progressExitGroupWin.AddProcess(p.Cmd.Process)
					if errAdd != nil {
						dice.Logger.Warn("添加到进程组失败，若主进程崩溃，lagrange 进程可能需要手动结束")
					}
				}
				err = p.Wait()
			}

			isInLogin := pa.IsInLogin()
			if isInLogin {
				conn.State = 3
				pa.GoCqhttpState = StateCodeLoginFailed
			} else {
				conn.State = 0
				pa.GoCqhttpState = GoCqhttpStateCodeClosed
			}

			if err != nil {
				helper.Info("lagrange 进程异常退出: ", err)
				pa.GoCqhttpState = StateCodeLoginFailed

				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					code := exitErr.ExitCode()
					switch code {
					case 137:
						// Failed to create CoreCLR, HRESULT: 0x8007054F
						// +++ exited with 137 +++
						helper.Info("你的设备尚未被支持，请等待后续更新。")
					case 134:
						// Resource temporarily unavailable
						// System.Net.Dns.GetHostEntryOrAddressesCore(String hostName, Boolean justAddresses, AddressFamily addressFamily, Int64 startingTimestamp)
						helper.Info("当前网络无法进行域名解析，请更换网络。")
					default:
						if time.Now().Unix()-processStartTime < 10 {
							helper.Info("进程在启动后10秒内即退出，请检查配置是否正确")
						} else {
							if pa.lagrangeRebootTimes > 5 {
								helper.Info("自动重启次数达到上限，放弃")
							} else {
								pa.lagrangeRebootTimes++
								if conn.Enable {
									helper.Info("5秒后，尝试对其进行重启")
									time.Sleep(5 * time.Second)
								}
								if conn.Enable {
									LagrangeServe(dice, conn, loginInfo)
								}
							}
						}
					}
				}
			} else {
				helper.Info("lagrange 进程退出")
			}
		}

		if loginInfo.IsAsyncRun {
			go run()
		} else {
			run()
		}
	} else if !pa.UseInPackClient {
		pa.GoCqhttpState = StateCodeLoginSuccessed
		pa.GoCqhttpLoginSucceeded = true
		dice.Save(false)
		go ServeQQ(dice, conn)
	}
}

var defaultLagrangeConfig = `
{
    "Logging": {
        "LogLevel": {
            "Default": "Information",
            "Microsoft": "Warning",
            "Microsoft.Hosting.Lifetime": "Information"
        }
    },
    "SignServerUrl": "{NTSignServer地址}",
    "Account": {
        "Uin": {账号UIN},
        "Password": "",
        "Protocol": "Linux",
        "AutoReconnect": true,
        "GetOptimumServer": true
    },
    "Message": {
        "IgnoreSelf": true,
        "StringPost": false
    },
    "QrCode": {
        "ConsoleCompatibilityMode": false
    },
    "Implementations": [
        {
            "Type": "ForwardWebSocket",
            "Host": "127.0.0.1",
            "Port": {WS端口},
            "HeartBeatInterval": 5000,
            "AccessToken": ""
        }
    ]
}
`
var defaultLagrangeGocqConfig = `
account:
  uin: {账号UIN}
  password: ''
  encrypt: false
  status: 0
  relogin:
    delay: 3
    interval: 3
    max-times: 0 
  use-sso-address: true
  allow-temp-session: false
  sign-servers:
    - url: '{NTSignServer地址}'
  max-check-count: 0
  sign-server-timeout: 60

heartbeat:
  interval: 5

message:
  post-format: array
  ignore-invalid-cqcode: false
  force-fragment: false
  fix-url: false
  proxy-rewrite: ''
  report-self-message: false
  remove-reply-at: false
  extra-reply-data: false
  skip-mime-scan: false
  convert-webp-image: false
  http-timeout: 15

output:
  log-level: warn
  log-aging: 15
  log-force-new: true
  log-colorful: true
  debug: false

default-middlewares: &default
  access-token: ''
  filter: ''
  rate-limit:
    enabled: false
    frequency: 1
    bucket: 1

database:
  leveldb:
    enable: true
  sqlite3:
    enable: false
    cachettl: 3600000000000

servers:
  - ws:
      address: 127.0.0.1:{WS端口}
      middlewares:
        <<: *default
`

// 在构建时注入
// var defaultNTSignServer = `https://lwxmagic.sealdice.com/api/sign`
// var lagrangeNTSignServer = "https://sign.lagrangecore.org/api/sign"

// 此处添加内置sign地址及对应标识字符串
var signServers = map[string]string{
	"sealdice": `https://lwxmagic.sealdice.com/api/sign`,
	"lagrange": "https://sign.lagrangecore.org/api/sign",
}

func GenerateLagrangeConfig(port int, signServerUrl string, signServerVersion string, info *EndPointInfo) string {
	if signServerUrl == "" {
		signServerUrl = "sealdice"
	}
	if url, exists := signServers[signServerUrl]; exists {
		signServerUrl = url
		if signServerVersion != "" && signServerVersion != "13107" {
			signServerUrl += "/" + signServerVersion
		}
	}
	pa := info.Adapter.(*PlatformAdapterGocq)
	conf := strings.ReplaceAll(defaultLagrangeConfig, "{WS端口}", strconv.Itoa(port))
	if pa.BuiltinMode == "lagrange-gocq" {
		conf = strings.ReplaceAll(defaultLagrangeGocqConfig, "{WS端口}", strconv.Itoa(port))
	}
	conf = strings.ReplaceAll(conf, "{NTSignServer地址}", signServerUrl)
	conf = strings.ReplaceAll(conf, "{账号UIN}", info.UserID[3:])
	return conf
}

func LagrangeServeRemoveSession(dice *Dice, conn *EndPointInfo) {
	workDir := gocqGetWorkDir(dice, conn)
	file := filepath.Join(workDir, "keystore.json")
	pa := conn.Adapter.(*PlatformAdapterGocq)
	if pa.BuiltinMode == "lagrange-gocq" {
		file = filepath.Join(workDir, "session.token")
	}
	if _, err := os.Stat(file); err == nil {
		_ = os.Remove(file)
	}
}

// 清理内置客户端配置文件目录
func LagrangeServeRemoveConfig(dice *Dice, conn *EndPointInfo) {
	workDir := lagrangeGetWorkDir(dice, conn)
	err := os.RemoveAll(workDir)
	if err != nil {
		dice.Logger.Errorf("清理内置客户端文件失败, 原因: %s, 请手动删除目录: %s", err.Error(), workDir)
	} else {
		dice.Logger.Infof("已自动清理内置客户端目录: %s", workDir)
	}
}

func RWLagrangeSignServerUrl(dice *Dice, conn *EndPointInfo, signServerUrl string, w bool, signServerVersion string) (string, string) {
	if signServerUrl == "" {
		signServerUrl = "sealdice"
	}
	if url, exists := signServers[signServerUrl]; exists {
		signServerUrl = url
		if signServerVersion != "" && signServerVersion != "13107" {
			signServerUrl += "/" + signServerVersion
		}
	}
	workDir := lagrangeGetWorkDir(dice, conn)
	configFilePath := filepath.Join(workDir, "appsettings.json")
	pa := conn.Adapter.(*PlatformAdapterGocq)

	if pa.BuiltinMode == "lagrange-gocq" {
		configFilePath = filepath.Join(workDir, "config.yml")
	}

	currentSignServerUrl := ""
	file, err := os.ReadFile(configFilePath)
	if err != nil {
		dice.Logger.Infof("读取内置客户端配置失败，账号：%s, 原因: %s", conn.UserID, err.Error())
		return "", ""
	}

	var result map[string]interface{}
	if pa.BuiltinMode == "lagrange" {
		err = json.Unmarshal(file, &result)
		if err != nil {
			dice.Logger.Infof("读取内置客户端配置失败，账号：%s, 原因: %s", conn.UserID, err.Error())
			return "", ""
		}
		if val, ok := result["SignServerUrl"].(string); ok {
			currentSignServerUrl = val
			if w {
				result["SignServerUrl"] = signServerUrl
				var c []byte
				c, err = json.MarshalIndent(result, "", "    ")
				if err != nil {
					dice.Logger.Infof("SignServerUrl字段无法正常覆写，账号：%s, 原因: %s", conn.UserID, err.Error())
				}
				_ = os.WriteFile(configFilePath, c, 0o644)
			}
		}
	} else {
		err = yaml.Unmarshal(file, &result)
		if err != nil {
			dice.Logger.Infof("读取内置gocq配置失败，账号：%s, 原因: %s", conn.UserID, err.Error())
			return "", ""
		}
		if val, ok := result["account"].(map[string]interface{})["sign-servers"].([]interface{})[0].(map[string]interface{})["url"].(string); ok {
			currentSignServerUrl = val
			if w {
				result["account"].(map[string]interface{})["sign-servers"].([]interface{})[0].(map[string]interface{})["url"] = signServerUrl
				var c []byte
				c, err = yaml.Marshal(&result)
				if err != nil {
					dice.Logger.Infof("SignServerUrl字段无法正常覆写，账号：%s, 原因: %s", conn.UserID, err.Error())
				}
				_ = os.WriteFile(configFilePath, c, 0o644)
			}
		}
	}
	if currentSignServerUrl == "" {
		currentSignServerUrl = signServers["sealdice"] + "/25765"
	}
	var version string
	for key, value := range signServers {
		if strings.HasPrefix(currentSignServerUrl, value) {
			version, _ = strings.CutPrefix(currentSignServerUrl, value)
			version, _ = strings.CutPrefix(version, "/")
			currentSignServerUrl = key
			break
		}
	}
	if _, exists := signServers[currentSignServerUrl]; !exists {
		// 此处填写signServer最新版本号，修复前端部分由自定义地址切换至其他选项时无法自动选中sign最新版本
		version = "25765"
	}
	if version == "" {
		// 此处填写sign版本号为空时默认版本号，修复前端部分由于signServerVersion丢失导致13107版本不会处于选中状态
		version = "13107"
	}
	return currentSignServerUrl, version
}
