package dice

import (
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

type AttrConfig struct {
	Display string `yaml:"display" json:"display"` // 展示形式，即st show时格式，默认为顺序展示

	Top          []string          `yaml:"top,flow" json:"top,flow"`         //nolint
	SortBy       string            `yaml:"sortBy" json:"sortBy"`             // time | Name | value desc
	Ignores      []string          `yaml:"ignores" json:"ignores"`           // 这里面的属性将不被显示
	ShowAs       map[string]string `yaml:"showAs" json:"showAs"`             // 展示形式，即st show时格式
	Setter       map[string]string `yaml:"setter" json:"setter"`             // st写入时执行这个，未实装
	ItemsPerLine int               `yaml:"itemsPerLine" json:"itemsPerLine"` // 每行显示几个属性，默认4
}

type NameTemplateItem struct {
	Template string `yaml:"template" json:"template"`
	HelpText string `yaml:"helpText" json:"helpText"`
}

type SetConfig struct {
	RelatedExt []string `yaml:"relatedExt" json:"relatedExt"` // 关联扩展
	DiceSides  int64    `yaml:"diceSides" json:"diceSides"`   // 骰子面数
	Keys       []string `yaml:"keys" json:"keys"`             // 可用于 .set xxx 的key
	EnableTip  string   `yaml:"enableTip" json:"enableTip"`   // 启用提示
}

type GameSystemTemplate struct {
	Name         string                      `yaml:"name" json:"name"`                 // 模板名字
	FullName     string                      `yaml:"fullName" json:"fullName"`         // 全名
	Authors      []string                    `yaml:"authors" json:"authors"`           // 作者
	Version      string                      `yaml:"version" json:"version"`           // 版本
	UpdatedTime  string                      `yaml:"updatedTime" json:"updatedTime"`   // 更新日期
	TemplateVer  string                      `yaml:"templateVer" json:"templateVer"`   // 模板版本
	NameTemplate map[string]NameTemplateItem `yaml:"nameTemplate" json:"nameTemplate"` // 名片模板
	AttrConfig   AttrConfig                  `yaml:"attrConfig" json:"attrConfig"`     // 默认展示顺序
	SetConfig    SetConfig                   `yaml:"setConfig" json:"setConfig"`       // .set 命令设置

	Defaults         map[string]int64    `yaml:"defaults" json:"defaults"`                 // 默认值
	DefaultsComputed map[string]string   `yaml:"defaultsComputed" json:"defaultsComputed"` // 计算类型
	Alias            map[string][]string `yaml:"alias" json:"alias"`                       // 别名/同义词

	TextMap         *TextTemplateWithWeightDict `yaml:"textMap" json:"textMap"` // UI文本
	TextMapHelpInfo *TextTemplateWithHelpDict   `yaml:"TextMapHelpInfo" json:"textMapHelpInfo"`

	// BasedOn           string                 `yaml:"based-on"`           // 基于规则

	AliasMap *SyncMap[string, string] `yaml:"-" json:"-"` // 别名/同义词
}

func (t *GameSystemTemplate) GetAlias(varname string) string {
	k := strings.ToLower(varname)
	v2, exists := t.AliasMap.Load(k)
	if exists {
		varname = v2
	} else {
		k = chsS2T.Read(k)
		v2, exists = t.AliasMap.Load(k)
		if exists {
			varname = v2
		}
	}

	return varname
}

func (t *GameSystemTemplate) GetDefaultValueEx0(ctx *MsgContext, varname string) (*VMValue, string, bool, bool) {
	name := t.GetAlias(varname)
	var detail string

	// 先计算computed
	if expr, exists := t.DefaultsComputed[name]; exists {
		ctx.SystemTemplate = t
		r, detail2, err := ctx.Dice.ExprEvalBase(expr, ctx, RollExtraFlags{
			DefaultDiceSideNum: getDefaultDicePoints(ctx),
		})

		// 使用showAs的值覆盖detail
		v, _ := t.getShowAs0(ctx, name)
		if v != nil {
			detail = v.ToString()
		}
		if detail == "" {
			detail = detail2
		}

		if err == nil {
			return &r.VMValue, detail, r.Parser.Calculated, true
		}
	}

	if val, exists := t.Defaults[name]; exists {
		return VMValueNew(VMTypeInt64, val), detail, false, true
	}

	return VMValueNew(VMTypeInt64, int64(0)), detail, false, false
}

func (t *GameSystemTemplate) GetDefaultValueEx(ctx *MsgContext, varname string) *VMValue {
	a, _, _, _ := t.GetDefaultValueEx0(ctx, varname)
	return a
}

func (t *GameSystemTemplate) getShowAs0(ctx *MsgContext, k string) (*VMValue, error) {
	// 有showas的情况
	if expr, exists := t.AttrConfig.ShowAs[k]; exists {
		ctx.SystemTemplate = t
		r, _, err := ctx.Dice.ExprTextBase(expr, ctx, RollExtraFlags{
			DefaultDiceSideNum: getDefaultDicePoints(ctx),
		})
		if err == nil {
			return &r.VMValue, nil
		}
		return nil, err
	}
	return nil, nil //nolint:nilnil
}

func (t *GameSystemTemplate) getShowAsBase(ctx *MsgContext, k string) (*VMValue, error) {
	// 有showas的情况
	v, err := t.getShowAs0(ctx, k)
	if v != nil || err != nil {
		return v, err
	}

	// 显示本体
	ch, _ := ctx.ChVarsGet()
	_v, exists := ch.Get(k)
	if exists {
		return _v.(*VMValue), nil
	}

	// 默认值
	v, _, _, exists = t.GetDefaultValueEx0(ctx, k)
	if v != nil && exists {
		return v, nil
	}

	// 不存在的值，返回nil
	return nil, nil //nolint:nilnil
}

func (t *GameSystemTemplate) GetShowAs(ctx *MsgContext, k string) (*VMValue, error) {
	r, err := t.getShowAsBase(ctx, k)
	if err != nil {
		return r, err
	}
	if r != nil {
		return r, err
	}
	// 返回值不存在，强行补0
	return &VMValue{TypeID: VMTypeInt64, Value: int64(0)}, nil
}

func (t *GameSystemTemplate) GetRealValue(ctx *MsgContext, k string) (*VMValue, error) {
	// 跟 showas 一样，但是不采用showas而是返回实际值1
	// 显示本体
	ch, _ := ctx.ChVarsGet()
	_v, exists := ch.Get(k)
	if exists {
		return _v.(*VMValue), nil
	}

	// 默认值
	v := t.GetDefaultValueEx(ctx, k)
	if v != nil {
		return v, nil
	}

	// 不存在的值，强行补0
	return &VMValue{TypeID: VMTypeInt64, Value: int64(0)}, nil
}
