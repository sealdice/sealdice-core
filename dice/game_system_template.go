package dice

import (
	"strings"

	"github.com/samber/lo"

	ds "github.com/sealdice/dicescript"
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
	ShowAsKey    map[string]string `yaml:"showAsKey" json:"showAsKey"`       // 展示形式，即st show时格式
	Setter       map[string]string `yaml:"setter" json:"setter"`             // st写入时执行这个，未实装
	ItemsPerLine int               `yaml:"itemsPerLine" json:"itemsPerLine"` // 每行显示几个属性，默认4
}

type NameTemplateItem struct {
	Template string `yaml:"template" json:"template"`
	HelpText string `yaml:"helpText" json:"helpText"`
}

type SetConfig struct {
	RelatedExt    []string `yaml:"relatedExt" json:"relatedExt"`       // 关联扩展
	DiceSides     int64    `yaml:"diceSides" json:"diceSides"`         // 骰子面数
	DiceSidesExpr string   `yaml:"diceSidesExpr" json:"diceSidesExpr"` // 骰子面数表达式
	Keys          []string `yaml:"keys" json:"keys"`                   // 可用于 .set xxx 的key
	EnableTip     string   `yaml:"enableTip" json:"enableTip"`         // 启用提示
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
	DetailOverwrite  map[string]string   `yaml:"detailOverwrite" json:"detailOverwrite"`   // 计算过程，如果有的话附加在st show或者计算中。用例见dnd5e模板
	Alias            map[string][]string `yaml:"alias" json:"alias"`                       // 别名/同义词

	TextMap         *TextTemplateWithWeightDict `yaml:"textMap" json:"textMap"` // UI文本
	TextMapHelpInfo *TextTemplateWithHelpDict   `yaml:"TextMapHelpInfo" json:"textMapHelpInfo"`

	PreloadCode string `yaml:"preloadCode" json:"preloadCode"` // 预加载代码
	// BasedOn           string                 `yaml:"based-on"`           // 基于规则

	AliasMap *SyncMap[string, string] `yaml:"-" json:"-"` // 别名/同义词
}

func (t *GameSystemTemplate) GetAlias(varname string) string {
	k := strings.ToLower(varname)
	if t == nil {
		return varname
	}
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

// GetDefaultValueEx0 获取默认值 四个返回值 val, detail, computed, exists
func (t *GameSystemTemplate) GetDefaultValueEx0(ctx *MsgContext, varname string) (*ds.VMValue, string, bool, bool) {
	name := t.GetAlias(varname)
	var detail string

	// 先计算computed
	if expr, exists := t.DefaultsComputed[name]; exists {
		ctx.SystemTemplate = t
		// 也许有点依赖这个东西？有更好的方式吗？可以更加全局的去加载吗（比如和vm创建伴生加载）？
		ctx.Eval(t.PreloadCode, nil)
		r := ctx.Eval(expr, nil)

		if r.vm.Error == nil {
			detail = r.vm.GetDetailText()
			return &r.VMValue, detail, r.vm.IsCalculateExists() || r.vm.IsComputedLoaded, true
		} else {
			return ds.NewStrVal("代码:" + expr + "\n" + "报错:" + r.vm.Error.Error()), "", true, true
		}
	}

	if val, exists := t.Defaults[name]; exists {
		return ds.NewIntVal(ds.IntType(val)), detail, false, true
	}

	// TODO: 以空值填充，这是vm v1的行为，未来需要再次评估其合理性
	return ds.NewIntVal(0), detail, false, false
}

// GetDefaultValueEx0V1 获取默认值，与现行版本不同的是里面调用了 getShowAs0，唯一的使用地点是 RollVM v1
func (t *GameSystemTemplate) GetDefaultValueEx0V1(ctx *MsgContext, varname string) (*ds.VMValue, string, bool, bool) {
	name := t.GetAlias(varname)
	var detail string

	// 先计算computed
	if expr, exists := t.DefaultsComputed[name]; exists {
		ctx.SystemTemplate = t
		r := ctx.Eval(expr, nil)
		detail2 := r.vm.GetDetailText()
		err := r.vm.Error

		// 使用showAs的值覆盖detail
		_, v, _ := t.getShowAs0(ctx, name)
		if v != nil {
			detail = v.ToString()
		}
		if detail == "" {
			detail = detail2
		}

		if err == nil {
			return &r.VMValue, detail, r.vm.IsCalculateExists() || r.vm.IsComputedLoaded, true
		}
	}

	if val, exists := t.Defaults[name]; exists {
		return ds.NewIntVal(ds.IntType(val)), detail, false, true
	}

	// TODO: 以空值填充，这是vm v1的行为，未来需要再次评估其合理性
	return ds.NewIntVal(0), detail, false, false
}

func (t *GameSystemTemplate) GetDefaultValueEx(ctx *MsgContext, varname string) *ds.VMValue {
	a, _, _, _ := t.GetDefaultValueEx0(ctx, varname)
	return a
}

func (t *GameSystemTemplate) getShowAs0(ctx *MsgContext, k string) (string, *ds.VMValue, error) {
	// 有showas的情况
	baseK := k
	if expr, exists := t.AttrConfig.ShowAsKey[k]; exists {
		ctx.SystemTemplate = t
		r, _, err := DiceExprTextBase(ctx, expr, RollExtraFlags{
			DefaultDiceSideNum: getDefaultDicePoints(ctx),
			V2Only:             true,
		})
		if err == nil {
			k = r.ToString()
		}
	}

	if expr, exists := t.AttrConfig.ShowAs[baseK]; exists {
		ctx.SystemTemplate = t
		r, _, err := DiceExprTextBase(ctx, expr, RollExtraFlags{
			DefaultDiceSideNum: getDefaultDicePoints(ctx),
			V2Only:             true,
		})
		if err == nil {
			return k, r.VMValue, nil
		}
		return k, nil, err
	}

	// 基础值
	if expr, exists := t.AttrConfig.ShowAs["*"]; exists {
		ctx.SystemTemplate = t
		ctx.CreateVmIfNotExists()
		ctx.vm.StoreNameLocal("name", ds.NewStrVal(baseK))
		r := ctx.EvalFString(expr, nil)
		if r.vm.Error == nil {
			return k, &r.VMValue, nil
		}
		return k, nil, r.vm.Error
	}

	return k, nil, nil //nolint:nilnil
}

func (t *GameSystemTemplate) getShowAsBase(ctx *MsgContext, k string) (string, *ds.VMValue, error) {
	// 有showas的情况
	newK, v, err := t.getShowAs0(ctx, k)
	if v != nil || err != nil {
		return newK, v, err
	}

	// 显示本体
	curAttrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))

	var exists bool
	v, exists = curAttrs.LoadX(k)
	if exists {
		return newK, v, nil
	}

	// 默认值
	v, _, _, exists = t.GetDefaultValueEx0(ctx, k)
	if v != nil && exists {
		return newK, v, nil
	}

	// 不存在的值，返回nil
	return k, nil, nil //nolint:nilnil
}

func (t *GameSystemTemplate) GetShowAs(ctx *MsgContext, k string) (string, *ds.VMValue, error) {
	k, v, err := t.getShowAsBase(ctx, k)
	if err != nil {
		return k, v, err
	}
	if v != nil {
		return k, v, err
	}
	// 返回值不存在，强行补0
	return k, ds.NewIntVal(0), nil
}

func (t *GameSystemTemplate) GetRealValueBase(ctx *MsgContext, k string) (*ds.VMValue, error) {
	// 跟 showas 一样，但是不采用showas而是返回实际值
	// 显示本体
	am := ctx.Dice.AttrsManager
	curAttrs := lo.Must(am.LoadByCtx(ctx))
	v, exists := curAttrs.LoadX(k)
	if exists {
		return v, nil
	}

	// 默认值
	v, _, _, exists = t.GetDefaultValueEx0(ctx, k)
	if exists {
		return v, nil
	}

	// 不存在的值
	return nil, nil
}

func (t *GameSystemTemplate) GetRealValue(ctx *MsgContext, k string) (*ds.VMValue, error) {
	v, err := t.GetRealValueBase(ctx, k)
	if v == nil && err == nil {
		// 不存在的值，强行补0
		return ds.NewIntVal(0), nil
	}
	return v, err
}
