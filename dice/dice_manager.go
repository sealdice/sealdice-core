package dice

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

type DiceManager struct {
	Dice         []*Dice
	ServeAddress string
}

type DiceConfigs struct {
	DiceConfigs  []DiceConfig `yaml:"diceConfigs"`
	ServeAddress string       `yaml:"serveAddress"`
}

func (dm *DiceManager) LoadDice() {
	os.MkdirAll("./data/images", 0644)
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

	dm.ServeAddress = dc.ServeAddress
	for _, i := range dc.DiceConfigs {
		newDice := new(Dice)
		newDice.BaseConfig = i
		dm.Dice = append(dm.Dice, newDice)
	}
}

func (dm *DiceManager) Save() {
	var dc DiceConfigs
	dc.ServeAddress = dm.ServeAddress
	for _, i := range dm.Dice {
		dc.DiceConfigs = append(dc.DiceConfigs, i.BaseConfig)
	}

	data, err := yaml.Marshal(dc)
	if err == nil {
		ioutil.WriteFile("./data/dice.yaml", data, 0644)
	}
}

func (dm *DiceManager) InitDice() {
	for _, i := range dm.Dice {
		i.Init()
	}
}

func (dm *DiceManager) TryCreateDefault() {
	if dm.ServeAddress == "" {
		dm.ServeAddress = ":3211"
	}
	if len(dm.Dice) == 0 {
		defaultDice := new(Dice)
		defaultDice.BaseConfig.Name = "default"
		defaultDice.BaseConfig.IsLogPrint = true
		dm.Dice = append(dm.Dice, defaultDice)
	}
}
