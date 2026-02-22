package api

import (
	"encoding/base64"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

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
		ID string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
				return c.JSON(http.StatusOK, i)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsGetQQVersions(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	versions := make([]string, 0, len(dice.GocqAppVersionMap))
	for version := range dice.GocqAppVersionMap {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return Success(&c, Response{
		"versions": versions,
	})
}

func ImConnectionsSetEnable(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ID     string `form:"id"     json:"id"`
		Enable bool   `form:"enable" json:"enable"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
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
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ID                  string `form:"id"                  json:"id"`
		Protocol            int    `form:"protocol"            json:"protocol"`
		AppVersion          string `form:"appVersion"          json:"appVersion"`
		IgnoreFriendRequest bool   `json:"ignoreFriendRequest"` // 忽略好友请求
		UseSignServer       bool   `json:"useSignServer"`
		ExtraArgs           string `json:"extraArgs"`
		SignServerConfig    *dice.SignServerConfig
	}{}

	err := c.Bind(&v)
	if err != nil {
		myDice.Save(false)
		return c.JSON(http.StatusNotFound, nil)
	}
	for _, i := range myDice.ImSession.EndPoints {
		if i.ID != v.ID {
			continue
		}
		switch i.ProtocolType {
		case "walle-q":
			ad := i.Adapter.(*dice.PlatformAdapterWalleQ)
			ad.SetQQProtocol(v.Protocol)
			ad.IgnoreFriendRequest = v.IgnoreFriendRequest
		case "milky":
			ad := i.Adapter.(*dice.PlatformAdapterMilky)
			ad.IgnoreFriendRequest = v.IgnoreFriendRequest
		case "pureonebot":
			ad := i.Adapter.(*dice.PlatformAdapterOnebot)
			ad.IgnoreFriendRequest = v.IgnoreFriendRequest
		default:
			ad := i.Adapter.(*dice.PlatformAdapterGocq)
			if i.ProtocolType != "onebot" {
				i.ProtocolType = "onebot"
			}
			ad.SetQQProtocol(v.Protocol)
			ad.InPackGoCqhttpAppVersion = v.AppVersion
			if v.UseSignServer {
				ad.SetSignServer(v.SignServerConfig)
				ad.UseSignServer = v.UseSignServer
				ad.SignServerConfig = v.SignServerConfig
			}
			ad.IgnoreFriendRequest = v.IgnoreFriendRequest
			ad.ExtraArgs = v.ExtraArgs
		}
		return c.JSON(http.StatusOK, i)
	}
	myDice.Save(false)
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsGetSignInfo(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	data, err := dice.LagrangeGetSignInfo(myDice)
	if err != nil {
		return Error(&c, "读取SignInfo失败", Response{})
	}
	return Success(&c, Response{
		"data": data,
	})
}

func ImConnectionsDel(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ID string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for index, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
				// 禁用该endpoint防止出问题
				i.SetEnable(myDice, false)
				// 待删除的EPInfo落库，保留其统计数据
				i.StatsDump(myDice)
				// TODO: 注意 这个好像很不科学
				// i.diceServing = false
				switch i.Platform {
				case "QQ":
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					if i.ProtocolType == "onebot" {
						pa := i.Adapter.(*dice.PlatformAdapterGocq)
						if pa.BuiltinMode == "lagrange" || pa.BuiltinMode == "lagrange-gocq" {
							dice.BuiltinQQServeProcessKillBase(myDice, i, true)
							// 经测试，若不延时，可能导致清理对应目录失败（原因：文件被占用）
							time.Sleep(1 * time.Second)
							dice.LagrangeServeRemoveConfig(myDice, i)
						} else {
							dice.BuiltinQQServeProcessKill(myDice, i)
						}
					}
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
				case "DINGTALK":
					i.Adapter.SetEnable(false)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				case "SLACK":
					i.Adapter.SetEnable(false)
					myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints[:index], myDice.ImSession.EndPoints[index+1:]...)
					return c.JSON(http.StatusOK, i)
				case "SEALCHAT":
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
		ID string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	for _, i := range myDice.ImSession.EndPoints {
		if i.ID != v.ID {
			continue
		}
		switch i.ProtocolType {
		case "onebot", "":
			pa := i.Adapter.(*dice.PlatformAdapterGocq)
			if pa.GoCqhttpState == dice.StateCodeInLoginQrCode {
				return c.JSON(http.StatusOK, map[string]string{
					"img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.GoCqhttpQrcodeData),
				})
			}
		case "walle-q":
			pa := i.Adapter.(*dice.PlatformAdapterWalleQ)
			if pa.WalleQState == dice.WqStateCodeInLoginQrCode {
				return c.JSON(http.StatusOK, map[string]string{
					"img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.WalleQQrcodeData),
				})
			}
			// case "LagrangeGo":
			//	pa := i.Adapter.(*dice.PlatformAdapterLagrangeGo)
			//	if pa.CurState == dice.StateCodeInLoginQrCode {
			//		return c.JSON(http.StatusOK, map[string]string{
			//			"img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.QrcodeData),
			//		})
			//	}
		}
		return c.JSON(http.StatusOK, i)
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsCaptchaSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		ID   string `form:"id"   json:"id"`
		Code string `form:"code" json:"code"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return err
	}

	for _, i := range myDice.ImSession.EndPoints {
		if i.ID == v.ID {
			switch i.ProtocolType {
			case "onebot", "":
				pa := i.Adapter.(*dice.PlatformAdapterGocq)
				if pa.GoCqhttpState == dice.GoCqhttpStateCodeInLoginBar {
					pa.GoCqhttpLoginCaptcha = v.Code
					return c.String(http.StatusOK, "")
				}
			}
		}
	}
	return c.String(http.StatusNotFound, "")
}

func ImConnectionsSmsCodeSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		ID   string `form:"id"   json:"id"`
		Code string `form:"code" json:"code"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
				switch i.ProtocolType {
				case "onebot", "":
					pa := i.Adapter.(*dice.PlatformAdapterGocq)
					if pa.GoCqhttpState == dice.GoCqhttpStateCodeInLoginVerifyCode {
						pa.GoCqhttpLoginVerifyCode = v.Code
						return c.JSON(http.StatusOK, map[string]string{})
					}
				}
				return c.JSON(http.StatusOK, i)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsSmsCodeGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		ID string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
				switch i.ProtocolType {
				case "onebot", "":
					pa := i.Adapter.(*dice.PlatformAdapterGocq)
					return c.JSON(http.StatusOK, map[string]string{"tip": pa.GoCqhttpSmsNumberTip})
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
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ID string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
				myDice.Logger.Warnf("relogin %s", v.ID)
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
		ID string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
				i.Adapter.DoRelogin()
				return c.JSON(http.StatusOK, nil)
			}
		}
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsGocqConfigDownload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	id := c.QueryParam("id")
	for _, i := range myDice.ImSession.EndPoints {
		if i.ID == id {
			buf := packGocqConfig(i.RelWorkDir)
			return c.Blob(http.StatusOK, "", buf.Bytes())
		}
	}

	return c.String(http.StatusNotFound, "")
}

type AddDiscordEcho struct {
	Token              string
	ProxyURL           string
	ReverseProxyUrl    string
	ReverseProxyCDNUrl string
}

func ImConnectionsAddDiscord(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := &AddDiscordEcho{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewDiscordConnItem(dice.AddDiscordEcho(*v))
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterDiscord)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeDiscord(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "") // 这个是非标的，呃。。
}

func ImConnectionsAddKook(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Token string `json:"token" yaml:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewKookConnItem(v.Token)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterKook)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeKook(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddTelegram(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Token    string `json:"token"    yaml:"token"`
		ProxyURL string `json:"proxyURL" yaml:"proxyURL"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewTelegramConnItem(v.Token, v.ProxyURL)
		conn.Session = myDice.ImSession

		// myDice.Logger.Infof("成功创建endpoint")
		pa := conn.Adapter.(*dice.PlatformAdapterTelegram)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeTelegram(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddMinecraft(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		URL string `json:"url" yaml:"url"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewMinecraftConnItem(v.URL)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterMinecraft)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeMinecraft(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddSealChat(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		URL   string `json:"url"   yaml:"url"`
		Token string `json:"token" yaml:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewSealChatConnItem(v.URL, v.Token)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterSealChat)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeSealChat(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddDodo(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ClientID string `json:"clientID" yaml:"clientID"`
		Token    string `json:"token"    yaml:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewDodoConnItem(v.ClientID, v.Token)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterDodo)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeDodo(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddDingTalk(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		ClientID  string `json:"clientID"  yaml:"clientID"`
		Token     string `json:"token"     yaml:"token"`
		Nickname  string `json:"nickname"  yaml:"nickname"`
		RobotCode string `json:"robotCode" yaml:"robotCode"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewDingTalkConnItem(v.ClientID, v.Token, v.Nickname, v.RobotCode)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterDingTalk)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeDingTalk(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddSlack(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		BotToken string `json:"botToken" yaml:"botToken"`
		AppToken string `json:"appToken"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewSlackConnItem(v.AppToken, v.BotToken)
		pa := conn.Adapter.(*dice.PlatformAdapterSlack)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeSlack(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddMilky(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		WsGateway   string `json:"wsGateway"   yaml:"wsGateway"`
		RestGateway string `json:"restGateway" yaml:"restGateway"`
		Token       string `json:"token"       yaml:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewMilkyConnItem(dice.AddMilkyEcho{
			Token:       v.Token,
			WsGateway:   v.WsGateway,
			RestGateway: v.RestGateway,
		})
		pa := conn.Adapter.(*dice.PlatformAdapterMilky)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeMilky(myDice, conn)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddBuiltinGocq(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Account string `json:"account"          yaml:"account"`
		//nolint:gosec
		Password         string                 `json:"password"         yaml:"password"`
		Protocol         int                    `json:"protocol"`
		AppVersion       string                 `json:"appVersion"`
		UseSignServer    bool                   `json:"useSignServer"`
		SignServerConfig *dice.SignServerConfig `json:"signServerConfig"`
		// ConnectUrl        string `yaml:"connectUrl" json:"connectUrl"`               // 连接地址
		// Platform          string `yaml:"platform" json:"platform"`                   // 平台，如QQ、QQ频道
		// Enable            bool   `yaml:"enable" json:"enable"`                       // 是否启用
		// Type              string `yaml:"type" json:"type"`                           // 协议类型，如onebot、koishi等
		// UseInPackGoCqhttp bool   `yaml:"useInPackGoCqhttp" json:"useInPackGoCqhttp"` // 是否使用内置的gocqhttp
	}{}

	err := c.Bind(&v)
	if err == nil {
		uid := v.Account
		if checkUidExists(c, uid) {
			return nil
		}

		conn := dice.NewGoCqhttpConnectInfoItem(v.Account)
		conn.UserID = dice.FormatDiceIDQQ(uid)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterGocq)
		pa.InPackGoCqhttpProtocol = v.Protocol
		pa.InPackGoCqhttpPassword = v.Password
		pa.InPackGoCqhttpAppVersion = v.AppVersion
		pa.Session = myDice.ImSession
		pa.UseSignServer = v.UseSignServer
		pa.SignServerConfig = v.SignServerConfig

		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()

		dice.GoCqhttpServe(myDice, conn, dice.GoCqhttpLoginInfo{
			Password:         v.Password,
			Protocol:         v.Protocol,
			AppVersion:       v.AppVersion,
			IsAsyncRun:       true,
			UseSignServer:    v.UseSignServer,
			SignServerConfig: v.SignServerConfig,
		})
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddGocqSeparate(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Account    string `json:"account"     yaml:"account"`
		ConnectURL string `json:"connectUrl"  yaml:"connectUrl"` // 连接地址
		//nolint:gosec
		AccessToken string `json:"accessToken" yaml:"accessToken"` // 访问令牌
	}{}

	err := c.Bind(&v)
	if err == nil {
		uid := v.Account
		if checkUidExists(c, uid) {
			return nil
		}
		conn := dice.NewOnebotConnItem(dice.AddOnebotEcho{
			Token:         v.AccessToken,
			ConnectURL:    v.ConnectURL,
			ReverseURL:    "",
			ReverseSuffix: "",
			Mode:          "client",
		})
		conn.UserID = dice.FormatDiceIDQQ(uid)
		pa := conn.Adapter.(*dice.PlatformAdapterOnebot)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		// 设置正在使用中 千万不要设置这个
		// conn.SetEnable(myDice, true)
		// 像Milky一样使用
		go dice.ServePureOnebot(myDice, conn)
		// 上次更新的时间
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddReverseWs(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Account     string `json:"account"     yaml:"account"`
		ReverseAddr string `json:"reverseAddr" yaml:"reverseAddr"`
	}{}

	err := c.Bind(&v)
	if err == nil {
		uid := v.Account
		if checkUidExists(c, uid) {
			return nil
		}
		conn := dice.NewOnebotConnItem(dice.AddOnebotEcho{
			Token:         "", // TODO：反向怎么可怜巴巴的，连个Token都没有吗
			ConnectURL:    "",
			ReverseURL:    v.ReverseAddr,
			ReverseSuffix: "/ws",
			Mode:          "server",
		})
		conn.UserID = dice.FormatDiceIDQQ(uid)
		pa := conn.Adapter.(*dice.PlatformAdapterOnebot)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		// 设置正在使用中 千万不要设置这个
		// conn.SetEnable(myDice, true)
		// 像Milky一样使用
		go dice.ServePureOnebot(myDice, conn)
		// 上次更新的时间
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		return c.JSON(http.StatusOK, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddRed(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Success(&c, Response{"testMode": true})
	}

	v := struct {
		Host  string `json:"host"  yaml:"host"`
		Port  int    `json:"port"  yaml:"port"`
		Token string `json:"token" yaml:"token"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewRedConnItem(v.Host, v.Port, v.Token)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterRed)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServeRed(myDice, conn)
		return Success(&c, Response{})
	}
	return c.String(430, "")
}

func ImConnectionsAddOfficialQQ(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Success(&c, Response{"testMode": true})
	}

	v := struct {
		AppID       uint64 `json:"appID"       yaml:"appID"`
		Token       string `json:"token"       yaml:"token"`
		AppSecret   string `json:"appSecret"   yaml:"appSecret"`
		OnlyQQGuild bool   `json:"onlyQQGuild" yaml:"onlyQQGuild"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		conn := dice.NewOfficialQQConnItem(v.AppID, v.Token, v.AppSecret, v.OnlyQQGuild)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterOfficialQQ)
		pa.Session = myDice.ImSession
		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		go dice.ServerOfficialQQ(myDice, conn)
		return Success(&c, Response{})
	}
	return c.String(430, "")
}

func ImConnectionsAddSatori(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Success(&c, Response{"testMode": true})
	}

	v := struct {
		Platform string `json:"platform" yaml:"platform"`
		Host     string `json:"host"     yaml:"host"`
		Port     int    `json:"port"     yaml:"port"`
		Token    string `json:"token"    yaml:"token"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, "")
	}

	conn := dice.NewSatoriConnItem(v.Platform, v.Host, v.Port, v.Token)
	conn.Session = myDice.ImSession
	pa := conn.Adapter.(*dice.PlatformAdapterSatori)
	pa.Session = myDice.ImSession
	myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
	myDice.LastUpdatedTime = time.Now().Unix()
	myDice.Save(false)
	go dice.ServeQQ(myDice, conn)
	return Success(&c, Response{})
}

func ImConnectionsAddBuiltinLagrange(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(http.StatusOK, Response{"testMode": true})
	}

	v := struct {
		Account           string `json:"account"           yaml:"account"`
		SignServerName    string `json:"signServerName"    yaml:"signServerName"`
		SignServerVersion string `json:"signServerVersion" yaml:"signServerVersion"`
		IsGocq            bool   `json:"isGocq"            yaml:"isGocq"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		uid := v.Account
		if checkUidExists(c, uid) {
			return nil
		}

		conn := dice.NewLagrangeConnectInfoItem(v.Account, v.IsGocq)
		conn.UserID = dice.FormatDiceIDQQ(uid)
		conn.Session = myDice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterGocq)
		// pa.InPackGoCqhttpProtocol = v.Protocol
		pa.Session = myDice.ImSession

		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		myDice.LastUpdatedTime = time.Now().Unix()
		uin, err := strconv.ParseInt(v.Account, 10, 64)
		if err != nil {
			return err
		}
		pa.SignServerName = v.SignServerName
		pa.SignServerVer = v.SignServerVersion
		dice.LagrangeServe(myDice, conn, dice.LagrangeLoginInfo{
			UIN:               uin,
			SignServerName:    v.SignServerName,
			SignServerVersion: v.SignServerVersion,
			IsAsyncRun:        true,
		})
		return c.JSON(http.StatusOK, v)
	}

	return c.String(430, "")
}

// func ImConnectionsAddLagrangeGO(c echo.Context) error {
//	if !doAuth(c) {
//		return c.JSON(http.StatusForbidden, nil)
//	}
//	if dm.JustForTest {
//		return Success(&c, Response{"testMode": true})
//	}
//
//	v := struct {
//		Account       string `yaml:"account" json:"account"`
//		CustomSignUrl string `yaml:"signServerUrl" json:"signServerUrl"`
//	}{}
//	err := c.Bind(&v)
//	if err == nil {
//		uid := v.Account
//		if checkUidExists(c, uid) {
//			return nil
//		}
//		uin, err := strconv.ParseInt(v.Account, 10, 64)
//		if err != nil {
//			return err
//		}
//		conn := dice.NewLagrangeGoConnItem(uint32(uin), v.CustomSignUrl)
//		conn.UserID = dice.FormatDiceIDQQ(uid)
//		conn.Session = myDice.ImSession
//		pa := conn.Adapter.(*dice.PlatformAdapterLagrangeGo)
//		pa.Session = myDice.ImSession
//
//		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
//		myDice.LastUpdatedTime = time.Now().Unix()
//
//		dice.ServeLagrangeGo(myDice, conn)
//		return c.JSON(http.StatusOK, v)
//	}
//
//	return c.String(430, "")
// }
