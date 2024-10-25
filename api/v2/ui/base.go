package ui

import (
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"sealdice-core/api/v2/model"
	"sealdice-core/dice"
	"sealdice-core/utils/web/response"
)

// TODO: 注释现在不一定对的上地址，这个还得接着改喵

var startTime = time.Now().Unix()

type fStopEcho struct {
	Key string `json:"key"`
}

// BaseApi
// 曾经这里直接嵌入了Dice，但是后来考虑到这个东西应该是和BaseApi分开
// BaseApi不应该能用Dice的方法（而是通过方法控制Dice）
type BaseApi struct {
	dice *dice.Dice
}

// PreInfo
// @Tags      Base
// @Summary   获取测试模式信息
// @Security  ApiKeyAuth
// @Accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Result{data=map[string]interface{},msg=string}  "返回测试模式信息"
// @Router    /base/preInfo [post]
func (b *BaseApi) PreInfo(c echo.Context) error {
	return response.OkWithData(map[string]interface{}{
		"testMode": b.dice.Parent.JustForTest,
	}, c)
}

// BaseInfo
// @Tags      Base
// @Summary   获取基础信息
// @Security  ApiKeyAuth
// @Accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Result{data=model.BaseInfoResponse,msg=string}  "返回基础信息，包括应用名称、版本、内存使用等"
// @Router    /base/baseinfo [get]
func (b *BaseApi) BaseInfo(c echo.Context) error {
	// 鉴权后使用
	dm := b.dice.Parent
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var versionNew string
	var versionNewNote string
	var versionNewCode int64
	if dm.AppVersionOnline != nil {
		versionNew = dm.AppVersionOnline.VersionLatestDetail
		versionNewNote = dm.AppVersionOnline.VersionLatestNote
		versionNewCode = dm.AppVersionOnline.VersionLatestCode
	}

	extraTitle := b.getName()

	versionDetail := model.VersionDetail{
		Major:         dice.VERSION.Major(),
		Minor:         dice.VERSION.Minor(),
		Patch:         dice.VERSION.Patch(),
		Prerelease:    dice.VERSION.Prerelease(),
		BuildMetaData: dice.VERSION.Metadata(),
	}
	infoResponse := model.BaseInfoResponse{
		AppName:        dice.APPNAME,
		AppChannel:     dice.APP_CHANNEL,
		Version:        dice.VERSION.String(),
		VersionSimple:  dice.VERSION_MAIN + dice.VERSION_PRERELEASE,
		VersionDetail:  versionDetail,
		VersionNew:     versionNew,
		VersionNewNote: versionNewNote,
		VersionCode:    dice.VERSION_CODE,
		VersionNewCode: versionNewCode,
		MemoryAlloc:    m.Alloc,
		MemoryUsedSys:  m.Sys - m.HeapReleased,
		Uptime:         time.Now().Unix() - startTime,
		ExtraTitle:     extraTitle,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		JustForTest:    dm.JustForTest,
		ContainerMode:  dm.ContainerMode,
	}

	return response.OkWithData(infoResponse, c)
}

// ForceStop
// @Tags      Base
// @Summary   安卓专属：强制停止程序
// @Security  ApiKeyAuth
// @Accept    application/json
// @Produce   application/json
// @Param     key  body  fStopEcho  true  "强制停止的密钥"
// @Success   200  {object}  response.Result{msg=string}  "执行成功"
// @Failure   400  {object}  response.Result{msg=string}  "参数绑定错误/执行错误/Key不匹配等等"
// @Router    /base/force-stop [post]
func (b *BaseApi) ForceStop(c echo.Context) error {
	// 此处不再判断是否为安卓，直接控制若是安卓再注册对应API即可
	defer b.cleanupAndExit()
	// this is a dangerous api, so we need to check the key
	haskey := false
	for _, s := range os.Environ() {
		if strings.HasPrefix(s, "FSTOP_KEY=") {
			key := strings.Split(s, "=")[1]
			v := fStopEcho{}
			err := c.Bind(&v)
			if err != nil {
				return response.FailWithMessage(response.GetGenericErrorMsg(err), c)
			}
			if v.Key == key {
				haskey = true
				break
			} else {
				return response.FailWithMessage("检查到FSTOP_KEY不对应，停止执行", c)
			}
		}
	}
	if !haskey {
		return response.FailWithMessage("检查到环境中不含有FSTOP_KEY，停止执行", c)
	}
	return response.OkWithMessage("执行成功", c)
}

// HeartBeat
// @Tags      Base
// @Summary   心跳检测
// @Security  ApiKeyAuth
// @Accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Result{msg=string}  "返回心跳检测信息"
// @Router    /base/heartbeat [get]
func (b *BaseApi) HeartBeat(c echo.Context) error {
	// 需要鉴权
	return response.OkWithMessage("HEARTBEATS", c)
}

// CheckSecurity
// @Tags      Base
// @Summary   检查是否需要进行安全提醒
// @Security  ApiKeyAuth
// @Accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Result{data=map[string]bool,msg=string}  "返回安全检查结果"
// @Router    /base/security/check [get]
func (b *BaseApi) CheckSecurity(c echo.Context) error {
	// 需要鉴权
	isPublicService := strings.HasPrefix(b.dice.Parent.ServeAddress, "0.0.0.0") || b.dice.Parent.ServeAddress == ":3211"
	isEmptyPassword := b.dice.Parent.UIPasswordHash == ""
	return response.OkWithData(map[string]bool{
		"isOk": !(isEmptyPassword && isPublicService),
	}, c)
}

// 工具函数
// cleanupAndExit 清理资源并退出程序
func (b *BaseApi) cleanupAndExit() {
	logger := b.dice.Logger
	logger.Info("程序即将退出，进行清理……")

	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("异常: %v\n堆栈: %v", err, string(debug.Stack()))
		}
	}()
	// TODO：已经是单例模式了，这里其实不需要这么套娃
	for _, i := range b.dice.Parent.Dice {
		if i.IsAlreadyLoadConfig {
			i.Config.BanList.SaveChanged(i)
			i.AttrsManager.CheckForSave()
			i.Save(true)

			// 关闭存储
			for _, j := range i.ExtList {
				if j.Storage != nil {
					if err := j.StorageClose(); err != nil {
						logger.Errorf("异常: %v\n堆栈: %v", err, string(debug.Stack()))
					}
				}
			}
			i.IsAlreadyLoadConfig = false
		}

		b.closeDBConnections()
	}

	// 清理gocqhttp
	for _, i := range b.dice.Parent.Dice {
		if i.ImSession != nil && i.ImSession.EndPoints != nil {
			for _, j := range i.ImSession.EndPoints {
				dice.BuiltinQQServeProcessKill(i, j)
			}
		}
	}

	if b.dice.Parent.Help != nil {
		b.dice.Parent.Help.Close()
	}
	if b.dice.Parent.IsReady {
		b.dice.Parent.Save()
	}
	if b.dice.Parent.Cron != nil {
		b.dice.Parent.Cron.Stop()
	}
	logger.Info("清理完成，程序即将退出")
	os.Exit(0) //nolint:gocritic
}

// closeDBConnections 关闭数据库连接
func (b *BaseApi) closeDBConnections() {
	diceInstance := b.dice
	closeConnection := func(db *sqlx.DB) {
		if db != nil {
			_ = db.Close()
		}
	}

	closeConnection(diceInstance.DBData)
	closeConnection(diceInstance.DBLogs)

	if cm := diceInstance.CensorManager; cm != nil {
		closeConnection(cm.DB)
	}
}

func (b *BaseApi) getName() string {
	defer func() {
		// 防止报错
		_ = recover()
	}()

	ctx := &dice.MsgContext{Dice: b.dice, EndPoint: nil, Session: b.dice.ImSession}
	return dice.DiceFormatTmpl(ctx, "核心:骰子名字")
}
