package main

func (self *Dice) registerBuiltinExtTemple() {
	self.extList = append(self.extList, &ExtInfo{
		Name:       "name", // 扩展的名称，需要用于指令中，写简短点
		version:    "0.0.1",
		Brief: "一行字简介",
		autoActive: true, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func (i *ExtInfo) string {
			text := "> " + i.Brief + "\n" + "提供命令:\n"
			for _, i := range i.cmdMap {
				brief := i.Brief
				if brief != "" {
					brief = " // " + brief
				}
				text += "." + i.name + brief + "\n"
			}
			return text
		},
		cmdMap: CmdMapCls{
			"command": &CmdItemInfo{
				name: "command",
				solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if ctx.isCurGroupBotOn {
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
