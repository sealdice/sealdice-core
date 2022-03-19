package dice

import (
	"fmt"
	"strconv"
	"strings"
)

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
		CmdMap: CmdMapCls{
			"dnd": &CmdItemInfo{
				Name: ".dnd",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					if ctx.IsCurGroupBotOn || ctx.IsPrivate {
						n, _ := cmdArgs.GetArgN(1)
						val, err := strconv.ParseInt(n, 10, 64)
						if err != nil {
							// 数量不存在时，视为1次
							val = 1
						}
						if val > 10 {
							val = 10
						}
						var i int64

						var ss []string
						for i = 0; i < val; i++ {
							result, _, err := self.ExprText(`力量:{$t1=3+1d17} 体质:{$t2=3+1d17} 敏捷:{$t3=3+1d17} 智力:{$t4=3+1d17} 感知:{$t5=3+1d17} 魅力:{$t6=3+1d17} 共计:{$t1+$t2+$t3+$t4+$t5+$t6}`, ctx)
							if err != nil {
								break
							}
							result = strings.ReplaceAll(result, `\n`, "\n")
							ss = append(ss, result)
						}
						info := strings.Join(ss, "\n")
						ReplyToSender(ctx, msg, fmt.Sprintf("<%s>的DND5e人物作成:\n%s", ctx.Player.Name, info))
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					return CmdExecuteResult{Matched: true, Solved: false}
				},
			},
		},
	}

	self.RegisterExtension(theExt)
}
