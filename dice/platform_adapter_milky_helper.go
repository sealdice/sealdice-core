package dice

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"sealdice-core/logger"
	"sealdice-core/utils/procs"
)

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
				// TODO: kill process
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
	milkyExePath, _ := filepath.Abs(filepath.Join(diceWorkdir, "milky/milky"))
	milkyExePath = filepath.ToSlash(milkyExePath) // windows平台需要这个替换
	if runtime.GOOS == "windows" {
		milkyExePath += ".exe" //nolint:ineffassign
	}
	_ = os.MkdirAll(workDir, 0o755)
	// TODO: generate config file
	command := fmt.Sprintf(`"%s"`, milkyExePath)
	p := procs.NewProcess(command)
	p.Dir = workDir
	p.Env = []string{
		fmt.Sprintf("APP_LAUNCHER_SIG=%s", BuildSignature(uint64(uin))),
	}
	chQrCode := make(chan int, 1)
	qrcodeFilePath := filepath.Join(workDir, "qrcode.png")
	pa.BuiltInLoginState = MilkyLoginStateInit
	p.OutputHandler = func(line string, _type string) string {
		// 登录中
		if pa.BuiltInLoginState < MilkyLoginStateConnecting {
			qrcodeSignal := "Fetch QrCode Success"
			onlineSignal := "successfully logged in"
			qrcodeExpiredSignal := "QrCode Expired, Please Fetch QrCode Again"
			// 读取二维码
			if strings.Contains(line, qrcodeSignal) {
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
				// TODO: kill process
			}
		}

		if _type == "stderr" {
			log.Error("Milky Internal: ", strings.TrimSpace(line))
		} else {
			log.Info("Milky Internal: ", strings.TrimSpace(line))
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
