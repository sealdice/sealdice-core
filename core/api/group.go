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
			item.Active = !item.Active
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
		GroupId string `yaml:"groupId" json:"groupId"`
		DiceId  string `yaml:"diceId" json:"diceId"`
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

					ctx := &dice.MsgContext{Dice: myDice, EndPoint: ep}
					ctx.Notice(_txt)
					group.Active = false
					time.Sleep(6 * time.Second)
					group.NotInGroup = true
					ep.Adapter.QuitGroup(ctx, v.GroupId)
					break
				}
			}
		}
	}

	return c.String(430, "")
}
