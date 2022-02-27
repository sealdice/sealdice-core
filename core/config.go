package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type AttributeOrderOthers struct {
	SortBy string `yaml:"sortBy"` // time | name | value desc
}

type AttributeOrder struct {
	Top []string `yaml:"top,flow"`
	Others AttributeOrderOthers`yaml:"others"`
}

type AttributeConfigs struct {
	Alias map[string][]string `yaml:"alias"`
	Order AttributeOrder `yaml:"order"`
}

type TextTemplateWithWeight = map[string]map[string]uint
type TextTemplateWithWeightDict = map[string]TextTemplateWithWeight

const BASE_CONFIG = "./data/config.yaml"
const CONFIG_ATTRIBUTE_FILE = "./data/configs/attribute.yaml"
const CONFIG_TEXT_TEMPLATE_FILE = "./data/configs/text-template.yaml"


func configInit() {
	os.MkdirAll("./data/configs", 0644)

	a := AttributeConfigs{
		Alias: map[string][]string{
			"理智": {"san", "san值", "理智值"},
			"力量": {"str"},
			"体质": {"con"},
			"体型": {"siz"},
			"敏捷": {"dex"},
			"外貌": {"app"},
			"意志": {"pow"},
			"教育": {"edu", "知识"}, // 教育和知识等值而不是一回事，注意
			"智力": {"int", "灵感"}, // 智力和灵感等值而不是一回事，注意

			"幸运": {"luck", "幸运值"},
			"生命值": {"hp", "生命", "血量"},
			"魔法值": {"mp", "魔法", "魔力", "魔力值"},
			"克苏鲁神话": {"cm", "克苏鲁"},
			"图书馆使用": {"图书馆"},
			"链枷": {"连枷"},
		},
		Order: AttributeOrder{
			Top: []string{"力量", "敏捷", "体质", "体型", "外貌", "智力", "意志", "教育", "理智", "克苏鲁神话", "生命值", "魔法值"},
			Others: AttributeOrderOthers{ SortBy: "name" },
		},
	}

	buf, err := yaml.Marshal(a)
	if err != nil {
		fmt.Println(err)
	} else {
		ioutil.WriteFile(CONFIG_ATTRIBUTE_FILE, buf, 0644)
	}

	b := TextTemplateWithWeightDict{
		"COC7": TextTemplateWithWeight{
			"设置房规-0": {
				"已切换房规为0: 出1大成功，不满50出96-100大失败，满50出100大失败(COC7规则书)": 1,
			},
			"设置房规-1": {
				"已切换房规为1: 不满50出1大成功，不满50出96-100大失败，满50出100大失败": 1,
			},
			"设置房规-2": {
				"已切换房规为2: 出1-5且判定成功为大成功，出96-100且判定失败为大失败": 1,
			},
			"设置房规-3": {
				"已切换房规为3: 出1-5大成功，出96-100大失败(即大成功/大失败时无视判定结果)": 1,
			},
			"设置房规-4": {
				"已切换房规为4: 出1-5且≤(成功率/10)为大成功，不满50出>=96+(成功率/10)为大失败，满50出100大失败": 1,
			},
			"设置房规-5": {
				"已切换房规为5: 出1-2且≤(成功率/5)为大成功，不满50出96-100大失败，满50出99-100大失败": 1,
			},
			"设置房规-当前": {
				"当前房规: {$t房规}": 1,
			},
			"判定-大失败": {
				"大失败！": 1,
			},
			"判定-失败": {
				"失败！": 1,
			},
			"判定-成功-普通": {
				"成功": 1,
			},
			"判定-成功-困难": {
				"成功(困难)": 1,
			},
			"判定-成功-极难": {
				"成功(极难)": 1,
			},
			"判定-大成功": {
				"运气不错，大成功！": 1,
				"大成功!": 1,
			},
		},
		"核心": TextTemplateWithWeight{
			"名字": {
				"SealDice": 1,
			},
			"骰子开启": {
				"{核心:名字} 已启用(开发中) {常量:VERSION}": 1,
			},
			"骰子关闭": {
				"{核心:名字} 停止服务": 1,
			},
			"骰子退群预告": {
				"收到指令，5s后将退出当前群组": 1,
			},
			"骰子保存设置": {
				"数据已保存": 1,
			},
			//"roll前缀":{
			//	"为了{$t原因}": 1,
			//},
			//"roll": {
			//	"{$t原因}{$t玩家} 掷出了 {$t骰点参数}{$t计算过程}={$t结果}${tASM}": 1,
			//},
			"暗骰-群内": {
				"黑暗的角落里，传来命运转动的声音": 1,
				"诸行无常，命数无定，答案只有少数人知晓": 1,
				"命运正在低语！": 1,
			},
			//"暗骰-私发": {
			//	"来自群<{$群名}>({$t群号})的暗骰，{核心:roll}": 1,
			//},
			"昵称-重置": {
				"{$tQQ昵称}({$tQQ})的昵称已重置为{$t玩家}": 1,
			},
			"昵称-改名": {
				"{$tQQ昵称}({$tQQ})的昵称被设定为{$t玩家}": 1,
			},
			"设定默认骰子面数": {
				"设定默认骰子面数为 {$t个人骰子面数}": 1,
			},
			"设定默认骰子面数-错误": {
				"设定默认骰子面数: 格式错误": 1,
			},
			"设定默认骰子面数-重置": {
				"重设默认骰子面数为默认值": 1,
			},
			"角色管理-加载成功": {
				"角色{$t玩家}加载成功，欢迎回来": 1,
			},
			"角色管理-角色不存在": {
				"无法加载角色：你所指定的角色不存在": 1,
			},
			"角色管理-序列化失败": {
				"无法加载/保存角色：序列化失败": 1,
			},
			"角色管理-储存成功": {
				"角色{$t新角色名}储存成功": 1,
			},
			"角色管理-删除成功": {
				"角色{$t新角色名}删除成功": 1,
			},
			// 角色管理-
		},
	}

	buf, err = yaml.Marshal(b)
	if err != nil {
		fmt.Println(err)
	} else {
		ioutil.WriteFile(CONFIG_TEXT_TEMPLATE_FILE, buf, 0644)
	}
}
