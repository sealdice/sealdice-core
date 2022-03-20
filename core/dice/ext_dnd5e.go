package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type RIListItem struct {
	name string
	val  int64
}

type ByRIListValue []*RIListItem

func (lst ByRIListValue) Len() int {
	return len(lst)
}
func (s ByRIListValue) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByRIListValue) Less(i, j int) bool {
	return s[i].val > s[j].val
}

func RegisterBuiltinExtDnd5e(self *Dice) {
	theExt := &ExtInfo{
		Name:       "dnd5e", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "正在努力完成的DND模块",
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
				Help: `.ri <先攻值> <角色名> // 角色名省略为当前角色
.ri +2 <角色名> // 先攻值格式1，解析为D20+2
.ri =D20+3 <角色名> // 先攻值格式2，解析为D20+3
.ri 12 <角色名> // 先攻值格式3，解析为12
.ri <单项>, <单项>, ... // 允许连写，逗号分隔`,
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					if ctx.IsCurGroupBotOn || ctx.IsPrivate {
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
					}

					return CmdExecuteResult{Matched: true, Solved: false}
				},
			},
			"init": &CmdItemInfo{
				Name: "init",
				Help: ".init // 查看先攻列表\n" +
					".init del <单位1> <单位2> ... // 从先攻列表中删除\n" +
					".init set <单位名称> <先攻表达式> // 设置单位的先攻\n" +
					".init clr // 清除先攻列表\n" +
					".init help // 显示本帮助",
				Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
					if ctx.IsCurGroupBotOn || ctx.IsPrivate {
						n, _ := cmdArgs.GetArgN(1)
						switch n {
						case "", "list":
							textOut := "当前先攻列表为:\n"
							riMap := dndGetRiMapList(ctx)

							var lst ByRIListValue
							for k, v := range riMap {
								lst = append(lst, &RIListItem{k, v})
							}

							sort.Sort(lst)
							for order, i := range lst {
								textOut += fmt.Sprintf("%2d. %s: %d\n", order+1, i.name, i.val)
							}

							if len(lst) == 0 {
								textOut += "- 没有找到任何单位"
							}

							ReplyToSender(ctx, msg, textOut)
						case "del", "rm":
							names := cmdArgs.Args[1:]
							riMap := dndGetRiMapList(ctx)
							deleted := []string{}
							for _, i := range names {
								_, exists := riMap[i]
								if exists {
									deleted = append(deleted, i)
									delete(riMap, i)
								}
							}
							textOut := "以下单位从先攻列表中移除:\n"
							for order, i := range deleted {
								textOut += fmt.Sprintf("%2d. %s: %d\n", order+1, i)
							}
							if len(deleted) == 0 {
								textOut += "- 没有找到任何单位"
							}
							ReplyToSender(ctx, msg, textOut)
						case "set":
							name, exists := cmdArgs.GetArgN(2)
							_, exists2 := cmdArgs.GetArgN(3)
							if !exists || !exists2 {
								ReplyToSender(ctx, msg, "错误的格式，应为: .init set <单位名称> <先攻表达式>")
								return CmdExecuteResult{Matched: true, Solved: true}
							}

							expr := strings.Join(cmdArgs.Args[2:], "")
							r, _detail, err := ctx.Dice.ExprEvalBase(expr, ctx, RollExtraFlags{})
							if err != nil || r.TypeId != VMTypeInt64 {
								ReplyToSender(ctx, msg, "错误的格式，应为: .init set <单位名称> <先攻表达式>")
								return CmdExecuteResult{Matched: true, Solved: true}
							}

							riMap := dndGetRiMapList(ctx)
							riMap[name] = r.Value.(int64)
							var detail string
							if _detail != "" {
								detail = _detail + "="
							}
							textOut := fmt.Sprintf("已设置 %s 的先攻点为 %s%s", name, detail, r.Value.(int64))

							ReplyToSender(ctx, msg, textOut)
						case "clr", "clear":
							dndClearRiMapList(ctx)
							ReplyToSender(ctx, msg, "先攻列表已清除")
						case "help":
							return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
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
	riList := ctx.Group.ValueMap[mapName].Value
	return riList.(map[string]int64)
}

func dndClearRiMapList(ctx *MsgContext) {
	ctx.LoadGroupVars()
	mapName := "riMapList"
	ctx.Group.ValueMap[mapName] = &VMValue{-1, map[string]int64{}}
}
