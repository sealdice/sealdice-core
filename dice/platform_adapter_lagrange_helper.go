package dice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"

	"sealdice-core/utils/procs"
)

func lagrangeGetWorkDir(dice *Dice, conn *EndPointInfo) string {
	workDir := filepath.Join(dice.BaseConfig.DataDir, conn.RelWorkDir)
	return workDir
}

func NewLagrangeConnectInfoItem(account string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "QQ"
	conn.ProtocolType = "onebot"
	conn.Enable = false
	conn.RelWorkDir = "extra/lagrange-qq" + account

	conn.Adapter = &PlatformAdapterGocq{
		EndPoint:          conn,
		UseInPackGoCqhttp: true,
		BuiltinMode:       "lagrange",
	}
	return conn
}
func LagrangeServe(dice *Dice, conn *EndPointInfo, loginInfo GoCqhttpLoginInfo) {
	pa := conn.Adapter.(*PlatformAdapterGocq)

	pa.CurLoginIndex++
	loginIndex := pa.CurLoginIndex
	pa.GoCqhttpState = StateCodeInLogin

	if pa.UseInPackGoCqhttp && pa.BuiltinMode == "lagrange" {
		workDir := lagrangeGetWorkDir(dice, conn)
		_ = os.MkdirAll(workDir, 0o755)

		exeFilePath, _ := filepath.Abs(filepath.Join(workDir, "Lagrange.OneBot"))
		exeFilePath = strings.ReplaceAll(exeFilePath, "\\", "/") // windows平台需要这个替换
		if runtime.GOOS == "windows" {
			exeFilePath += ".exe"
		}
		qrcodeFilePath := filepath.Join(workDir, "qr-0.png")
		configFilePath := filepath.Join(workDir, "appsettings.json")
		deviceFilePath := filepath.Join(workDir, "device.json")
		keystoreFilePath := filepath.Join(workDir, "keystore.json")

		log := dice.Logger
		if _, err := os.Stat(qrcodeFilePath); err == nil {
			// 如果已经存在二维码文件，将其删除
			_ = os.Remove(qrcodeFilePath)
			log.Info("onebot: 删除已存在的二维码文件")
		}

		if !pa.GoCqhttpLoginSucceeded {
			// 并未登录成功，删除记录文件
			_ = os.Remove(exeFilePath)
			_ = os.Remove(configFilePath)
			_ = os.Remove(deviceFilePath)
			_ = os.Remove(keystoreFilePath)
		}

		// 创建配置文件
		if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
			// 如果不存在，进行创建
			p, _ := GetRandomFreePort()
			pa.ConnectURL = fmt.Sprintf("ws://localhost:%d", p)
			c := GenerateLagrangeConfig(p, loginInfo)
			_ = os.WriteFile(configFilePath, []byte(c), 0o644)
		}

		if _, err := os.Stat(exeFilePath); errors.Is(err, os.ErrNotExist) {
			// 拷贝一份到实际目录
			wd, _ := os.Getwd()
			lagrangeExePath, _ := filepath.Abs(filepath.Join(wd, "lagrange/Lagrange.OneBot"))
			if runtime.GOOS == "windows" {
				lagrangeExePath += ".exe"
			}
			lagrangeExePath = strings.ReplaceAll(lagrangeExePath, "\\", "/") // windows平台需要这个替换
			lagrangeExe, err := os.OpenFile(lagrangeExePath, os.O_RDONLY, 0o644)
			if err != nil {
				log.Error("onebot: 找不到 Lagrange.OneBot")
				return
			}
			dst, err := os.OpenFile(exeFilePath, os.O_WRONLY|os.O_CREATE, 0o755)

			writer := bufio.NewWriter(dst)
			_, _ = io.Copy(writer, lagrangeExe)

			lagrangeExe.Close()
			dst.Close()
		}

		// 启动客户端
		_ = os.Chmod(exeFilePath, 0o755)
		log.Info("onebot: 正在启动 onebot 客户端…… ", exeFilePath)

		p := procs.NewProcess(exeFilePath)
		p.Dir = workDir

		if runtime.GOOS == "android" {
			p.Env = os.Environ()
		}

		chQrCode := make(chan int, 1)
		isSeldKilling := false

		p.OutputHandler = func(line string) string {
			if loginIndex != pa.CurLoginIndex {
				// 当前连接已经无用，进程自杀
				if !isSeldKilling {
					log.Infof("检测到新的连接序号 %d，当前连接 %d 将自动退出", pa.CurLoginIndex, loginIndex)
					// 注: 这里不要调用kill
					isSeldKilling = true
					_ = p.Stop()
				}
				return ""
			}

			// 登录中
			if pa.IsInLogin() {
				// 读取二维码
				if strings.Contains(line, "QrCode Fetched") {
					chQrCode <- 1
				}

				// 登录成功
				if strings.Contains(line, "Success") {
					pa.GoCqhttpState = StateCodeLoginSuccessed
					pa.GoCqhttpLoginSucceeded = true
					log.Infof("onebot: 登录成功，账号：<%s>(%s)", conn.Nickname, conn.UserID)
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)

					go ServeQQ(dice, conn)
				}
			}

			return ""
		}

		go func() {
			<-chQrCode
			time.Sleep(3 * time.Second)
			if _, err := os.Stat(qrcodeFilePath); err == nil {
				log.Info("onebot: 二维码已就绪")
				qrdata, err := os.ReadFile(qrcodeFilePath)
				if err == nil {
					pa.GoCqhttpState = StateCodeInLoginQrCode
					pa.GoCqhttpQrcodeData = qrdata
					log.Info("onebot: 读取二维码成功")
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)
				} else {
					pa.GoCqhttpState = StateCodeLoginFailed
					pa.GoCqhttpQrcodeData = nil
					pa.GocqhttpLoginFailedReason = "读取二维码失败"
					dice.LastUpdatedTime = time.Now().Unix()
					dice.Save(false)
					log.Infof("onebot: 读取二维码失败：%s", err)
				}
			}
		}()

		run := func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("onebot: 异常: %v 堆栈: %v", r, string(debug.Stack()))
				}
			}()

			pa.GoCqhttpProcess = p
			err := p.Start()

			if err == nil {
				if dice.Parent.progressExitGroupWin != 0 && p.Cmd != nil {
					errAdd := dice.Parent.progressExitGroupWin.AddProcess(p.Cmd.Process)
					if errAdd != nil {
						dice.Logger.Warn("添加到进程组失败，若主进程崩溃，lagrange 进程可能需要手动结束")
					}
				}
				_ = p.Wait()
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
				log.Info("lagrange 进程异常退出: ", err)
			} else {
				log.Info("lagrange 进程退出")
			}
		}

		if loginInfo.IsAsyncRun {
			go run()
		} else {
			run()
		}

	} else {
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
        "Uin": 0,
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

// 在构建时注入
var defaultNTSignServer = ``

func GenerateLagrangeConfig(port int, info GoCqhttpLoginInfo) string {
	conf := strings.ReplaceAll(defaultLagrangeConfig, "{WS端口}", fmt.Sprintf("%d", port))
	conf = strings.ReplaceAll(conf, "{NTSignServer地址}", defaultNTSignServer)
	return conf
}

func LagrangeServeProcessKill(dice *Dice, conn *EndPointInfo) {
	defer func() {
		defer func() {
			if r := recover(); r != nil {
				dice.Logger.Error("lagrange 清理报错: ", r)
				// lagrange 进程退出: exit status 1
			}
		}()

		pa, ok := conn.Adapter.(*PlatformAdapterGocq)
		if !ok {
			return
		}
		if pa.UseInPackGoCqhttp && pa.BuiltinMode == "lagrange" {
			// 重置状态
			conn.State = 0
			pa.GoCqhttpState = 0

			pa.DiceServing = false
			pa.GoCqhttpQrcodeData = nil

			workDir := lagrangeGetWorkDir(dice, conn)
			qrcodeFile := filepath.Join(workDir, "qr-0.png")
			if _, err := os.Stat(qrcodeFile); err == nil {
				// 如果已经存在二维码文件，将其删除
				_ = os.Remove(qrcodeFile)
				dice.Logger.Info("onebot: 删除已存在的二维码文件")
			}

			// 注意这个会panic，因此recover捕获了
			if pa.GoCqhttpProcess != nil {
				p := pa.GoCqhttpProcess
				pa.GoCqhttpProcess = nil
				// sigintwindows.SendCtrlBreak(p.Cmds[0].Process.Pid)
				_ = p.Stop()
				_ = p.Wait() // 等待进程退出，因为Stop内部是Kill，这是不等待的
			}
		}
	}()
}

func LagrangeServeRemoveSession(dice *Dice, conn *EndPointInfo) {
	workDir := gocqGetWorkDir(dice, conn)
	if _, err := os.Stat(filepath.Join(workDir, "keystore.json")); err == nil {
		_ = os.Remove(filepath.Join(workDir, "keystore.json"))
	}
}
