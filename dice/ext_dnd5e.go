package dice

import (
	"fmt"
	"regexp"
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
				Name: "dnd",
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
						ReplyToSender(ctx, msg, fmt.Sprintf("<%s>的DnD5e人物作成:\n%s", ctx.Player.Name, info))
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					return CmdExecuteResult{Matched: true, Solved: false}
				},
			},
			"ri": &CmdItemInfo{
				Name: "ri",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					text := cmdArgs.CleanArgs
					mctx := setupMCtx(ctx, cmdArgs, 0)

					readOne := func() (int, string, int64, string) {
						text = strings.TrimSpace(text)
						var name string
						var val int64
						var detail string

						// 遇到加值
						if strings.HasPrefix(text, "+") {
							// 加值情况1，D20+
							r, _detail, err := ctx.Dice.ExprEvalBase("D20"+text, mctx, RollExtraFlags{})
							if err != nil {
								// 情况1，加值输入错误
								return 1, name, val, detail
							}
							detail = _detail
							val = r.Value.(int64)
							text = r.restInput
						} else if strings.HasPrefix(text, "=") {
							// 加值情况1，=表达式
							r, _, err := ctx.Dice.ExprEvalBase(text[1:], mctx, RollExtraFlags{})
							if err != nil {
								// 情况1，加值输入错误
								return 1, name, val, detail
							}
							val = r.Value.(int64)
							text = r.restInput
						} else {
							// 加值情况3，数字
							reNum := regexp.MustCompile(`^(\d+)`)
							m := reNum.FindStringSubmatch(text)
							if len(m) > 0 {
								val, _ = strconv.ParseInt(m[0], 10, 64)
								text = text[len(m[0]):]
							}
						}

						// 清理读取了第一项文本之后的空格
						text = strings.TrimSpace(text)

						//|| strings.HasPrefix(text, "，")
						if strings.HasPrefix(text, ",") || text == "" {
							if strings.HasPrefix(text, ",") {
								// 句末有,的话，吃掉
								text = text[1:]
							}
							// 情况1，名字是自己
							name = mctx.Player.Name
							// 情况2，名字是自己，没有加值
							if val == 0 {
								val = DiceRoll64(20)
							}
							return 0, name, val, detail
						}

						// 情况3: 是名字
						reName := regexp.MustCompile(`^([^\s\d,，][^\s,，]*)\s*,?`)
						m := reName.FindStringSubmatch(text)
						if len(m) > 0 {
							name = m[1]
							text = text[len(m[0]):]
						} else {
							// 不知道是啥，报错
							return 2, name, val, detail
						}

						return 0, name, val, detail
					}

					solved := true
					tryOnce := true
					var items []struct {
						name   string
						val    int64
						detail string
					}

					for tryOnce || text != "" {
						code, name, val, detail := readOne()
						items = append(items, struct {
							name   string
							val    int64
							detail string
						}{name, val, detail})

						if code != 0 {
							solved = false
							break
						}
						tryOnce = false
					}

					if solved {
						riMap := dndGetRiMapList(ctx)
						textOut := "先攻点数设置: \n"

						for order, i := range items {
							var detail string
							riMap[i.name] = i.val
							if i.detail != "" {
								detail = i.detail + "="
							}
							textOut += fmt.Sprintf("%2d. %s: %s%d\n", order+1, i.name, detail, i.val)
						}

						ReplyToSender(ctx, msg, textOut)
					} else {
						ReplyToSender(ctx, msg, "ri 格式不正确!")
					}

					return CmdExecuteResult{Matched: true, Solved: solved}
				},
			},
			"init": &CmdItemInfo{
				Name: "init",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					if ctx.IsCurGroupBotOn || ctx.IsPrivate {
						n, exists := cmdArgs.GetArgN(1)
						if exists {
							fmt.Println(n)
						} else {

						}

						return CmdExecuteResult{Matched: true, Solved: true}
					}
					return CmdExecuteResult{Matched: true, Solved: false}
				},
			},
		},
	}

	self.RegisterExtension(theExt)
}

func dndGetRiMapList(ctx *MsgContext) map[string]int64 {
	ctx.LoadGroupVars()
	mapName := "riMapList"
	_, exists := ctx.Group.ValueMap[mapName]
	if !exists {
		ctx.Group.ValueMap[mapName] = &VMValue{-1, map[string]int64{}}
	}
	if ctx.Group.ValueMap[mapName].Value == nil {
		ctx.Group.ValueMap[mapName] = &VMValue{-1, map[string]int64{}}
	}
	riList, ok := ctx.Group.ValueMap[mapName].Value.(map[string]int64)
	if !ok {
		ctx.Group.ValueMap[mapName].Value = map[string]int64{}
		return ctx.Group.ValueMap[mapName].Value.(map[string]int64)
	}
	return riList
}
