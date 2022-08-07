package dice

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

func CustomReplyConfigRead(dice *Dice, filename string) *ReplyConfig {
	attrConfigFn := dice.GetExtConfigFilePath("reply", filename)
	rc := &ReplyConfig{Enable: false, Filename: filename}

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

func CustomReplyConfigCheckExists(dice *Dice, filename string) bool {
	attrConfigFn := dice.GetExtConfigFilePath("reply", filename)
	if _, err := os.Stat(attrConfigFn); err == nil {
		return true
	}
	return false
}

func CustomReplyConfigNew(dice *Dice, filename string) *ReplyConfig {
	for _, i := range dice.CustomReplyConfig {
		if strings.ToLower(i.Filename) == strings.ToLower(filename) {
			return nil
		}
	}

	rc := &ReplyConfig{Enable: true, Filename: filename, Items: []*ReplyItem{}}
	dice.CustomReplyConfig = append(dice.CustomReplyConfig, rc)
	rc.Save(dice)
	return rc
}

func CustomReplyConfigDelete(dice *Dice, filename string) bool {
	attrConfigFn := dice.GetExtConfigFilePath("reply", filename)
	if _, err := os.Stat(attrConfigFn); err == nil {
		err := os.Remove(attrConfigFn)
		if err == nil {
			rcs := []*ReplyConfig{}
			for _, i := range dice.CustomReplyConfig {
				if i.Filename != filename {
					rcs = append(rcs, i)
				}
			}
			dice.CustomReplyConfig = rcs
		}
		return true
	}
	return false
}

func RegisterBuiltinExtReply(dice *Dice) {
	//rc := CustomReplyConfigRead(dice, "reply.yaml")
	//rc.Save(dice)
	//dice.CustomReplyConfig = append(dice.CustomReplyConfig, rc)

	filenames := []string{"reply.yaml"}
	filepath.Walk(dice.GetExtDataDir("reply"), func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() && strings.EqualFold(info.Name(), "assets") {
			return fs.SkipDir
		}
		if info.IsDir() && strings.EqualFold(info.Name(), "images") {
			return fs.SkipDir
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), ".reply") {
			return nil
		}
		if info.Name() == "info.yaml" {
			return nil
		}

		if !info.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".yaml" || ext == "" {
				if info.Name() != "reply.yaml" {
					filenames = append(filenames, info.Name())
				}
			}
		}
		return nil
	})

	for _, i := range filenames {
		dice.Logger.Info("读取自定义回复配置:", i)
		rc := CustomReplyConfigRead(dice, i)
		rc.Save(dice)
		dice.CustomReplyConfig = append(dice.CustomReplyConfig, rc)
	}

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
		Version:    "1.1.0",
		Brief:      "自定义回复模块，支持各种文本匹配和简易脚本",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		OnNotCommandReceived: func(ctx *MsgContext, msg *Message) {
			// 当前，只有非指令才会匹配
			rcs := ctx.Dice.CustomReplyConfig
			for _, rc := range rcs {
				if rc.Enable {
					log := ctx.Dice.Logger
					condIndex := -1
					defer func() {
						if r := recover(); r != nil {
							//  + fmt.Sprintf("%s", r)
							log.Errorf("异常: %v 堆栈: %v", r, string(debug.Stack()))
							if condIndex != -1 {
								ReplyToSender(ctx, msg, fmt.Sprintf(
									"自定义回复匹配成功(序号%d)，但回复内容触发异常，请联系骰主修改:\n%s",
									condIndex, DiceFormatTmpl(ctx, "核心:骰子执行异常")))
							}
						}
					}()

					checkInCoolDown := func() bool {
						lastTime := ctx.Group.LastCustomReplyTime
						now := float64(time.Now().UnixMilli()) / 1000
						interval := rc.Interval
						if interval < 2 {
							interval = 2
						}

						if now-lastTime < interval {
							return true // 未达到冷却，退出
						}
						ctx.Group.LastCustomReplyTime = now
						return false
					}

					cleanText, _ := AtParse(msg.Message, "")
					cleanText = strings.TrimSpace(cleanText)
					VarSetValueInt64(ctx, "$t文本长度", int64(len(cleanText)))

					for index, i := range rc.Items {
						if i.Enable {
							checkTrue := true
							for _, i := range i.Conditions {
								if !i.Check(ctx, msg, nil, cleanText) {
									checkTrue = false
									break
								}
							}
							condIndex = index
							if len(i.Conditions) > 0 && checkTrue {
								inCoolDown := checkInCoolDown()
								if inCoolDown {
									// 仍在冷却，拒绝回复
									log.Infof("自定义回复: 条件满足，但正处于冷却")
									return
								}
							}

							//fmt.Println("!!!!", cleanText, checkTrue, len(i.Conditions), len(i.Results))
							//if checkTrue {
							//	fmt.Println("!!xx", cleanText)
							//}

							if len(i.Conditions) > 0 && checkTrue {
								SetTempVars(ctx, msg.Sender.Nickname)
								VarSetValueStr(ctx, "$tMsgID", fmt.Sprintf("%v", msg.RawId))
								for _, j := range i.Results {
									j.Execute(ctx, msg, nil)
								}
								break
							}
						}
					}
				}
			}
		},
		GetDescText: func(i *ExtInfo) string {
			return GetExtensionDesc(i)
		},
		CmdMap: CmdMapCls{},
	}

	dice.RegisterExtension(theExt)
}
