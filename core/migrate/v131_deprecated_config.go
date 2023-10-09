package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sealdice-core/dice"

	"gopkg.in/yaml.v3"
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

	if _, err := os.Stat(confPath); err != nil {
		// 如果文件没找到，那直接算成功，跳过流程
		return nil
	}

	customTextPath := filepath.Join("./data/default/configs/text-template.yaml")
	customTextBakPath := filepath.Join("./data/default/configs/text-template.yaml.bak")
	confData, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}

	deprecatedConf := v131DeprecatedConfig{}
	err = yaml.Unmarshal(confData, &deprecatedConf)
	if err != nil {
		return err
	}

	customTextData, err := os.ReadFile(customTextPath)
	if err != nil {
		return err
	}
	var customTexts dice.TextTemplateWithWeightDict
	err = yaml.Unmarshal(customTextData, &customTexts)
	if err != nil {
		return err
	}

	if customTexts == nil {
		customTexts = make(dice.TextTemplateWithWeightDict)
	}
	if customTexts["核心"] == nil {
		customTexts["核心"] = make(dice.TextTemplateWithWeight)
	}
	if customTexts["核心"] == nil {
		customTexts["其他"] = make(dice.TextTemplateWithWeight)
	}

	var needUpdateCustomText bool
	if deprecatedConf.HelpMasterInfo != "" {
		needUpdateCustomText = true
		if len(customTexts["核心"]["骰子帮助文本_骰主"]) == 0 {
			customTexts["核心"]["骰子帮助文本_骰主"] = dice.TextTemplateItemList{dice.TextTemplateItem{"", 0}}
		}
		customTexts["核心"]["骰子帮助文本_骰主"][0][0] = deprecatedConf.HelpMasterInfo
		customTexts["核心"]["骰子帮助文本_骰主"][0][1] = 1
	}
	if deprecatedConf.HelpMasterLicense != "" {
		needUpdateCustomText = true
		if len(customTexts["核心"]["骰子帮助文本_协议"]) == 0 {
			customTexts["核心"]["骰子帮助文本_协议"] = dice.TextTemplateItemList{dice.TextTemplateItem{"", 0}}
		}
		customTexts["核心"]["骰子帮助文本_协议"][0][0] = deprecatedConf.HelpMasterLicense
		customTexts["核心"]["骰子帮助文本_协议"][0][1] = 1
	}
	if deprecatedConf.CustomBotExtraText != "" {
		needUpdateCustomText = true
		if len(customTexts["核心"]["骰子状态附加文本"]) == 0 {
			customTexts["核心"]["骰子状态附加文本"] = dice.TextTemplateItemList{dice.TextTemplateItem{"", 0}}
		}
		customTexts["核心"]["骰子状态附加文本"][0][0] = deprecatedConf.CustomBotExtraText
		customTexts["核心"]["骰子状态附加文本"][0][1] = 1
	}
	if deprecatedConf.CustomDrawKeysText != "" && deprecatedConf.CustomDrawKeysTextEnable {
		needUpdateCustomText = true
		if len(customTexts["其他"]["抽牌_列表"]) == 0 {
			customTexts["其他"]["抽牌_列表"] = dice.TextTemplateItemList{dice.TextTemplateItem{"", 0}}
		}
		customTexts["其他"]["抽牌_列表"][0][0] = deprecatedConf.CustomDrawKeysText
		customTexts["其他"]["抽牌_列表"][0][1] = 1
	}

	if needUpdateCustomText {
		fmt.Println("检测到旧设置项需要迁移到自定义文案，试图进行迁移")

		// 保存修改了的 custom text 设置
		newData, err := yaml.Marshal(customTexts)
		if err != nil {
			return err
		}
		// 先备份
		err = os.WriteFile(customTextBakPath, customTextData, 0644)
		if err != nil {
			return err
		}
		err = os.WriteFile(customTextPath, newData, 0644)
		if err != nil {
			return err
		}

		// 保存迁移后的 config.yaml
		newConf := make(map[string]interface{})
		err = yaml.Unmarshal(confData, &newConf)
		if err != nil {
			return err
		}
		delete(newConf, "helpMasterInfo")
		delete(newConf, "helpMasterLicense")
		delete(newConf, "customBotExtraText")
		delete(newConf, "customDrawKeysText")
		delete(newConf, "customDrawKeysTextEnable")

		newData2, err := yaml.Marshal(newConf)
		if err != nil {
			return err
		}
		err = os.WriteFile(confPath, newData2, 0644)
		if err != nil {
			return err
		}

		fmt.Println("旧设置项迁移到自定义文案成功")
	}

	return nil
}
