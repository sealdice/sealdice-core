package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
	"gopkg.in/yaml.v3"
)

// Attrs keeps attribute defaults and computed expressions.
type Attrs struct {
	Defaults         map[string]int    `yaml:"defaults"`
	DefaultsComputed map[string]string `yaml:"defaultsComputed"`
	DetailOverwrite  map[string]string `yaml:"detailOverwrite"`

	DefaultsComputedReal map[string]*ds.VMValue `json:"-" yaml:"-"`
}

// Alias defines alias dictionary for attributes.
type Alias map[string][]string

// Commands wraps command-related configuration.
type Commands struct {
	Set SetConfig `yaml:"set"`
	Sn  SnConfig  `yaml:"sn"`
	St  StConfig  `yaml:"st"`
}

// SetConfig configures the set command.
type SetConfig struct {
	DiceSideExpr string   `yaml:"diceSides"`
	EnableTip    string   `yaml:"enableTip"`
	Keys         []string `yaml:"keys"`
	RelatedExt   []string `yaml:"relatedExt"`
}

// SnConfig configures name templates.
type SnConfig map[string]SnTemplate

// SnTemplate describes a single sn entry.
type SnTemplate struct {
	Template string `yaml:"template"`
	HelpText string `yaml:"helpText"`
}

// StConfig configures st command.
type StConfig struct {
	Show StShowConfig `yaml:"show"`
}

// StShowConfig controls st show behaviour.
type StShowConfig struct {
	Top               []string          `yaml:"top"`
	SortBy            string            `yaml:"sortBy"`
	Ignores           []string          `yaml:"ignores"`
	ShowKeyAs         map[string]string `yaml:"showKeyAs"`
	ShowValueAs       map[string]string `yaml:"showValueAs"`
	ShowValueAsIfMiss map[string]string `yaml:"showValueAsIfMiss"` // 影响单个属性的右侧内容
	ItemsPerLine      int               `yaml:"itemsPerLine"`
}

// NameTemplateItem describes an sn template entry.
type NameTemplateItem struct {
	Template string `json:"template" yaml:"template"`
	HelpText string `json:"helpText" yaml:"helpText"`
}

// LegacySetConfig keeps backward compatible view of set configuration.
type LegacySetConfig struct {
	DiceSideExpr string
	DiceSides    int64
	EnableTip    string
	Keys         []string
	RelatedExt   []string
}

// GameSystemTemplate is the core template definition compatible with the smallseal format.
// GameSystemTemplateV2 mirrors the template definition used in smallseal.
type GameSystemTemplateV2 struct {
	Name        string   `yaml:"name"`
	FullName    string   `yaml:"fullName"`
	Authors     []string `yaml:"authors"`
	Version     string   `yaml:"version"`
	UpdatedTime string   `yaml:"updatedTime"`
	TemplateVer string   `yaml:"templateVer"`
	InitScript  string   `yaml:"initScript"`
	Attrs       Attrs    `yaml:"attrs"`
	Alias       Alias    `yaml:"alias"`
	Commands    Commands `yaml:"commands"`

	AliasMap          *SyncMap[string, string]                                                                                                                  `json:"-" yaml:"-"`
	HookValueLoadPost func(ctx *ds.Context, name string, curVal *ds.VMValue, doCompute func(curVal *ds.VMValue) *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue `json:"-" yaml:"-"`
}

// GameSystemTemplate extends GameSystemTemplateV2 with compatibility helpers.
type GameSystemTemplate struct {
	*GameSystemTemplateV2 `yaml:",inline"`

	// 这几个都是出于兼容目的，因为有一部分V1机制还在用这个
	TextMap         *TextTemplateWithWeightDict `yaml:"textMap"`
	TextMapHelpInfo *TextTemplateWithHelpDict   `yaml:"textMapHelpInfo"`

	SetConfig    LegacySetConfig             `yaml:"-"`
	NameTemplate map[string]NameTemplateItem `yaml:"-"`

	inited bool
}

func (t *GameSystemTemplateV2) Init() {
	if t == nil {
		return
	}

	aliasMap := new(SyncMap[string, string])
	for key, items := range t.Alias {
		canonical := strings.ToLower(key)
		aliasMap.Store(canonical, key)
		for _, alias := range items {
			aliasMap.Store(strings.ToLower(alias), key)
		}
	}
	t.AliasMap = aliasMap

	t.Attrs.DefaultsComputedReal = make(map[string]*ds.VMValue)
	for k, v := range t.Attrs.DefaultsComputed {
		t.Attrs.DefaultsComputedReal[k] = ds.NewComputedVal(v)
	}
}

func (t *GameSystemTemplateV2) GetAlias(varname string) string {
	if t == nil {
		return varname
	}
	k := strings.ToLower(varname)
	if t.AliasMap != nil {
		if v, ok := t.AliasMap.Load(k); ok {
			return v
		}
	}
	return varname
}

func (t *GameSystemTemplateV2) GetDefaultValue(varname string) (*ds.VMValue, string, bool, bool) {
	if t == nil {
		return nil, "", false, false
	}
	if computed, exists := t.Attrs.DefaultsComputedReal[varname]; exists {
		return computed, "", true, true
	}
	if val, exists := t.Attrs.Defaults[varname]; exists {
		return ds.NewIntVal(ds.IntType(val)), "", false, true
	}
	return nil, "", false, false
}

func (t *GameSystemTemplateV2) GetShowKeyAs(ctx *MsgContext, k string) (string, error) {
	if t == nil || ctx == nil {
		return k, nil
	}
	if expr, exists := t.Commands.St.Show.ShowKeyAs[k]; exists && expr != "" {
		res := ctx.EvalFString(expr, &ds.RollConfig{
			DefaultDiceSideExpr: strconv.FormatInt(getDefaultDicePoints(ctx), 10),
		})
		if res != nil && res.vm.Error == nil {
			return res.ToString(), nil
		}
		if res != nil && res.vm.Error != nil {
			return k, res.vm.Error
		}
	}
	return k, nil
}

func (t *GameSystemTemplateV2) GetShowValueAs(ctx *MsgContext, k string) (*ds.VMValue, error) {
	// 先看一下是否有值
	_, exists := t.GetAttrValue(ctx, k)

	getVal := func(expr string) (*VMResultV2, error) {
		v := ctx.EvalFString(expr, nil)
		return v, nil
	}

	if !exists {
		// 无值情况多一种匹配，用于虽然这个值有默认ShowValueAs，但是用户进行了赋值的情况
		if expr, exists := t.Commands.St.Show.ShowValueAsIfMiss[k]; exists {
			if r, err := getVal(expr); err == nil {
				return &r.VMValue, nil
			}
		}
		// 通配值
		if expr, exists := t.Commands.St.Show.ShowValueAsIfMiss["*"]; exists {
			// 这里存入k是因为接下来模板可能会用到原始key，例如 loadRaw(name)
			ctx.vm.StoreNameLocal("name", ds.NewStrVal(k))
			if r, err := getVal(expr); err == nil {
				return &r.VMValue, nil
			}
			return ds.NewIntVal(0), nil
		}
	}

	if expr, exists := t.Commands.St.Show.ShowValueAs[k]; exists && expr != "" {
		res := ctx.EvalFString(expr, nil)
		if res != nil && res.vm.Error == nil {
			return &res.VMValue, nil
		}
		if res != nil && res.vm.Error != nil {
			return nil, res.vm.Error
		}
	}
	if expr, exists := t.Commands.St.Show.ShowValueAs["*"]; exists && expr != "" {
		ctx.CreateVmIfNotExists()
		ctx.vm.StoreNameLocal("name", ds.NewStrVal(k))
		res := ctx.EvalFString(expr, nil)
		if res != nil && res.vm.Error == nil {
			return &res.VMValue, nil
		}
		if res != nil && res.vm.Error != nil {
			return ds.NewIntVal(0), res.vm.Error
		}
		return ds.NewIntVal(0), nil
	}

	// 最后返回真实值
	return t.GetRealValue(ctx, k)
}

// 获取用户录入的值
func (t *GameSystemTemplateV2) GetAttrValue(ctx *MsgContext, k string) (*ds.VMValue, bool) {
	am := ctx.Dice.AttrsManager
	curAttrs := lo.Must(am.Load(ctx.Group.GroupID, ctx.Player.UserID))
	return curAttrs.LoadX(k)
}

func (t *GameSystemTemplateV2) GetRealValueBase(ctx *MsgContext, k string) (*ds.VMValue, error) {
	if ctx.Dice != nil {
		curAttrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
		if v, exists := curAttrs.LoadX(k); exists {
			return v, nil
		}
	}
	if v, _, _, exists := t.GetDefaultValue(k); exists {
		return v, nil
	}
	return nil, nil //nolint:nilnil
}

func (t *GameSystemTemplateV2) GetRealValue(ctx *MsgContext, k string) (*ds.VMValue, error) {
	v, err := t.GetRealValueBase(ctx, k)
	if v == nil && err == nil {
		return ds.NewIntVal(0), nil
	}
	return v, err
}

var errTemplateValueNotFound = errors.New("template value not found")

// Init prepares runtime caches for the template.
func (t *GameSystemTemplate) Init() {
	if t == nil {
		return
	}

	// 我注意到有重复init
	if t.inited {
		return
	}

	if t.GameSystemTemplateV2 == nil {
		// ensure the embedded template is always available
		t.GameSystemTemplateV2 = &GameSystemTemplateV2{}
	}

	t.GameSystemTemplateV2.Init()

	t.SetConfig = LegacySetConfig{
		DiceSideExpr: t.Commands.Set.DiceSideExpr,
		EnableTip:    t.Commands.Set.EnableTip,
		Keys:         append([]string(nil), t.Commands.Set.Keys...),
		RelatedExt:   append([]string(nil), t.Commands.Set.RelatedExt...),
	}
	if v, err := strconv.ParseInt(strings.TrimSpace(t.SetConfig.DiceSideExpr), 10, 64); err == nil {
		t.SetConfig.DiceSides = v
	}

	t.NameTemplate = map[string]NameTemplateItem{}
	for key, item := range t.Commands.Sn {
		lower := strings.ToLower(key)
		entry := NameTemplateItem(item)
		t.NameTemplate[key] = entry
		if lower != key {
			t.NameTemplate[lower] = entry
		}
	}

	if t.Attrs.Defaults == nil {
		t.Attrs.Defaults = map[string]int{}
	}
	if t.Attrs.DefaultsComputed == nil {
		t.Attrs.DefaultsComputed = map[string]string{}
	}
	if t.Attrs.DetailOverwrite == nil {
		t.Attrs.DetailOverwrite = map[string]string{}
	}

	t.inited = true
}

func (t *GameSystemTemplate) GetAlias(varname string) string {
	k := strings.ToLower(varname)
	if t == nil || t.GameSystemTemplateV2 == nil || t.AliasMap == nil {
		return varname
	}

	if v, ok := t.AliasMap.Load(k); ok {
		return v
	}

	k = chsS2T.Read(k)
	if v, ok := t.AliasMap.Load(k); ok {
		return v
	}

	return varname
}

func (t *GameSystemTemplate) runInitScript(ctx *MsgContext) {
	if ctx == nil || t == nil || t.GameSystemTemplateV2 == nil {
		return
	}
	ctx.SystemTemplate = t
	if t.InitScript == "" {
		return
	}
	ctx.Eval(t.InitScript, nil)
}

// GetDefaultValueEx0 获取默认值 四个返回值 val, detail, computed, exists
func (t *GameSystemTemplate) GetDefaultValueEx0(ctx *MsgContext, varname string) (*ds.VMValue, string, bool, bool) {
	if t == nil || t.GameSystemTemplateV2 == nil {
		return ds.NewIntVal(0), "", false, false
	}

	name := t.GetAlias(varname)
	var detail string

	if expr, exists := t.Attrs.DefaultsComputed[name]; exists {
		t.runInitScript(ctx)
		if result := ctx.Eval(expr, nil); result != nil {
			if result.vm.Error == nil {
				detail = result.vm.GetDetailText()
				return &result.VMValue, detail, result.vm.IsCalculateExists() || result.vm.IsComputedLoaded, true
			}
			return ds.NewStrVal("代码:" + expr + "\n" + "报错:" + result.vm.Error.Error()), "", true, true
		}
	}

	if val, exists := t.Attrs.Defaults[name]; exists {
		return ds.NewIntVal(ds.IntType(val)), detail, false, true
	}

	return ds.NewIntVal(0), detail, false, false
}

// GetDefaultValueEx0V1 获取默认值，与现行版本不同的是里面调用了 getShowAs0，唯一的使用地点是 RollVM v1
func (t *GameSystemTemplate) GetDefaultValueEx0V1(ctx *MsgContext, varname string) (*ds.VMValue, string, bool, bool) {
	if t == nil || t.GameSystemTemplateV2 == nil {
		return ds.NewIntVal(0), "", false, false
	}

	name := t.GetAlias(varname)
	var detail string

	if expr, exists := t.Attrs.DefaultsComputed[name]; exists {
		t.runInitScript(ctx)
		if result := ctx.Eval(expr, nil); result != nil {
			detailBackup := result.vm.GetDetailText()
			err := result.vm.Error

			_, v, _ := t.getShowAs0(ctx, name)
			if v != nil {
				detail = v.ToString()
			}
			if detail == "" {
				detail = detailBackup
			}

			if err == nil {
				return &result.VMValue, detail, result.vm.IsCalculateExists() || result.vm.IsComputedLoaded, true
			}
		}
	}

	if val, exists := t.Attrs.Defaults[name]; exists {
		return ds.NewIntVal(ds.IntType(val)), detail, false, true
	}

	return ds.NewIntVal(0), detail, false, false
}

func (t *GameSystemTemplate) GetDefaultValueEx(ctx *MsgContext, varname string) *ds.VMValue {
	v, _, _, _ := t.GetDefaultValueEx0(ctx, varname)
	return v
}

func (t *GameSystemTemplate) getShowAs0(ctx *MsgContext, k string) (string, *ds.VMValue, error) {
	if t == nil || t.GameSystemTemplateV2 == nil {
		return k, nil, nil
	}

	baseK := k
	if expr, exists := t.Commands.St.Show.ShowKeyAs[k]; exists && expr != "" {
		t.runInitScript(ctx)
		if r, _, err := DiceExprTextBase(ctx, expr, RollExtraFlags{DefaultDiceSideNum: getDefaultDicePoints(ctx), V2Only: true}); err == nil {
			k = r.ToString()
		}
	}

	if expr, exists := t.Commands.St.Show.ShowValueAs[baseK]; exists && expr != "" {
		t.runInitScript(ctx)
		r, _, err := DiceExprTextBase(ctx, expr, RollExtraFlags{DefaultDiceSideNum: getDefaultDicePoints(ctx), V2Only: true})
		if err == nil {
			return k, r.VMValue, nil
		}
		return k, nil, err
	}

	if expr, exists := t.Commands.St.Show.ShowValueAs["*"]; exists && expr != "" {
		t.runInitScript(ctx)
		ctx.CreateVmIfNotExists()
		ctx.vm.StoreNameLocal("name", ds.NewStrVal(baseK))
		r := ctx.EvalFString(expr, nil)
		if r.vm.Error == nil {
			return k, &r.VMValue, nil
		}
		return k, nil, r.vm.Error
	}

	return k, nil, nil
}

func (t *GameSystemTemplate) GetRealValueBase(ctx *MsgContext, k string) (*ds.VMValue, error) {
	if t == nil || t.GameSystemTemplateV2 == nil {
		return nil, errTemplateValueNotFound
	}

	if ctx == nil || ctx.Dice == nil {
		return nil, errTemplateValueNotFound
	}

	curAttrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if v, exists := curAttrs.LoadX(k); exists {
		return v, nil
	}

	// if v, _, _, exists := t.GetDefaultValueEx0(ctx, k); exists {
	// 	return v, nil
	// }

	// 默认值
	v, _, _, exists := t.GetDefaultValue(k)
	if exists {
		return v, nil
	}

	return nil, errTemplateValueNotFound
}

func (t *GameSystemTemplate) GetRealValue(ctx *MsgContext, k string) (*ds.VMValue, error) {
	if t == nil || t.GameSystemTemplateV2 == nil {
		return ds.NewIntVal(0), nil
	}
	v, err := t.GetRealValueBase(ctx, k)
	if errors.Is(err, errTemplateValueNotFound) {
		return ds.NewIntVal(0), nil
	}
	return v, err
}

func loadGameSystemTemplateFromData(data []byte, format string) (*GameSystemTemplate, error) {
	if len(data) == 0 {
		return nil, errors.New("empty template data")
	}

	format = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(format), "."))
	if format == "" {
		format = "yaml"
	}

	if isLegacyTemplateData(data) {
		legacy, err := convertLegacyTemplate(data, format)
		if err != nil {
			return nil, err
		}
		return legacy, nil
	}

	tmpl := &GameSystemTemplate{}
	switch format {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, tmpl); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, tmpl); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported template format: %s", format)
	}

	if tmpl.GameSystemTemplateV2 == nil {
		tmpl.GameSystemTemplateV2 = &GameSystemTemplateV2{}
	}
	tmpl.Init()
	return tmpl, nil
}

func isLegacyTemplateData(data []byte) bool {
	var probe struct {
		TemplateVer      string         `json:"templateVer"      yaml:"templateVer"`
		Attrs            map[string]any `json:"attrs"            yaml:"attrs"`
		Defaults         map[string]any `json:"defaults"         yaml:"defaults"`
		DefaultsComputed map[string]any `json:"defaultsComputed" yaml:"defaultsComputed"`
	}
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return false
	}
	ver := strings.TrimSpace(strings.ToLower(probe.TemplateVer))
	if ver != "" {
		if strings.HasPrefix(ver, "2") || strings.HasPrefix(ver, "v2") {
			return false
		}
		return true
	}
	if len(probe.Attrs) > 0 {
		return false
	}
	if len(probe.Defaults) > 0 || len(probe.DefaultsComputed) > 0 {
		return true
	}
	return false
}

// LoadGameSystemTemplateFromReader loads a template from an io.Reader with the given format.
func LoadGameSystemTemplateFromReader(r io.Reader, format string) (*GameSystemTemplate, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}
	return loadGameSystemTemplateFromData(data, format)
}

// LoadGameSystemTemplateFromFile loads a template from a file path.
func LoadGameSystemTemplateFromFile(filename string) (*GameSystemTemplate, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}
	return loadGameSystemTemplateFromData(data, filepath.Ext(filename))
}

// LoadGameSystemTemplateFromBytes loads a template from raw bytes with the provided format.
func LoadGameSystemTemplateFromBytes(data []byte, format string) (*GameSystemTemplate, error) {
	return loadGameSystemTemplateFromData(data, format)
}
