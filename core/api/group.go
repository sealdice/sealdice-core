package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"sealdice-core/dice"
	"strings"
	"time"
)

func groupList(c echo.Context) error {
	items := []*dice.GroupInfo{}
	for index, i := range myDice.ImSession.ServiceAtNew {
		if !strings.HasPrefix(i.GroupId, "PG-") {
			item := myDice.ImSession.ServiceAtNew[index]
			if item != nil {
				if item.NotInGroup {
					continue
				}
				exts := []string{}
				item.TmpPlayerNum = int64(len(i.Players))
				for _, i := range item.ActivatedExtList {
					exts = append(exts, i.Name)
				}
				i.TmpExtList = exts
				items = append(items, i)
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

	v := dice.GroupInfo{}
	err := c.Bind(&v)
	if err == nil {
		item, exists := myDice.ImSession.ServiceAtNew[v.GroupId]
		if exists {
			// TODO: 要改 但暂时不好改
			item.Active = v.Active
			//for _, i := range myDice.ImSession.EndPoints {
			//	dice.SetBotOnAtGroup()
			//}
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
		GroupId   string `yaml:"groupId" json:"groupId"`
		DiceId    string `yaml:"diceId" json:"diceId"`
		Silence   bool   `yaml:"silence" json:"silence"`
		ExtraText string `yaml:"extraText" json:"extraText"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		// 不太好弄，主要会出现多个帐号在群的情况
		group, exists := myDice.ImSession.ServiceAtNew[v.GroupId]
		if exists {
			for _, ep := range myDice.ImSession.EndPoints {
				if ep.UserId == v.DiceId {
					// 就是这个
					_txt := fmt.Sprintf("Master后台操作退群: 于群组<%s>(%s)中告别", group.GroupName, group.GroupId)
					myDice.Logger.Info(_txt)

					ctx := &dice.MsgContext{Dice: myDice, EndPoint: ep, Session: myDice.ImSession}
					ctx.Notice(_txt)
					dice.SetBotOffAtGroup(ctx, group.GroupId)

					if !v.Silence {
						txtPost := "因长期不使用等原因，骰主后台操作退群"
						if v.ExtraText != "" {
							txtPost += "\n骰主留言: " + v.ExtraText
						}
						dice.ReplyGroup(ctx, &dice.Message{GroupId: v.GroupId}, txtPost)
					}

					time.Sleep(6 * time.Second)
					group.NotInGroup = true

					ep.Adapter.QuitGroup(ctx, v.GroupId)
					return c.String(http.StatusOK, "")
				}
			}
		}
	}

	return c.String(430, "")
}
