package migrate

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"sealdice-core/dice"
)

type v131DeprecatedConfig struct {
	// 转移到自定义文案的 核心:骰子帮助文本_骰主
	HelpMasterInfo string `yaml:"helpMasterInfo"` // help中骰主信息
	// 转移到自定义文案的 核心:骰子帮助文本_协议
	HelpMasterLicense string `yaml:"helpMasterLicense"` // help中使用协议
	// 转移到自定义文案的 核心:骰子状态附加文本
	CustomBotExtraText string `yaml:"customBotExtraText"` // bot自定义文本
	// 转移到自定义文案的 其他:抽牌_列表
	CustomDrawKeysText string `yaml:"customDrawKeysText"` // draw keys自定义文本
	// 迁移到自定义文案的 其他:抽牌_列表 后该值弃用
	CustomDrawKeysTextEnable bool `yaml:"customDrawKeysTextEnable"` // 应用draw keys自定义文本
}

func V131DeprecatedConfig2CustomText() error {
	confPath := filepath.Join("./data/default/serve.yaml")
	customTextPath := filepath.Join("./data/default/configs/text-template.yaml")
	data, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}

	deprecatedConf := v131DeprecatedConfig{}
	err = yaml.Unmarshal(data, &deprecatedConf)
	if err != nil {
		return err
	}

	data2, err := os.ReadFile(customTextPath)
	if err != nil {
		return err
	}
	var customTexts dice.TextTemplateWithWeightDict
	err = yaml.Unmarshal(data2, &customTexts)
	if err != nil {
		return err
	}

	var needUpdateCustomText bool
	if deprecatedConf.HelpMasterInfo != "" {
		needUpdateCustomText = true
		customTexts["核心"]["骰子帮助文本_骰主"][0][0] = deprecatedConf.HelpMasterInfo
		customTexts["核心"]["骰子帮助文本_骰主"][0][1] = 1
	}
	if deprecatedConf.HelpMasterLicense != "" {
		needUpdateCustomText = true
		customTexts["核心"]["骰子帮助文本_协议"][0][0] = deprecatedConf.HelpMasterLicense
		customTexts["核心"]["骰子帮助文本_协议"][0][1] = 1
	}
	if deprecatedConf.CustomBotExtraText != "" {
		needUpdateCustomText = true
		customTexts["核心"]["骰子状态附加文本"][0][0] = deprecatedConf.CustomBotExtraText
		customTexts["核心"]["骰子状态附加文本"][0][1] = 1
	}
	if deprecatedConf.CustomDrawKeysText != "" && deprecatedConf.CustomDrawKeysTextEnable {
		needUpdateCustomText = true
		customTexts["其他"]["抽牌_列表"][0][0] = deprecatedConf.CustomDrawKeysText
		customTexts["其他"]["抽牌_列表"][0][1] = 1
	}

	if needUpdateCustomText {
		fmt.Println("检测到旧设置项需要迁移到自定义文案，试图进行迁移")
		// 保存迁移后的 config.yaml
		newConf := make(map[string]interface{})
		err = yaml.Unmarshal(data, &newConf)
		if err != nil {
			return err
		}
		delete(newConf, "helpMasterInfo")
		delete(newConf, "helpMasterLicense")
		delete(newConf, "customBotExtraText")
		delete(newConf, "customDrawKeysText")
		delete(newConf, "customDrawKeysTextEnable")

		newData, err := yaml.Marshal(newConf)
		if err != nil {
			return err
		}
		err = os.WriteFile(confPath, newData, 0644)
		if err != nil {
			return err
		}

		// 保存修改了的 custom text 设置
		newData2, err := yaml.Marshal(customTexts)
		if err != nil {
			return err
		}
		err = os.WriteFile(customTextPath, newData2, 0644)
		if err != nil {
			return err
		}
		fmt.Println("旧设置项迁移到自定义文案成功")
	}

	return nil
}
