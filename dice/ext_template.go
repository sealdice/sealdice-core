package dice

func RegisterBuiltinExtTemple(dice *Dice) {
	theExt := &ExtInfo{
		Name:       "name", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "一行字简介",
		Author:     "",
		AutoActive: true, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"command": &CmdItemInfo{
				Name: "command",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					// do something
					return CmdExecuteResult{Matched: true, Solved: false}
				},
			},
		},
	}

	dice.RegisterExtension(theExt)
}
