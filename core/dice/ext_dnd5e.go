package dice

func RegisterBuiltinExtDnd5e(self *Dice) {
	theExt := &ExtInfo{
		Name:       "dnd5e", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "不要看了，还没开始。咕咕咕",
		Author:     "木落",
		AutoActive: false, // 是否自动开启
		ConflictWith: []string{
			"coc7",
		},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func(i *ExtInfo) string {
			return ""
		},
		CmdMap: CmdMapCls{},
	}

	self.RegisterExtension(theExt)
}
