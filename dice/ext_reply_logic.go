package dice

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
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
		re, err := regexp.Compile(m.Value)
		if err == nil {
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
	r, _, err := ctx.Dice.ExprEval(m.Value, ctx)
	if err != nil {
		ctx.Dice.Logger.Infof("自定义回复表达式执行失败: %s", m.Value)
		return false
	}

	if r.restInput != "" {
		ctx.Dice.Logger.Infof("自定义回复表达式执行失败(后半部分不能识别 %s): %s", r.restInput, m.Value)
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

func (m *ReplyResultReplyToSender) Execute(ctx *MsgContext, msg *Message, _ *CmdArgs) {
	// go func() {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	p := m.Message.toRandomPool()
	ctx.Player.TempValueAlias = nil // 防止dnd的hp被转为“生命值”
	ReplyToSender(ctx, msg, DiceFormat(ctx, p.Pick().(string)))
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
	ReplyPerson(ctx, msg, DiceFormat(ctx, p.Pick().(string)))
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
	ReplyGroup(ctx, msg, DiceFormat(ctx, p.Pick().(string)))
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
	_, _, _ = ctx.Dice.ExprTextBase(m.Message, ctx, RollExtraFlags{})
}

type ReplyItem struct {
	Enable     bool                 `yaml:"enable" json:"enable"`
	Conditions []ReplyConditionBase `yaml:"conditions" json:"conditions"`
	Results    []ReplyResultBase    `yaml:"results" json:"results"`
}

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

func (ri *ReplyItem) UnmarshalJSON(data []byte) error {
	var err error
	m := map[string]interface{}{}

	if err = json.Unmarshal(data, &m); err != nil {
		return err
	}

	ri.Enable, _ = m["enable"].(bool)

	tryUnmarshal := func(input interface{}, t reflect.Type) (interface{}, error) {
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

	if m["conditions"] != nil {
		ri.Conditions = []ReplyConditionBase{}
		cs, ok := m["conditions"].([]interface{})
		if ok {
			typeMap := map[string]reflect.Type{
				"textMatch":    reflect.TypeOf(ReplyConditionTextMatch{}),
				"exprTrue":     reflect.TypeOf(ReplyConditionExprTrue{}),
				"textLenLimit": reflect.TypeOf(ReplyConditionTextLenLimit{}),
			}

			for _, i := range cs {
				mm, ok := i.(map[string]interface{})
				if ok && mm["condType"] != nil {
					name, _ := mm["condType"].(string)
					theType := typeMap[name]
					if theType != nil {
						val, err := tryUnmarshal(i, theType)
						if err != nil {
							return err
						}
						ri.Conditions = append(ri.Conditions, val.(ReplyConditionBase))
					}
				}
			}
		}
	}

	if m["results"] != nil {
		ri.Results = []ReplyResultBase{}
		rs, ok := m["results"].([]interface{})
		if ok {
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
						val, err := tryUnmarshal(i, theType)
						if err != nil {
							return err
						}
						ri.Results = append(ri.Results, val.(ReplyResultBase))
					}
				}
			}
		}
	}
	return nil
}

func (ri *ReplyItem) UnmarshalYAML(value *yaml.Node) error {
	var err error
	m := map[string]interface{}{}

	err = value.Decode(m)
	if err != nil {
		return err
	}

	ri.Enable, _ = m["enable"].(bool)

	tryUnmarshal := func(input interface{}, t reflect.Type) (interface{}, error) {
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

	if m["conditions"] != nil {
		ri.Conditions = []ReplyConditionBase{}
		cs, ok := m["conditions"].([]interface{})
		if ok {
			typeMap := map[string]reflect.Type{
				"textMatch":    reflect.TypeOf(ReplyConditionTextMatch{}),
				"exprTrue":     reflect.TypeOf(ReplyConditionExprTrue{}),
				"textLenLimit": reflect.TypeOf(ReplyConditionTextLenLimit{}),
			}

			for _, i := range cs {
				mm, ok := i.(map[string]interface{})
				if ok && mm["condType"] != nil {
					name, _ := mm["condType"].(string)
					theType := typeMap[name]
					if theType != nil {
						val, err := tryUnmarshal(i, theType)
						if err != nil {
							return err
						}
						ri.Conditions = append(ri.Conditions, val.(ReplyConditionBase))
					}
				}
			}
		}
	}

	if m["results"] != nil {
		ri.Results = []ReplyResultBase{}
		rs, ok := m["results"].([]interface{})
		if ok {
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
						val, err := tryUnmarshal(i, theType)
						if err != nil {
							return err
						}
						ri.Results = append(ri.Results, val.(ReplyResultBase))
					}
				}
			}
		}
	}
	return nil
}

func (c *ReplyConfig) Clean() {
	for _, i := range c.Items {
		for _, j := range i.Conditions {
			j.Clean()
		}
		for _, j := range i.Results {
			j.Clean()
		}
	}
}
