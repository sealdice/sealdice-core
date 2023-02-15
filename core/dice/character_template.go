package dice

import (
	"fmt"
	"strings"
)

type AttributeOrderOthers struct {
	SortBy string `yaml:"sortBy"` // time | Name | value desc
}

type AttributeOrder struct {
	Top    []string             `yaml:"top,flow"`
	Others AttributeOrderOthers `yaml:"others"`
}

type AttributeConfigs struct {
	Alias map[string][]string `yaml:"alias"`
	Order AttributeOrder      `yaml:"order"`
}

// ---------

type AttrSettings struct {
	Top     []string          `yaml:"top,flow" json:"top,flow"`
	SortBy  string            `yaml:"sortBy" json:"sortBy"`   // time | Name | value desc
	Ignores []string          `yaml:"ignores" json:"ignores"` // 这里面的属性将不被显示
	ShowAs  map[string]string `yaml:"showAs" json:"showAs"`   // 展示形式，即st show时格式
	Setter  map[string]string `yaml:"setter" json:"setter"`   // st写入时执行这个
}

type NameTemplateItem struct {
	Template string `yaml:"template" json:"template"`
	HelpText string `yaml:"helpText" json:"helpText"`
}

type CharacterTemplate struct {
	KeyName      string                      `yaml:"keyName" json:"keyName"`           // 模板名字
	FullName     string                      `yaml:"fullName" json:"fullName"`         // 全名
	Authors      []string                    `yaml:"authors" json:"authors"`           // 作者
	Version      string                      `yaml:"version" json:"version"`           // 版本
	UpdatedTime  string                      `yaml:"updatedTime" json:"updatedTime"`   // 更新日期
	TemplateVer  string                      `yaml:"templateVer" json:"templateVer"`   // 模板版本
	NameTemplate map[string]NameTemplateItem `yaml:"nameTemplate" json:"nameTemplate"` // 名片模板
	AttrSettings AttrSettings                `yaml:"attrSettings" json:"attrSettings"` // 默认展示顺序

	Defaults         map[string]int64    `yaml:"defaults" json:"defaults"`                 // 默认值
	DefaultsComputed map[string]string   `yaml:"defaultsComputed" json:"defaultsComputed"` // 计算类型
	Alias            map[string][]string `yaml:"alias" json:"alias"`                       // 别名/同义词

	TextMap         *TextTemplateWithWeightDict `yaml:"textMap" json:"textMap"` // UI文本
	TextMapHelpInfo *TextTemplateWithHelpDict   `yaml:"TextMapHelpInfo" json:"textMapHelpInfo"`

	//BasedOn           string                 `yaml:"based-on"`           // 基于规则

	AliasMap *SyncMap[string, string] `yaml:"-" json:"-"` // 别名/同义词
}

func (t *CharacterTemplate) GetAlias(varname string) string {
	v2, exists := t.AliasMap.Load(strings.ToLower(varname))
	if exists {
		varname = v2
	}
	return varname
}

func (t *CharacterTemplate) GetDefaultValueEx(ctx *MsgContext, varname string) *VMValue {
	name := t.GetAlias(varname)
	if val, exists := t.Defaults[name]; exists {
		return VMValueNew(VMTypeInt64, val)
	}

	if expr, exists := t.DefaultsComputed[name]; exists {
		r, _, err := ctx.Dice.ExprEvalBase(expr, ctx, RollExtraFlags{
			DefaultDiceSideNum: getDefaultDicePoints(ctx),
		})
		if err == nil {
			return &r.VMValue
		}
	}

	return VMValueNew(VMTypeInt64, int64(0))
}

func (t *CharacterTemplate) GetShowAs(ctx *MsgContext, k string) (*VMValue, error) {
	// 有showas的情况
	if expr, exists := t.AttrSettings.ShowAs[k]; exists {
		ctx.SystemTemplate = t
		fmt.Println("!!!", expr)
		r, _, err := ctx.Dice.ExprTextBase(expr, ctx, RollExtraFlags{
			DefaultDiceSideNum: getDefaultDicePoints(ctx),
		})
		if err == nil {
			return &r.VMValue, nil
		}
		return nil, err
	}
	// 显示本体
	if ch, exists := ctx.ChVarsGet(); exists {
		_v, exists := ch.Get(k)
		if exists {
			return _v.(*VMValue), nil
		}
	}
	// 默认值
	v := t.GetDefaultValueEx(ctx, k)
	if v != nil {
		return v, nil
	}

	// 不存在的值，强行补0
	return &VMValue{TypeId: VMTypeInt64, Value: int64(0)}, nil
}
