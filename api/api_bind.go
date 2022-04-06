package api

import (
	"encoding/base64"
	"net/http"
	"runtime"
	"sealdice-core/dice"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const CODE_ALREADY_EXISTS = 602

var startTime = time.Now().Unix()

func baseInfo(c echo.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return c.JSON(http.StatusOK, struct {
		AppName       string `json:"appName"`
		Version       string `json:"version"`
		MemoryAlloc   uint64 `json:"memoryAlloc"`
		Uptime        int64  `json:"uptime"`
		MemoryUsedSys uint64 `json:"memoryUsedSys"`
	}{
		AppName:       dice.APPNAME,
		Version:       dice.VERSION,
		MemoryAlloc:   m.Alloc,
		MemoryUsedSys: m.Sys,
		Uptime:        time.Now().Unix() - startTime,
	})
}

func hello2(c echo.Context) error {
	//dice.CmdRegister("aaa", "bb");
	//a := dice.CmdList();
	//b, _ := json.Marshal(a)
	return c.String(http.StatusOK, string(""))
}

var myDice *dice.Dice
var dm *dice.DiceManager

func customText(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"texts":    myDice.TextMapRaw,
		"helpInfo": myDice.TextMapHelpInfo,
	})
}

func customTextSave(c echo.Context) error {
	v := struct {
		Category string                      `form:"category" json:"category"`
		Data     dice.TextTemplateWithWeight `form:"data" json:"data"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, v1 := range v.Data {
			for _, v2 := range v1 {
				v2[1] = int(v2[1].(float64))
			}
		}
		myDice.TextMapRaw[v.Category] = v.Data
		dice.SetupTextHelpInfo(myDice, myDice.TextMapHelpInfo, myDice.TextMapRaw, "configs/text-template.yaml")
		myDice.GenerateTextMap()
		myDice.SaveText()
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}

func ImConnections(c echo.Context) error {
	return c.JSON(http.StatusOK, myDice.ImSession.EndPoints)
}

func ImConnectionsGet(c echo.Context) error {
	v := struct {
		Id string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				return c.JSON(http.StatusOK, i)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsSetEnable(c echo.Context) error {
	v := struct {
		Id     string `form:"id" json:"id"`
		Enable bool   `form:"enable" json:"enable"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				i.SetEnable(myDice, v.Enable)
				return c.JSON(http.StatusOK, i)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsDel(c echo.Context) error {
	v := struct {
		Id string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for index, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				// TODO: 注意 这个好像很不科学
				//i.DiceServing = false
				dice.GoCqHttpServeProcessKill(myDice, i)
				myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
				return c.JSON(http.StatusOK, i)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsQrcodeGet(c echo.Context) error {
	v := struct {
		Id string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				pa := i.Adapter.(*dice.PlatformAdapterQQOnebot)
				if pa.InPackGoCqHttpQrcodeReady {
					return c.JSON(http.StatusOK, map[string]string{
						"img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.InPackGoCqHttpQrcodeData),
					})
				}
				return c.JSON(http.StatusOK, i)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsGocqhttpRelogin(c echo.Context) error {
	v := struct {
		Id string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				i.Adapter.DoRelogin()
				return c.JSON(http.StatusOK, nil)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsAdd(c echo.Context) error {
	v := struct {
		Account  string `yaml:"account" json:"account"`
		Password string `yaml:"password" json:"password"`
		Protocol int    `json:"protocol"`
		//ConnectUrl        string `yaml:"connectUrl" json:"connectUrl"`               // 连接地址
		//Platform          string `yaml:"platform" json:"platform"`                   // 平台，如QQ、QQ频道
		//Enable            bool   `yaml:"enable" json:"enable"`                       // 是否启用
		//Type              string `yaml:"type" json:"type"`                           // 协议类型，如onebot、koishi等
		//UseInPackGoCqhttp bool   `yaml:"useInPackGoCqhttp" json:"useInPackGoCqhttp"` // 是否使用内置的gocqhttp
	}{}

	err := c.Bind(&v)
	if err == nil {
		uid, err := strconv.ParseInt(v.Account, 10, 64)
		if err != nil {
			return c.String(430, "")
		}

		for _, i := range myDice.ImSession.EndPoints {
			if i.UserId == dice.FormatDiceIdQQ(uid) {
				return c.JSON(CODE_ALREADY_EXISTS, i)
			}
		}

		conn := dice.NewGoCqhttpConnectInfoItem(v.Account)
		conn.UserId = dice.FormatDiceIdQQ(uid)
		pa := conn.Adapter.(*dice.PlatformAdapterQQOnebot)
		pa.InPackGoCqHttpProtocol = v.Protocol
		pa.InPackGoCqHttpPassword = v.Password
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		dice.GoCqHttpServe(myDice, conn, v.Password, v.Protocol, true)
		myDice.Save(false)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func logFetchAndClear(c echo.Context) error {
	info := c.JSON(http.StatusOK, myDice.LogWriter.Items)
	//myDice.LogWriter.Items = myDice.LogWriter.Items[:0]
	return info
}

var lastExecTime int64

func DiceExec(c echo.Context) error {
	v := struct {
		Id      string `form:"id" json:"id"`
		Message string `form:"message"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.JSON(400, "格式错误")
	}
	if v.Message == "" {
		return c.JSON(400, "格式错误")
	}
	now := time.Now().Unix()
	if now-lastExecTime < 3 {
		return c.JSON(400, "过于频繁")
	}
	lastExecTime = now

	pa := dice.PlatformAdapterHttp{}
	tmpEp := &dice.EndPointInfo{
		EndPointInfoBase: dice.EndPointInfoBase{
			Id:       "1",
			Nickname: "海豹核心",
			State:    2,
			UserId:   "UI:1000",
			Platform: "UI",
			Enable:   true,
		},
		Adapter: &pa,
	}
	msg := &dice.Message{
		MessageType: "private",
		Message:     v.Message,
		Sender: dice.SenderBase{
			Nickname: "User",
			UserId:   "UI:1001",
		},
	}
	myDice.ImSession.Execute(tmpEp, msg, true)
	return c.JSON(200, pa.RecentMessage)
}

type DiceConfigInfo struct {
	CommandPrefix           []string `json:"commandPrefix"`           // 指令前导
	DiceMasters             []string `json:"diceMasters"`             // 骰主设置，需要格式: 平台:帐号
	OnlyLogCommandInGroup   bool     `json:"onlyLogCommandInGroup"`   // 日志中仅记录命令
	OnlyLogCommandInPrivate bool     `json:"onlyLogCommandInPrivate"` // 日志中仅记录命令
	MessageDelayRangeStart  float64  `json:"messageDelayRangeStart"`  // 指令延迟区间
	MessageDelayRangeEnd    float64  `json:"messageDelayRangeEnd"`
	UIPassword              string   `json:"uiPassword"`
	MasterUnlockCode        string   `json:"masterUnlockCode"`
}

func DiceConfig(c echo.Context) error {
	password := myDice.UIPassword
	p2 := []string{}
	if len(password) > 0 {
		for _ = range password {
			p2 = append(p2, "*")
		}
		password = strings.Join(p2, "")
	}

	info := DiceConfigInfo{
		CommandPrefix:           myDice.CommandPrefix,
		DiceMasters:             myDice.DiceMasters,
		OnlyLogCommandInPrivate: myDice.OnlyLogCommandInPrivate,
		OnlyLogCommandInGroup:   myDice.OnlyLogCommandInGroup,
		MessageDelayRangeStart:  myDice.MessageDelayRangeStart,
		MessageDelayRangeEnd:    myDice.MessageDelayRangeEnd,
		UIPassword:              password,
		MasterUnlockCode:        myDice.MasterUnlockCode,
	}
	return c.JSON(http.StatusOK, info)
}

func DiceConfigSet(c echo.Context) error {
	return c.JSON(http.StatusOK, nil)
}

func Bind(e *echo.Echo, _myDice *dice.DiceManager) {
	dm = _myDice
	myDice = _myDice.Dice[0]
	e.GET("/baseInfo", baseInfo)
	e.GET("/cmd/register", hello2)
	e.GET("/log/fetchAndClear", logFetchAndClear)
	e.GET("/configs/customText", customText)
	e.GET("/im_connections/list", ImConnections)
	e.GET("/im_connections/get", ImConnectionsGet)

	e.POST("/im_connections/qrcode", ImConnectionsQrcodeGet)
	e.POST("/configs/customText/save", customTextSave)
	e.POST("/im_connections/add", ImConnectionsAdd)
	e.POST("/im_connections/del", ImConnectionsDel)
	e.POST("/im_connections/set_enable", ImConnectionsSetEnable)
	e.POST("/im_connections/gocqhttpRelogin", ImConnectionsGocqhttpRelogin)

	e.GET("/dice/config/get", DiceConfig)
	e.POST("/dice/config/set", DiceConfigSet)
	e.POST("/dice/exec", DiceExec)
}
