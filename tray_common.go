//go:build windows || darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fy0/systray"
	"gopkg.in/yaml.v3"

	"sealdice-core/dice"
	"sealdice-core/logger"
)

const defaultTrayTooltip = "海豹TRPG骰点核心"

type trayAccountConfig struct {
	IMSession struct {
		EndPoints []struct {
			BaseInfo struct {
				Nickname string `yaml:"nickname"`
				UserID   string `yaml:"userId"`
			} `yaml:"baseInfo"`
		} `yaml:"endPoints"`
	} `yaml:"imSession"`
}

type trayAccountMenu struct {
	root       *systray.MenuItem
	addAccount *systray.MenuItem
	account    []*systray.MenuItem
	lastError  string
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
	titles, err := loadTrayAccountTitles(dm)
	if err != nil {
		message := err.Error()
		if message != menu.lastError {
			logger.M().Warnf("刷新托盘账号列表失败: %v", err)
			menu.lastError = message
		}
		return
	}
	if menu.lastError != "" {
		logger.M().Info("托盘账号列表已恢复刷新")
		menu.lastError = ""
	}

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

func loadTrayAccountTitles(dm *dice.DiceManager) ([]string, error) {
	var titles []string
	for _, instance := range dm.Dice {
		path := filepath.Join(instance.BaseConfig.DataDir, "serve.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("读取 %s: %w", path, err)
		}

		var config trayAccountConfig
		if err = yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析 %s: %w", path, err)
		}
		for _, endpoint := range config.IMSession.EndPoints {
			titles = append(titles, fmt.Sprintf("%s(%s)", endpoint.BaseInfo.Nickname, endpoint.BaseInfo.UserID))
		}
	}
	return titles, nil
}
