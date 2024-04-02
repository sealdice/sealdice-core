package dice

import (
	"errors"
	"fmt"
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

	if pa.UseInPackGoCqhttp && pa.BuiltinMode == "lagrange" { //nolint:nestif
		workDir := lagrangeGetWorkDir(dice, conn)
		_ = os.MkdirAll(workDir, 0o755)
		wd, _ := os.Getwd()
		exeFilePath, _ := filepath.Abs(filepath.Join(wd, "lagrange/Lagrange.OneBot"))
		exeFilePath = filepath.ToSlash(exeFilePath) // windows平台需要这个替换
		if runtime.GOOS == "windows" {
			exeFilePath += ".exe"
		}
		qrcodeFilePath := filepath.Join(workDir, fmt.Sprintf("qr-%s.png", conn.UserID[3:]))
		configFilePath := filepath.Join(workDir, "appsettings.json")

		log := dice.Logger
		if _, err := os.Stat(qrcodeFilePath); err == nil {
			// 如果已经存在二维码文件，将其删除
			_ = os.Remove(qrcodeFilePath)
			log.Info("onebot: 删除已存在的二维码文件")
		}

		// 创建配置文件
		if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
			// 如果不存在，进行创建
			p, _ := GetRandomFreePort()
			pa.ConnectURL = fmt.Sprintf("ws://127.0.0.1:%d", p)
			c := GenerateLagrangeConfig(p, conn)
			_ = os.WriteFile(configFilePath, []byte(c), 0o644)
		}

		// 启动客户端
		_ = os.Chmod(exeFilePath, 0o755)

		command := ""
		if runtime.GOOS == "android" {
			for i, s := range os.Environ() {
				if strings.HasPrefix(s, "RUNNER_PATH=") {
					log.Infof("RUNNER_PATH: %s", os.Environ()[i][12:])
					command = os.Environ()[i][12:]
					break
				}
			}
			command, _ = filepath.Abs(filepath.Join(command, "start.sh"))
			_ = os.Chmod(command, 0o755)
			command += " " + exeFilePath
		} else {
			command = exeFilePath
		}
		log.Info("onebot: 正在启动 onebot 客户端…… ", command)
		conn.State = 2
		conn.Enable = true
		p := procs.NewProcess(command)
		p.Dir = workDir

		chQrCode := make(chan int, 1)
		isSelfKilling := false

		p.OutputHandler = func(line string) string {
			if loginIndex != pa.CurLoginIndex {
				// 当前连接已经无用，进程自杀
				if !isSelfKilling {
					log.Infof("检测到新的连接序号 %d，当前连接 %d 将自动退出", pa.CurLoginIndex, loginIndex)
					// 注: 这里不要调用kill
					isSelfKilling = true
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

				log.Warn("onebot | ", line)
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
	} else if !pa.UseInPackGoCqhttp {
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

// 在构建时注入
var defaultNTSignServer = `https://lwxmagic.sealdice.com/api/sign`

func GenerateLagrangeConfig(port int, info *EndPointInfo) string {
	conf := strings.ReplaceAll(defaultLagrangeConfig, "{WS端口}", fmt.Sprintf("%d", port))
	conf = strings.ReplaceAll(conf, "{NTSignServer地址}", defaultNTSignServer)
	conf = strings.ReplaceAll(conf, "{账号UIN}", info.UserID[3:])
	return conf
}

func LagrangeServeRemoveSession(dice *Dice, conn *EndPointInfo) {
	workDir := gocqGetWorkDir(dice, conn)
	if _, err := os.Stat(filepath.Join(workDir, "keystore.json")); err == nil {
		_ = os.Remove(filepath.Join(workDir, "keystore.json"))
	}
}
