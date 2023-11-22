package dice

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func CustomReplyConfigRead(dice *Dice, filename string) (*ReplyConfig, error) {
	attrConfigFn := dice.GetExtConfigFilePath("reply", filename)
	rc := &ReplyConfig{Enable: false, Filename: filename}

	if _, err := os.Stat(attrConfigFn); err == nil {
		// 如果文件存在，那么读取
		af, err := os.ReadFile(attrConfigFn)
		if err == nil {
			err = yaml.Unmarshal(af, rc)
			if err != nil {
				dice.Logger.Error("读取自定义回复配置文件发生异常，请检查格式是否正确")
				return nil, err
			}
		}
	}

	if rc.Items == nil {
		rc.Items = []*ReplyItem{}
	}

	return rc, nil
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
		if strings.EqualFold(i.Filename, filename) {
			return nil
		}
	}

	nowTime := time.Now().Unix()
	rc := &ReplyConfig{Enable: true, Filename: filename, Name: filename, Items: []*ReplyItem{}, UpdateTimestamp: nowTime, CreateTimestamp: nowTime, Author: []string{"无名海豹"}}
	dice.CustomReplyConfig = append(dice.CustomReplyConfig, rc)
	rc.Save(dice)
	return rc
}

func CustomReplyConfigDelete(dice *Dice, filename string) bool {
	attrConfigFn := dice.GetExtConfigFilePath("reply", filename)
	if _, err := os.Stat(attrConfigFn); err == nil {
		err := os.Remove(attrConfigFn)
		if err == nil {
			var rcs []*ReplyConfig
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

func ReplyReload(dice *Dice) {
	var rcs []*ReplyConfig
	filenames := []string{"reply.yaml"}
	_ = filepath.Walk(dice.GetExtDataDir("reply"), func(path string, info fs.FileInfo, err error) error {
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
		rc, err := CustomReplyConfigRead(dice, i)
		if err == nil {
			dice.Logger.Info("读取自定义回复配置:", i)
			rc.Save(dice)
			rcs = append(rcs, rc)
		} else {
			dice.Logger.Info("读取自定义回复配置 - 失败:", i)
			dice.Logger.Error(err)
		}
	}

	dice.CustomReplyConfig = rcs
}

func RegisterBuiltinExtReply(dice *Dice) {
	ReplyReload(dice)

	theExt := &ExtInfo{
		Name:       "reply", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.2.0",
		Brief:      "自定义回复模块，支持各种文本匹配和简易脚本",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		Official:   true,
		OnNotCommandReceived: func(ctx *MsgContext, msg *Message) {
			// 当前，只有非指令才会匹配
			rcs := ctx.Dice.CustomReplyConfig
			if !ctx.Dice.CustomReplyConfigEnable {
				return
			}
			executed := false
			log := ctx.Dice.Logger

			cleanText, _ := AtParse(msg.Message, "")
			cleanText = strings.TrimSpace(cleanText)
			VarSetValueInt64(ctx, "$t文本长度", int64(len(cleanText)))

			if dice.ReplyDebugMode {
				log.Infof("[回复调试]当前文本:“%s” hex: %x 字节形式: %v", cleanText, cleanText, []byte(cleanText))
			}

			// 在判定条件前，先设置一轮变量，以免条件中的变量出问题
			SetTempVars(ctx, msg.Sender.Nickname)

			for _, rc := range rcs {
				if executed {
					break
				}
				if rc.Enable {
					condIndex := -1
					defer func() {
						if r := recover(); r != nil {
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
									log.Infof("自定义回复[%s]: 条件满足，但正处于冷却", rc.Filename)
									return
								}
							}

							if len(i.Conditions) > 0 && checkTrue {
								log.Infof("自定义回复[%s]: 条件满足", rc.Filename)

								SetTempVars(ctx, msg.Sender.Nickname)
								VarSetValueStr(ctx, "$tMsgID", fmt.Sprintf("%v", msg.RawID))
								for _, j := range i.Results {
									j.Execute(ctx, msg, nil)
								}
								executed = true
								break
							}
						}
					}
				}
			}
		},
		GetDescText: GetExtensionDesc,
		CmdMap:      CmdMapCls{},
	}

	dice.RegisterExtension(theExt)
}
