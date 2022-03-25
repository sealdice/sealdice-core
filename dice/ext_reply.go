package dice

func RegisterBuiltinExtReply(self *Dice) {
	theExt := &ExtInfo{
		Name:       "reply", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "[尚未实现]智能回复模块，支持关键字精确匹配和模糊匹配",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
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

	self.RegisterExtension(theExt)
}
