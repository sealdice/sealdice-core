package api

import (
	"bufio"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sealdice-core/dice"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func banConfigGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, myDice.BanList)
}

func banConfigSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := dice.BanListInfo{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}

	if v.ThresholdWarn == 0 {
		v.ThresholdWarn = 100
	}

	if v.ThresholdBan == 0 {
		v.ThresholdBan = 200
	}

	myDice.BanList.BanBehaviorRefuseReply = v.BanBehaviorRefuseReply
	myDice.BanList.BanBehaviorRefuseInvite = v.BanBehaviorRefuseInvite
	myDice.BanList.BanBehaviorQuitLastPlace = v.BanBehaviorQuitLastPlace
	myDice.BanList.BanBehaviorQuitPlaceImmediately = v.BanBehaviorQuitPlaceImmediately
	myDice.BanList.BanBehaviorQuitIfAdmin = v.BanBehaviorQuitIfAdmin
	myDice.BanList.ScoreReducePerMinute = v.ScoreReducePerMinute
	myDice.BanList.ThresholdWarn = v.ThresholdWarn
	myDice.BanList.ThresholdBan = v.ThresholdBan
	myDice.BanList.ScoreGroupMuted = v.ScoreGroupMuted
	myDice.BanList.ScoreGroupKicked = v.ScoreGroupKicked
	myDice.BanList.ScoreTooManyCommand = v.ScoreTooManyCommand

	myDice.BanList.JointScorePercentOfGroup = v.JointScorePercentOfGroup
	myDice.BanList.JointScorePercentOfInviter = v.JointScorePercentOfInviter
	myDice.MarkModified()

	return c.JSON(http.StatusOK, myDice.BanList)
}

func banMapList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	lst := myDice.GetBanList()
	return c.JSON(http.StatusOK, lst)
}

func banMapDeleteOne(c echo.Context) error {
	v := dice.BanListInfoItem{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}
	myDice.BanList.DeleteByID(myDice, v.ID)
	return c.JSON(http.StatusOK, nil)
}

func banMapAddOne(c echo.Context) error {
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := dice.BanListInfoItem{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}
	if v.Rank == dice.BanRankBanned {
		score := myDice.BanList.ThresholdBan
		reason := "骰主后台设置"
		if len(v.Reasons) > 0 {
			reason = v.Reasons[0]
		}

		prefix := strings.Split(v.ID, ":")[0]
		platform := strings.Replace(prefix, "-Group", "", 1)
		for _, i := range myDice.ImSession.EndPoints {
			if i.Platform == platform && i.Enable {
				v2 := myDice.BanList.AddScoreBase(v.ID, score, "海豹后台", reason, &dice.MsgContext{Dice: myDice, EndPoint: i})
				if v2 != nil {
					if v.Name != "" {
						v2.Name = v.Name
					}
				}
			}
		}
	}
	if v.Rank == dice.BanRankTrusted {
		myDice.BanList.SetTrustByID(v.ID, "海豹后台", "骰主后台设置")
	}

	return c.JSON(http.StatusOK, nil)
}

//
// func banMapSet(c echo.Context) error {
//	if !doAuth(c) {
//		return c.JSON(http.StatusForbidden, nil)
//	}
//
//	v := struct {
//		data []byte
//	}{}
//	err := c.Bind(&v)
//	if err != nil {
//		return c.String(430, err.Error())
//	}
//
//	myDice.BanList.LoadMapFromJSON(v.data)
//	return c.JSON(http.StatusOK, nil)
//}

func banExport(c echo.Context) error {
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	lst := myDice.GetBanList()

	temp, _ := os.CreateTemp("", "黑白名单-*.json")
	defer func() {
		_ = temp.Close()
		_ = os.RemoveAll(temp.Name())
	}()
	writer := bufio.NewWriter(temp)
	err := json.NewEncoder(writer).Encode(&lst)
	_ = writer.Flush()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	c.Response().Header().Add("Cache-Control", "no-store")
	return c.Attachment(temp.Name(), "黑白名单.json")
}

func banImport(c echo.Context) error {
	if !doAuth(c) {
		return c.NoContent(http.StatusForbidden)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	src, err := file.Open()
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	defer func(src multipart.File) {
		_ = src.Close()
	}(src)

	var lst []*dice.BanListInfoItem
	data, err := io.ReadAll(src)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	err = json.Unmarshal(data, &lst)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}

	now := time.Now()
	for _, item := range lst {
		item.UpdatedAt = now.Unix()
		var newReasons []string
		for _, reason := range item.Reasons {
			if !strings.HasSuffix(reason, "（来自导入）") {
				newReasons = append(newReasons, reason+"（来自导入）")
			}
		}
		item.Reasons = newReasons
		myDice.BanList.Map.Store(item.ID, item)
	}
	myDice.BanList.SaveChanged(myDice)

	return Success(&c, Response{})
}
