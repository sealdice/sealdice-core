package api

import (
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"runtime"
	"sealdice-core/dice"
	"sort"
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
	//fmt.Println("!!!!", m.Alloc-m.Frees, m.HeapReleased, m.HeapInuse)
	//
	//const meg = 1024 * 1024
	//fmt.Printf("env: %v, sys: %4d MB, alloc: %4d MB, idel: %4d MB, released: %4d MB, inuse: %4d MB stack:%d\n",
	//	os.Getenv("GODEBUG"), m.HeapSys/meg, m.HeapAlloc/meg, m.HeapIdle/meg, m.HeapReleased/meg, m.HeapInuse/meg,
	//	m.StackSys/meg)

	var versionNew string
	if dm.AppVersionOnline != nil {
		versionNew = dm.AppVersionOnline.VersionLatestDetail
	}

	return c.JSON(http.StatusOK, struct {
		AppName       string `json:"appName"`
		Version       string `json:"version"`
		VersionNew    string `json:"versionNew"`
		MemoryAlloc   uint64 `json:"memoryAlloc"`
		Uptime        int64  `json:"uptime"`
		MemoryUsedSys uint64 `json:"memoryUsedSys"`
	}{
		AppName:       dice.APPNAME,
		Version:       dice.VERSION,
		VersionNew:    versionNew,
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
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
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
	timeNeed := int64(500)
	if dm.JustForTest {
		timeNeed = 80
	}
	if now-lastExecTime < timeNeed {
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

func DiceAllCommand(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var cmdLst []string
	for k := range myDice.CmdMap {
		cmdLst = append(cmdLst, k)
	}

	for _, i := range myDice.ExtList {
		for k := range i.CmdMap {
			cmdLst = append(cmdLst, k)
		}
	}
	sort.Sort(dice.ByLength(cmdLst))
	return c.JSON(200, cmdLst)
}

func Bind(e *echo.Echo, _myDice *dice.DiceManager) {
	dm = _myDice
	myDice = _myDice.Dice[0]

	var prefix string
	prefix = "/sd-api"

	e.GET(prefix+"/baseInfo", baseInfo)
	e.GET(prefix+"/hello", hello2)
	e.GET(prefix+"/log/fetchAndClear", logFetchAndClear)
	e.GET(prefix+"/im_connections/list", ImConnections)
	e.GET(prefix+"/im_connections/get", ImConnectionsGet)

	e.POST(prefix+"/im_connections/qrcode", ImConnectionsQrcodeGet)
	e.POST(prefix+"/im_connections/add", ImConnectionsAdd)
	e.POST(prefix+"/im_connections/del", ImConnectionsDel)
	e.POST(prefix+"/im_connections/set_enable", ImConnectionsSetEnable)
	e.POST(prefix+"/im_connections/gocqhttpRelogin", ImConnectionsGocqhttpRelogin)

	e.GET(prefix+"/configs/customText", customText)
	e.POST(prefix+"/configs/customText/save", customTextSave)

	e.GET(prefix+"/configs/custom_reply", customReply)
	e.POST(prefix+"/configs/custom_reply/save", customReplySave)

	e.GET(prefix+"/dice/config/get", DiceConfig)
	e.POST(prefix+"/dice/config/set", DiceConfigSet)
	e.POST(prefix+"/dice/exec", DiceExec)
	e.GET(prefix+"/dice/cmdList", DiceAllCommand)

	e.POST(prefix+"/signin", doSignIn)
	e.GET(prefix+"/signin/salt", doSignInGetSalt)
	e.GET(prefix+"/checkSecurity", checkSecurity)

	e.GET(prefix+"/backup/list", backupGetList)
	e.POST(prefix+"/backup/do_backup", backupSimple)
	e.GET(prefix+"/backup/config_get", backupConfigGet)
	e.POST(prefix+"/backup/config_set", backupConfigSave)
	e.GET(prefix+"/backup/download", backupDownload)

	e.GET(prefix+"/group/list", groupList)
	e.POST(prefix+"/group/set_one", groupSetOne)
	e.POST(prefix+"/group/quit_one", groupQuit)

	e.GET(prefix+"/banconfig/get", banConfigGet)
	e.POST(prefix+"/banconfig/set", banConfigSet)

	e.GET(prefix+"/banconfig/map_get", banMapGet)
	e.POST(prefix+"/banconfig/map_delete_one", banMapDeleteOne)
	e.POST(prefix+"/banconfig/map_add_one", banMapAddOne)
	//e.POST(prefix+"/banconfig/map_set", banMapSet)
}
