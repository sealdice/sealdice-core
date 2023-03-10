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
	var versionNewNote string
	var versionNewCode int64
	if dm.AppVersionOnline != nil {
		versionNew = dm.AppVersionOnline.VersionLatestDetail
		versionNewNote = dm.AppVersionOnline.VersionLatestNote
		versionNewCode = dm.AppVersionOnline.VersionLatestCode
	}

	extraTitle := ""
	getName := func() string {
		defer func() {
			// 防止报错
			if r := recover(); r != nil {
			}
		}()

		ctx := &dice.MsgContext{Dice: myDice, EndPoint: nil, Session: myDice.ImSession}
		return dice.DiceFormatTmpl(ctx, "核心:骰子名字")
	}

	extraTitle = getName()

	return c.JSON(http.StatusOK, struct {
		AppName        string `json:"appName"`
		Version        string `json:"version"`
		VersionNew     string `json:"versionNew"`
		VersionNewNote string `json:"versionNewNote"`
		VersionCode    int64  `json:"versionCode"`
		VersionNewCode int64  `json:"versionNewCode"`
		MemoryAlloc    uint64 `json:"memoryAlloc"`
		Uptime         int64  `json:"uptime"`
		MemoryUsedSys  uint64 `json:"memoryUsedSys"`
		ExtraTitle     string `json:"extraTitle"`
		OS             string `json:"OS"`
		Arch           string `json:"arch"`
	}{
		AppName:        dice.APPNAME,
		Version:        dice.VERSION,
		VersionNew:     versionNew,
		VersionNewNote: versionNewNote,
		VersionCode:    dice.VERSION_CODE,
		VersionNewCode: versionNewCode,
		MemoryAlloc:    m.Alloc,
		MemoryUsedSys:  m.Sys,
		Uptime:         time.Now().Unix() - startTime,
		ExtraTitle:     extraTitle,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
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
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
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

	myDice.LastUpdatedTime = time.Now().Unix()
	myDice.Save(false)
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsSetData(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Id                  string `form:"id" json:"id"`
		Protocol            int    `form:"protocol" json:"protocol"`
		IgnoreFriendRequest bool   `json:"ignoreFriendRequest"` // 忽略好友请求
	}{}

	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				ad := i.Adapter.(*dice.PlatformAdapterGocq)
				ad.SetQQProtocol(v.Protocol)
				ad.IgnoreFriendRequest = v.IgnoreFriendRequest
				return c.JSON(http.StatusOK, i)
			}
		}
	}

	myDice.Save(false)
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsDel(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
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
				switch i.Platform {
				case "QQ":
					dice.GoCqHttpServeProcessKill(myDice, i)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				case "DISCORD":
					i.Adapter.SetEnable(false)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				case "KOOK":
					i.Adapter.SetEnable(false)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				case "TG":
					i.Adapter.SetEnable(false)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				case "MC":
					i.Adapter.SetEnable(false)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				case "DODO":
					i.Adapter.SetEnable(false)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				}
			}
		}
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
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
	//fmt.Println(err)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			//fmt.Println(i.Id, i.ProtocolType, i.ProtocolType)
			if i.Id == v.Id {
				switch i.ProtocolType {
				case "onebot":
					pa := i.Adapter.(*dice.PlatformAdapterGocq)
					if pa.GoCqHttpState == dice.StateCodeInLoginQrCode {
						return c.JSON(http.StatusOK, map[string]string{
							"img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.GoCqHttpQrcodeData),
						})
					}
				case "walle-q":
					pa := i.Adapter.(*dice.PlatformAdapterWalleQ)
					if pa.WalleQState == dice.WqStateCodeInLoginQrCode {
						//fmt.Println("qrcode:", base64.StdEncoding.EncodeToString(pa.WalleQQrcodeData))
						return c.JSON(http.StatusOK, map[string]string{
							"img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.WalleQQrcodeData),
						})
					}
				}
				return c.JSON(http.StatusOK, i)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsAddWalleQ(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	v := struct {
		Account  string `yaml:"account" json:"account"`
		Password string `yaml:"password" json:"password"`
		Protocol int    `json:"protocol"`
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

		conn := dice.NewWqConnectInfoItem(v.Account)
		conn.UserId = dice.FormatDiceIdQQ(uid)
		conn.ProtocolType = "walle-q"
		pa := conn.Adapter.(*dice.PlatformAdapterWalleQ)
		pa.InPackWalleQProtocol = v.Protocol
		pa.InPackWalleQPassword = v.Password
		pa.Session = myDice.ImSession

		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		go dice.WalleQServe(myDice, conn, v.Password, v.Protocol, false)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func ImConnectionsGocqhttpRelogin(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
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

func ImConnectionsWalleQRelogin(c echo.Context) error {
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

func ImConnectionsAddDiscord(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//myDice.Logger.Infof("后端add调用")
	v := struct {
		Token string `yaml:"token" json:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		//myDice.Logger.Infof("bind无异常")
		conn := dice.NewDiscordConnItem(v.Token)
		//myDice.Logger.Infof("成功创建endpoint")
		pa := conn.Adapter.(*dice.PlatformAdapterDiscord)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeDiscord(myDice, conn)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddKook(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//myDice.Logger.Infof("后端add调用")
	v := struct {
		Token string `yaml:"token" json:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		//myDice.Logger.Infof("bind无异常")
		conn := dice.NewKookConnItem(v.Token)
		//myDice.Logger.Infof("成功创建endpoint")
		pa := conn.Adapter.(*dice.PlatformAdapterKook)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeKook(myDice, conn)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddTelegram(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//myDice.Logger.Infof("后端add调用")
	v := struct {
		Token string `yaml:"token" json:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		//myDice.Logger.Infof("bind无异常")
		conn := dice.NewTelegramConnItem(v.Token)
		//myDice.Logger.Infof("成功创建endpoint")
		pa := conn.Adapter.(*dice.PlatformAdapterTelegram)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeTelegram(myDice, conn)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddMinecraft(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Url string `yaml:"url" json:"url"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		//myDice.Logger.Infof("bind无异常")
		conn := dice.NewMinecraftConnItem(v.Url)
		//myDice.Logger.Infof("成功创建endpoint")
		pa := conn.Adapter.(*dice.PlatformAdapterMinecraft)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeMinecraft(myDice, conn)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddDodo(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ClientID string `yaml:"clientID" json:"clientID"`
		Token    string `yaml:"token" json:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		//myDice.Logger.Infof("bind无异常")
		conn := dice.NewDodoConnItem(v.ClientID, v.Token)
		//myDice.Logger.Infof("成功创建endpoint")
		pa := conn.Adapter.(*dice.PlatformAdapterDodo)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeDodo(myDice, conn)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
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
		pa := conn.Adapter.(*dice.PlatformAdapterGocq)
		pa.InPackGoCqHttpProtocol = v.Protocol
		pa.InPackGoCqHttpPassword = v.Password
		pa.Session = myDice.ImSession

		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()

		dice.GoCqHttpServe(myDice, conn, v.Password, v.Protocol, true)
		myDice.LastUpdatedTime = time.Now().Unix()
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
		myDice.LastUpdatedTime = time.Now().Unix()
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
		Platform:    "UI",
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

func onebotTool(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Port int64 `form:"port" json:"port"`
	}{}
	err := c.Bind(&v)

	port := int64(13325)
	if v.Port != 0 {
		port = v.Port
	}

	errText := ""
	var ip string
	ip, err = socksOpen(myDice, port)
	if err != nil {
		errText = err.Error()
	}

	resp := c.JSON(200, map[string]interface{}{
		"ok":      err == nil,
		"ip":      ip,
		"errText": errText,
	})
	return resp
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
	e.POST(prefix+"/im_connections/addDiscord", ImConnectionsAddDiscord)
	e.POST(prefix+"/im_connections/addKook", ImConnectionsAddKook)
	e.POST(prefix+"/im_connections/addTelegram", ImConnectionsAddTelegram)
	e.POST(prefix+"/im_connections/addMinecraft", ImConnectionsAddMinecraft)
	e.POST(prefix+"/im_connections/addDodo", ImConnectionsAddDodo)
	e.POST(prefix+"/im_connections/addWalleQ", ImConnectionsAddWalleQ)
	e.POST(prefix+"/im_connections/del", ImConnectionsDel)
	e.POST(prefix+"/im_connections/set_enable", ImConnectionsSetEnable)
	e.POST(prefix+"/im_connections/set_data", ImConnectionsSetData)
	e.POST(prefix+"/im_connections/gocqhttpRelogin", ImConnectionsGocqhttpRelogin)
	e.POST(prefix+"/im_connections/walleQRelogin", ImConnectionsWalleQRelogin)

	e.GET(prefix+"/configs/customText", customText)
	e.POST(prefix+"/configs/customText/save", customTextSave)

	e.GET(prefix+"/configs/custom_reply", customReplyGet)
	e.POST(prefix+"/configs/custom_reply/save", customReplySave)
	e.GET(prefix+"/configs/custom_reply/file_list", customReplyFileList)
	e.POST(prefix+"/configs/custom_reply/file_new", customReplyFileNew)
	e.POST(prefix+"/configs/custom_reply/file_delete", customReplyFileDelete)
	e.GET(prefix+"/configs/custom_reply/file_download", customReplyFileDownload)
	e.POST(prefix+"/configs/custom_reply/file_upload", customReplyFileUpload)
	e.GET(prefix+"/configs/custom_reply/debug_mode", customReplyDebugModeGet)
	e.POST(prefix+"/configs/custom_reply/debug_mode", customReplyDebugModeSet)

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

	e.GET(prefix+"/banconfig/list", banMapList)
	e.GET(prefix+"/banconfig/get", banConfigGet)
	e.POST(prefix+"/banconfig/set", banConfigSet)

	//e.GET(prefix+"/banconfig/map_get", banMapGet)
	e.POST(prefix+"/banconfig/map_delete_one", banMapDeleteOne)
	e.POST(prefix+"/banconfig/map_add_one", banMapAddOne)
	//e.POST(prefix+"/banconfig/map_set", banMapSet)

	e.GET(prefix+"/deck/list", deckList)
	e.POST(prefix+"/deck/reload", deckReload)
	e.POST(prefix+"/deck/upload", deckUpload)
	e.POST(prefix+"/deck/enable", deckEnable)
	e.POST(prefix+"/deck/delete", deckDelete)

	e.POST(prefix+"/dice/upgrade", upgrade)

	e.POST(prefix+"/js/reload", jsReload)
	e.POST(prefix+"/js/execute", jsExec)
	e.POST(prefix+"/js/upload", jsUpload)
	e.GET(prefix+"/js/list", jsList)
	e.POST(prefix+"/js/delete", jsDelete)
	e.GET(prefix+"/js/get_record", jsGetRecord)

	e.POST(prefix+"/tool/onebot", onebotTool)
	e.GET(prefix+"/utils/ga/:uid", getGithubAvatar)
	e.GET(prefix+"/utils/news", getNews)
}
