package dice

func RegisterBuiltinStory(self *Dice) {
	theExt := &ExtInfo{
		Name:       "story", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "提供随机姓名、线索板、安科等功能",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func(i *ExtInfo) string {
			return GetExtensionDesc(i)
		},
		OnLoad: func() {

		},
		CmdMap: CmdMapCls{
			"name": &CmdItemInfo{
				Name: "name",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					if ctx.IsCurGroupBotOn || ctx.IsPrivate {
						// do something
					}
					return CmdExecuteResult{Matched: true, Solved: false}
				},
			},
		},
	}

	self.RegisterExtension(theExt)
}
