package main

func (self *Dice) registerBuiltinExtLog() {
	self.extList = append(self.extList, &ExtInfo{
		Name:       "log",
		version:    "0.0.1",
		Brief: "跑团辅助扩展，提供日志、染色等功能",
		autoActive: true,
		EntryHook: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		cmdMap: CmdMapCls{
			"log": &CmdItemInfo{
				name: "log",
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
