package api

import (
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
)

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
		c.String(http.StatusBadRequest, "")
		return nil, true
	}

	return value, false
}
