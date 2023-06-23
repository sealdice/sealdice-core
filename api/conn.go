package api

import (
	"encoding/base64"
	"net/http"
	"sealdice-core/dice"
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
					ad.IgnoreFriendRequest = v.IgnoreFriendRequest
				}
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
				case "onebot", "":
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

func ImConnectionsSmsCodeSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		Id   string `form:"id" json:"id"`
		Code string `form:"code" json:"code"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				switch i.ProtocolType {
				case "onebot", "":
					pa := i.Adapter.(*dice.PlatformAdapterGocq)
					if pa.GoCqHttpState == dice.GoCqHttpStateCodeInLoginVerifyCode {
						pa.GoCqHttpLoginVerifyCode = v.Code
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
		Id string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		for _, i := range myDice.ImSession.EndPoints {
			if i.Id == v.Id {
				switch i.ProtocolType {
				case "onebot", "":
					pa := i.Adapter.(*dice.PlatformAdapterGocq)
					return c.JSON(http.StatusOK, map[string]string{"tip": pa.GoCqHttpSmsNumberTip})
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

	//myDice.Logger.Infof("后端add调用")
	v := &AddDiscordEcho{}
	err := c.Bind(&v)
	if err == nil {
		//myDice.Logger.Infof("bind无异常")
		conn := dice.NewDiscordConnItem(dice.AddDiscordEcho(*v))
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
		Account    string `yaml:"account" json:"account"`
		ConnectUrl string `yaml:"connectUrl" json:"connectUrl"` // 连接地址
		RelWorkDir string `yaml:"relWorkDir" json:"relWorkDir"` //
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
		pa.Session = myDice.ImSession

		// 三项设置
		conn.RelWorkDir = v.RelWorkDir
		pa.ConnectUrl = v.ConnectUrl
		pa.UseInPackGoCqhttp = false

		myDice.ImSession.EndPoints = append(myDice.ImSession.EndPoints, conn)
		conn.SetEnable(myDice, true)

		myDice.LastUpdatedTime = time.Now().Unix()
		myDice.Save(false)
		return c.JSON(200, conn)
	}
	return c.String(430, "")
}
