package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	"sealdice-core/dice/model"
)

func storyGetInfo(c echo.Context) error {
	info, err := model.LogGetInfo(myDice.DBLogs)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, info)
}

// Deprecated: replaced by page
func storyGetLogs(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	logs, err := model.LogGetLogs(myDice.DBLogs)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, logs)
}

func storyGetLogPage(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	v := model.QueryLogPage{}
	err := c.Bind(&v)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	if v.PageNum < 1 {
		v.PageNum = 1
	}
	if v.PageSize <= 0 {
		v.PageSize = 20
	}

	logs, err := model.LogGetLogPage(myDice.DBLogs, &v)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, logs)
}

// Deprecated: replaced by page
func storyGetItems(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	lines, err := model.LogGetAllLines(myDice.DBLogs, c.QueryParam("groupId"), c.QueryParam("name"))
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, lines)
}

func storyGetItemPage(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := model.QueryLogLinePage{}
	err := c.Bind(&v)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	if v.PageNum < 1 {
		v.PageNum = 1
	}
	if v.PageSize <= 0 {
		v.PageSize = 10
	}

	lines, err := model.LogGetLinePage(myDice.DBLogs, &v)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, lines)
}

func storyDelLog(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	v := &model.LogInfo{}
	err := c.Bind(&v)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	is := model.LogDelete(myDice.DBLogs, v.GroupID, v.Name)
	if !is {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, false)
	}
	return c.JSON(http.StatusOK, true)
}

func storyUploadLog(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	v := &model.LogInfo{}
	_ = c.Bind(&v)
	url, err := logSendToBackend(v.GroupID, v.Name)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("跑团日志已上传服务器，链接如下：<br>%s", url))
}

func logSendToBackend(groupID string, logName string) (string, error) {
	ctx := &dice.MsgContext{
		Dice:     myDice,
		EndPoint: myDice.UIEndpoint,
	}

	return dice.LogSendToBackend(ctx, groupID, logName)
}
