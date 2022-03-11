package api

import (
	"encoding/base64"
	"net/http"
	"runtime"
	"sealdice-core/dice"
	"strconv"
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
	return c.JSON(http.StatusOK, myDice.TextMapRaw)
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
		myDice.GenerateTextMap()
		myDice.SaveText()
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}

func ImConnections(c echo.Context) error {
	return c.JSON(http.StatusOK, myDice.ImSession.Conns)
}

func ImConnectionsGet(c echo.Context) error {
	v := struct {
		Id string `form:"id" json:"id"`
	}{}
	err := c.Bind(&v)
	if err == nil {
		for _, i := range myDice.ImSession.Conns {
			if i.Id == v.Id {
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
		for index, i := range myDice.ImSession.Conns {
			if i.Id == v.Id {
				dice.GoCqHttpServeProcessKill(myDice, i)
				myDice.ImSession.Conns = append(myDice.ImSession.Conns[:index], myDice.ImSession.Conns[index+1:]...)
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
		for _, i := range myDice.ImSession.Conns {
			if i.Id == v.Id {
				if i.InPackGoCqHttpQrcodeReady {
					return c.JSON(http.StatusOK, map[string]string{
						"img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(i.InPackGoCqHttpQrcodeData),
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
		for _, i := range myDice.ImSession.Conns {
			if i.Id == v.Id {
				myDice.Logger.Infof("重新启动go-cqhttp进程，对应账号: <%s>(%d)", i.Nickname, i.UserId)
				dice.GoCqHttpServeProcessKill(myDice, i)
				time.Sleep(1 * time.Second)
				dice.GoCqHttpServeRemoveSessionToken(myDice, i) // 删除session.token
				dice.GoCqHttpServe(myDice, i, "", 1, true)
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

		for _, i := range myDice.ImSession.Conns {
			if i.UserId == uid {
				return c.JSON(CODE_ALREADY_EXISTS, i)
			}
		}

		conn := dice.NewGoCqhttpConnectInfoItem(v.Account)
		conn.UserId = uid
		myDice.ImSession.Conns = append(myDice.ImSession.Conns, conn)
		dice.GoCqHttpServe(myDice, conn, v.Password, 1, true)
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
	e.POST("/im_connections/gocqhttpRelogin", ImConnectionsGocqhttpRelogin)
}
