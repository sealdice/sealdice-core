package dice

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/antlabs/strsim"
	"gopkg.in/yaml.v3"
)

type ReplyConditionBase interface {
	Check(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, cleanText string) bool
	Clean()
}

type ReplyResultBase interface {
	Execute(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs)
	Clean()
}

// ReplyConditionTextMatch 文本匹配 // textMatch
type ReplyConditionTextMatch struct {
	CondType  string `yaml:"condType" json:"condType"`
	MatchType string `yaml:"matchType" json:"matchType"` // matchExact 精确  matchRegex 正则 matchFuzzy 模糊 matchContains 包含
	Value     string `yaml:"value" json:"value"`
}

// ReplyConditionMultiMatch 文本多重匹配
type ReplyConditionMultiMatch struct {
	CondType  string `yaml:"condType" json:"condType"`
	MatchType string `yaml:"matchType" json:"matchType"`
	Value     string `yaml:"value" json:"value"`
}

// ReplyConditionExprTrue 表达式为真 // exprTrue
type ReplyConditionExprTrue struct {
	CondType string `yaml:"condType" json:"condType"`
	Value    string `yaml:"value" json:"value"`
}

// ReplyConditionTextLenLimit 文本长度限制 // textLenLimit
type ReplyConditionTextLenLimit struct {
	CondType string `yaml:"condType" json:"condType"`
	MatchOp  string `yaml:"matchOp" json:"matchOp"` // 其实是ge或le
	Value    int    `yaml:"value" json:"value"`
}

// Jaro 和 hamming 平均，阈值设为0.7，别问我为啥，玄学决策的
func strCompare(a string, b string) float64 {
	va := strsim.Compare(a, b, strsim.Jaro())
	vb := strsim.Compare(a, b, strsim.Hamming())
	return (va + vb) / 2
}

func (m *ReplyConditionTextMatch) Clean() {
	m.Value = strings.TrimSpace(m.Value)
}

type replyRegexCacheType struct {
	cache SyncMap[string, *regexp.Regexp]
}

func (r *replyRegexCacheType) compile(expr string) *regexp.Regexp {
	if re, ok := r.cache.Load(expr); ok {
		return re
	}

	if ret, err := regexp.Compile(expr); err == nil {
		r.cache.Store(expr, ret)
		return ret
	} else {
		r.cache.Store(expr, nil)
		return nil
	}
}

var replyRegexCache replyRegexCacheType

func (m *ReplyConditionTextMatch) Check(ctx *MsgContext, _ *Message, _ *CmdArgs, cleanText string) bool {
	var ret bool
	switch m.MatchType {
	case "matchExact":
		ret = strings.EqualFold(cleanText, m.Value)
	case "matchMulti":
		texts := strings.Split(m.Value, "|")
		for _, i := range texts {
			if i == cleanText {
				VarSetValueStr(ctx, "$t0", cleanText)
				return true
			}
		}
		return false
	case "matchPrefix":
		ret = strings.HasPrefix(strings.ToLower(cleanText), strings.ToLower(m.Value))
	case "matchSuffix":
		ret = strings.HasSuffix(strings.ToLower(cleanText), strings.ToLower(m.Value))
	case "matchRegex":
		re := replyRegexCache.compile(m.Value)
		if re != nil {
			lst := re.FindStringSubmatch(cleanText)
			gName := re.SubexpNames()
			for index, s := range lst {
				VarSetValueStr(ctx, fmt.Sprintf("$t%d", index), s)
				if gName[index] != "" {
					VarSetValueStr(ctx, fmt.Sprintf("$t%s", gName[index]), s)
				}
			}
			ret = len(lst) != 0
		}
	case "matchFuzzy":
		return strCompare(strings.ToLower(m.Value), strings.ToLower(cleanText)) > 0.7
	case "matchContains":
		return strings.Contains(strings.ToLower(cleanText), strings.ToLower(m.Value))
	case "matchNotContains":
		return !strings.Contains(strings.ToLower(cleanText), strings.ToLower(m.Value))
	}
	if ret {
		VarSetValueStr(ctx, "$t0", cleanText)
	}
	return ret
}

func (m *ReplyConditionTextLenLimit) Clean() {
	if m.MatchOp != "ge" && m.MatchOp != "le" {
		m.MatchOp = "ge"
	}
}

func (m *ReplyConditionTextLenLimit) Check(_ *MsgContext, _ *Message, _ *CmdArgs, cleanText string) bool {
	textLen := len([]rune(cleanText))
	if m.MatchOp == "le" {
		return textLen <= m.Value
	}
	return textLen >= m.Value
}

func (m *ReplyConditionExprTrue) Clean() {
	m.Value = strings.TrimSpace(m.Value)
}

func (m *ReplyConditionExprTrue) Check(ctx *MsgContext, _ *Message, _ *CmdArgs, _ string) bool {
	// r := ctx.Eval(m.Value, ds.RollConfig{})
	flags := RollExtraFlags{V2Only: true}
	if ctx.Dice.VMVersionForReply == "v1" {
		flags.V1Only = true
	}
	r, _, err := DiceExprEvalBase(ctx, m.Value, flags)

	if err != nil {
		ctx.Dice.Logger.Infof("自定义回复表达式执行失败: %s", m.Value)
		return false
	}

	if r.GetRestInput() != "" {
		ctx.Dice.Logger.Infof("自定义回复表达式执行失败(后半部分不能识别 %s): %s", r.GetRestInput(), m.Value)
		return false
	}

	// fmt.Println("???", r, err, r.AsBool(), r.Value == int64(0), r.Value != int64(0))
	return r.AsBool()
}

// ReplyResultReplyToSender replyToSender
type ReplyResultReplyToSender struct {
	ResultType string               `yaml:"resultType" json:"resultType"`
	Delay      float64              `yaml:"delay" json:"delay"`
	Message    TextTemplateItemList `yaml:"message" json:"message"`
}

func (m *ReplyResultReplyToSender) Clean() {
	m.Message.Clean()
}

func formatExprForReply(ctx *MsgContext, expr string) string {
	var text string
	var err error

	if ctx.Dice.VMVersionForReply == "v1" {
		text, err = DiceFormatV1(ctx, expr)
		if err != nil {
			text = fmt.Sprintf("执行出错V1: %s", err.Error())
		}
	} else {
		text, err = DiceFormatV2(ctx, expr)
		if err != nil {
			text = fmt.Sprintf("执行出错V2: %s\n原始文本: %s", err.Error(), strconv.Quote(expr))
		}
	}

	return text
}

func (m *ReplyResultReplyToSender) Execute(ctx *MsgContext, msg *Message, _ *CmdArgs) {
	// go func() {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	p := m.Message.toRandomPool()
	ctx.Player.TempValueAlias = nil // 防止dnd的hp被转为“生命值”

	expr := p.Pick().(string)
	text := formatExprForReply(ctx, expr)
	ReplyToSender(ctx, msg, text)
	// }()
}

// ReplyResultReplyPrivate 回复到私人 replyPrivate
type ReplyResultReplyPrivate struct {
	ResultType string               `yaml:"resultType" json:"resultType"`
	Delay      float64              `yaml:"delay" json:"delay"`
	Message    TextTemplateItemList `yaml:"message" json:"message"`
}

func (m *ReplyResultReplyPrivate) Clean() {
	m.Message.Clean()
}

func (m *ReplyResultReplyPrivate) Execute(ctx *MsgContext, msg *Message, _ *CmdArgs) {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	p := m.Message.toRandomPool()

	expr := p.Pick().(string)
	text := formatExprForReply(ctx, expr)
	ReplyPerson(ctx, msg, text)
}

// ReplyResultReplyGroup 回复到群组 replyGroup
type ReplyResultReplyGroup struct {
	ResultType string                `yaml:"resultType" json:"resultType"`
	Delay      float64               `yaml:"delay" json:"delay"`
	Message    *TextTemplateItemList `yaml:"message" json:"message"`
}

func (m *ReplyResultReplyGroup) Clean() {
	m.Message.Clean()
}

func (m *ReplyResultReplyGroup) Execute(ctx *MsgContext, msg *Message, _ *CmdArgs) {
	// go func() {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	p := m.Message.toRandomPool()

	expr := p.Pick().(string)
	text := formatExprForReply(ctx, expr)
	ReplyGroup(ctx, msg, text)
	// }()
}

// ReplyResultRunText 同.text，但无输出  runText
type ReplyResultRunText struct {
	ResultType string  `yaml:"resultType" json:"resultType"`
	Delay      float64 `yaml:"delay" json:"delay"`
	Message    string  `yaml:"message" json:"message"`
}

func (m *ReplyResultRunText) Execute(ctx *MsgContext, _ *Message, _ *CmdArgs) {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	flags := RollExtraFlags{V2Only: true}
	if ctx.Dice.VMVersionForReply == "v1" {
		flags.V1Only = true
	}
	_, _, _ = DiceExprTextBase(ctx, m.Message, flags)
}

type ReplyConditions []ReplyConditionBase

var _ json.Unmarshaler = (*ReplyConditions)(nil)
var _ yaml.Unmarshaler = (*ReplyConditions)(nil)

type ReplyItem struct {
	Enable     bool              `yaml:"enable" json:"enable"`
	Conditions ReplyConditions   `yaml:"conditions" json:"conditions"`
	Results    []ReplyResultBase `yaml:"results" json:"results"`
}

var _ json.Unmarshaler = (*ReplyItem)(nil)
var _ yaml.Unmarshaler = (*ReplyItem)(nil)

type ReplyConfig struct {
	Enable   bool         `yaml:"enable" json:"enable"`
	Interval float64      `yaml:"interval" json:"interval"` // 响应间隔，最少为5
	Items    []*ReplyItem `yaml:"items" json:"items"`

	// 作者信息
	Name            string   `yaml:"name" json:"name"`
	Author          []string `yaml:"author" json:"author"`
	Version         string   `yaml:"version" json:"version"`
	CreateTimestamp int64    `yaml:"createTimestamp" json:"createTimestamp"`
	UpdateTimestamp int64    `yaml:"updateTimestamp" json:"updateTimestamp"`
	Desc            string   `yaml:"desc" json:"desc"`

	// 文件级别执行条件
	Conditions ReplyConditions `yaml:"conditions" json:"conditions"`

	// web专用
	Filename string `yaml:"-" json:"filename"`
}

func (c *ReplyConfig) Save(dice *Dice) {
	attrConfigFn := dice.GetExtConfigFilePath("reply", c.Filename)
	buf, err := yaml.Marshal(c)
	if err != nil {
		fmt.Println(err)
	} else {
		_ = os.WriteFile(attrConfigFn, buf, 0644)
	}
}

func (c *ReplyConfig) Clean() {
	for _, cond := range c.Conditions {
		cond.Clean()
	}
	for _, i := range c.Items {
		for _, j := range i.Conditions {
			j.Clean()
		}
		for _, j := range i.Results {
			j.Clean()
		}
	}
}

func (cond *ReplyConditions) UnmarshalJSON(data []byte) error {
	var err error
	var cs = []any{}
	typeMap := map[string]reflect.Type{
		"textMatch":    reflect.TypeOf(ReplyConditionTextMatch{}),
		"exprTrue":     reflect.TypeOf(ReplyConditionExprTrue{}),
		"textLenLimit": reflect.TypeOf(ReplyConditionTextLenLimit{}),
	}

	if err = json.Unmarshal(data, &cs); err != nil {
		return err
	}

	next := make([]ReplyConditionBase, 0, len(cs))

	for _, condRaw := range cs {
		condRawMap, ok := condRaw.(map[string]any)
		if !ok || condRawMap["condType"] == nil {
			continue
		}
		typeName, _ := condRawMap["condType"].(string)
		theType := typeMap[typeName]
		if theType != nil {
			val, err := tryUnmarshalYAML(condRaw, theType)
			if err != nil {
				return err
			}
			next = append(next, val.(ReplyConditionBase))
		}
	}

	*cond = next
	return nil
}

func (cond *ReplyConditions) UnmarshalYAML(data *yaml.Node) error {
	var err error
	var cs = []any{}
	typeMap := map[string]reflect.Type{
		"textMatch":    reflect.TypeOf(ReplyConditionTextMatch{}),
		"exprTrue":     reflect.TypeOf(ReplyConditionExprTrue{}),
		"textLenLimit": reflect.TypeOf(ReplyConditionTextLenLimit{}),
	}

	// HACK: 用更加符合 yaml 库原生设计的方式重新实现
	if err = data.Decode(&cs); err != nil {
		return err
	}

	next := make([]ReplyConditionBase, 0, len(cs))

	for _, condRaw := range cs {
		condRawMap, ok := condRaw.(map[string]any)
		if !ok || condRawMap["condType"] == nil {
			continue
		}
		typeName, _ := condRawMap["condType"].(string)
		theType := typeMap[typeName]
		if theType != nil {
			val, err := tryUnmarshalYAML(condRaw, theType)
			if err != nil {
				return err
			}
			next = append(next, val.(ReplyConditionBase))
		}
	}

	*cond = next
	return nil
}

func (ri *ReplyItem) UnmarshalJSON(data []byte) error {
	var err error
	m := map[string]json.RawMessage{}

	if err = json.Unmarshal(data, &m); err != nil {
		return err
	}

	if val, ok := m["enable"]; ok {
		_ = json.Unmarshal(val, &ri.Enable)
	}

	if val, ok := m["conditions"]; ok {
		err = json.Unmarshal(val, &ri.Conditions)
		if err != nil {
			return err
		}
	}

	if val, ok := m["results"]; ok {
		ri.Results = []ReplyResultBase{}

		rs := []any{}
		err = json.Unmarshal(val, &rs)
		if err != nil {
			return err
		}

		typeMap := map[string]reflect.Type{
			"replyPrivate":  reflect.TypeOf(ReplyResultReplyPrivate{}),
			"replyGroup":    reflect.TypeOf(ReplyResultReplyGroup{}),
			"replyToSender": reflect.TypeOf(ReplyResultReplyToSender{}),
			"runText":       reflect.TypeOf(ReplyResultRunText{}),
		}

		for _, i := range rs {
			m, ok := i.(map[string]interface{})
			if ok && m["resultType"] != nil {
				name, _ := m["resultType"].(string)
				theType := typeMap[name]
				if theType != nil {
					val, err := tryUnmarshalJSON(i, theType)
					if err != nil {
						return err
					}
					ri.Results = append(ri.Results, val.(ReplyResultBase))
				}
			}
		}
	}

	return nil
}

func (ri *ReplyItem) UnmarshalYAML(value *yaml.Node) error {
	var err error
	m := map[string]yaml.Node{}

	err = value.Decode(m)
	if err != nil {
		return err
	}

	if val, ok := m["enable"]; ok {
		_ = val.Decode(&ri.Enable)
	}

	if val, ok := m["conditions"]; ok {
		ri.Conditions = ReplyConditions{}
		err = val.Decode(&ri.Conditions)
		if err != nil {
			return err
		}
	}

	if val, ok := m["results"]; ok {
		ri.Results = []ReplyResultBase{}
		var rs []any
		err = val.Decode(&rs)
		if err != nil {
			return err
		}

		typeMap := map[string]reflect.Type{
			"replyPrivate":  reflect.TypeOf(ReplyResultReplyPrivate{}),
			"replyGroup":    reflect.TypeOf(ReplyResultReplyGroup{}),
			"replyToSender": reflect.TypeOf(ReplyResultReplyToSender{}),
			"runText":       reflect.TypeOf(ReplyResultRunText{}),
		}

		for _, i := range rs {
			m, ok := i.(map[string]interface{})
			if ok && m["resultType"] != nil {
				name, _ := m["resultType"].(string)
				theType := typeMap[name]
				if theType != nil {
					val, err := tryUnmarshalYAML(i, theType)
					if err != nil {
						return err
					}
					ri.Results = append(ri.Results, val.(ReplyResultBase))
				}
			}
		}
	}
	return nil
}

func tryUnmarshalJSON(input any, t reflect.Type) (any, error) {
	valueBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	value := reflect.New(t).Interface()
	if err = json.Unmarshal(valueBytes, &value); err != nil {
		return nil, err
	}

	return value, nil
}

func tryUnmarshalYAML(input any, t reflect.Type) (any, error) {
	// TODO(Xiangze Li): 真的需要区分json和yaml吗？输入已经是any了，中间格式用任何一种应该都可以
	valueBytes, err := yaml.Marshal(input)
	if err != nil {
		return nil, err
	}
	value := reflect.New(t).Interface()
	if err = yaml.Unmarshal(valueBytes, value); err != nil {
		return nil, err
	}

	return value, nil
}
