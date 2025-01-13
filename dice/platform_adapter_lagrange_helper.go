package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	log "sealdice-core/utils/kratos"
	"sealdice-core/utils/procs"
)

type LagrangeLoginInfo struct {
	UIN               int64
	SignServerName    string
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
		appinfoFilePath := filepath.Join(workDir, "appinfo.json")

		if pa.BuiltinMode == "lagrange-gocq" {
			exeFilePath, _ = filepath.Abs(filepath.Join(wd, "lagrange/go-cqhttp"))
			qrcodeFilePath = filepath.Join(workDir, "qrcode.png")
			configFilePath = filepath.Join(workDir, "config.yml")
			appinfoFilePath = filepath.Join(workDir, "data/versions/7.json")
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
			// 这里是为了防止用户手动删除配置，但数据库里还存有账号
			if loginInfo.SignServerName == "" {
				loginInfo.SignServerName = pa.SignServerName
			}
			if loginInfo.SignServerVersion == "" {
				loginInfo.SignServerVersion = pa.SignServerVer
			}
			// 生成appinfo和signserverurl写入文件
			a, c := GenerateLagrangeConfig(p, loginInfo.SignServerName, loginInfo.SignServerVersion, dice, conn)
			if a != nil {
				dir := filepath.Dir(appinfoFilePath)
				if _, err := os.Stat(dir); err != nil {
					_ = os.MkdirAll(dir, 0o755)
				}
				_ = os.WriteFile(appinfoFilePath, a, 0o644)
			}
			_ = os.WriteFile(configFilePath, c, 0o644)
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
				if strings.Contains(line, "Success") || strings.Contains(line, onlineSignal) || strings.Contains(line, "Bot Uin:") {
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

// 在构建时注入
// var defaultNTSignServer = `https://lwxmagic.sealdice.com/api/sign`
// var lagrangeNTSignServer = "https://sign.lagrangecore.org/api/sign"

func GenerateLagrangeConfig(port int, signServerName string, signServerVersion string, dice *Dice, info *EndPointInfo) ([]byte, []byte) {
	var appinfo []byte
	var signServerUrl string
	pa := info.Adapter.(*PlatformAdapterGocq)
	if signServerVersion == "自定义" {
		appinfo, _ = lagrangeGetAppinfoFromSignServer(signServerName)
		signServerUrl = signServerName
	} else {
		if len(signInfoGlobal) == 0 {
			_, _ = LagrangeGetSignInfo(dice)
		}
		appinfo, signServerUrl = lagrangeGetSignSeverFromInfo(signServerVersion, signServerName)
	}
	conf := strings.ReplaceAll(defaultLagrangeConfig, "{WS端口}", strconv.Itoa(port))
	if pa.BuiltinMode == "lagrange-gocq" {
		conf = strings.ReplaceAll(defaultLagrangeGocqConfig, "{WS端口}", strconv.Itoa(port))
	}
	conf = strings.ReplaceAll(conf, "{NTSignServer地址}", signServerUrl)
	conf = strings.ReplaceAll(conf, "{账号UIN}", info.UserID[3:])
	return appinfo, []byte(conf)
}

// 该函数后续考虑优化掉
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

// 云端SignInfo.Servers结构
type SignServerInfo struct {
	Name     string `json:"name"`
	Url      string `json:"url"`
	Latency  int    `json:"latency"`
	Selected bool   `json:"selected"`
	Ignored  bool   `json:"ignored"`
	Note     string `json:"note"`
}

// 云端SignInfo结构
type SignInfo struct {
	Version  string                 `json:"version"`
	Appinfo  map[string]interface{} `json:"appinfo"`
	Servers  []*SignServerInfo      `json:"servers"`
	Selected bool                   `json:"selected"`
	Ignored  bool                   `json:"ignored"`
	Note     string                 `json:"note"`
}

// 小概率出现并发读写，需上锁
var mu sync.Mutex
var signInfoGlobal []SignInfo

func LagrangeGetSignInfo(dice *Dice) ([]SignInfo, error) {
	mu.Lock()
	defer mu.Unlock()
	cachePath := filepath.Join(dice.BaseConfig.DataDir, "extra/SignInfo.cache")
	signInfo, err := lagrangeGetSignInfoFromCloud(cachePath)
	if err == nil && len(signInfo) > 0 {
		signInfoGlobal = append([]SignInfo(nil), signInfo...)
		return signInfo, nil
	}
	dice.Logger.Infof("无法从云端获取SignInfo，即将读取本地缓存数据, 原因: %s", err.Error())

	signInfo, err = lagrangeGetSignInfoFromCache(cachePath)
	if err == nil && len(signInfo) > 0 {
		signInfoGlobal = append([]SignInfo(nil), signInfo...)
		return signInfo, nil
	}
	dice.Logger.Infof("无法从本地缓存获取SignInfo，即将读取内置数据, 原因: %s", err.Error())

	if err = json.Unmarshal([]byte(signInfoJson), &signInfo); err == nil {
		lagrangeGetSignServerLatency(signInfo)
		signInfoGlobal = append([]SignInfo(nil), signInfo...)
		return signInfo, nil
	}
	dice.Logger.Infof("无法从内置数据获取SignInfo，请联系开发者上报问题, 原因: %s", err.Error())
	return nil, errors.New("内置SignInfo信息有误")
}

func lagrangeGetSignInfoFromCloud(cachePath string) ([]SignInfo, error) {
	now := time.Now()
	unixTimestamp := now.Unix()
	url := fmt.Sprintf("https://d1.sealdice.com/sealsign/signinfo.json?v=%v", unixTimestamp)
	c := http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var signInfo []SignInfo
	err = json.Unmarshal(body, &signInfo)
	if err != nil {
		return nil, err
	}
	_ = os.WriteFile(cachePath, body, 0o644)
	lagrangeGetSignServerLatency(signInfo)
	return signInfo, nil
}

func lagrangeGetSignInfoFromCache(cachePath string) ([]SignInfo, error) {
	var err error
	if _, err = os.Stat(cachePath); err == nil {
		var file []byte
		if file, err = os.ReadFile(cachePath); err == nil {
			var signInfo []SignInfo
			if err = json.Unmarshal(file, &signInfo); err == nil {
				lagrangeGetSignServerLatency(signInfo)
				return signInfo, nil
			}
		}
	}
	return nil, err
}

func lagrangeGetSignSeverFromInfo(serverVer string, serverName string) ([]byte, string) {
	mu.Lock()
	defer mu.Unlock()
	for _, info := range signInfoGlobal {
		if info.Version == serverVer {
			for _, server := range info.Servers {
				if server.Name == serverName {
					if appinfo, err := json.Marshal(info.Appinfo); err == nil {
						return appinfo, server.Url
					}
				}
			}
		}
	}
	return nil, ""
}

func lagrangeGetSignServerLatency(signInfo []SignInfo) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	c := &http.Client{
		Timeout: 3 * time.Second,
	}
	for _, si := range signInfo {
		for _, server := range si.Servers {
			wg.Add(1)
			go func(server *SignServerInfo) {
				defer wg.Done()
				latency := testLatency(c, server.Url)
				mu.Lock()
				server.Latency = latency
				mu.Unlock()
			}(server)
		}
	}
	wg.Wait()
}

func testLatency(c *http.Client, url string) int {
	start := time.Now()
	resp, err := c.Get(url)
	if err != nil {
		return 999
	}
	defer resp.Body.Close()
	duration := time.Since(start)
	return int(duration.Milliseconds())
}

// 当自定义签名地址时，从/appinfo路径获取appinfo信息
func lagrangeGetAppinfoFromSignServer(serverName string) ([]byte, error) {
	c := http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := c.Get(serverName + "/appinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var test map[string]interface{}
	err = json.Unmarshal(body, &test)
	if err != nil {
		return nil, err
	}
	return body, nil
}

var signInfoJson string = `
[
  {
    "version": "25765",
    "appinfo": {
      "AppClientVersion": 25765,
      "AppId": 1600001615,
      "AppIdQrCode": 13697054,
      "CurrentVersion": "3.2.10-25765",
      "Kernel": "Linux",
      "MainSigMap": 169742560,
      "MiscBitmap": 32764,
      "NTLoginType": 1,
      "Os": "Linux",
      "PackageName": "com.tencent.qq",
      "PtVersion": "2.0.0",
      "SsoVersion": 19,
      "SubAppId": 537234773,
      "SubSigMap": 0,
      "VendorOs": "linux",
      "WtLoginSdk": "nt.wtlogin.0.0.1"
    },
    "servers": [
      {
        "name": "海豹",
        "url": "https://lwxmagic.sealdice.com/api/sign/25765"
      },
	  {
        "name": "Lagrange",
        "url": "https://sign.lagrangecore.org/api/sign/25765"
      }
    ]
  },
  {
    "version": "30366",
    "appinfo": {
      "AppClientVersion": 30366,
      "AppId": 1600001615,
      "AppIdQrCode": 13697054,
      "CurrentVersion": "3.2.15-30366",
      "Kernel": "Linux",
      "MainSigMap": 169742560,
      "MiscBitmap": 32764,
      "NTLoginType": 1,
      "Os": "Linux",
      "PackageName": "com.tencent.qq",
      "PtVersion": "2.0.0",
      "SsoVersion": 19,
      "SubAppId": 537258424,
      "SubSigMap": 0,
      "VendorOs": "linux",
      "WtLoginSdk": "nt.wtlogin.0.0.1"
    },
    "servers": [
      {
        "name": "海豹",
        "url": "https://lwxmagic.sealdice.com/api/sign/30366",
		"selected": true,
		"note": "部分地区用户可能无法连接"
      },
	  {
        "name": "Lagrange",
        "url": "https://sign.lagrangecore.org/api/sign/30366"
      }
    ],
    "selected": true
  }
]
	`

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
