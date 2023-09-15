package api

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"path/filepath"
	"sealdice-core/dice"
	"sealdice-core/dice/model"
	"time"
)

func logPostLog(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	log := model.LogOne{}
	err := c.Bind(&log)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	for _, item := range log.Items {
		model.LogAppend(myDice.DBLogs, log.Info.GroupId, log.Info.Name, &item)
	}
	return c.JSON(http.StatusOK, true)
}

func storyGetInfo(c echo.Context) error {
	info, err := model.LogGetInfo(myDice.DBLogs)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}
	return c.JSON(http.StatusOK, info)
}

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
	is := model.LogDelete(myDice.DBLogs, v.GroupId, v.Name)
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
	url, err := logSendToBackend(v.GroupId, v.Name)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("跑团日志已上传服务器，链接如下：<br>%s", url))
}

func logSendToBackend(groupId string, logName string) (string, error) {
	dirpath := filepath.Join(myDice.BaseConfig.DataDir, "log-exports")
	_ = os.MkdirAll(dirpath, 0755)

	lines, err := model.LogGetAllLines(myDice.DBLogs, groupId, logName)

	if len(lines) == 0 {
		return "", errors.New("#此log不存在，或条目数为空，名字是否正确？")
	}

	if err != nil {
		return "", err
	}

	if err == nil {
		// 本地进行一个zip留档，以防万一
		fzip, _ := os.CreateTemp(dirpath, dice.FilenameReplace(groupId+"_"+logName)+".*.zip")
		writer := zip.NewWriter(fzip)

		text := ""
		for _, i := range lines {
			timeTxt := time.Unix(i.Time, 0).Format("2006-01-02 15:04:05")
			text += fmt.Sprintf("%s(%v) %s\n%s\n\n", i.Nickname, i.IMUserId, timeTxt, i.Message)
		}

		fileWriter, _ := writer.Create("文本log.txt")
		_, _ = fileWriter.Write([]byte(text))

		data, err := json.Marshal(map[string]interface{}{
			"version": dice.Story_version,
			"items":   lines,
		})
		if err == nil {
			fileWriter2, _ := writer.Create("海豹标准log-粘贴到染色器可格式化.txt")
			_, _ = fileWriter2.Write(data)
		}

		_ = writer.Close()
		_ = fzip.Close()
	}

	if err == nil {
		// 压缩log，发往后端
		data, err := json.Marshal(map[string]interface{}{
			"version": dice.Story_version,
			"items":   lines,
		})

		if err == nil {
			var zlibBuffer bytes.Buffer
			w := zlib.NewWriter(&zlibBuffer)
			_, _ = w.Write(data)
			_ = w.Close()

			return dice.UploadFileToWeizaima(myDice.Logger, logName, myDice.UIEndpoint.UserId, &zlibBuffer), nil
		}
	}
	return "", nil
}
