package api

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	//dice.CmdRegister("aaa", "bb");
	//a := dice.CmdList();
	//b, _ := json.Marshal(a)
	return c.JSON(http.StatusOK, nil)
}

var myDice *dice.Dice
var dm *dice.DiceManager

func ImConnections(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, myDice.ImSession.EndPoints)
}

func ImConnectionsGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		dice.GoCqHttpServe(myDice, conn, v.Password, v.Protocol, true)
		myDice.Save(false)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func doSignInGetSalt(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"salt": myDice.Parent.UIPasswordSalt,
	})
}

func checkSecurity(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	isPublicService := strings.HasPrefix(myDice.Parent.ServeAddress, "0.0.0.0") || myDice.Parent.ServeAddress == ":3211"
	isEmptyPassword := myDice.Parent.UIPasswordHash == ""
	return c.JSON(200, map[string]bool{
		"isOk": !(isEmptyPassword && isPublicService),
	})
}

func doSignIn(c echo.Context) error {
	v := struct {
		Password string `json:"password"`
	}{}

	err := c.Bind(&v)
	if err != nil {
		return c.JSON(400, nil)
	}

	generateToken := func() error {
		now := time.Now().Unix()
		head := hex.EncodeToString(Int64ToBytes(now))
		token := dice.RandStringBytesMaskImprSrcSB2(64) + ":" + head

		myDice.Parent.AccessTokens[token] = true
		myDice.Parent.Save()
		return c.JSON(http.StatusOK, map[string]string{
			"token": token,
		})
	}

	if myDice.Parent.UIPasswordHash == "" {
		return generateToken()
	}

	if myDice.Parent.UIPasswordHash == v.Password {
		return generateToken()
	}

	return c.JSON(400, nil)
}

func logFetchAndClear(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	info := c.JSON(http.StatusOK, myDice.LogWriter.Items)
	//myDice.LogWriter.Items = myDice.LogWriter.Items[:0]
	return info
}

var lastExecTime int64

func DiceExec(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

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
	now := time.Now().UnixMilli()
	if now-lastExecTime < 500 {
		return c.JSON(400, "过于频繁")
	}
	lastExecTime = now

	pa := dice.PlatformAdapterHttp{
		RecentMessage: []dice.HttpSimpleMessage{},
	}
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
	// 注：form其实不需要
	CommandPrefix           []string `json:"commandPrefix" form:"commandPrefix"`                     // 指令前缀
	DiceMasters             []string `json:"diceMasters" form:"diceMasters"`                         // 骰主设置，需要格式: 平台:帐号
	OnlyLogCommandInGroup   bool     `json:"onlyLogCommandInGroup" form:"onlyLogCommandInGroup"`     // 日志中仅记录命令
	OnlyLogCommandInPrivate bool     `json:"onlyLogCommandInPrivate" form:"onlyLogCommandInPrivate"` // 日志中仅记录命令
	WorkInQQChannel         bool     `json:"workInQQChannel"`                                        // 在QQ频道中开启
	MessageDelayRangeStart  float64  `json:"messageDelayRangeStart" form:"messageDelayRangeStart"`   // 指令延迟区间
	MessageDelayRangeEnd    float64  `json:"messageDelayRangeEnd" form:"messageDelayRangeEnd"`
	UIPassword              string   `json:"uiPassword" form:"uiPassword"`
	HelpDocEngineType       int      `json:"helpDocEngineType"`
	MasterUnlockCode        string   `json:"masterUnlockCode" form:"masterUnlockCode"`
	ServeAddress            string   `json:"serveAddress" form:"serveAddress"`
	MasterUnlockCodeTime    int64    `json:"masterUnlockCodeTime"`
	LogPageItemLimit        int64    `json:"logPageItemLimit"`
	FriendAddComment        string   `json:"friendAddComment"`
}

func DiceConfig(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	password := ""
	if myDice.Parent.UIPasswordHash != "" {
		password = "------"
	}

	limit := myDice.UILogLimit
	if limit == 0 {
		limit = 100
	}
	myDice.UnlockCodeUpdate()

	info := DiceConfigInfo{
		CommandPrefix:           myDice.CommandPrefix,
		DiceMasters:             myDice.DiceMasters,
		OnlyLogCommandInPrivate: myDice.OnlyLogCommandInPrivate,
		OnlyLogCommandInGroup:   myDice.OnlyLogCommandInGroup,
		MessageDelayRangeStart:  myDice.MessageDelayRangeStart,
		MessageDelayRangeEnd:    myDice.MessageDelayRangeEnd,
		UIPassword:              password,
		MasterUnlockCode:        myDice.MasterUnlockCode,
		MasterUnlockCodeTime:    myDice.MasterUnlockCodeTime,
		WorkInQQChannel:         myDice.WorkInQQChannel,
		LogPageItemLimit:        limit,
		FriendAddComment:        myDice.FriendAddComment,

		ServeAddress:      myDice.Parent.ServeAddress,
		HelpDocEngineType: myDice.Parent.HelpDocEngineType,
	}
	return c.JSON(http.StatusOK, info)
}

func DiceConfigSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	jsonMap := make(map[string]interface{})
	err := json.NewDecoder(c.Request().Body).Decode(&jsonMap)

	stringConvert := func(val interface{}) []string {
		var lst []string
		for _, i := range val.([]interface{}) {
			t := i.(string)
			if t != "" {
				lst = append(lst, t)
			}
		}
		return lst
	}

	if err == nil {
		if val, ok := jsonMap["commandPrefix"]; ok {
			myDice.CommandPrefix = stringConvert(val)
		}

		if val, ok := jsonMap["diceMasters"]; ok {
			myDice.DiceMasters = stringConvert(val)
		}

		if val, ok := jsonMap["onlyLogCommandInGroup"]; ok {
			myDice.OnlyLogCommandInGroup = val.(bool)
		}

		if val, ok := jsonMap["onlyLogCommandInPrivate"]; ok {
			myDice.OnlyLogCommandInPrivate = val.(bool)
		}

		if val, ok := jsonMap["workInQQChannel"]; ok {
			myDice.WorkInQQChannel = val.(bool)
		}

		if val, ok := jsonMap["UILogLimit"]; ok {
			val, err := strconv.ParseInt(val.(string), 10, 64)
			if err == nil {
				if val >= 0 {
					myDice.LogWriter.LogLimit = val
				}
			}
		}

		if val, ok := jsonMap["messageDelayRangeStart"]; ok {
			f, err := strconv.ParseFloat(val.(string), 64)
			if err == nil {
				if f < 0 {
					f = 0
				}
				if myDice.MessageDelayRangeEnd < f {
					myDice.MessageDelayRangeEnd = f
				}
				myDice.MessageDelayRangeStart = f
			}
		}

		if val, ok := jsonMap["messageDelayRangeEnd"]; ok {
			f, err := strconv.ParseFloat(val.(string), 64)
			if err == nil {
				if f < 0 {
					f = 0
				}

				if f >= myDice.MessageDelayRangeStart {
					myDice.MessageDelayRangeEnd = f
				}
			}
		}

		if val, ok := jsonMap["friendAddComment"]; ok {
			myDice.FriendAddComment = val.(string)
		}

		if val, ok := jsonMap["uiPassword"]; ok {
			myDice.Parent.UIPasswordHash = val.(string)
		}

		if val, ok := jsonMap["serveAddress"]; ok {
			myDice.Parent.ServeAddress = val.(string)
		}

		myDice.Parent.Save()
	} else {
		fmt.Println(err)
	}
	return c.JSON(http.StatusOK, nil)
}

func Bind(e *echo.Echo, _myDice *dice.DiceManager) {
	dm = _myDice
	myDice = _myDice.Dice[0]
	e.GET("/baseInfo", baseInfo)
	e.GET("/hello", hello2)
	e.GET("/log/fetchAndClear", logFetchAndClear)
	e.GET("/im_connections/list", ImConnections)
	e.GET("/im_connections/get", ImConnectionsGet)

	e.POST("/im_connections/qrcode", ImConnectionsQrcodeGet)
	e.POST("/im_connections/add", ImConnectionsAdd)
	e.POST("/im_connections/del", ImConnectionsDel)
	e.POST("/im_connections/set_enable", ImConnectionsSetEnable)
	e.POST("/im_connections/gocqhttpRelogin", ImConnectionsGocqhttpRelogin)

	e.GET("/configs/customText", customText)
	e.POST("/configs/customText/save", customTextSave)

	e.GET("/configs/custom_reply", customReply)
	e.POST("/configs/custom_reply/save", customReplySave)

	e.GET("/dice/config/get", DiceConfig)
	e.POST("/dice/config/set", DiceConfigSet)
	e.POST("/dice/exec", DiceExec)
	e.POST("/signin", doSignIn)
	e.GET("/signin/salt", doSignInGetSalt)
	e.GET("/checkSecurity", checkSecurity)
}
