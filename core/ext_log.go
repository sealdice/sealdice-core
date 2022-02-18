package main

import "sort"

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
					}
					replyToSender(session.Socket, msg, "尚未开发完成，敬请期待");
					return struct{ success bool }{
						success: true,
					}
				},
			},
		},
	})
}
