package dice

import (
	"encoding/json"
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

// GameTemplateAttrs keeps attribute defaults and computed expressions.
type GameTemplateAttrs struct {
	Defaults         map[string]int64  `yaml:"defaults"`
	DefaultsComputed map[string]string `yaml:"defaultsComputed"`
	DetailOverwrite  map[string]string `yaml:"detailOverwrite"`
}

// Alias defines alias dictionary for attributes.
type Alias map[string][]string

// GameTemplateCommands wraps command-related configuration.
type GameTemplateCommands struct {
	Set GameTemplateSetConfig `yaml:"set"`
	Sn  GameTemplateSnConfig  `yaml:"sn"`
	St  GameTemplateStConfig  `yaml:"st"`
}

// GameTemplateSetConfig configures the set command.
type GameTemplateSetConfig struct {
	DiceSideExpr string   `yaml:"diceSideExpr"`
	EnableTip    string   `yaml:"enableTip"`
	Keys         []string `yaml:"keys"`
	RelatedExt   []string `yaml:"relatedExt"`
}

// GameTemplateSnConfig configures name templates.
type GameTemplateSnConfig map[string]GameTemplateSnTemplate

// GameTemplateSnTemplate describes a single sn entry.
type GameTemplateSnTemplate struct {
	Template string `yaml:"template"`
	HelpText string `yaml:"helpText"`
}

// GameTemplateStConfig configures st command.
type GameTemplateStConfig struct {
	Show GameTemplateStShowConfig `yaml:"show"`
}

// GameTemplateStShowConfig controls st show behaviour.
type GameTemplateStShowConfig struct {
	Top          []string          `yaml:"top"`
	SortBy       string            `yaml:"sortBy"`
	Ignores      []string          `yaml:"ignores"`
	ShowValueAs  map[string]string `yaml:"showValueAs"`
	ShowKeyAs    map[string]string `yaml:"showKeyAs"`
	ItemsPerLine int               `yaml:"itemsPerLine"`
}

// AttrConfig keeps backward compatible view of show configuration.
type AttrConfig struct {
	Top          []string
	SortBy       string
	Ignores      []string
	ShowAs       map[string]string
	ShowAsKey    map[string]string
	ItemsPerLine int
}

// NameTemplateItem describes an sn template entry.
type NameTemplateItem struct {
	Template string `json:"template" yaml:"template"`
	HelpText string `json:"helpText" yaml:"helpText"`
}

// SetConfig keeps backward compatible view of set configuration.
type SetConfig struct {
	DiceSideExpr string
	DiceSides    int64
	EnableTip    string
	Keys         []string
	RelatedExt   []string
}

// GameSystemTemplate is the core template definition compatible with the smallseal format.
type GameSystemTemplate struct {
	Name        string   `yaml:"name"`
	FullName    string   `yaml:"fullName"`
	Authors     []string `yaml:"authors"`
	Version     string   `yaml:"version"`
	UpdatedTime string   `yaml:"updatedTime"`
	TemplateVer string   `yaml:"templateVer"`
	InitScript  string   `yaml:"initScript"`

	Attrs    GameTemplateAttrs    `yaml:"attrs"`
	Alias    Alias                `yaml:"alias"`
	Commands GameTemplateCommands `yaml:"commands"`

	TextMap         *TextTemplateWithWeightDict `yaml:"textMap"`
	TextMapHelpInfo *TextTemplateWithHelpDict   `yaml:"textMapHelpInfo"`

	AliasMap *SyncMap[string, string] `yaml:"-"`

	HookValueLoadPost func(ctx *ds.Context, name string, curVal *ds.VMValue, doCompute func(curVal *ds.VMValue) *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue `yaml:"-"`

	AttrConfig   AttrConfig                  `yaml:"-"`
	SetConfig    SetConfig                   `yaml:"-"`
	NameTemplate map[string]NameTemplateItem `yaml:"-"`
}

// Init prepares runtime caches for the template.
func (t *GameSystemTemplate) Init() {
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

	show := t.Commands.St.Show
	t.AttrConfig = AttrConfig{
		Top:          append([]string(nil), show.Top...),
		SortBy:       show.SortBy,
		Ignores:      append([]string(nil), show.Ignores...),
		ShowAs:       show.ShowValueAs,
		ShowAsKey:    show.ShowKeyAs,
		ItemsPerLine: show.ItemsPerLine,
	}

	t.SetConfig = SetConfig{
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
		entry := NameTemplateItem{Template: item.Template, HelpText: item.HelpText}
		t.NameTemplate[key] = entry
		if lower != key {
			t.NameTemplate[lower] = entry
		}
	}

	if t.Attrs.Defaults == nil {
		t.Attrs.Defaults = map[string]int64{}
	}
	if t.Attrs.DefaultsComputed == nil {
		t.Attrs.DefaultsComputed = map[string]string{}
	}
	if t.Attrs.DetailOverwrite == nil {
		t.Attrs.DetailOverwrite = map[string]string{}
	}
}

func (t *GameSystemTemplate) GetAlias(varname string) string {
	k := strings.ToLower(varname)
	if t == nil {
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
	if ctx == nil || t == nil {
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
	if t == nil {
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
	if t == nil {
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
	if t == nil {
		return k, nil, nil
	}

	baseK := k
	if expr, exists := t.AttrConfig.ShowAsKey[k]; exists && expr != "" {
		t.runInitScript(ctx)
		if r, _, err := DiceExprTextBase(ctx, expr, RollExtraFlags{DefaultDiceSideNum: getDefaultDicePoints(ctx), V2Only: true}); err == nil {
			k = r.ToString()
		}
	}

	if expr, exists := t.AttrConfig.ShowAs[baseK]; exists && expr != "" {
		t.runInitScript(ctx)
		r, _, err := DiceExprTextBase(ctx, expr, RollExtraFlags{DefaultDiceSideNum: getDefaultDicePoints(ctx), V2Only: true})
		if err == nil {
			return k, r.VMValue, nil
		}
		return k, nil, err
	}

	if expr, exists := t.AttrConfig.ShowAs["*"]; exists && expr != "" {
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

func (t *GameSystemTemplate) getShowAsBase(ctx *MsgContext, k string) (string, *ds.VMValue, error) {
	newK, v, err := t.getShowAs0(ctx, k)
	if v != nil || err != nil {
		return newK, v, err
	}

	curAttrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if v, exists := curAttrs.LoadX(k); exists {
		return newK, v, nil
	}

	if v, _, _, exists := t.GetDefaultValueEx0(ctx, k); v != nil && exists {
		return newK, v, nil
	}

	return k, nil, nil
}

func (t *GameSystemTemplate) GetShowAs(ctx *MsgContext, k string) (string, *ds.VMValue, error) {
	k, v, err := t.getShowAsBase(ctx, k)
	if err != nil {
		return k, v, err
	}
	if v != nil {
		return k, v, nil
	}
	return k, ds.NewIntVal(0), nil
}

func (t *GameSystemTemplate) GetRealValueBase(ctx *MsgContext, k string) (*ds.VMValue, error) {
	if t == nil {
		return nil, nil
	}

	curAttrs := lo.Must(ctx.Dice.AttrsManager.LoadByCtx(ctx))
	if v, exists := curAttrs.LoadX(k); exists {
		return v, nil
	}

	if v, _, _, exists := t.GetDefaultValueEx0(ctx, k); exists {
		return v, nil
	}

	return nil, nil
}

func (t *GameSystemTemplate) GetRealValue(ctx *MsgContext, k string) (*ds.VMValue, error) {
	v, err := t.GetRealValueBase(ctx, k)
	if v == nil && err == nil {
		return ds.NewIntVal(0), nil
	}
	return v, err
}

func loadGameSystemTemplateFromData(data []byte, format string) (*GameSystemTemplate, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty template data")
	}

	tmpl := &GameSystemTemplate{}
	format = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(format), "."))
	if format == "" {
		format = "yaml"
	}

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

	tmpl.Init()
	return tmpl, nil
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
