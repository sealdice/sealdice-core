package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
	"sealdice-core/dice/dao"
	"sealdice-core/model"
	log "sealdice-core/utils/kratos"
)

func storyGetInfo(c echo.Context) error {
	info, err := dao.LogGetInfo(myDice.DBOperator)
	if err != nil {
		log.Error("storyGetInfo", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, info)
}

// Deprecated: replaced by page
func storyGetLogs(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	logs, err := dao.LogGetLogs(myDice.DBOperator)
	if err != nil {
		log.Error("storyGetLogs", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, logs)
}

func storyGetLogPage(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	v := dao.QueryLogPage{}
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

	total, page, err := dao.LogGetLogPage(myDice.DBOperator, &v)
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
	lines, err := dao.LogGetAllLines(myDice.DBOperator, c.QueryParam("groupId"), c.QueryParam("name"))
	if err != nil {
		log.Error("storyGetItems", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, lines)
}

func storyGetItemPage(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := dao.QueryLogLinePage{}
	err := c.Bind(&v)
	if err != nil {
		log.Error("storyGetItemPage", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	if v.PageNum < 1 {
		v.PageNum = 1
	}
	if v.PageSize <= 0 {
		v.PageSize = 10
	}

	lines, err := dao.LogGetLinePage(myDice.DBOperator, &v)
	if err != nil {
		log.Error("storyGetItemPage", err)
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
		log.Error("storyDelLog", err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	is := dao.LogDelete(myDice.DBOperator, v.GroupID, v.Name)
	if !is {
		log.Error("storyDelLog", "failed to delete")
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
	unofficial, url, err := logSendToBackend(v.GroupID, v.Name)
	if err != nil {
		log.Error("storyUploadLog", err)
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	ret := fmt.Sprintf("跑团日志已上传服务器，链接如下：<br/>%s", url)
	if unofficial {
		ret += "<br/>[注意：该链接非海豹官方染色器]"
	}
	return c.JSON(http.StatusOK, ret)
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

func storyDownloadLogBackup(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	name := c.QueryParam("name")
	path, err := dice.StoryLogBackupDownloadPath(myDice, name)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return c.Attachment(path, name)
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

func logSendToBackend(groupID string, logName string) (bool, string, error) {
	ctx := &dice.MsgContext{
		Dice:     myDice,
		EndPoint: myDice.UIEndpoint,
	}

	return dice.LogSendToBackend(ctx, groupID, logName)
}
