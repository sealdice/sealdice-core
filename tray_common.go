//go:build windows || darwin

package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/fy0/systray"

	"sealdice-core/api"
	"sealdice-core/dice"
)

const (
	defaultTrayTooltip = "海豹TRPG骰点核心"
	defaultTrayPort    = "3211"
)

var (
	trayPort atomic.Pointer[string]
)

func getTrayPort() string {
	port := trayPort.Load()
	if port == nil {
		return defaultTrayPort
	}
	return *port
}

func setTrayPort(port string) {
	trayPort.Store(&port)
}

type trayAccountMenu struct {
	root                *systray.MenuItem
	addAccount          *systray.MenuItem
	account             []*systray.MenuItem
	openAccountSettings func()
}

func formatTrayTooltip(dm *dice.DiceManager, version, port string) string {
	text := ""
	if dm != nil {
		text = dice.NormalizeTrayTooltipPrefix(dm.GetTrayTooltip())
	}
	if text == "" {
		text = defaultTrayTooltip
	}
	return fmt.Sprintf("%s %s #%s", text, version, port)
}

func currentTrayTooltip(version, port string) string {
	tooltip := formatTrayTooltip(nil, version, port)
	api.WithCurrentRuntime(func(manager *dice.DiceManager) {
		tooltip = formatTrayTooltip(manager, version, port)
	})
	return tooltip
}

func startTrayAccountMenu(openAccountSettings func()) {
	root := systray.AddMenuItem("账号列表", "查看已配置的平台账号")
	menu := &trayAccountMenu{
		root:                root,
		addAccount:          root.AddSubMenuItem("添加账号", "打开账号设置"),
		openAccountSettings: openAccountSettings,
	}

	go func() {
		for range menu.addAccount.ClickedCh {
			openAccountSettings()
		}
	}()

	menu.refresh(currentTrayAccountTitles())
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			menu.refresh(currentTrayAccountTitles())
		}
	}()
}

func currentTrayAccountTitles() []string {
	var titles []string
	api.WithCurrentRuntime(func(manager *dice.DiceManager) {
		titles = loadTrayAccountTitles(manager.DiceSnapshot())
	})
	return titles
}

func (menu *trayAccountMenu) refresh(titles []string) {
	for len(menu.account) < len(titles) {
		item := menu.root.AddSubMenuItem(titles[len(menu.account)], "")
		go func(item *systray.MenuItem) {
			for range item.ClickedCh {
				menu.openAccountSettings()
			}
		}(item)
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

func loadTrayAccountTitles(instances []*dice.Dice) []string {
	var titles []string
	for _, instance := range instances {
		if instance == nil || instance.ImSession == nil {
			continue
		}
		for _, endpoint := range instance.ImSession.EndPointsSnapshot() {
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
