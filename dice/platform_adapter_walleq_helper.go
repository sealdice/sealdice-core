package dice

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"

	"sealdice-core/utils/procs"
)

const (
	WqStateCodeInit              = 0
	WqStateCodeInLogin           = 1
	WqStateCodeInLoginQrCode     = 2
	WqStateCodeInLoginBar        = 3
	WqStateCodeInLoginVerifyCode = 6
	WqStateCodeInLoginDeviceLock = 7
	WqStateCodeLoginSuccessed    = 10
	WqStateCodeLoginFailed       = 11
	WqStateCodeClosed            = 20
)

type WqQQ struct {
	Password string `toml:"password,omitempty"`
	Protocol int    `toml:"protocol"`
}

type WqMeta struct {
	LogLevel       string `toml:"log_level"`
	EventCacheSize int    `toml:"event_cache_size"`
	Sled           bool   `toml:"sled"`
	Leveldb        bool   `toml:"leveldb"`
}

type WqWS struct { // 虽然支持多个但是用不到啊
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type WalleQConfig struct {
	// wq 支持复数的 qq 登录 // 多进程 or 单进程 ？
	QQ map[string]WqQQ `toml:"qq"`
	// 元数据
	Meta WqMeta `toml:"meta"`
	// 连接方式必须完整，哪怕没有用到 http 连接方式
	Onebot struct {
		HTTP         []interface{} `toml:"http"`
		HTTPWebhook  []interface{} `toml:"http_webhook"`
		WebsocketRev []interface{} `toml:"websocket_rev"`
		Websocket    []WqWS        `toml:"websocket"`
		// 心跳
		Heartbeat struct {
			Enabled  bool `toml:"enabled"`
			Interval int  `toml:"interval"`
		} `toml:"heartbeat"`
	} `toml:"onebot"`
}

func NewWqConnectInfoItem(account string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "walle-q"
	conn.Enable = false
	conn.RelWorkDir = "extra/walle-q-" + account

	conn.Adapter = &PlatformAdapterWalleQ{
		EndPoint:        conn,
		UseInPackWalleQ: true,
	}
	return conn
}

func WalleQServeRemoveSessionToken(dice *Dice, conn *EndPointInfo) {
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	if _, err := os.Stat(filepath.Join(workDir, "session.token")); err == nil {
		_ = os.Remove(filepath.Join(workDir, "session.token"))
	}
}

func WalleQServeProcessKill(dice *Dice, conn *EndPointInfo) {
	defer func() {
		defer func() {
			if r := recover(); r != nil {
				dice.Logger.Error("Walle-q 清理报错: ", r)
				// go-cqhttp 进程退出: exit status 1
			}
		}()

		pa, ok := conn.Adapter.(*PlatformAdapterWalleQ)
		if !ok {
			return
		}
		if pa.UseInPackWalleQ {
			// 重置状态
			conn.State = 0
			pa.WalleQState = 0

			pa.DiceServing = false
			pa.WalleQQrcodeData = nil
			pa.WalleQLoginDeviceLockURL = ""

			workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
			qrcodeFile := filepath.Join(workDir, "qrcode.png")
			if _, err := os.Stat(qrcodeFile); err == nil {
				// 如果已经存在二维码文件，将其删除
				_ = os.Remove(qrcodeFile)
				dice.Logger.Info("onebot: 删除已存在的二维码文件")
			}

			// 注意这个会panic，因此recover捕获了
			if pa.WalleQProcess != nil {
				p := pa.WalleQProcess
				pa.WalleQProcess = nil
				// sigintwindows.SendCtrlBreak(p.Cmds[0].Process.Pid)
				_ = p.Stop()
				_ = p.Wait() // 等待进程退出，因为Stop内部是Kill，这是不等待的
			}
		}
	}()
}

func WalleQServe(dice *Dice, conn *EndPointInfo, password string, protocol int, isAsyncRun bool) {
	pa := conn.Adapter.(*PlatformAdapterWalleQ)
	pa.CurLoginIndex++
	loginIndex := pa.CurLoginIndex
	pa.WalleQState = WqStateCodeInLogin
	fmt.Println("WalleQServe begin")
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	_ = os.MkdirAll(workDir, 0o755)
	log := dice.Logger
	qrcodeFile := filepath.Join(workDir, "qrcode.png")
	// deviceFilePath := filepath.Join(workDir, "device.json") // 暂时让他自己写
	configFilePath := filepath.Join(workDir, "walle-q.toml")
	if _, err := os.Stat(qrcodeFile); err == nil {
		// 如果已经存在二维码文件，将其删除
		_ = os.Remove(qrcodeFile)
		log.Info("onebot: 删除已存在的二维码文件")
	}

	// 创建配置文件
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		p, _ := GetRandomFreePort()
		pa.ConnectURL = fmt.Sprintf("ws://127.0.0.1:%d", p)
		id, _ := pa.mustExtractID(conn.UserID)
		wqc := WalleQConfig{}
		wqc.QQ = make(map[string]WqQQ)
		wqc.QQ[id] = WqQQ{
			password, protocol,
		}
		wqc.Meta.LogLevel = "info"
		wqc.Meta.EventCacheSize = 10
		wqc.Meta.Leveldb = true
		wqc.Meta.Sled = false
		wqc.Onebot.Websocket = append(wqc.Onebot.Websocket, WqWS{"127.0.0.1", p})
		wqc.Onebot.HTTP = make([]interface{}, 0)
		wqc.Onebot.WebsocketRev = make([]interface{}, 0)
		wqc.Onebot.HTTPWebhook = make([]interface{}, 0)
		b := new(bytes.Buffer)
		_ = toml.NewEncoder(b).Encode(wqc)
		_ = os.WriteFile(configFilePath, b.Bytes(), 0o644)
	} else { //nolint
		// 如果决定使用单进程 wq
		/*
			wqc := new(WalleQConfig)
			_, err = toml.DecodeFile(configFilePath,wqc)
			if err != nil {
				dice.Logger.Error("读取 Walle-q 配置文件失败，请检查！")
				return
			}
			id, _ := pa.mustExtractId(conn.UserId)
			wqc.QQ[id] = WqQQ{
				password, protocol,
			}
			b := new(bytes.Buffer)
			err = toml.NewEncoder(b).Encode(wqc)
			_ = os.WriteFile(configFilePath, []byte(b.String()), 0644)
		*/
	}
	wd, _ := os.Getwd()
	wqExePath, _ := filepath.Abs(filepath.Join(wd, "walle-q/walle-q"))
	wqExePath = strings.ReplaceAll(wqExePath, "\\", "/") // windows平台需要这个替换
	_ = os.Chmod(wqExePath, 0o755)

	log.Info("onebot: 正在启动onebot客户端…… ", wqExePath)
	p := procs.NewProcess(fmt.Sprintf(`"%s"`, wqExePath))
	p.Dir = workDir
	chQrCode := make(chan int, 1)
	isSeldKilling := false

	p.OutputHandler = func(line string) string {
		fmt.Println(line)
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

		if pa.IsInLogin() {
			if strings.Contains(line, "扫描二维码登录") {
				chQrCode <- 1
			}

			if strings.Contains(line, "note: run with `RUST_BACKTRACE=1`") {
				log.Info("wq 崩溃，可能的触发条件：在没有网络的时候请求二维码。")
				pa.WalleQState = WqStateCodeLoginFailed
			}

			if strings.Contains(line, "输入ticket:") {
				log.Info("需要")
			}

			if strings.Contains(line, "Walle-Q Login success with") {
				go ServeQQ(dice, conn)
			}
		}

		if strings.Contains(line, "扫描二维码登录") { //nolint
			// TODO
		}
		return line
	}

	go func() {
		<-chQrCode
		if _, err := os.Stat(qrcodeFile); err == nil {
			log.Info("如控制台二维码不好扫描，可以手动打开 ./data/default/extra/walle-q数字 目录下qrcode.png")
			qrdata, err := os.ReadFile(qrcodeFile)
			if err == nil {
				pa.WalleQState = WqStateCodeInLoginQrCode
				pa.WalleQQrcodeData = qrdata
				dice.Logger.Info("获取二维码成功")
				_ = os.Rename(qrcodeFile, qrcodeFile+".bak.png")
				dice.LastUpdatedTime = time.Now().Unix()
				dice.Save(false)
			} else {
				pa.WalleQQrcodeData = nil
				pa.WalleQState = WqStateCodeLoginFailed
				pa.WalleQLoginFailedReason = "获取二维码失败"
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
		pa.WalleQProcess = p
		err := p.Start()

		if err == nil {
			if dice.Parent.progressExitGroupWin != 0 && p.Cmd != nil {
				err = dice.Parent.progressExitGroupWin.AddProcess(p.Cmd.Process)
				if err != nil {
					dice.Logger.Warn("添加到进程组失败，若主进程崩溃，walle-q 进程可能需要手动结束")
				}
			}
			fmt.Println("wait！")
			err = p.Wait()
			fmt.Println(err)
		}

		if err != nil {
			dice.Logger.Info("walle-q 进程退出: ", err)
		} else {
			dice.Logger.Info("进程退出", nil)
		}
	}

	if isAsyncRun {
		go run()
	} else {
		run()
	}
}

func (pa *PlatformAdapterWalleQ) SetQQProtocol(protocol int) bool {
	pa.InPackWalleQProtocol = protocol
	uid := pa.EndPoint.UserID
	log := pa.EndPoint.Session.Parent.Logger

	wd := filepath.Join(pa.Session.Parent.BaseConfig.DataDir, pa.EndPoint.RelWorkDir)
	wqc := new(WalleQConfig)
	_, err := toml.DecodeFile(wd, wqc)
	if err != nil {
		log.Error("读取 Walle-q 配置文件失败，请检查！")
		return false
	}
	wqc.QQ[uid] = WqQQ{
		wqc.QQ[uid].Password, protocol,
	}
	b := new(bytes.Buffer)
	_ = toml.NewEncoder(b).Encode(wqc)
	_ = os.WriteFile(wd, b.Bytes(), 0o644)
	return true
}

func (pa *PlatformAdapterWalleQ) IsInLogin() bool {
	return pa.WalleQState < WqStateCodeLoginSuccessed
}

func (pa *PlatformAdapterWalleQ) IsLoginSuccessed() bool {
	return pa.WalleQState == WqStateCodeLoginSuccessed
}
