package dice

import (
	"errors"
	"fmt"
	"strings"

	ds "github.com/sealdice/dicescript"
)

// TrpgStValue is the stable JS-facing representation of a DiceScript value.
// Opaque values remain readable and are preserved if a hook does not replace
// them with one of the constructors exposed by seal.trpg.st.
type TrpgStValue struct {
	Type        string  `jsbind:"type"`
	IntValue    int64   `jsbind:"intValue"`
	FloatValue  float64 `jsbind:"floatValue"`
	StringValue string  `jsbind:"stringValue"`
	Expression  string  `jsbind:"expression"`
	Repr        string  `jsbind:"repr"`

	raw *ds.VMValue
}

func trpgStValueFromVM(value *ds.VMValue) *TrpgStValue {
	if value == nil {
		return nil
	}
	ret := &TrpgStValue{raw: value, Repr: value.ToRepr()}
	switch value.TypeId {
	case ds.VMTypeInt:
		ret.Type = "int"
		v, _ := value.ReadInt()
		ret.IntValue = int64(v)
	case ds.VMTypeFloat:
		ret.Type = "float"
		ret.FloatValue, _ = value.ReadFloat()
	case ds.VMTypeString:
		ret.Type = "string"
		ret.StringValue, _ = value.ReadString()
	case ds.VMTypeNull:
		ret.Type = "null"
	case ds.VMTypeComputedValue:
		ret.Type = "computed"
		if computed, ok := value.ReadComputed(); ok && computed != nil {
			ret.Expression = computed.Expr
		}
	default:
		ret.Type = "opaque"
	}
	return ret
}

func (value *TrpgStValue) toVMValue() (*ds.VMValue, error) {
	if value == nil {
		return nil, errors.New("st value is nil")
	}
	switch value.Type {
	case "int":
		return ds.NewIntVal(ds.IntType(value.IntValue)), nil
	case "float":
		return ds.NewFloatVal(value.FloatValue), nil
	case "string":
		return ds.NewStrVal(value.StringValue), nil
	case "null":
		return ds.NewNullVal(), nil
	case "computed":
		if value.raw != nil && value.raw.TypeId == ds.VMTypeComputedValue {
			if computed, ok := value.raw.ReadComputed(); ok && computed != nil && computed.Expr == value.Expression {
				return value.raw, nil
			}
		}
		return ds.NewComputedVal(value.Expression), nil
	case "opaque":
		if value.raw != nil {
			return value.raw, nil
		}
		return nil, errors.New("opaque st value cannot be constructed")
	default:
		return nil, fmt.Errorf("unsupported st value type %q", value.Type)
	}
}

type TrpgStOperation struct {
	Kind          string       `jsbind:"kind"`
	Name          string       `jsbind:"name"`
	Operator      string       `jsbind:"operator"`
	Expression    string       `jsbind:"expression"`
	Operand       *TrpgStValue `jsbind:"operand"`
	Extra         *TrpgStValue `jsbind:"extra"`
	OldValue      *TrpgStValue `jsbind:"oldValue"`
	ProposedValue *TrpgStValue `jsbind:"proposedValue"`
	Skip          bool         `jsbind:"skip"`
	RejectReason  string       `jsbind:"rejectReason"`
	AppendText    string       `jsbind:"appendText"`
}

type TrpgStCommandEvent struct {
	Actor        *MsgContext        `jsbind:"actor"`
	Target       *MsgContext        `jsbind:"target"`
	Message      *Message           `jsbind:"message"`
	Args         *CmdArgs           `jsbind:"args"`
	System       string             `jsbind:"system"`
	Operations   []*TrpgStOperation `jsbind:"operations"`
	Handled      bool               `jsbind:"handled"`
	Reply        string             `jsbind:"reply"`
	ReplySuffix  string             `jsbind:"replySuffix"`
	RejectReason string             `jsbind:"rejectReason"`
}

type TrpgStRenderEvent struct {
	Command *TrpgStCommandEvent `jsbind:"command"`
	Name    string              `jsbind:"name"`
	Value   *TrpgStValue        `jsbind:"value"`
	Text    string              `jsbind:"text"`
	Skip    bool                `jsbind:"skip"`
}

// TrpgStHooks is intentionally versioned as one registration per extension.
// Registering again replaces the prior set, which also makes script reloads
// deterministic.
type TrpgStHooks struct {
	Systems       []string                                                    `jsbind:"systems"`
	BeforeCommand func(event *TrpgStCommandEvent)                             `jsbind:"beforeCommand"`
	AfterParse    func(event *TrpgStCommandEvent)                             `jsbind:"afterParse"`
	BeforeApply   func(event *TrpgStCommandEvent, operation *TrpgStOperation) `jsbind:"beforeApply"`
	AfterEvaluate func(event *TrpgStCommandEvent, operation *TrpgStOperation) `jsbind:"afterEvaluate"`
	BeforeCommit  func(event *TrpgStCommandEvent)                             `jsbind:"beforeCommit"`
	AfterCommit   func(event *TrpgStCommandEvent)                             `jsbind:"afterCommit"`
	OnShow        func(event *TrpgStRenderEvent)                              `jsbind:"onShow"`
	OnExport      func(event *TrpgStRenderEvent)                              `jsbind:"onExport"`
}

func registerTrpgStHooks(ext *ExtInfo, hooks *TrpgStHooks) error {
	if ext == nil {
		return errors.New("注册trpg.st hook时必须提供扩展对象")
	}
	if hooks == nil {
		return errors.New("trpg.st hook不能为空")
	}
	ext.TrpgStHooks = hooks
	return nil
}

func (hooks *TrpgStHooks) appliesTo(system string) bool {
	if hooks == nil {
		return false
	}
	if len(hooks.Systems) == 0 {
		return true
	}
	for _, item := range hooks.Systems {
		if strings.EqualFold(strings.TrimSpace(item), system) {
			return true
		}
	}
	return false
}

func trpgStHookExtensions(ctx *MsgContext, system string) []*ExtInfo {
	if ctx == nil || ctx.Group == nil || ctx.Dice == nil {
		return nil
	}
	tmpl, ok := ctx.Dice.GameSystemMap.Load(system)
	if !ok || tmpl == nil || len(tmpl.SetConfig.RelatedExt) == 0 {
		return nil
	}

	related := make(map[string]struct{})
	graph := ctx.Dice.activeWithGraph()
	for _, name := range tmpl.SetConfig.RelatedExt {
		name = ctx.Dice.ExtAliasToName(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		related[strings.ToLower(name)] = struct{}{}
		for _, chained := range collectChainedNames(ctx.Dice.Logger, graph, name, maxChainDepth) {
			related[strings.ToLower(chained)] = struct{}{}
		}
	}

	var ret []*ExtInfo
	for _, wrapper := range commandExtensionOrder(ctx.Group, ctx.Dice) {
		if wrapper == nil {
			continue
		}
		if _, ok := related[strings.ToLower(wrapper.Name)]; !ok {
			continue
		}
		ext := wrapper.GetRealExt()
		if ext == nil || ext.TrpgStHooks == nil || !ext.TrpgStHooks.appliesTo(system) {
			continue
		}
		ret = append(ret, ext)
	}
	return ret
}

func invokeTrpgStHook(d *Dice, ext *ExtInfo, hook func()) (panicErr error) {
	if ext == nil || hook == nil {
		return nil
	}
	if ext.IsJsExt && !d.Config.JsEnable {
		return fmt.Errorf("扩展<%s>的JS运行环境未启用", ext.Name)
	}
	ext.callWithJsCheck(d, func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicErr = fmt.Errorf("扩展<%s>执行trpg.st hook异常: %v", ext.Name, recovered)
			}
		}()
		hook()
	})
	return panicErr
}

func trpgStRejectReason(event *TrpgStCommandEvent, operation *TrpgStOperation) string {
	if event != nil && event.RejectReason != "" {
		return event.RejectReason
	}
	if operation != nil {
		return operation.RejectReason
	}
	return ""
}
