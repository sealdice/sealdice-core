package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"sealdice-core/dice"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
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
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ID     string `form:"id" json:"id"`
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
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		ID                  string `form:"id" json:"id"`
		Protocol            int    `form:"protocol" json:"protocol"`
		AppVersion          string `form:"appVersion" json:"appVersion"`
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
		if i.ProtocolType == "walle-q" {
			ad := i.Adapter.(*dice.PlatformAdapterWalleQ)
			ad.SetQQProtocol(v.Protocol)
			ad.IgnoreFriendRequest = v.IgnoreFriendRequest
		} else {
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
				// i.DiceServing = false
				switch i.Platform {
				case "QQ":
					dice.GoCqhttpServeProcessKill(myDice, i)
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
				case "DINGTALK":
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
		}
		return c.JSON(http.StatusOK, i)
	}
	return c.JSON(http.StatusNotFound, nil)
}

func ImConnectionsSmsCodeSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		ID   string `form:"id" json:"id"`
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
			if i.UserID == dice.FormatDiceIDQQ(uid) {
				return c.JSON(CodeAlreadyExists, i)
			}
		}

		conn := dice.NewWqConnectInfoItem(v.Account)
		conn.UserID = dice.FormatDiceIDQQ(uid)
		conn.Session = myDice.ImSession
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
		ID string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.ID == v.ID {
				fmt.Print("!!! relogin ", v.ID)
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

type AddDiscordEcho struct {
	Token    string
	ProxyURL string
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

	v := struct {
		Token string `yaml:"token" json:"token"`
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

	v := struct {
		Token    string `yaml:"token" json:"token"`
		ProxyURL string `yaml:"proxyURL" json:"proxyURL"`
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
		URL string `yaml:"url" json:"url"`
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
		conn := dice.NewDodoConnItem(v.ClientID, v.Token)
		conn.Session = myDice.ImSession
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

func ImConnectionsAddDingTalk(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		ClientID  string `yaml:"clientID" json:"clientID"`
		Token     string `yaml:"token" json:"token"`
		Nickname  string `yaml:"nickname" json:"nickname"`
		RobotCode string `yaml:"robotCode" json:"robotCode"`
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
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddSlack(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		BotToken string `yaml:"botToken" json:"botToken"`
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
		Account          string                 `yaml:"account" json:"account"`
		Password         string                 `yaml:"password" json:"password"`
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
		uid, err := strconv.ParseInt(v.Account, 10, 64)
		if err != nil {
			return c.String(430, "")
		}

		for _, i := range myDice.ImSession.EndPoints {
			if i.UserID == dice.FormatDiceIDQQ(uid) {
				return c.JSON(CodeAlreadyExists, i)
			}
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
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}

func ImConnectionsAddGocqSeparate(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Account     string `yaml:"account" json:"account"`
		ConnectURL  string `yaml:"connectUrl" json:"connectUrl"`   // 连接地址
		AccessToken string `yaml:"accessToken" json:"accessToken"` // 访问令牌
		RelWorkDir  string `yaml:"relWorkDir" json:"relWorkDir"`   //
	}{}

	err := c.Bind(&v)
	if err == nil {
		uid, err := strconv.ParseInt(v.Account, 10, 64)
		if err != nil {
			return c.String(430, "")
		}

		for _, i := range myDice.ImSession.EndPoints {
			if i.UserID == dice.FormatDiceIDQQ(uid) {
				return c.JSON(CodeAlreadyExists, i)
			}
		}

		conn := dice.NewGoCqhttpConnectInfoItem(v.Account)
		conn.UserID = dice.FormatDiceIDQQ(uid)
		conn.Session = myDice.ImSession

		pa := conn.Adapter.(*dice.PlatformAdapterGocq)
		pa.Session = myDice.ImSession

		// 三项设置
		conn.RelWorkDir = v.RelWorkDir
		pa.ConnectURL = v.ConnectURL
		pa.AccessToken = v.AccessToken

		pa.UseInPackGoCqhttp = false

		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		conn.SetEnable(myDice, true)

		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		return c.JSON(200, conn)
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
		Host  string `yaml:"host" json:"host"`
		Port  int    `yaml:"port" json:"port"`
		Token string `yaml:"token" json:"token"`
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
