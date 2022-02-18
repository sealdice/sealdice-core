package main

func (self *Dice) registerBuiltinExtTemple() {
	self.extList = append(self.extList, &ExtInfo{
		Name:       "name", // 扩展的名称，需要用于指令中，写简短点
		version:    "0.0.1",
		Brief: "一行字简介",
		autoActive: true, // 是否自动开启
		EntryHook: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		cmdMap: CmdMapCls{
			"command": &CmdItemInfo{
				name: "command",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if isCurGroupBotOn(session, msg) {
						//p := getPlayerInfoBySender(session, msg)
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
		},
	})
}
