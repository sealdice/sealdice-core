package dice

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func CustomReplyConfigRead(dice *Dice) *ReplyConfig {
	attrConfigFn := dice.GetExtConfigFilePath("reply", "reply.yaml")
	rc := &ReplyConfig{Enable: false}

	if _, err := os.Stat(attrConfigFn); err == nil {
		// 如果文件存在，那么读取
		af, err := ioutil.ReadFile(attrConfigFn)
		if err == nil {
			err = yaml.Unmarshal(af, rc)
			if err != nil {
				dice.Logger.Error("读取自定义回复配置文件发生异常，请检查格式是否正确")
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
	//a.Condition = &ReplyConditionTextMatch{"match", "match_exact", "asd"}
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
	//fmt.Println(333, ri.Condition, ri.Condition.(*ReplyConditionTextMatch))
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
		Brief:      "自定义回复模块，支持各种文本匹配和简易脚本",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		OnNotCommandReceived: func(ctx *MsgContext, msg *Message) {
			// 当前，只有非指令才会匹配
			rc := ctx.Dice.CustomReplyConfig
			if rc.Enable {
				lastTime := ctx.Group.LastCustomReplyTime
				now := float64(time.Now().UnixMilli()) / 1000
				interval := rc.Interval
				if interval < 5 {
					interval = 5
				}

				if now-lastTime < interval {
					return // 未达到冷却，退出
				}
				ctx.Group.LastCustomReplyTime = now

				cleanText, _ := AtParse(msg.Message, "")
				cleanText = strings.TrimSpace(cleanText)
				for _, i := range rc.Items {
					//fmt.Println("???", i.Enable, i)
					if i.Enable {
						checkTrue := true
						for _, i := range i.Conditions {
							if !i.Check(ctx, msg, nil, cleanText) {
								checkTrue = false
								break
							}
						}
						if len(i.Conditions) > 0 && checkTrue {
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
					dice.Parent.Backup(AllBackupConfig{
						Decks:   true,
						HelpDoc: true,
						Dices: map[string]*OneBackupConfig{
							"default": &OneBackupConfig{
								MiscConfig:  true,
								PlayerData:  true,
								CustomReply: true,
								CustomText:  true,
								Accounts:    true,
							},
						},
					}, "bak.zip")

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
