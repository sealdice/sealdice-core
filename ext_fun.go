package main

func (self *Dice) registerBuiltinExtFun() {
	cmdGugu := CmdItemInfo{
		name: "gugu",
		solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if isCurGroupBotOn(session, msg) || msg.MessageType == "private" {
				//p := getPlayerInfoBySender(session, msg)
				replyToSender(session.Socket, msg, "🕊️: 我有点事，你们先开")
			}
			return struct{ success bool }{
				success: true,
			}
		},
	}

	self.extList = append(self.extList, &ExtInfo{
		Name:       "fun", // 扩展的名称，需要用于指令中，写简短点
		version:    "0.0.1",
		Brief: "娱乐扩展，主打抽牌功能、智能鸽子",
		autoActive: true, // 是否自动开启
		EntryHook: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		cmdMap: CmdMapCls{
			"gugu": &cmdGugu,
			"咕咕": &cmdGugu,
		},
	})
}
