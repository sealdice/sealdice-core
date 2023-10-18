package api

import (
	"fmt"
	"net/http"
	"sealdice-core/dice"
	"sealdice-core/dice/model"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func groupList(c echo.Context) error {
	var items []*dice.GroupInfo
	for groupID, item := range myDice.ImSession.ServiceAtNew {
		item.GroupID = groupID
		if !strings.HasPrefix(item.GroupID, "PG-") {
			if item != nil {
				var exts []string
				item.TmpPlayerNum, _ = model.GroupPlayerNumGet(myDice.DBData, item.GroupID)
				// item.TmpPlayerNum = int64(len(i.Players))
				for _, i := range item.ActivatedExtList {
					exts = append(exts, i.Name)
				}
				item.TmpExtList = exts

				if item.DiceIDExistsMap.Len() > 0 {
					items = append(items, item)
				}
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

func groupSetOne(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Active  bool   `yaml:"active" json:"active"`
		GroupID string `yaml:"groupId" json:"groupId"`
		DiceID  string `yaml:"diceId" json:"diceId"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		_, exists := myDice.ImSession.ServiceAtNew[v.GroupID]
		if exists {
			for _, ep := range myDice.ImSession.EndPoints {
				// if ep.UserId == v.DiceId {
				ctx := &dice.MsgContext{Dice: myDice, EndPoint: ep, Session: myDice.ImSession}
				if v.Active {
					dice.SetBotOnAtGroup(ctx, v.GroupID)
				} else {
					dice.SetBotOffAtGroup(ctx, v.GroupID)
				}
				//}
			}
		}
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}

func groupQuit(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}
	v := struct {
		GroupID   string `yaml:"groupId" json:"groupId"`
		DiceID    string `yaml:"diceId" json:"diceId"`
		Silence   bool   `yaml:"silence" json:"silence"`
		ExtraText string `yaml:"extraText" json:"extraText"`
	}{}
	err := c.Bind(&v)

	if err != nil {
		return c.String(430, "")
	}

	// 不太好弄，主要会出现多个帐号在群的情况
	group, exists := myDice.ImSession.ServiceAtNew[v.GroupID]
	if !exists {
		return c.String(430, "")
	}

	for _, ep := range myDice.ImSession.EndPoints {
		if ep.UserID != v.DiceID {
			continue
		}
		// 就是这个
		_txt := fmt.Sprintf("Master后台操作退群: 于群组<%s>(%s)中告别", group.GroupName, group.GroupID)
		myDice.Logger.Info(_txt)

		ctx := &dice.MsgContext{Dice: myDice, EndPoint: ep, Session: myDice.ImSession}
		ctx.Notice(_txt)
		// dice.SetBotOffAtGroup(ctx, group.GroupId)

		if !v.Silence {
			txtPost := dice.DiceFormatTmpl(ctx, "核心:提示_手动退群前缀")
			if v.ExtraText != "" {
				txtPost += "\n骰主留言: " + v.ExtraText
			}
			dice.ReplyGroup(ctx, &dice.Message{GroupID: v.GroupID}, txtPost)
		}

		group.DiceIDExistsMap.Delete(v.DiceID)
		time.Sleep(6 * time.Second)
		group.UpdatedAtTime = time.Now().Unix()

		ep.Adapter.QuitGroup(ctx, v.GroupID)
		return c.String(http.StatusOK, "")
	}
	return c.String(430, "")
}
