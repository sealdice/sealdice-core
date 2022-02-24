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

var CONFIG_ATTRIBUTE_FILE = "./data/configs/attribute.yaml"


func configInit() {
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
			Top: []string{"力量", "敏捷", "体质", "体型", "外貌", "智力", "意志", "教育", "理智", "克苏鲁神话"},
			Others: AttributeOrderOthers{ SortBy: "name" },
		},
	}

	buf, err := yaml.Marshal(a)
	if err != nil {
		fmt.Println(err)
	} else {
		os.MkdirAll("./data/configs", 0644)
		ioutil.WriteFile(CONFIG_ATTRIBUTE_FILE, buf, 0644)
	}

}
