package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/acarl005/stripansi"
	"github.com/fy0/procs"
	"github.com/google/uuid"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
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

func GenerateDeviceJsonIOS(protocol int) ([]byte, error) {
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

	return json.Marshal(deviceJson)
}

func GenerateDeviceJsonAllRandom(protocol int) ([]byte, error) {
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

	return json.Marshal(deviceJson)
}

func GenerateDeviceJson(protocol int) ([]byte, error) {
	switch protocol {
	case 0, 2, 3:
		return GenerateDeviceJsonIOS(protocol)
	case 1:
		return GenerateDeviceJsonAndroid(protocol)
	default:
		return GenerateDeviceJsonAllRandom(protocol)
	}
}

func GenerateDeviceJsonAndroid(protocol int) ([]byte, error) {
	rand.Seed(time.Now().Unix())
	bootId := uuid.New()
	imei := goluhn.Generate(15) // 注意，这个imei是完全胡乱创建的，并不符合imei规则
	androidId := fmt.Sprintf("%X", rand.Uint64())

	deviceJson := deviceFile{
		Display:      "MIUI V9.5.3.0", // Rom的名字 比如 Flyme 1.1.2（魅族rom）  JWR66V（Android nexus系列原生4.3rom）
		Product:      "sagit",         // 产品名，比如这是小米6的代号
		Device:       "sagit",
		Board:        "msm8998",                                                                     // 主板:骁龙835                                                                    //
		Brand:        "Xiaomi",                                                                      // 品牌
		Model:        "MI 6",                                                                        // 型号
		Bootloader:   "unknown",                                                                     // unknown不需要改
		FingerPrint:  "Xiaomi/sagit/sagit:8.0.0/OPR1.170623.027/V9.5.3.0.OCACNFA:user/release-keys", // 指纹
		BootId:       bootId.String(),
		ProcVersion:  "Linux version 3.10.61-7254923", // 很长，后面 builder省略了
		BaseBand:     "",                              // 基带版本 4.3CPL2-... 一大堆，直接不写
		SimInfo:      "",
		OSType:       "android",
		MacAddress:   randomMacAddress(),
		IpAddress:    []int32{192, 168, rand.Int31() % 255, rand.Int31()%253 + 2}, // 192.168.x.x
		WifiBSSID:    randomMacAddress(),
		WifiSSID:     "<unknown ssid>",
		IMEI:         imei,
		AndroidId:    androidId, // 原版的 androidId和Display内容一样，我没看协议，但是按android文档上说应该是64-bit number的hex，姑且这么做
		APN:          "wifi",
		VendorName:   "MIUI", // 这个和下面一个选项(VendorOSName)都属于意义不明，找不到相似对应，不知道是啥
		VendorOSName: "xiaomi",
		Protocol:     protocol,
		Version: &osVersionFile{
			Incremental: "OCACNFA", // Build.Version.INCREMENTAL, MIUI12: V12.5.3.0.RJBCNXM
			Release:     "11",
			Codename:    "REL",
			Sdk:         29,
		},
	}

	return json.Marshal(deviceJson)
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

func GenerateConfig(qq int64, password string, port int) string {
	ret := strings.ReplaceAll(defaultConfig, "{WS端口}", fmt.Sprintf("%d", port))
	ret = strings.Replace(ret, "{QQ帐号}", fmt.Sprintf("%d", qq), 1)

	password2, _ := json.Marshal(password)
	ret = strings.Replace(ret, "{QQ密码}", fmt.Sprintf("%s", string(password2)), 1)
	return ret
}

func NewGoCqhttpConnectInfoItem(account string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.Id = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "onebot"
	conn.Enable = false
	conn.RelWorkDir = "extra/go-cqhttp-qq" + account

	conn.Adapter = &PlatformAdapterQQOnebot{
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

		pa, ok := conn.Adapter.(*PlatformAdapterQQOnebot)
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

			workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
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

func GoCqHttpServeRemoveSessionToken(dice *Dice, conn *EndPointInfo) {
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	if _, err := os.Stat(filepath.Join(workDir, "session.token")); err == nil {
		_ = os.Remove(filepath.Join(workDir, "session.token"))
	}
}

func GoCqHttpServe(dice *Dice, conn *EndPointInfo, password string, protocol int, isAsyncRun bool) {
	pa := conn.Adapter.(*PlatformAdapterQQOnebot)
	//if pa.GoCqHttpState != GoCqHttpStateCodeInit {
	//	return
	//}

	pa.CurLoginIndex += 1
	loginIndex := pa.CurLoginIndex
	pa.GoCqHttpState = GoCqHttpStateCodeInLogin

	fmt.Println("GoCqHttpServe begin")
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	_ = os.MkdirAll(workDir, 0755)

	qrcodeFile := filepath.Join(workDir, "qrcode.png")
	deviceFilePath := filepath.Join(workDir, "device.json")
	configFilePath := filepath.Join(workDir, "config.yml")
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
		deviceInfo, err := GenerateDeviceJson(protocol)
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
		c := GenerateConfig(qqid, password, p)
		_ = os.WriteFile(configFilePath, []byte(c), 0644)
	}

	// 启动客户端
	wd, _ := os.Getwd()
	gocqhttpExePath, _ := filepath.Abs(filepath.Join(wd, "go-cqhttp/go-cqhttp"))
	gocqhttpExePath = strings.Replace(gocqhttpExePath, "\\", "/", -1) // windows平台需要这个替换

	// 随手执行一下
	_ = os.Chmod(gocqhttpExePath, 0755)

	dice.Logger.Info("onebot: 正在启动onebot客户端…… ", gocqhttpExePath)
	p := procs.NewProcess(fmt.Sprintf(`"%s" faststart`, gocqhttpExePath))
	p.Dir = workDir

	chQrCode := make(chan int, 1)
	riskCount := 0
	isSeldKilling := false
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
				pa.GoCqHttpState = GoCqHttpStateCodeLoginFailed
			}

			// 未知错误，gocqhttp崩溃
			if strings.Contains(line, "Packet failed to sendPacket: connection closed") {
				dice.Logger.Infof("登录异常，gocqhttp崩溃")
				pa.GoCqHttpState = GoCqHttpStateCodeLoginFailed
			}

			if strings.Contains(line, "WARNING") && strings.Contains(line, "账号已开启设备锁，请前往") {
				re := regexp.MustCompile(`-> (.+?) <-`)
				m := re.FindStringSubmatch(line)
				dice.Logger.Info("触发设备锁流程: ", len(m) > 0)
				if len(m) > 0 {
					// 设备锁流程，因为需要重新登录，进行一个“已成功登录过”的标记，这样配置文件不会被删除
					pa.GoCqHttpState = GoCqHttpStateCodeInLoginDeviceLock
					pa.GoCqHttpLoginSucceeded = true
					pa.GoCqHttpLoginDeviceLockUrl = m[1]
					fmt.Println("??????????????????? why")
				}
			}

			if strings.Contains(line, " [WARNING]: 请输入短信验证码：") {
				fmt.Println("!!!!!!!!!!!!!!!!!!!! 短信验证码")
				_, _ = p.Cmds[0].Stdout.Write([]byte("\n"))
			}

			if strings.Contains(line, "发送验证码失败，可能是请求过于频繁.") {
				pa.GoCqHttpState = GoCqHttpStateCodeLoginFailed
				pa.GocqhttpLoginFailedReason = "发送验证码失败，可能是请求过于频繁"
			}

			// 登录成功
			if strings.Contains(line, "CQ WebSocket 服务器已启动") {
				// CQ WebSocket 服务器已启动
				// 登录成功 欢迎使用
				pa.GoCqHttpState = GoCqHttpStateCodeLoginSuccessed
				pa.GoCqHttpLoginSucceeded = true
				dice.Logger.Infof("gocqhttp登录成功，帐号: <%s>(%s)", conn.Nickname, conn.UserId)

				go ServeQQ(dice, conn)
			}
		}

		if strings.Contains(line, "请使用手机QQ扫描二维码以继续登录") {
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

		if pa.IsInLogin() || strings.Contains(line, "风控") || strings.Contains(line, "WARNING") || strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") {
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
					fmt.Printf("onebot | %s\n", line)
				}
			} else {
				fmt.Printf("onebot | %s\n", line)

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
		return line
	}

	go func() {
		<-chQrCode
		if _, err := os.Stat(qrcodeFile); err == nil {
			dice.Logger.Info("onebot: 二维码已经就绪")
			fmt.Println("如控制台二维码不好扫描，可以手动打开 ./data/default/extra/go-cqhttp-qqXXXXX 目录下qrcode.png")
			qrdata, err := os.ReadFile(qrcodeFile)
			if err == nil {
				pa.GoCqHttpState = GoCqHttpStateCodeInLoginQrCode
				pa.GoCqHttpQrcodeData = qrdata
				dice.Logger.Info("获取二维码成功")
				_ = os.Rename(qrcodeFile, qrcodeFile+".bak.png")
			} else {
				pa.GoCqHttpQrcodeData = nil
				pa.GoCqHttpState = GoCqHttpStateCodeLoginFailed
				pa.GocqhttpLoginFailedReason = "获取二维码失败"
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
			if dice.Parent.progressExitGroupWin != 0 && len(p.Cmds) > 0 && p.Cmds[0] != nil {
				err := dice.Parent.progressExitGroupWin.AddProcess(p.Cmds[0].Process)
				if err != nil {
					dice.Logger.Warn("添加到进程组失败，若主进程崩溃，gocqhttp进程可能需要手动结束")
				}
			}
			_ = p.Wait()
		}

		isInLogin := pa.IsInLogin()
		isDeviceLockLogin := pa.GoCqHttpState == GoCqHttpStateCodeInLoginDeviceLock
		if !isDeviceLockLogin {
			// 如果在设备锁流程中，不清空数据
			GoCqHttpServeProcessKill(dice, conn)

			if isInLogin {
				conn.State = 3
				pa.GoCqHttpState = GoCqHttpStateCodeLoginFailed
			}
		}

		if err != nil {
			dice.Logger.Info("go-cqhttp 进程退出: ", err)
		} else {
			dice.Logger.Info("go-cqhttp 进程退出")
		}
	}

	if isAsyncRun {
		go run()
	} else {
		run()
	}
}

// 注意：放在这里并不科学，记得重构
func ServeQQ(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "QQ" {
		conn := ep.Adapter.(*PlatformAdapterQQOnebot)

		if !conn.DiceServing {
			conn.DiceServing = true
		} else {
			return
		}

		ep.Enable = true
		ep.State = 2 // 连接中

		checkQuit := func() bool {
			if conn.GoCqHttpState == GoCqHttpStateCodeInLoginDeviceLock {
				d.Logger.Infof("检测到设备锁流程，暂时不再连接")
				ep.State = 0
				return true
			}
			if !conn.DiceServing {
				// 退出连接
				d.Logger.Infof("检测到连接关闭，不再进行此onebot服务的重连: <%s>(%s)", ep.Nickname, ep.UserId)
				return true
			}
			return false
		}

		lastRetryTime := time.Now().Unix()
		waitTimes := 0
		for {
			if checkQuit() {
				break
			}

			// 骰子开始连接
			d.Logger.Infof("开始连接 onebot 服务，帐号 <%s>(%s)，重试计数[%d/%d]", ep.Nickname, ep.UserId, waitTimes, 5)
			ret := ep.Adapter.Serve()

			if time.Now().Unix()-lastRetryTime > 8*60 {
				lastRetryTime = 0
			}
			lastRetryTime = time.Now().Unix()

			if ret == 0 {
				break
			}

			if checkQuit() {
				break
			}

			waitTimes += 1
			if waitTimes > 5 {
				d.Logger.Infof("onebot 连接重试次数过多，先行中断: <%s>(%s)", ep.Nickname, ep.UserId)
				conn.DiceServing = false
				break
			}

			//d.Logger.Infof("onebot 连接失败[%d/%d]，将在15秒后重新连接，帐号 <%s>(%s)", waitTimes, 5, ep.Nickname, ep.UserId)
			time.Sleep(15 * time.Second)
		}
	}
}
