package dice

func registerBuiltinExtTemple(self *Dice) {
	self.ExtList = append(self.ExtList, &ExtInfo{
		Name:       "name", // 扩展的名称，需要用于指令中，写简短点
		Version:    "0.0.1",
		Brief:      "一行字简介",
		AutoActive: true, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func(i *ExtInfo) string {
			text := "> " + i.Brief + "\n" + "提供命令:\n"
			for _, i := range i.CmdMap {
				brief := i.Brief
				if brief != "" {
					brief = " // " + brief
				}
				text += "." + i.Name + brief + "\n"
			}
			return text
		},
		CmdMap: CmdMapCls{
			"command": &CmdItemInfo{
				Name: "command",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					if ctx.IsCurGroupBotOn {
						//p := getPlayerInfoBySender(session, msg)
					}
					return CmdExecuteResult{Success: true}
				},
			},
		},
	})
}
