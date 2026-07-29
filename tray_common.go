//go:build windows || darwin

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fy0/systray"

	"sealdice-core/dice"
)

const defaultTrayTooltip = "海豹TRPG骰点核心"

type trayAccountMenu struct {
	root       *systray.MenuItem
	addAccount *systray.MenuItem
	account    []*systray.MenuItem
}

func formatTrayTooltip(dm *dice.DiceManager, version, port string) string {
	text := strings.TrimSpace(dm.GetTrayTooltip())
	if text == "" {
		text = defaultTrayTooltip
	}
	return fmt.Sprintf("%s %s #%s", text, version, port)
}

func startTrayAccountMenu(dm *dice.DiceManager, openAccountSettings func()) {
	root := systray.AddMenuItem("账号列表", "查看已配置的平台账号")
	menu := &trayAccountMenu{
		root:       root,
		addAccount: root.AddSubMenuItem("添加账号", "打开账号设置"),
	}

	go func() {
		for range menu.addAccount.ClickedCh {
			openAccountSettings()
		}
	}()

	menu.refresh(dm)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			menu.refresh(dm)
		}
	}()
}

func (menu *trayAccountMenu) refresh(dm *dice.DiceManager) {
	titles := loadTrayAccountTitles(dm)

	for len(menu.account) < len(titles) {
		item := menu.root.AddSubMenuItem(titles[len(menu.account)], "")
		item.Disable()
		menu.account = append(menu.account, item)
	}

	for index, item := range menu.account {
		if index < len(titles) {
			item.SetTitle(titles[index])
			item.Show()
		} else {
			item.Hide()
		}
	}

	if len(titles) == 0 {
		menu.addAccount.Show()
	} else {
		menu.addAccount.Hide()
	}
}

func loadTrayAccountTitles(dm *dice.DiceManager) []string {
	var titles []string
	for _, instance := range dm.Dice {
		if instance == nil || instance.ImSession == nil {
			continue
		}
		for _, endpoint := range instance.ImSession.EndPoints {
			if endpoint == nil {
				continue
			}
			titles = append(titles, fmt.Sprintf(
				"%s(%s) [%s]",
				endpoint.Nickname,
				endpoint.UserID,
				trayEndpointStateText(endpoint.State),
			))
		}
	}
	return titles
}

func trayEndpointStateText(state dice.EndpointState) string {
	switch state {
	case dice.StateDisconnected:
		return "已断开"
	case dice.StateConnected:
		return "已连接"
	case dice.StateConnecting:
		return "连接中"
	case dice.StateConnectionFailed:
		return "连接失败"
	default:
		return "未知状态"
	}
}
