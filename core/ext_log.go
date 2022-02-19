package main

import (
	"fmt"
	"sort"
	"strings"
)

func (self *Dice) registerBuiltinExtLog() {
	self.extList = append(self.extList, &ExtInfo{
		Name:       "log",
		version:    "0.0.1",
		Brief: "跑团辅助扩展，提供日志、染色等功能",
		autoActive: true,
		OnPrepare: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func (ei *ExtInfo) string {
			text := "> " + ei.Brief + "\n" + "提供命令:\n"
			keys := make([]string, 0, len(ei.cmdMap))
			for k := range ei.cmdMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, i := range keys {
				i := ei.cmdMap[i]
				brief := i.Brief
				if brief != "" {
					brief = " // " + brief
				}
				text += i.name + brief + "\n"
			}

			return text
		},
		cmdMap: CmdMapCls{
			"log": &CmdItemInfo{
				name: ".log",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if isCurGroupBotOn(session, msg) {
						//p := getPlayerInfoBySender(session, msg)
						group := session.ServiceAt[msg.GroupId]

						if len(cmdArgs.Args) == 0 {
							onText := "关闭"
							if group.LogOn {
								onText = "开启"
							}
							text := fmt.Sprintf("日志指令，当前状态: %s\n已记录文本%d条", onText, group.LogCurLines)
							replyToSender(session.Socket, msg, text);
						} else {
							if cmdArgs.isArgEqual(1, "on") {
								group.LogOn = true
								text := fmt.Sprintf("日志已经开启，当前已记录文本%d条", group.LogCurLines)
								replyToSender(session.Socket, msg, text);
							} else if cmdArgs.isArgEqual(1, "off") {
								group.LogOn = false
								text := fmt.Sprintf("日志已经关闭，当前已记录文本%d条", group.LogCurLines)
								replyToSender(session.Socket, msg, text);
							} else if cmdArgs.isArgEqual(1, "show") {
								group.LogOn = false
								info := strings.Join(group.tmpTexts, "\n")
								text := fmt.Sprintf("%d条记录如下:\n%s\n", group.LogCurLines, info)
								replyToSender(session.Socket, msg, text);
							} else if cmdArgs.isArgEqual(1, "clear", "clr") {
								group.LogCurLines = 0
								group.tmpTexts = group.tmpTexts[:]
								replyToSender(session.Socket, msg, "日志行数和记录已清零");
							} else if cmdArgs.isArgEqual(1, "new") {
								group.LogCurLines = 0
								group.tmpTexts = group.tmpTexts[:]
								replyToSender(session.Socket, msg, "新的故事开始了，祝旅途愉快！");
							}
						}
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
		},
	})
}
