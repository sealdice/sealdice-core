package api

import (
	"fmt"
	"net/http"
	"strings"

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
		return Error(&c, err.Error(), Response{})
	}

	if v.PageNum < 1 {
		v.PageNum = 1
	}
	if v.PageSize <= 0 {
		v.PageSize = 20
	}

	total, page, err := model.LogGetLogPage(myDice.DBLogs, &v)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	return Success(&c, Response{
		"data":     page,
		"total":    total,
		"pageNum":  v.PageNum,
		"pageSize": len(page),
	})
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

func storyGetLogBackupList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	list, err := dice.StoryLogBackupList(myDice)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{
		"data": list,
	})
}

func storyDeleteLogBackup(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	name := c.QueryParam("name")
	names := []string{name}
	fail := dice.StoryLogBackupBatchDelete(myDice, names)
	if len(fail) > 0 {
		return Error(&c, "备份删除失败", Response{})
	}
	return Success(&c, Response{})
}

func storyBatchDeleteLogBackup(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		Names []string `json:"names"`
	}{}
	err := c.Bind(&v)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	fails := dice.StoryLogBackupBatchDelete(myDice, v.Names)
	if len(fails) > 0 {
		return Error(&c, fmt.Sprintf("部分备份删除失败：%s", strings.Join(fails, "，")), Response{})
	}
	return Success(&c, Response{})
}

func logSendToBackend(groupID string, logName string) (string, error) {
	ctx := &dice.MsgContext{
		Dice:     myDice,
		EndPoint: myDice.UIEndpoint,
	}

	return dice.LogSendToBackend(ctx, groupID, logName)
}
