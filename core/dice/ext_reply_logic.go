package dice

import (
	"encoding/json"
	"fmt"
	"github.com/antlabs/strsim"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type ReplyConditionBase interface {
	Check(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, cleanText string) bool
}

type ReplyResultBase interface {
	Execute(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs)
}

// ReplyConditionTextMatch 文本匹配 // textMatch
type ReplyConditionTextMatch struct {
	CondType  string `yaml:"condType" json:"condType"`
	MatchType string `yaml:"matchType" json:"matchType"` // matchExact 精确 matchRegex 正则 matchFuzzy 模糊
	Value     string `yaml:"value" json:"value"`
}

// ReplyConditionExprTrue 表达式为真 // exprTrue
type ReplyConditionExprTrue struct {
	CondType string `yaml:"condType" json:"condType"`
	Value    string `yaml:"value" json:"value"`
}

// Jaro 和 hamming 平均，阈值设为0.7，别问我为啥，玄学决策的
func strCompare(a string, b string) float64 {
	va := strsim.Compare(a, b, strsim.Jaro())
	vb := strsim.Compare(a, b, strsim.Hamming())
	return (va + vb) / 2
}

func (m *ReplyConditionTextMatch) Check(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, cleanText string) bool {
	var ret bool
	switch m.MatchType {
	case "matchExact":
		ret = strings.ToLower(cleanText) == strings.ToLower(m.Value)
	case "matchPrefix":
		ret = strings.HasPrefix(strings.ToLower(cleanText), strings.ToLower(m.Value))
	case "matchSuffix":
		ret = strings.HasSuffix(strings.ToLower(cleanText), strings.ToLower(m.Value))
	case "matchRegex":
		re, err := regexp.Compile(m.Value)
		if err == nil {
			ret = re.MatchString(cleanText)
		}
	case "matchFuzzy":
		return strCompare(strings.ToLower(m.Value), strings.ToLower(cleanText)) > 0.7
	}
	return ret
}

// ReplyResultReplyToSender replyToSender
type ReplyResultReplyToSender struct {
	ResultType string               `yaml:"resultType" json:"resultType"`
	Delay      float64              `yaml:"delay" json:"delay"`
	Message    TextTemplateItemList `yaml:"message" json:"message"`
}

func (m *ReplyResultReplyToSender) Execute(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	//go func() {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	p := m.Message.toRandomPool()
	ReplyToSender(ctx, msg, DiceFormat(ctx, p.Pick().(string)))
	//}()
}

// ReplyResultReplyPrivate 回复到私人 replyPrivate
type ReplyResultReplyPrivate struct {
	ResultType string               `yaml:"resultType" json:"resultType"`
	Delay      float64              `yaml:"delay" json:"delay"`
	Message    TextTemplateItemList `yaml:"message" json:"message"`
}

func (m *ReplyResultReplyPrivate) Execute(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	//go func() {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	p := m.Message.toRandomPool()
	ReplyPerson(ctx, msg, DiceFormat(ctx, p.Pick().(string)))
	//ReplyPerson(ctx, msg, DiceFormat(ctx, m.Message))
	//}()
}

// ReplyResultReplyGroup 回复到群组 replyGroup
type ReplyResultReplyGroup struct {
	ResultType string                `yaml:"resultType" json:"resultType"`
	Delay      float64               `yaml:"delay" json:"delay"`
	Message    *TextTemplateItemList `yaml:"message" json:"message"`
}

func (m *ReplyResultReplyGroup) Execute(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	//go func() {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	p := m.Message.toRandomPool()
	ReplyGroup(ctx, msg, DiceFormat(ctx, p.Pick().(string)))
	//}()
}

// ReplyResultRunText 同.text，但无输出  runText
type ReplyResultRunText struct {
	ResultType string  `yaml:"resultType" json:"resultType"`
	Delay      float64 `yaml:"delay" json:"delay"`
	Message    string  `yaml:"message" json:"message"`
}

func (m *ReplyResultRunText) Execute(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	//go func() {
	time.Sleep(time.Duration(m.Delay * float64(time.Second)))
	_, _, _ = ctx.Dice.ExprTextBase(m.Message, ctx)
	//}()
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
}

func (c *ReplyConfig) Save(dice *Dice) {
	attrConfigFn := dice.GetExtConfigFilePath("reply", "reply.yaml")
	buf, err := yaml.Marshal(c)
	if err != nil {
		fmt.Println(err)
	} else {
		ioutil.WriteFile(attrConfigFn, buf, 0644)
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
				"textMatch": reflect.TypeOf(ReplyConditionTextMatch{}),
				"exprTrue":  reflect.TypeOf(ReplyConditionExprTrue{}),
			}

			for _, i := range cs {
				m, ok := i.(map[string]interface{})
				if ok && m["condType"] != nil {
					name, _ := m["condType"].(string)
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
				"textMatch": reflect.TypeOf(ReplyConditionTextMatch{}),
				"exprTrue":  reflect.TypeOf(ReplyConditionExprTrue{}),
			}

			for _, i := range cs {
				m, ok := i.(map[string]interface{})
				if ok && m["condType"] != nil {
					name, _ := m["condType"].(string)
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
