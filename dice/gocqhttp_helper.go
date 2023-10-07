package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sealdice-core/utils/procs"
	"strings"
	"time"

	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/acarl005/stripansi"
	"github.com/google/uuid"
)

type deviceFile struct {
	Display      string         `json:"display"`
	Product      string         `json:"product"`
	Device       string         `json:"device"`
	Board        string         `json:"board"`
	Model        string         `json:"model"`
	FingerPrint  string         `json:"finger_print"`
	BootId       string         `json:"boot_id"`
	ProcVersion  string         `json:"proc_version"`
	Protocol     int            `json:"protocol"` // 0: iPad 1: Android 2: AndroidWatch  // 3 macOS 4 企点
	IMEI         string         `json:"imei"`
	Brand        string         `json:"brand"`
	Bootloader   string         `json:"bootloader"`
	BaseBand     string         `json:"base_band"`
	SimInfo      string         `json:"sim_info"`
	OSType       string         `json:"os_type"`
	MacAddress   string         `json:"mac_address"`
	IpAddress    []int32        `json:"ip_address"`
	WifiBSSID    string         `json:"wifi_bssid"`
	WifiSSID     string         `json:"wifi_ssid"`
	ImsiMd5      string         `json:"imsi_md5"`
	AndroidId    string         `json:"android_id"`
	APN          string         `json:"apn"`
	VendorName   string         `json:"vendor_name"`
	VendorOSName string         `json:"vendor_os_name"`
	Version      *osVersionFile `json:"version"`
}

type osVersionFile struct {
	Incremental string `json:"incremental"`
	Release     string `json:"release"`
	Codename    string `json:"codename"`
	Sdk         uint32 `json:"sdk"`
}

func randomMacAddress() string {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return "00:16:ea:ae:3c:40"
	}
	// Set the local bit
	buf[0] |= 2
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
}

func RandString(len int) string {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}

//model	设备
//"iPhone11,2"	iPhone XS
//"iPhone11,8"	iPhone XR
//"iPhone12,1"	iPhone 11
//"iPhone13,2"	iPhone 12
//"iPad8,1"	iPad Pro
//"iPad11,2"	iPad mini
//"iPad13,2"	iPad Air 4
//"Apple Watch"	Apple Watch

func GenerateDeviceJsonIOS(protocol int) (string, []byte, error) {
	rand.Seed(time.Now().Unix())
	bootId := uuid.New()
	imei := goluhn.Generate(15) // 注意，这个imei是完全胡乱创建的，并不符合imei规则
	androidId := fmt.Sprintf("%X", rand.Uint64())

	deviceJson := deviceFile{
		Display:      "iPhone",      // Rom的名字 比如 Flyme 1.1.2（魅族rom）  JWR66V（Android nexus系列原生4.3rom）
		Product:      RandString(6), // 产品名，比如这是小米6的代号
		Device:       RandString(6),
		Board:        RandString(6),  // 主板:骁龙835                                                                    //
		Brand:        "Apple",        // 品牌
		Model:        "iPhone13,2",   // 型号
		Bootloader:   "unknown",      // unknown不需要改
		FingerPrint:  RandString(24), // 指纹
		BootId:       bootId.String(),
		ProcVersion:  "1.0", // 很长，后面 builder省略了
		BaseBand:     "",    // 基带版本 4.3CPL2-... 一大堆，直接不写
		SimInfo:      "",
		OSType:       "iOS",
		MacAddress:   randomMacAddress(),
		IpAddress:    []int32{192, 168, rand.Int31() % 255, rand.Int31()%253 + 2}, // 192.168.x.x
		WifiBSSID:    randomMacAddress(),
		WifiSSID:     "<unknown ssid>",
		IMEI:         imei,
		AndroidId:    androidId, // 原版的 androidId和Display内容一样，我没看协议，但是按android文档上说应该是64-bit number的hex，姑且这么做
		APN:          "wifi",
		VendorName:   "Apple", // 这个和下面一个选项(VendorOSName)都属于意义不明，找不到相似对应，不知道是啥
		VendorOSName: "Apple",
		Protocol:     protocol,
		Version: &osVersionFile{
			Incremental: "OCACNFA", // Build.Version.INCREMENTAL, MIUI12: V12.5.3.0.RJBCNXM
			Release:     "11",
			Codename:    "REL",
			Sdk:         29,
		},
	}

	if protocol == 2 {
		deviceJson.Model = "Apple Watch"
	}

	if protocol == 3 {
		deviceJson.Model = "mac OS X"
	}

	a, b := json.Marshal(deviceJson)
	return deviceJson.Model, a, b
}

func GenerateDeviceJsonAndroidWatch(protocol int) (string, []byte, error) {
	rand.Seed(time.Now().Unix())
	bootId := uuid.New()
	imei := goluhn.Generate(15) // 注意，这个imei是完全胡乱创建的，并不符合imei规则
	androidId := fmt.Sprintf("%X", rand.Uint64())

	deviceJson := deviceFile{
		Display:      "MIRAI.142521.001", // Rom的名字 比如 Flyme 1.1.2（魅族rom）  JWR66V（Android nexus系列原生4.3rom）
		Product:      "mirai",            // 产品名，比如这是小米6的代号
		Device:       "mirai",
		Board:        "mirai",                                                           // 主板:骁龙835                                                                    //
		Brand:        "Apple",                                                           // 品牌
		Model:        "mirai",                                                           // 型号
		Bootloader:   "unknown",                                                         // unknown不需要改
		FingerPrint:  "mamoe/mirai/mirai:10/MIRAI.200122.001/9108230:user/release-keys", // 指纹
		BootId:       bootId.String(),
		ProcVersion:  "Linux version 3.0.31-zli0DMkg (android-build@xxx.xxx.xxx.xxx.com)", // 很长，后面 builder省略了
		BaseBand:     "",                                                                  // 基带版本 4.3CPL2-... 一大堆，直接不写
		SimInfo:      "T-Mobile",
		OSType:       "android",
		MacAddress:   randomMacAddress(),
		IpAddress:    []int32{192, 168, rand.Int31() % 255, rand.Int31()%253 + 2}, // 192.168.x.x
		WifiBSSID:    randomMacAddress(),
		WifiSSID:     "<unknown ssid>",
		IMEI:         imei,
		AndroidId:    androidId, // 原版的 androidId和Display内容一样，我没看协议，但是按android文档上说应该是64-bit number的hex，姑且这么做
		APN:          "wifi",
		VendorName:   "MIUI", // 这个和下面一个选项(VendorOSName)都属于意义不明，找不到相似对应，不知道是啥
		VendorOSName: "mirai",
		Protocol:     protocol,
		Version: &osVersionFile{
			Incremental: "5891938", // Build.Version.INCREMENTAL, MIUI12: V12.5.3.0.RJBCNXM
			Release:     "10",
			Codename:    "REL",
			Sdk:         29,
		},
	}

	a, b := json.Marshal(deviceJson)
	return deviceJson.Model, a, b
}

func GenerateDeviceJsonAllRandom(protocol int) (string, []byte, error) {
	rand.Seed(time.Now().Unix())
	bootId := uuid.New()
	imei := goluhn.Generate(15) // 注意，这个imei是完全胡乱创建的，并不符合imei规则
	androidId := fmt.Sprintf("%X", rand.Uint64())

	deviceJson := deviceFile{
		Display:      RandString(6), // Rom的名字 比如 Flyme 1.1.2（魅族rom）  JWR66V（Android nexus系列原生4.3rom）
		Product:      RandString(6), // 产品名，比如这是小米6的代号
		Device:       RandString(6),
		Board:        RandString(6),  // 主板:骁龙835                                                                    //
		Brand:        RandString(12), // 品牌
		Model:        RandString(24), // 型号
		Bootloader:   "unknown",      // unknown不需要改
		FingerPrint:  RandString(24), // 指纹
		BootId:       bootId.String(),
		ProcVersion:  "1.0", // 很长，后面 builder省略了
		BaseBand:     "",    // 基带版本 4.3CPL2-... 一大堆，直接不写
		SimInfo:      "",
		OSType:       "android",
		MacAddress:   randomMacAddress(),
		IpAddress:    []int32{192, 168, rand.Int31() % 255, rand.Int31()%253 + 2}, // 192.168.x.x
		WifiBSSID:    randomMacAddress(),
		WifiSSID:     "<unknown ssid>",
		IMEI:         imei,
		AndroidId:    androidId, // 原版的 androidId和Display内容一样，我没看协议，但是按android文档上说应该是64-bit number的hex，姑且这么做
		APN:          "wifi",
		VendorName:   RandString(12), // 这个和下面一个选项(VendorOSName)都属于意义不明，找不到相似对应，不知道是啥
		VendorOSName: RandString(12),
		Protocol:     protocol,
		Version: &osVersionFile{
			Incremental: "OCACNFA", // Build.Version.INCREMENTAL, MIUI12: V12.5.3.0.RJBCNXM
			Release:     "11",
			Codename:    "REL",
			Sdk:         29,
		},
	}

	a, b := json.Marshal(deviceJson)
	return deviceJson.Model, a, b
}

func GenerateDeviceJson(dice *Dice, protocol int) (string, []byte, error) {
	switch protocol {
	case 0, 3:
		return GenerateDeviceJsonIOS(protocol)
	case 2:
		return GenerateDeviceJsonAndroidWatch(protocol)
	case 1:
		return GenerateDeviceJsonAndroid(dice, protocol)
	default:
		return GenerateDeviceJsonAllRandom(protocol)
	}
}

func GenerateDeviceJsonAndroid(dice *Dice, protocol int) (string, []byte, error) {
	// check if ./my_device.json exists
	if _, err := os.Stat("./my_device.json"); err == nil {
		dice.Logger.Info("检测到my_device.json，将使用该文件中的设备信息")
		// file exists
		data, err := os.ReadFile("./my_device.json")
		if err == nil {
			deviceJson := deviceFile{}
			err = json.Unmarshal(data, &deviceJson)
			if err == nil {
				deviceJson.Protocol = protocol
				a, b := json.Marshal(deviceJson)
				return deviceJson.Model, a, b
			} else {
				dice.Logger.Warn("读取./my_device.json失败，将使用随机设备信息。原因为JSON解析错误: " + err.Error())
			}
		} else {
			dice.Logger.Warn("读取./my_device.json失败，将使用随机设备信息")
		}
	}

	pool := androidDevicePool
	//rand.Seed(time.Now().Unix())
	//bootId := uuid.New()
	imei := goluhn.Generate(15) // 注意，这个imei是完全胡乱创建的，并不符合imei规则
	androidId := fmt.Sprintf("%X", rand.Uint64())

	m := pool[rand.Int()%len(pool)]
	deviceJson := m.data

	deviceJson.MacAddress = randomMacAddress()
	deviceJson.IpAddress = []int32{192, 168, rand.Int31() % 255, rand.Int31()%253 + 2} // 192.168.x.x
	deviceJson.IMEI = imei
	deviceJson.AndroidId = androidId
	deviceJson.Protocol = protocol

	a, b := json.Marshal(deviceJson)
	return deviceJson.Model, a, b
}

var defaultConfig = `
# go-cqhttp 默认配置文件

account: # 账号相关
  uin: {QQ帐号} # QQ账号
  password: {QQ密码} # 密码为空时使用扫码登录
  encrypt: false  # 是否开启密码加密
  status: 0      # 在线状态 请参考 https://docs.go-cqhttp.org/guide/config.html#在线状态
  relogin: # 重连设置
    delay: 3   # 首次重连延迟, 单位秒
    interval: 3   # 重连间隔
    max-times: 0  # 最大重连次数, 0为无限制

  # 是否使用服务器下发的新地址进行重连
  # 注意, 此设置可能导致在海外服务器上连接情况更差
  use-sso-address: true
  # 是否允许发送临时会话消息
  allow-temp-session: false
{旧版签名服务相关配置信息}
{新版签名服务相关配置信息}

heartbeat:
  # 心跳频率, 单位秒
  # -1 为关闭心跳
  interval: 5

message:
  # 上报数据类型
  # 可选: string,array
  post-format: string
  # 是否忽略无效的CQ码, 如果为假将原样发送
  ignore-invalid-cqcode: false
  # 是否强制分片发送消息
  # 分片发送将会带来更快的速度
  # 但是兼容性会有些问题
  force-fragment: false
  # 是否将url分片发送
  fix-url: false
  # 下载图片等请求网络代理
  proxy-rewrite: ''
  # 是否上报自身消息
  report-self-message: false
  # 移除服务端的Reply附带的At
  remove-reply-at: false
  # 为Reply附加更多信息
  extra-reply-data: false
  # 跳过 Mime 扫描, 忽略错误数据
  skip-mime-scan: false

output:
  # 日志等级 trace,debug,info,warn,error
  log-level: warn
  # 日志时效 单位天. 超过这个时间之前的日志将会被自动删除. 设置为 0 表示永久保留.
  log-aging: 15
  # 是否在每次启动时强制创建全新的文件储存日志. 为 false 的情况下将会在上次启动时创建的日志文件续写
  log-force-new: true
  # 是否启用日志颜色
  log-colorful: true
  # 是否启用 DEBUG
  debug: false # 开启调试模式

# 默认中间件锚点
default-middlewares: &default
  # 访问密钥, 强烈推荐在公网的服务器设置
  access-token: ''
  # 事件过滤器文件目录
  filter: ''
  # API限速设置
  # 该设置为全局生效
  # 原 cqhttp 虽然启用了 rate_limit 后缀, 但是基本没插件适配
  # 目前该限速设置为令牌桶算法, 请参考:
  # https://baike.baidu.com/item/%E4%BB%A4%E7%89%8C%E6%A1%B6%E7%AE%97%E6%B3%95/6597000?fr=aladdin
  rate-limit:
    enabled: false # 是否启用限速
    frequency: 1  # 令牌回复频率, 单位秒
    bucket: 1     # 令牌桶大小

database: # 数据库相关设置
  leveldb:
    # 是否启用内置leveldb数据库
    # 启用将会增加10-20MB的内存占用和一定的磁盘空间
    # 关闭将无法使用 撤回 回复 get_msg 等上下文相关功能
    enable: true

  # 媒体文件缓存， 删除此项则使用缓存文件(旧版行为)
  cache:
    image: data/image.db
    video: data/video.db

# 连接服务列表
servers:
  # 添加方式，同一连接方式可添加多个，具体配置说明请查看文档
  #- http: # http 通信
  #- ws:   # 正向 Websocket
  #- ws-reverse: # 反向 Websocket
  #- pprof: #性能分析服务器
  # 正向WS设置
  - ws:
      # 正向WS服务器监听地址
      host: 127.0.0.1
      # 正向WS服务器监听端口
      port: {WS端口}
      # rc3
      address: 127.0.0.1:{WS端口}
      middlewares:
        <<: *default # 引用默认中间件
`

func GenerateConfig(qq int64, port int, info GoCqHttpLoginInfo) string {
	ret := strings.ReplaceAll(defaultConfig, "{WS端口}", fmt.Sprintf("%d", port))
	ret = strings.Replace(ret, "{QQ帐号}", fmt.Sprintf("%d", qq), 1)
	ret = strings.Replace(ret, "{QQ密码}", info.Password, 1)

	if info.UseSignServer && info.SignServerConfig != nil {
		ret = strings.Replace(ret, "{旧版签名服务相关配置信息}", generateOldSignServerConfigStr(info.SignServerConfig), 1)
		ret = strings.Replace(ret, "{新版签名服务相关配置信息}", generateNewSignServerConfigStr(info.SignServerConfig), 1)
	} else {
		ret = strings.Replace(ret, "{旧版签名服务相关配置信息}", "", 1)
		ret = strings.Replace(ret, "{新版签名服务相关配置信息}", "", 1)
	}
	return ret
}

func generateOldSignServerConfigStr(config *SignServerConfig) string {
	if config.SignServers != nil {
		mainServer := config.SignServers[0]
		return fmt.Sprintf(`
  # 旧版签名服务相关配置信息
  sign-server: '%s'
  # 如果签名服务器的版本在1.1.0及以下, 请将下面的参数改成true
  # 该字段在新签名配置信息中也存在，防止重复此处不配置
  # is-below-110: false
  # 签名服务器所需要的apikey, 如果签名服务器的版本在1.1.0及以下则此项无效
  key: '%s'
`, mainServer.Url, mainServer.Key)
	} else {
		return ""
	}
}

func generateNewSignServerConfigStr(config *SignServerConfig) string {
	var signServers []string
	for _, server := range config.SignServers {
		signServers = append(
			signServers,
			fmt.Sprintf(`    - url: '%s'
      key: "%s"
      authorization: "%s"`, server.Url, server.Key, server.Authorization),
		)
	}
	signServersStr := "  sign-servers:\n" + strings.Join(signServers, "\n")

	return fmt.Sprintf(`  # 新版签名服务相关配置信息
  # 数据包的签名服务器列表，第一个作为主签名服务器，后续作为备用
  # 兼容 https://github.com/fuqiuluo/unidbg-fetch-qsign
  # 如果遇到 登录 45 错误, 或者发送信息风控的话需要填入一个或多个服务器
  # 不建议设置过多，设置主备各一个即可，超过 5 个只会取前五个
  # 示例:
  # sign-servers: 
  #   - url: 'http://127.0.0.1:8080' # 本地签名服务器
  #     key: "114514"  # 相应 key
  #     authorization: "-"   # authorization 内容, 依服务端设置
  #   - url: 'https://signserver.example.com' # 线上签名服务器
  #     key: "114514"  
  #     authorization: "-"   
  #   ...
  # 
  # 服务器可使用docker在本地搭建或者使用他人开放的服务
%s

  # 判断签名服务不可用（需要切换）的额外规则
  # 0: 不设置 （此时仅在请求无法返回结果时判定为不可用）
  # 1: 在获取到的 sign 为空 （若选此建议关闭 auto-register，一般为实例未注册但是请求签名的情况）
  # 2: 在获取到的 sign 或 token 为空（若选此建议关闭 auto-refresh-token ）
  rule-change-sign-server: %d

  # 连续寻找可用签名服务器最大尝试次数
  # 为 0 时会在连续 3 次没有找到可用签名服务器后保持使用主签名服务器，不再尝试进行切换备用
  # 否则会在达到指定次数后 **退出** 主程序
  max-check-count: %d
  # 签名服务请求超时时间(s)
  sign-server-timeout: %d
  # 如果签名服务器的版本在1.1.0及以下, 请将下面的参数改成true
  # 建议使用 1.1.6 以上版本，低版本普遍半个月冻结一次
  is-below-110: false
  # 在实例可能丢失（获取到的签名为空）时是否尝试重新注册
  # 为 true 时，在签名服务不可用时可能每次发消息都会尝试重新注册并签名。
  # 为 false 时，将不会自动注册实例，在签名服务器重启或实例被销毁后需要重启 go-cqhttp 以获取实例
  # 否则后续消息将不会正常签名。关闭此项后可以考虑开启签名服务器端 auto_register 避免需要重启
  # 由于实现问题，当前建议关闭此项，推荐开启签名服务器的自动注册实例
  auto-register: %v
  # 是否在 token 过期后立即自动刷新签名 token（在需要签名时才会检测到，主要防止 token 意外丢失）
  # 独立于定时刷新
  auto-refresh-token: %v
  # 定时刷新 token 间隔时间，单位为分钟, 建议 30~40 分钟, 不可超过 60 分钟
  # 目前丢失token也不会有太大影响，可设置为 0 以关闭，推荐开启
  refresh-interval: %d
`,
		signServersStr,
		config.RuleChangeSignServer,
		config.MaxCheckCount,
		config.SignServerTimeout,
		config.AutoRegister,
		config.AutoRefreshToken,
		config.RefreshInterval,
	)
}

func NewGoCqhttpConnectInfoItem(account string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.Id = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "onebot"
	conn.Enable = false
	conn.RelWorkDir = "extra/go-cqhttp-qq" + account

	conn.Adapter = &PlatformAdapterGocq{
		EndPoint:          conn,
		UseInPackGoCqhttp: true,
	}
	return conn
}

func GoCqHttpServeProcessKill(dice *Dice, conn *EndPointInfo) {
	defer func() {
		defer func() {
			if r := recover(); r != nil {
				dice.Logger.Error("go-cqhttp清理报错: ", r)
				// go-cqhttp 进程退出: exit status 1
			}
		}()

		pa, ok := conn.Adapter.(*PlatformAdapterGocq)
		if !ok {
			return
		}
		if pa.UseInPackGoCqhttp {
			// 重置状态
			conn.State = 0
			pa.GoCqHttpState = 0

			pa.DiceServing = false
			pa.GoCqHttpQrcodeData = nil
			pa.GoCqHttpLoginDeviceLockUrl = ""

			workDir := gocqGetWorkDir(dice, conn)
			qrcodeFile := filepath.Join(workDir, "qrcode.png")
			if _, err := os.Stat(qrcodeFile); err == nil {
				// 如果已经存在二维码文件，将其删除
				_ = os.Remove(qrcodeFile)
				dice.Logger.Info("onebot: 删除已存在的二维码文件")
			}

			// 注意这个会panic，因此recover捕获了
			if pa.GoCqHttpProcess != nil {
				p := pa.GoCqHttpProcess
				pa.GoCqHttpProcess = nil
				//sigintwindows.SendCtrlBreak(p.Cmds[0].Process.Pid)
				_ = p.Stop()
				_ = p.Wait() // 等待进程退出，因为Stop内部是Kill，这是不等待的
			}
		}
	}()
}

func gocqGetWorkDir(dice *Dice, conn *EndPointInfo) string {
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	//pa := conn.Adapter.(*PlatformAdapterGocq)
	//if !pa.UseInPackGoCqhttp {
	//	return "#$%Abort^?*" // 使其尽量非法，从而跳过连接外所有流程
	//}
	return workDir
}

func GoCqHttpServeRemoveSessionToken(dice *Dice, conn *EndPointInfo) {
	workDir := gocqGetWorkDir(dice, conn)
	if _, err := os.Stat(filepath.Join(workDir, "session.token")); err == nil {
		_ = os.Remove(filepath.Join(workDir, "session.token"))
	}
}

type GoCqHttpLoginInfo struct {
	Password         string
	Protocol         int
	AppVersion       string
	IsAsyncRun       bool
	UseSignServer    bool
	SignServerConfig *SignServerConfig
}

type SignServerConfig struct {
	SignServers          []*SignServer `yaml:"signServers" json:"signServers"`
	RuleChangeSignServer int           `yaml:"ruleChangeSignServer" json:"ruleChangeSignServer"`
	MaxCheckCount        int           `yaml:"maxCheckCount" json:"maxCheckCount"`
	SignServerTimeout    int           `yaml:"signServerTimeout" json:"signServerTimeout"`
	AutoRegister         bool          `yaml:"autoRegister" json:"autoRegister"`
	AutoRefreshToken     bool          `yaml:"autoRefreshToken" json:"autoRefreshToken"`
	RefreshInterval      int           `yaml:"refreshInterval" json:"refreshInterval"`
}

type SignServer struct {
	Url           string `yaml:"url" json:"url"`
	Key           string `yaml:"key" json:"key"`
	Authorization string `yaml:"authorization" json:"authorization"`
}

func GoCqHttpServe(dice *Dice, conn *EndPointInfo, loginInfo GoCqHttpLoginInfo) {
	pa := conn.Adapter.(*PlatformAdapterGocq)
	//if pa.GoCqHttpState != StateCodeInit {
	//	return
	//}

	pa.CurLoginIndex += 1
	loginIndex := pa.CurLoginIndex
	pa.GoCqHttpState = StateCodeInLogin

	fmt.Println("GoCqHttpServe begin")
	if pa.UseInPackGoCqhttp {
		workDir := gocqGetWorkDir(dice, conn)
		_ = os.MkdirAll(workDir, 0755)

		qrcodeFile := filepath.Join(workDir, "qrcode.png")
		deviceFilePath := filepath.Join(workDir, "device.json")
		configFilePath := filepath.Join(workDir, "config.yml")
		versionDirPath := filepath.Join(workDir, "data", "versions")
		if _, err := os.Stat(qrcodeFile); err == nil {
			// 如果已经存在二维码文件，将其删除
			_ = os.Remove(qrcodeFile)
			dice.Logger.Info("onebot: 删除已存在的二维码文件")
		}

		//if _, err := os.Stat(filepath.Join(workDir, "session.token")); errors.Is(err, os.ErrNotExist) {
		if !pa.GoCqHttpLoginSucceeded {
			// 并未登录成功，删除记录文件
			dice.Logger.Info("onebot: 之前并未登录成功，删除设备文件和配置文件")
			_ = os.Remove(configFilePath)
			_ = os.Remove(deviceFilePath)
		}

		// 创建设备配置文件
		if _, err := os.Stat(deviceFilePath); errors.Is(err, os.ErrNotExist) {
			_, deviceInfo, err := GenerateDeviceJson(dice, loginInfo.Protocol)
			if err == nil {
				_ = os.WriteFile(deviceFilePath, deviceInfo, 0644)
				dice.Logger.Info("onebot: 成功创建设备文件")
			}
		}

		// 创建配置文件
		if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
			// 如果不存在 config.yml 那么启动一次，让它自动生成
			// 改为：如果不存在，帮他创建
			p, _ := GetRandomFreePort()
			pa.ConnectUrl = fmt.Sprintf("ws://localhost:%d", p)
			qqid, _ := pa.mustExtractId(conn.UserId)
			c := GenerateConfig(qqid, p, loginInfo)
			_ = os.WriteFile(configFilePath, []byte(c), 0644)
		}

		if versions, ok := GocqAppVersionMap[loginInfo.AppVersion]; ok {
			if targetVersion, ok2 := versions[ProtocolType(loginInfo.Protocol)]; ok2 {
				// 删除旧协议版本文件
				_ = os.RemoveAll(versionDirPath)
				// 创建协议版本文件
				_ = os.MkdirAll(versionDirPath, 0755)
				versionFilePath := filepath.Join(versionDirPath, fmt.Sprintf("%d.json", loginInfo.Protocol))
				jsonData, _ := json.Marshal(targetVersion)
				_ = os.WriteFile(versionFilePath, jsonData, 0644)
			}
		}

		// 启动客户端
		wd, _ := os.Getwd()
		gocqhttpExePath, _ := filepath.Abs(filepath.Join(wd, "go-cqhttp/go-cqhttp"))
		gocqhttpExePath = strings.Replace(gocqhttpExePath, "\\", "/", -1) // windows平台需要这个替换

		// 随手执行一下
		_ = os.Chmod(gocqhttpExePath, 0755)

		dice.Logger.Info("onebot: 正在启动onebot客户端…… ", gocqhttpExePath)
		p := procs.NewProcess(fmt.Sprintf(`"%s" -faststart`, gocqhttpExePath))
		p.Dir = workDir

		if runtime.GOOS == "android" {
			p.Env = os.Environ()
		}
		chQrCode := make(chan int, 1)
		riskCount := 0
		isSeldKilling := false

		slideMode := 0
		chSMS := make(chan string, 1)

		p.OutputHandler = func(line string) string {
			if loginIndex != pa.CurLoginIndex {
				// 当前连接已经无用，进程自杀
				if !isSeldKilling {
					dice.Logger.Infof("检测到新的连接序号 %d，当前连接 %d 将自动退出", pa.CurLoginIndex, loginIndex)
					// 注: 这里不要调用kill
					isSeldKilling = true
					_ = p.Stop()
				}
				return ""
			}

			// 登录中
			if pa.IsInLogin() {
				// 请使用手机QQ扫描二维码 (qrcode.png) :
				if strings.Contains(line, "qrcode.png") {
					chQrCode <- 1
				}

				// 获取二维码失败，登录失败
				if strings.Contains(line, "fetch qrcode error: Packet timed out ") {
					dice.Logger.Infof("从QQ服务器获取二维码错误（超时），帐号: <%s>(%s)", conn.Nickname, conn.UserId)
					pa.GoCqHttpState = StateCodeLoginFailed
				}

				// 未知错误，gocqhttp崩溃
				if strings.Contains(line, "Packet failed to sendPacket: connection closed") {
					dice.Logger.Infof("登录异常，gocqhttp崩溃")
					pa.GoCqHttpState = StateCodeLoginFailed
				}

				if strings.Contains(line, "按 Enter 继续....") {
					// 直接输入继续，基本都是登录失败
					return "\n"
				}

				if strings.Contains(line, "WARNING") && strings.Contains(line, "账号已开启设备锁，请前往") {
					re := regexp.MustCompile(`-> (.+?) <-`)
					m := re.FindStringSubmatch(line)
					dice.Logger.Info("触发设备锁流程: ", len(m) > 0)
					if len(m) > 0 {
						// 设备锁流程，因为需要重新登录，进行一个“已成功登录过”的标记，这样配置文件不会被删除
						pa.GoCqHttpState = StateCodeInLoginDeviceLock
						pa.GoCqHttpLoginSucceeded = true
						pa.GoCqHttpLoginDeviceLockUrl = m[1]
						dice.LastUpdatedTime = time.Now().Unix()
						dice.Save(false)
					}
				}

				if strings.Contains(line, " 发送短信验证码") && strings.Contains(line, " [WARNING]: 1. 向手机 ") {
					re := regexp.MustCompile(`\[WARNING\]: 发送短信验证码 (.+?) 发送短信验证码`)
					m := re.FindStringSubmatch(line)
					if len(m) > 0 {
						pa.GoCqHttpSmsNumberTip = m[1]
					}
				}

				if strings.Contains(line, " [WARNING]: 登录需要滑条验证码, 请验证后重试.") {
					slideMode = 1
				}

				if strings.Contains(line, " [WARNING]: 账号已开启设备锁，请选择验证方式") {
					slideMode = 2
				}

				// 直接短信验证，不过滑条
				if slideMode == 2 {
					// 账号已开启设备锁，请选择验证方式
					// 1. 向手机 %v 发送短信验证码
					// 2. 使用手机QQ扫码验证
					// 请输入(1 - 2)：
					if strings.Contains(line, "WARNING") && strings.Contains(line, "[WARNING]: 请输入(1 - 2)：") {
						// gocq的tty检测太辣鸡了
						return "1\n"
					}
				}

				// 滑条流程
				if slideMode == 1 {
					if strings.Contains(line, "WARNING") && strings.Contains(line, "[WARNING]: 请输入(1 - 2)：") {
						// gocq的tty检测太辣鸡了
						return "1\n"
					}

					if strings.Contains(line, "WARNING") && strings.Contains(line, "请前往该地址验证") {
						re := regexp.MustCompile(`-> (.+)`)
						m := re.FindStringSubmatch(line)
						dice.Logger.Info("触发滑条流程: ", len(m) > 0)

						if len(m) > 0 {
							pa.GoCqHttpState = GoCqHttpStateCodeInLoginBar
							pa.GoCqHttpLoginDeviceLockUrl = strings.TrimSpace(m[1])
						}
					}
				}

				if strings.Contains(line, " [WARNING]: 请输入短信验证码：") {
					dice.Logger.Info("进入短信验证码流程，等待输入")
					pa.GoCqHttpState = GoCqHttpStateCodeInLoginVerifyCode
					pa.GoCqHttpLoginVerifyCode = ""
					go func() {
						// 检查是否有短信验证码
						for i := 0; i < 100; i += 1 {
							if pa.GoCqHttpState != GoCqHttpStateCodeInLoginVerifyCode {
								break
							}
							time.Sleep(6 * time.Duration(time.Second))
							if pa.GoCqHttpLoginVerifyCode != "" {
								chSMS <- pa.GoCqHttpLoginVerifyCode
								break
							}
						}
					}()
					code := <-chSMS
					dice.Logger.Infof("即将输入短信验证码: %v", code)
					return code + "\n"
				}

				if strings.Contains(line, "发送验证码失败，可能是请求过于频繁.") {
					pa.GoCqHttpState = StateCodeLoginFailed
					pa.GocqhttpLoginFailedReason = "发送验证码失败，可能是请求过于频繁"
				}

				// 登录成功
				if strings.Contains(line, "CQ WebSocket 服务器已启动") {
					// CQ WebSocket 服务器已启动
					// 登录成功 欢迎使用
					pa.GoCqHttpState = StateCodeLoginSuccessed
					pa.GoCqHttpLoginSucceeded = true
					dice.Logger.Infof("gocqhttp登录成功，帐号: <%s>(%s)", conn.Nickname, conn.UserId)
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)

					go ServeQQ(dice, conn)
				}
			}

			if strings.Contains(line, "请使用手机QQ扫描二维码以继续登录") {
				//TODO
				fmt.Println("请使用手机QQ扫描二维码以继续登录")
			}

			if (pa.IsLoginSuccessed() && strings.Contains(line, "[ERROR]:") && strings.Contains(line, "Protocol -> sendPacket msg error: 120")) || strings.Contains(line, "账号可能被风控####2测试触发语句") {
				// 这种情况应该是被禁言，提前减去以免出事
				riskCount -= 1
				dice.Logger.Infof("因禁言无法发言: 下方可能会提示遭遇风控")
			}

			if (pa.IsLoginSuccessed() && strings.Contains(line, "WARNING") && strings.Contains(line, "账号可能被风控")) || strings.Contains(line, "账号可能被风控####测试触发语句") {
				//群消息发送失败: 账号可能被风控
				now := time.Now().Unix()
				if now-pa.GoCqHttpLastRestrictedTime < 5*60 {
					// 阈值是5分钟内2次
					riskCount += 1
				}
				pa.GoCqHttpLastRestrictedTime = now
				if riskCount >= 2 {
					riskCount = 0
					if dice.AutoReloginEnable {
						// 大于5分钟触发
						if now-pa.GoCqLastAutoLoginTime > 5*60 {
							dice.Logger.Warnf("自动重启: 达到风控重启阈值 <%s>(%s)", conn.Nickname, conn.UserId)
							if pa.InPackGoCqHttpPassword != "" {
								pa.DoRelogin()
							} else {
								dice.Logger.Warnf("自动重启: 未输入密码，放弃")
							}
						}
					}
				}
			}

			if pa.IsInLogin() || strings.Contains(line, "[WARNING]") || strings.Contains(line, "[ERROR]") || strings.Contains(line, "[FATAL]") {
				//  [WARNING]: 登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录
				if pa.IsLoginSuccessed() {
					skip := false

					if strings.Contains(line, "WARNING") {
						if strings.Contains(line, "检查更新失败") || strings.Contains(line, "Protocol -> device lock is disable.") {
							skip = true
						}
						if strings.Contains(line, "语音文件") && strings.Contains(line, "下载失败") {
							skip = true
						}
					}

					if strings.Contains(line, "ERROR") {
						if strings.Contains(line, "panic on decoder MsgPush.PushGroupProMsg") {
							skip = true
						}
					}

					if !skip {
						dice.Logger.Infof("onebot | %s", stripansi.Strip(line))
					} else {
						if strings.HasSuffix(line, "\n") {
							fmt.Printf("onebot | %s", line)
						}
					}
				} else {
					if strings.HasSuffix(line, "\n") {
						fmt.Printf("onebot | %s", line)
					}

					skip := false
					if strings.Contains(line, "WARNING") && strings.Contains(line, "使用了过时的配置格式，请更新配置文件") {
						skip = true
					}

					if strings.Contains(line, "WARNING") {
						if strings.Contains(line, "检查更新失败") || strings.Contains(line, "Protocol -> device lock is disable.") {
							skip = true
						}
					}

					// error 之类错误无条件警告
					if !skip {
						if strings.Contains(line, "WARNING") || strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") {
							dice.Logger.Infof("onebot | %s", stripansi.Strip(line))
						}
					}
				}
			}
			return ""
		}

		go func() {
			<-chQrCode
			if _, err := os.Stat(qrcodeFile); err == nil {
				dice.Logger.Info("onebot: 二维码已经就绪")
				fmt.Println("如控制台二维码不好扫描，可以手动打开 ./data/default/extra/go-cqhttp-qqXXXXX 目录下qrcode.png")
				qrdata, err := os.ReadFile(qrcodeFile)
				if err == nil {
					pa.GoCqHttpState = StateCodeInLoginQrCode
					pa.GoCqHttpQrcodeData = qrdata
					dice.Logger.Info("获取二维码成功")
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)
					_ = os.Rename(qrcodeFile, qrcodeFile+".bak.png")
				} else {
					pa.GoCqHttpQrcodeData = nil
					pa.GoCqHttpState = StateCodeLoginFailed
					pa.GocqhttpLoginFailedReason = "获取二维码失败"
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)
					dice.Logger.Info("获取二维码失败，错误为: ", err.Error())
				}
			}
		}()

		run := func() {
			defer func() {
				if r := recover(); r != nil {
					dice.Logger.Errorf("onebot: 异常: %v 堆栈: %v", r, string(debug.Stack()))
				}
			}()

			// 启动gocqhttp，开始登录
			pa.GoCqHttpProcess = p
			err := p.Start()

			if err == nil {
				if dice.Parent.progressExitGroupWin != 0 && p.Cmd != nil {
					err := dice.Parent.progressExitGroupWin.AddProcess(p.Cmd.Process)
					if err != nil {
						dice.Logger.Warn("添加到进程组失败，若主进程崩溃，gocqhttp进程可能需要手动结束")
					}
				}
				_ = p.Wait()
			}

			isInLogin := pa.IsInLogin()
			isDeviceLockLogin := pa.GoCqHttpState == StateCodeInLoginDeviceLock
			if !isDeviceLockLogin {
				// 如果在设备锁流程中，不清空数据
				GoCqHttpServeProcessKill(dice, conn)

				if isInLogin {
					conn.State = 3
					pa.GoCqHttpState = StateCodeLoginFailed
				} else {
					conn.State = 0
					pa.GoCqHttpState = GoCqHttpStateCodeClosed
				}
			}

			if err != nil {
				dice.Logger.Info("go-cqhttp 进程退出: ", err)
			} else {
				dice.Logger.Info("go-cqhttp 进程退出")
			}
		}

		if loginInfo.IsAsyncRun {
			go run()
		} else {
			run()
		}
	} else {
		pa.GoCqHttpState = StateCodeLoginSuccessed
		pa.GoCqHttpLoginSucceeded = true
		dice.Save(false)
		go ServeQQ(dice, conn)
	}
}
