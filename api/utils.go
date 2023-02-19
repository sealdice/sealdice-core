package api

import (
	"encoding/binary"
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
)

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func doAuth(c echo.Context) bool {
	token := c.Request().Header.Get("token")
	if token == "" {
		token = c.QueryParam("token")
	}
	if myDice.Parent.AccessTokens[token] {
		return true
	}
	return false
}

func GetHexData(c echo.Context, method string, name string) (value []byte, finished bool) {
	var err error
	var strValue string
	// var exists bool

	if method == "GET" {
		strValue = c.Param(name)
	} else if method == "POST" {
		strValue = c.FormValue(name)
	}

	// if !exists {
	// 	c.String(http.StatusNotAcceptable, "")
	// 	return nil, true
	// }

	value, err = hex.DecodeString(strValue)
	if err != nil {
		_ = c.String(http.StatusBadRequest, "")
		return nil, true
	}

	return value, false
}
