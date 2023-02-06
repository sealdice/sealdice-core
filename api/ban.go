package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"sealdice-core/dice"
	"strings"
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

func banMapGet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSONBlob(http.StatusOK, myDice.BanList.MapToJSON())
}

func banMapDeleteOne(c echo.Context) error {
	v := dice.BanListInfoItem{}
	err := c.Bind(&v)
	if err != nil {
		return c.String(430, err.Error())
	}
	myDice.BanList.DeleteById(myDice, v.ID)
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
		myDice.BanList.SetTrustById(v.ID, "海豹后台", "骰主后台设置")
	}

	return c.JSON(http.StatusOK, nil)
}

//
//func banMapSet(c echo.Context) error {
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
