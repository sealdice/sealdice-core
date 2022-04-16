package dice

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

func CustomReplyConfigRead(dice *Dice) *ReplyConfig {
	attrConfigFn := dice.GetExtConfigFilePath("reply", "reply.yaml")
	rc := &ReplyConfig{Enable: true}

	if _, err := os.Stat(attrConfigFn); err == nil {
		// 如果文件存在，那么读取
		af, err := ioutil.ReadFile(attrConfigFn)
		if err == nil {
			err = yaml.Unmarshal(af, rc)
			if err != nil {
				panic(err)
			}
		}
	}

	if rc.Items == nil {
		rc.Items = []*ReplyItem{}
	}

	return rc
}

func RegisterBuiltinExtReply(dice *Dice) {
	rc := CustomReplyConfigRead(dice)
	rc.Save(dice)
	dice.CustomReplyConfig = rc

	//a := ReplyItem{}
	//a.Enable = true
	//a.Condition = &ReplyConditionMatch{"match", "match_exact", "asd"}
	//a.Results = []ReplyResultBase{
	//	&ReplyResultReplyToSender{
	//		"replyToSender",
	//		0.3,
	//		"text",
	//	},
	//}

	//txt, _ := yaml.Marshal(a)
	//fmt.Println(string(txt))
	//{"enable":true,"condition":{"condType":"match","matchType":"match_exact","value":"asd"},"results":[{"resultType":"replyToSender","delay":0.3,"message":"text"}]}
	////{"enable":false,"condition":{"condType":"match","matchType":"match_exact","value":"asd"},"results":null}
	//
	//ri := ReplyItem{}
	//fmt.Println(yaml.Unmarshal(txt, &ri))
	//fmt.Println(333, ri.Condition, ri.Condition.(*ReplyConditionMatch))
	//
	//rc := ReplyConfig{
	//	Enable: true,
	//	Items: []*ReplyItem{
	//		&ri,
	//	},
	//}

	theExt := &ExtInfo{
		Name:       "reply", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "[尚未实现]智能回复模块，支持关键字精确匹配和模糊匹配",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		OnNotCommandReceived: func(ctx *MsgContext, msg *Message) {
			// 当前，只有非指令才会匹配
			rc := ctx.Dice.CustomReplyConfig
			if rc.Enable {
				for _, i := range rc.Items {
					//fmt.Println("???", i.Enable, i)
					if i.Enable {
						if i.Condition.Check(ctx, msg, nil) {
							for _, j := range i.Results {
								j.Execute(ctx, msg, nil)
							}
							break
						}
					}
				}
			}
		},
		GetDescText: func(i *ExtInfo) string {
			return GetExtensionDesc(i)
		},
		CmdMap: CmdMapCls{
			"reply": &CmdItemInfo{
				Name: "reply",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					if ctx.IsCurGroupBotOn {
						// do something
						ReplyToSender(ctx, msg, "并不存在的指令，或许敬请期待？")
					}
					return CmdExecuteResult{Matched: true, Solved: false}
				},
			},
		},
	}

	dice.RegisterExtension(theExt)
}
