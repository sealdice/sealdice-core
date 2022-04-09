package dice

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

type DiceManager struct {
	Dice                 []*Dice
	ServeAddress         string
	Help                 *HelpManager
	IsHelpReloading      bool
	UseDictForTokenizer  bool
	HelpDocEngineType    int
	progressExitGroupWin ProcessExitGroup

	UIPasswordHash string
	UIPasswordSalt string
	AccessTokens   map[string]bool
}

type DiceConfigs struct {
	DiceConfigs       []DiceConfig `yaml:"diceConfigs"`
	ServeAddress      string       `yaml:"serveAddress"`
	WebUIAddress      string       `yaml:"webUIAddress"`
	HelpDocEngineType int          `yaml:"helpDocEngineType"`

	UIPasswordSalt string   `yaml:"UIPasswordFrontendSalt"`
	UIPasswordHash string   `yaml:"uiPasswordHash"`
	AccessTokens   []string `yaml:"accessTokens"`
}

func (dm *DiceManager) InitHelp() {
	os.MkdirAll("./data/helpdoc", 0755)
	dm.Help = new(HelpManager)
	dm.Help.Parent = dm
	dm.Help.EngineType = dm.HelpDocEngineType
	dm.Help.Load()
}

func (dm *DiceManager) LoadDice() {
	os.MkdirAll("./data/images", 0755)
	os.MkdirAll("./data/decks", 0755)
	os.MkdirAll("./data/names", 0755)
	ioutil.WriteFile("./data/images/sealdice.png", ICON_PNG, 0644)

	data, err := ioutil.ReadFile("./data/dice.yaml")
	if err != nil {
		return
	}

	var dc DiceConfigs
	err = yaml.Unmarshal(data, &dc)
	if err != nil {
		fmt.Println("读取 data/dice.yaml 发生错误: 配置文件格式不正确")
		panic(err)
	}

	if dc.UIPasswordSalt == "" {
		dc.UIPasswordSalt = RandStringBytesMaskImprSrcSB2(32)
	}

	dm.ServeAddress = dc.ServeAddress
	dm.HelpDocEngineType = dc.HelpDocEngineType
	dm.UIPasswordHash = dc.UIPasswordHash
	dm.UIPasswordSalt = dc.UIPasswordSalt
	dm.AccessTokens = map[string]bool{}

	for _, i := range dc.AccessTokens {
		dm.AccessTokens[i] = true
	}

	for _, i := range dc.DiceConfigs {
		newDice := new(Dice)
		newDice.BaseConfig = i
		dm.Dice = append(dm.Dice, newDice)
	}
}

func (dm *DiceManager) Save() {
	var dc DiceConfigs
	dc.ServeAddress = dm.ServeAddress
	dc.HelpDocEngineType = dm.HelpDocEngineType
	dc.UIPasswordSalt = dm.UIPasswordSalt
	dc.UIPasswordHash = dm.UIPasswordHash
	dc.AccessTokens = []string{}

	for k, _ := range dm.AccessTokens {
		dc.AccessTokens = append(dc.AccessTokens, k)
	}

	for _, i := range dm.Dice {
		dc.DiceConfigs = append(dc.DiceConfigs, i.BaseConfig)
	}

	data, err := yaml.Marshal(dc)
	if err == nil {
		ioutil.WriteFile("./data/dice.yaml", data, 0644)
	}
}

func (dm *DiceManager) InitDice() {
	dm.InitHelp()

	g, err := NewProcessExitGroup()
	if err != nil {
		fmt.Println("进程组创建失败，若进程崩溃，gocqhttp进程可能需要手动结束。")
	} else {
		dm.progressExitGroupWin = g
	}

	for _, i := range dm.Dice {
		i.Parent = dm
		i.Init()
	}

	if len(dm.Dice) >= 1 {
		dm.AddHelpWithDice(dm.Dice[0])
	}
}

func (dm *DiceManager) TryCreateDefault() {
	if dm.ServeAddress == "" {
		dm.ServeAddress = "0.0.0.0:3211"
	}

	if len(dm.Dice) == 0 {
		defaultDice := new(Dice)
		defaultDice.BaseConfig.Name = "default"
		defaultDice.BaseConfig.IsLogPrint = true
		defaultDice.MessageDelayRangeStart = 0.4
		defaultDice.MessageDelayRangeEnd = 0.9
		dm.Dice = append(dm.Dice, defaultDice)
	}
}
