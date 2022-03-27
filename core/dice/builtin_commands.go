package dice

import (
	"encoding/json"
	"fmt"
	"github.com/juliangruber/go-intersect"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func FormatDiceIdQQ(diceQQ int64) string {
	return fmt.Sprintf("QQ:%s", strconv.FormatInt(diceQQ, 10))
}

func SetBotOnAtGroup(ctx *MsgContext, msg *Message) {
	session := ctx.Session
	group := session.ServiceAt[msg.GroupId]
	if group != nil {
		group.Active = true
	} else {
		extLst := []*ExtInfo{}
		for _, i := range session.Parent.ExtList {
			if i.AutoActive {
				extLst = append(extLst, i)
			}
		}
		session.ServiceAt[msg.GroupId] = &ServiceAtItem{
			Active:           true,
			ActivatedExtList: extLst,
			Players:          map[int64]*PlayerInfo{},
			GroupId:          msg.GroupId,
			ValueMap:         map[string]*VMValue{},
			DiceIds:          map[string]bool{},
		}
		group = session.ServiceAt[msg.GroupId]
	}

	if group.DiceIds == nil {
		group.DiceIds = map[string]bool{}
	}
	if group.BotList == nil {
		group.BotList = map[string]bool{}
	}

	group.DiceIds[FormatDiceIdQQ(ctx.conn.UserId)] = true
}

/** 这几条指令不能移除 */
func (d *Dice) registerCoreCommands() {
	HelpForFind := ".find <关键字> // 查找文档。关键字可以多个，用空格分割\n" +
		".find <数字ID> // 显示该ID的词条\n" +
		".find --rand // 显示随机词条"
	cmdSearch := &CmdItemInfo{
		Name:     "find",
		Help:     HelpForFind,
		LongHelp: "查询指令，通常使用全文搜索(x86版)或快速查询(arm, 移动版)\n" + HelpForFind + "\n注: 默认搭载的《怪物之锤查询》文档来自蜜瓜包、October整理\n默认搭载的DND系列文档来自DicePP项目",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				var id string
				if cmdArgs.GetKwarg("rand") != nil || cmdArgs.GetKwarg("随机") != nil {
					_id := rand.Uint64()%d.Parent.Help.CurId + 1
					id = strconv.FormatUint(_id, 10)
				}

				if id == "" {
					if _id, exists := cmdArgs.GetArgN(1); exists {
						_, err2 := strconv.ParseInt(_id, 10, 64)
						if err2 == nil {
							id = _id
						}
					}
				}

				if id != "" {
					text, exists := d.Parent.Help.TextMap[id]
					if exists {
						ReplyToSender(ctx, msg, fmt.Sprintf("词条: %s:%s\n%s", text.PackageName, text.Title, text.Content))
					} else {
						ReplyToSender(ctx, msg, "未发现对应ID的词条")
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if _, exists := cmdArgs.GetArgN(1); exists {
					search, err := d.Parent.Help.Search(ctx, cmdArgs.CleanArgs, false)
					if err == nil {
						if len(search.Hits) > 0 {
							var bestResult string
							hasSecond := len(search.Hits) >= 2
							best := d.Parent.Help.TextMap[search.Hits[0].ID]
							others := ""

							for _, i := range search.Hits {
								t := d.Parent.Help.TextMap[i.ID]
								others += fmt.Sprintf("[序号%s]【%s:%s】 匹配度 %.2f\n", i.ID, t.PackageName, t.Title, i.Score)
							}

							var showBest bool
							if hasSecond {
								offset := d.Parent.Help.GetShowBestOffset()
								val := search.Hits[1].Score - search.Hits[0].Score
								if val < 0 {
									val = -val
								}
								if val > float64(offset) {
									showBest = true
								}
							} else {
								showBest = true
							}

							if showBest {
								bestResult = fmt.Sprintf("最优先结果:\n词条: %s:%s\n%s\n\n", best.PackageName, best.Title, best.Content)
							}

							suffix := d.Parent.Help.GetSuffixText()
							ReplyToSender(ctx, msg, fmt.Sprintf("%s全部结果:\n%s\n%s", bestResult, others, suffix))
						} else {
							ReplyToSender(ctx, msg, "未找到搜索结果")
						}
					} else {
						ReplyToSender(ctx, msg, "搜索故障: "+err.Error())
					}
				} else {
					ReplyToSender(ctx, msg, "想要问什么呢？\n.查询 <数字ID> // 显示该ID的词条\n.查询 <任意文本> // 查询关联内容\n.查询 --rand // 随机词条")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["查询"] = cmdSearch
	d.CmdMap["find"] = cmdSearch

	HelpForHelp := ".help // 查看本帮助\n" +
		".help 指令 // 查看某指令信息\n" +
		".help 扩展模块 // 查看扩展信息，如.help coc7\n" +
		".help 关键字 // 查看任意帮助，同.find"
	cmdHelp := &CmdItemInfo{
		Name:     "help",
		Help:     HelpForHelp,
		LongHelp: "帮助指令，用于查看指令帮助和helpdoc中录入的信息\n" + HelpForHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if _, exists := cmdArgs.GetArgN(1); exists {
					search, err := d.Parent.Help.Search(ctx, cmdArgs.CleanArgs, true)
					if err == nil {
						if len(search.Hits) > 0 {
							a := d.Parent.Help.TextMap[search.Hits[0].ID]
							ReplyToSender(ctx, msg, fmt.Sprintf("%s:%s\n%s", a.PackageName, a.Title, a.Content))
						} else {
							ReplyToSender(ctx, msg, "未找到搜索结果")
						}
					} else {
						ReplyToSender(ctx, msg, "搜索故障: "+err.Error())
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				text := "海豹核心 " + VERSION + "\n"
				text += "-----------------------------------------------\n"
				text += ".help 骰点" + "\n"
				text += ".help 娱乐" + "\n"
				text += ".help 扩展" + "\n"
				text += ".help 跑团" + "\n"
				text += ".help 日志" + "\n"
				text += ".help 骰主" + "\n"
				text += ".help 其他" + "\n"
				text += "官网(建设中): https://sealdice.com/" + "\n"
				text += "日志着色器: http://log.weizaima.com/" + "\n"
				text += "测试群: 524364253" + "\n"
				//text += "扩展指令请输入 .ext 和 .ext <扩展名称> 进行查看\n"
				text += "-----------------------------------------------\n"
				text += DiceFormatTmpl(ctx, "核心:骰子帮助文本_附加说明")
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["help"] = cmdHelp

	cmdBot := &CmdItemInfo{
		Name: "bot on/off/about/bye",
		Help: ".bot on/off/about/bye // 开启、关闭、查看信息、退群",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			inGroup := msg.MessageType == "group"
			AtSomebodyButNotMe := len(cmdArgs.At) > 0 && !cmdArgs.AmIBeMentioned // 喊的不是当前骰子

			if len(cmdArgs.Args) == 0 || cmdArgs.IsArgEqual(1, "about") {
				if AtSomebodyButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}
				count := 0
				serveCount := 0
				for _, i := range d.ImSession.ServiceAt {
					if !i.NotInGroup && i.GroupId != 0 {
						if i.Active {
							count += 1
						}
						serveCount += 1
					}
				}
				lastSavedTimeText := "从未"
				if d.LastSavedTime != nil {
					lastSavedTimeText = d.LastSavedTime.Format("2006-01-02 15:04:05") + " UTC"
				}
				text := fmt.Sprintf("SealDice %s\n供职于%d个群，其中%d个处于开启状态\n上次自动保存时间: %s", VERSION, serveCount, count, lastSavedTimeText)

				if inGroup {
					isActive := ctx.Group != nil && ctx.Group.Active
					activeText := "开启"
					if !isActive {
						activeText = "关闭"
					}
					text += "\n群内工作状态: " + activeText
					ReplyGroup(ctx, msg, text)
				} else {
					ReplyPerson(ctx, msg, text)
				}
			} else {
				if inGroup && !AtSomebodyButNotMe {
					if len(cmdArgs.Args) >= 1 {
						if cmdArgs.IsArgEqual(1, "on") {
							SetBotOnAtGroup(ctx, msg)
							ctx.Group = ctx.Session.ServiceAt[msg.GroupId]
							ctx.IsCurGroupBotOn = true
							// "SealDice 已启用(开发中) " + VERSION
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子开启"))
							return CmdExecuteResult{Matched: true, Solved: true}
						} else if cmdArgs.IsArgEqual(1, "off") {
							//if len(ctx.Group.ActivatedExtList) == 0 {
							//	delete(ctx.Session.ServiceAt, msg.GroupId)
							//} else {
							ctx.Group.Active = false
							//}
							// 停止服务
							ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子关闭"))
							return CmdExecuteResult{Matched: false, Solved: true}
						} else if cmdArgs.IsArgEqual(1, "bye", "exit", "quit") {
							// 收到指令，5s后将退出当前群组
							ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子退群预告"))
							d.Logger.Infof("指令退群: 于群组(%d)中告别，操作者:(%d)", msg.GroupId, msg.UserId)
							ctx.Group.Active = false
							time.Sleep(6 * time.Second)
							QuitGroup(ctx, msg.GroupId)
							return CmdExecuteResult{Matched: true, Solved: true}
						} else if cmdArgs.IsArgEqual(1, "save") {
							d.Save(false)
							// 数据已保存
							ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子保存设置"))
							return CmdExecuteResult{Matched: true, Solved: true}
						}
					}
				}
			}

			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["bot"] = cmdBot

	readIdList := func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) []string {
		uidLst := []string{}
		for _, i := range cmdArgs.At {
			uidLst = append(uidLst, i.UID)
		}

		if len(cmdArgs.Args) > 1 {
			for _, i := range cmdArgs.Args[1:] {
				if i == "me" {
					uid := FormatDiceIdQQ(ctx.Player.UserId)
					uidLst = append(uidLst, uid)
					continue
				}
				qq, err := strconv.ParseInt(i, 10, 64)
				if err == nil {
					uid := FormatDiceIdQQ(qq)
					uidLst = append(uidLst, uid)
				}
			}
		}
		return uidLst
	}

	botListHelp := ".botlist add @A @B @C // 标记群内其他机器人，以免发生误触和无限对话\n" +
		".botlist add @A @B --s  // 同上，不过骰子不会做出回复\n" +
		".botlist del @A @B @C // 去除机器人标记\n" +
		".botlist list // 查看当前列表"
	cmdBotList := &CmdItemInfo{
		Name:     "botlist",
		Help:     botListHelp,
		LongHelp: "机器人列表:\n" + botListHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn {
				subCmd, _ := cmdArgs.GetArgN(1)
				switch subCmd {
				case "add":
					existsCount := 0
					newCount := 0
					for _, uid := range readIdList(ctx, msg, cmdArgs) {
						if ctx.Group.BotList[uid] {
							existsCount += 1
						} else {
							ctx.Group.BotList[uid] = true
							newCount += 1
						}
					}

					if cmdArgs.GetKwarg("s") == nil || cmdArgs.GetKwarg("slience") == nil {
						ReplyToSender(ctx, msg, fmt.Sprintf("新增标记了%d个帐号，这些账号将被视为机器人。\n因此别人@他们时，海豹将不会回复。\n他们的指令也会被海豹忽略，避免发生循环回复事故", newCount))
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				case "del", "rm":
					existsCount := 0
					for _, uid := range readIdList(ctx, msg, cmdArgs) {
						if ctx.Group.BotList[uid] {
							existsCount += 1
							delete(ctx.Group.BotList, uid)
						}
					}
					ReplyToSender(ctx, msg, fmt.Sprintf("删除标记了%d个帐号，这些账号将不再被视为机器人。\n海豹将继续回应他们的命令", existsCount))
					return CmdExecuteResult{Matched: true, Solved: true}
				case "list":
					text := ""
					for i, _ := range ctx.Group.BotList {
						// uid := FormatDiceIdQQ(i)
						text += "- " + i + "\n"
					}
					if text == "" {
						text = "无"
					}
					ReplyToSender(ctx, msg, fmt.Sprintf("群内其他机器人列表:\n%s", text))
					return CmdExecuteResult{Matched: true, Solved: true}
				default:
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}
			} else if ctx.IsPrivate {
				ReplyToSender(ctx, msg, fmt.Sprintf("私聊中不支持这个指令"))
			}

			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["botlist"] = cmdBotList

	masterListHelp := `.master add me // 将自己标记为骰主
.master add @A @B // 将别人标记为骰主
.master del @A @B @C // 去除骰主标记
.master unlock <密码> // 清空骰主列表，并使自己成为骰主
.master list // 查看当前骰主列表`
	cmdMaster := &CmdItemInfo{
		Name:     "master",
		Help:     masterListHelp,
		LongHelp: "骰主指令:\n" + masterListHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || msg.MessageType == "private" {
				if ctx.IsCurGroupBotOn && cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				pRequired := 0
				if len(ctx.Dice.DiceMasters) >= 1 {
					pRequired = 100
				}
				if ctx.PrivilegeLevel < pRequired {
					subCmd, _ := cmdArgs.GetArgN(1)
					if subCmd == "unlock" {
						// 特殊解锁指令
					}

					ReplyToSender(ctx, msg, fmt.Sprintf("你不具备Master权限"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				subCmd, _ := cmdArgs.GetArgN(1)
				switch subCmd {
				case "add":
					newCount := 0
					for _, uid := range readIdList(ctx, msg, cmdArgs) {
						ctx.Dice.MasterAdd(uid)
						newCount += 1
					}
					ctx.Dice.Save(false)
					ReplyToSender(ctx, msg, fmt.Sprintf("海豹将新增%d位master", newCount))
					return CmdExecuteResult{Matched: true, Solved: true}
				case "del", "rm":
					existsCount := 0
					for _, uid := range readIdList(ctx, msg, cmdArgs) {
						if ctx.Dice.MasterRemove(uid) {
							existsCount += 1
						}
					}
					ctx.Dice.Save(false)
					ReplyToSender(ctx, msg, fmt.Sprintf("海豹移除了%d名master", existsCount))
					return CmdExecuteResult{Matched: true, Solved: true}
				case "list":
					text := ""
					for _, i := range ctx.Dice.DiceMasters {
						// uid := FormatDiceIdQQ(i)
						text += "- " + i + "\n"
					}
					if text == "" {
						text = "无"
					}
					ReplyToSender(ctx, msg, fmt.Sprintf("Master列表:\n%s", text))
					return CmdExecuteResult{Matched: true, Solved: true}
				default:
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["master"] = cmdMaster

	cmdSend := &CmdItemInfo{
		Name:     "send",
		Help:     ".send // 向骰主留言",
		LongHelp: "留言指令:\n.send // 向骰主留言",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || msg.MessageType == "private" {
				if ctx.IsCurGroupBotOn && cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if _, exists := cmdArgs.GetArgN(1); exists {
					for _, i := range ctx.Dice.DiceMasters {
						lst := strings.SplitN(i, ":", 2)
						if lst[0] == "QQ" {
							val, _ := strconv.ParseInt(lst[1], 10, 64)
							text := ""

							if ctx.IsCurGroupBotOn {
								text += fmt.Sprintf("一条来自群组<%s>(%d)，作者<%s>(%d)的留言:\n", ctx.Group.GroupName, ctx.Group.GroupId, ctx.Player.Name, ctx.Player.UserId)
							} else {
								text += fmt.Sprintf("一条来自私聊，作者<%s>(%d)的留言:\n", ctx.Player.Name, ctx.Player.UserId)
							}

							text += cmdArgs.CleanArgs
							replyPersonRaw(ctx, val, text, "")
						}
					}
					ReplyToSender(ctx, msg, "您的留言已被记录，另外注意不要滥用此功能，祝您生活愉快，再会。")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["send"] = cmdSend

	cmdRoll := &CmdItemInfo{
		Name: "roll",
		Help: ".r <表达式> <原因> // 骰点指令\n.rh <表达式> <原因> // 暗骰",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || msg.MessageType == "private" {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				var text string
				var diceResult int64
				var diceResultExists bool
				var detail string
				disableLoadVarname := !(cmdArgs.Command == "rx" || cmdArgs.Command == "rhx")

				if ctx.Dice.CommandCompatibleMode {
					if (cmdArgs.Command == "rd" || cmdArgs.Command == "rhd") && len(cmdArgs.Args) >= 1 {
						if m, _ := regexp.MatchString(`^\d`, cmdArgs.CleanArgs); m {
							cmdArgs.CleanArgs = "d" + cmdArgs.CleanArgs
						}
					}
				}

				var r *VmResult
				rollOne := func() *CmdExecuteResult {
					forWhat := ""
					if len(cmdArgs.Args) >= 1 {
						var err error
						r, detail, err = ctx.Dice.ExprEvalBase(cmdArgs.CleanArgs, ctx, RollExtraFlags{
							DisableLoadVarname: disableLoadVarname,
							DefaultDiceSideNum: getDefaultDicePoints(ctx),
						})

						if r != nil && r.TypeId == 0 {
							diceResult = r.Value.(int64)
							diceResultExists = true
							//return errors.New("错误的类型")
						}

						if err == nil {
							forWhat = r.restInput
						} else {
							errs := string(err.Error())
							if strings.HasPrefix(errs, "E1:") {
								ReplyToSender(ctx, msg, errs)
								//ReplyGroup(ctx, msg.GroupId, errs)
								return &CmdExecuteResult{Matched: true, Solved: false}
							}
							forWhat = cmdArgs.CleanArgs
						}
					}

					VarSetValueStr(ctx, "$t原因", forWhat)
					if forWhat != "" {
						forWhatText := DiceFormatTmpl(ctx, "核心:骰点_原因")
						VarSetValueStr(ctx, "$t原因句子", forWhatText)
					} else {
						VarSetValueStr(ctx, "$t原因句子", "")
					}

					if diceResultExists {
						detailWrap := ""
						if detail != "" {
							detailWrap = "=" + detail
						}

						VarSetValueStr(ctx, "$t表达式文本", r.Matched)
						VarSetValueStr(ctx, "$t计算过程", detailWrap)
						VarSetValueInt64(ctx, "$t计算结果", diceResult)
						//text = fmt.Sprintf("%s<%s>掷出了 %s%s=%d", prefix, ctx.Player.Name, cmdArgs.Args[0], detailWrap, diceResult)
					} else {
						dicePoints := getDefaultDicePoints(ctx)
						val := DiceRoll64(int64(dicePoints))

						VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("D%d", dicePoints))
						VarSetValueStr(ctx, "$t计算过程", "")
						VarSetValueInt64(ctx, "$t计算结果", val)
						//text = fmt.Sprintf("%s<%s>掷出了 D%d=%d", prefix, ctx.Player.Name, dicePoints, val)
					}
					return nil
				}

				if cmdArgs.SpecialExecuteTimes > 1 {
					VarSetValueInt64(ctx, "$t次数", int64(cmdArgs.SpecialExecuteTimes))
					if cmdArgs.SpecialExecuteTimes > 12 {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:检定_轮数过多警告"))
						return CmdExecuteResult{Matched: true, Solved: false}
					}
					texts := []string{}
					for i := 0; i < cmdArgs.SpecialExecuteTimes; i++ {
						ret := rollOne()
						if ret != nil {
							return *ret
						}
						texts = append(texts, DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
					}
					VarSetValueStr(ctx, "$t结果文本", strings.Join(texts, `\n`))
					text = DiceFormatTmpl(ctx, "核心:骰点_多轮")
				} else {
					ret := rollOne()
					if ret != nil {
						return *ret
					}
					VarSetValueStr(ctx, "$t结果文本", DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
					text = DiceFormatTmpl(ctx, "核心:骰点")
				}

				if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
					asm := r.Parser.GetAsmText()
					text += "\n" + asm
				}

				if cmdArgs.Command == "rh" || cmdArgs.Command == "rhd" {
					if ctx.Group != nil {
						ctx.CommandHideFlag = ctx.Group.GroupId
						//prefix := fmt.Sprintf("来自群<%s>(%d)的暗骰，", ctx.Group.GroupName, msg.GroupId)
						prefix := DiceFormatTmpl(ctx, "核心:暗骰_私聊_前缀")
						ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:暗骰_群内"))
						ReplyPerson(ctx, msg, prefix+text)
					} else {
						ReplyToSender(ctx, msg, text)
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				} else {
					ReplyToSender(ctx, msg, text)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["r"] = cmdRoll
	d.CmdMap["rd"] = cmdRoll
	d.CmdMap["roll"] = cmdRoll
	d.CmdMap["rh"] = cmdRoll
	d.CmdMap["rhd"] = cmdRoll
	d.CmdMap["rx"] = cmdRoll
	d.CmdMap["rhx"] = cmdRoll

	cmdExt := &CmdItemInfo{
		Name: "ext",
		Help: ".ext // 查看扩展列表",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
			}
			if ctx.IsCurGroupBotOn {
				showList := func() {
					text := "检测到以下扩展：\n"
					for index, i := range ctx.Dice.ExtList {
						state := "关"
						for _, j := range ctx.Group.ActivatedExtList {
							if i.Name == j.Name {
								state = "开"
								break
							}
						}
						author := i.Author
						if author == "" {
							author = "<未注明>"
						}
						text += fmt.Sprintf("%d. [%s]%s - 版本:%s 作者:%s\n", index+1, state, i.Name, i.Version, author)
					}
					text += "使用命令: .ext <扩展名> on/off 可以在当前群开启或关闭某扩展。\n"
					text += "命令: .ext <扩展名> 可以查看扩展介绍及帮助"
					ReplyGroup(ctx, msg, text)
				}

				if len(cmdArgs.Args) == 0 {
					showList()
				}

				if len(cmdArgs.Args) >= 1 {
					if cmdArgs.IsArgEqual(1, "list") {
						showList()
					} else if cmdArgs.IsArgEqual(2, "on") {
						extName := cmdArgs.Args[0]
						checkConflict := func(ext *ExtInfo) []string {
							actived := []string{}
							for _, i := range ctx.Group.ActivatedExtList {
								actived = append(actived, i.Name)
							}

							if ext.ConflictWith != nil {
								ret := []string{}
								for _, i := range intersect.Simple(actived, ext.ConflictWith) {
									ret = append(ret, i.(string))
								}
								return ret
							}
							return []string{}
						}

						for _, i := range d.ExtList {
							if i.Name == extName {
								text := fmt.Sprintf("打开扩展 %s", extName)

								conflicts := checkConflict(i)
								if len(conflicts) > 0 {
									text += "\n检测到可能冲突的扩展，建议关闭: " + strings.Join(conflicts, ",")
									text += "\n若扩展中存在同名指令，则越晚开启的扩展，优先级越高。"
								}

								ctx.Group.ExtActive(i)
								ReplyGroup(ctx, msg, text)
								break
							}
						}
					} else if cmdArgs.IsArgEqual(2, "off") {
						extName := cmdArgs.Args[0]
						ei := ctx.Group.ExtInactive(extName)
						if ei != nil {
							ReplyGroup(ctx, msg, fmt.Sprintf("关闭扩展 %s", extName))
						} else {
							ReplyGroup(ctx, msg, fmt.Sprintf("未找到此扩展，可能已经关闭: %s", extName))
						}
						return CmdExecuteResult{Matched: true, Solved: true}
					} else {
						extName := cmdArgs.Args[0]
						for _, i := range d.ExtList {
							if i.Name == extName {
								text := fmt.Sprintf("> [%s] 版本%s 作者%s\n", i.Name, i.Version, i.Author)
								ReplyToSender(ctx, msg, text+i.GetDescText(i))
								return CmdExecuteResult{Matched: true, Solved: true}
							}
						}
					}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["ext"] = cmdExt

	cmdNN := &CmdItemInfo{
		Name: "nn <角色名>",
		Help: ".nn <角色名> // 跟角色名则改为指定角色名，不带则重置角色名",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if msg.MessageType == "group" {
				if ctx.IsCurGroupBotOn {
					if len(cmdArgs.Args) == 0 {
						p := ctx.Player
						p.Name = msg.Sender.Nickname
						VarSetValue(ctx, "$t玩家", &VMValue{VMTypeString, fmt.Sprintf("<%s>", ctx.Player.Name)})

						ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_重置"))
						//replyGroup(ctx, msg.GroupId, fmt.Sprintf("%s(%d) 的昵称已重置为<%s>", msg.Sender.Nickname, msg.Sender.UserId, p.Name))
					}
					if len(cmdArgs.Args) >= 1 {
						p := ctx.Player
						p.Name = cmdArgs.Args[0]
						VarSetValue(ctx, "$t玩家", &VMValue{VMTypeString, fmt.Sprintf("<%s>", ctx.Player.Name)})

						ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_改名"))
						//replyGroup(ctx, msg.GroupId, fmt.Sprintf("%s(%d) 的昵称被设定为<%s>", msg.Sender.Nickname, msg.Sender.UserId, p.Name))
					}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["nn"] = cmdNN

	helpSet := ".set <面数> // 设置默认骰子面数"
	cmdSet := &CmdItemInfo{
		Name:     "set",
		Help:     helpSet,
		LongHelp: "设定骰子面数:\n" + helpSet,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				p := ctx.Player
				isSetGroup := true
				my := cmdArgs.GetKwarg("my")
				if my != nil {
					isSetGroup = false
				}

				arg1, exists := cmdArgs.GetArgN(1)
				if exists {
					num, err := strconv.ParseInt(cmdArgs.Args[0], 10, 64)
					if num < 0 {
						num = 0
					}
					if err == nil {
						if isSetGroup {
							ctx.Group.DiceSideNum = num
							VarSetValue(ctx, "$t群组骰子面数", &VMValue{VMTypeInt64, ctx.Group.DiceSideNum})
							VarSetValue(ctx, "$t当前骰子面数", &VMValue{VMTypeInt64, getDefaultDicePoints(ctx)})
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认群组骰子面数"))
						} else {
							p.DiceSideNum = int(num)
							VarSetValue(ctx, "$t个人骰子面数", &VMValue{VMTypeInt64, int64(ctx.Player.DiceSideNum)})
							VarSetValue(ctx, "$t当前骰子面数", &VMValue{VMTypeInt64, getDefaultDicePoints(ctx)})
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数"))
							//replyGroup(ctx, msg.GroupId, fmt.Sprintf("设定默认骰子面数为 %d", num))
						}
					} else {
						switch arg1 {
						case "clr":
							if isSetGroup {
								ctx.Group.DiceSideNum = 0
							} else {
								p.DiceSideNum = 0
								//replyGroup(ctx, msg.GroupId, fmt.Sprintf("重设默认骰子面数为初始"))
							}
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数_重置"))
						case "help":
							return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
						default:
							//replyGroup(ctx, msg.GroupId, fmt.Sprintf("设定默认骰子面数: 格式错误"))
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数_错误"))
						}
					}
				} else {
					ReplyToSender(ctx, msg, DiceFormat(ctx, `个人骰子面数: {$t个人骰子面数}\n`+
						`群组骰子面数: {$t群组骰子面数}\n当前骰子面数: {$t当前骰子面数}`))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["set"] = cmdSet

	textHelp := ".text <文本模板> // 文本指令，例: .text 看看手气: {1d16}"
	cmdText := &CmdItemInfo{
		Name:     "text",
		Help:     textHelp,
		LongHelp: "文本模板指令:\n" + textHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				_, exists := cmdArgs.GetArgN(1)
				if exists {
					r, _, err := d.ExprTextBase(cmdArgs.RawArgs, ctx)

					if err == nil && (r.TypeId == VMTypeString || r.TypeId == VMTypeNone) {
						text := r.Value.(string)

						if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
							asm := r.Parser.GetAsmText()
							text += "\n" + asm
						}

						ReplyToSender(ctx, msg, text)
					} else {
						ReplyToSender(ctx, msg, "格式错误")
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["text"] = cmdText

	cmdChar := &CmdItemInfo{
		Name: "ch",
		//Help: ".ch save <角色名> // 保存角色，角色名省略则为当前昵称\n.ch load <角色名> // 加载角色\n.ch list // 列出当前角色",
		Help: ".ch list/save/load/del // 角色管理",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn {
				getNickname := func() string {
					name, _ := cmdArgs.GetArgN(2)
					if name == "" {
						name = ctx.Player.Name
					}
					return name
				}

				if cmdArgs.IsArgEqual(1, "list") {
					vars := ctx.LoadPlayerVars()
					characters := []string{}
					for k, _ := range vars.ValueMap {
						if strings.HasPrefix(k, "$ch:") {
							characters = append(characters, k[4:])
						}
					}
					if len(characters) == 0 {
						ReplyToSender(ctx, msg, fmt.Sprintf("<%s>当前还没有角色列表", ctx.Player.Name))
					} else {
						ReplyToSender(ctx, msg, fmt.Sprintf("<%s>的角色列表为:\n%s", ctx.Player.Name, strings.Join(characters, "\n")))
					}
				} else if cmdArgs.IsArgEqual(1, "load") {
					name := getNickname()
					vars := ctx.LoadPlayerVars()
					data, exists := vars.ValueMap["$ch:"+name]

					if exists {
						ctx.Player.ValueMap = make(map[string]*VMValue)
						err := JsonValueMapUnmarshal([]byte(data.Value.(string)), &ctx.Player.ValueMap)
						if err == nil {
							ctx.Player.Name = name
							VarSetValue(ctx, "$t玩家", &VMValue{VMTypeString, fmt.Sprintf("<%s>", ctx.Player.Name)})

							//replyToSender(ctx, msg, fmt.Sprintf("角色<%s>加载成功，欢迎回来。", Name))
							ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_加载成功"))
						} else {
							//replyToSender(ctx, msg, "无法加载角色：序列化失败")
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_序列化失败"))
						}
					} else {
						//replyToSender(ctx, msg, "无法加载角色：你所指定的角色不存在")
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
					}
				} else if cmdArgs.IsArgEqual(1, "Save") {
					name := getNickname()
					vars := ctx.LoadPlayerVars()
					v, err := json.Marshal(ctx.Player.ValueMap)

					if err == nil {
						vars.ValueMap["$ch:"+name] = &VMValue{
							VMTypeString,
							string(v),
						}
						VarSetValue(ctx, "$t新角色名", &VMValue{VMTypeString, fmt.Sprintf("<%s>", name)})
						//replyToSender(ctx, msg, fmt.Sprintf("角色<%s>储存成功", Name))
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_储存成功"))
					} else {
						//replyToSender(ctx, msg, "无法储存角色：序列化失败")
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_序列化失败"))
					}
				} else if cmdArgs.IsArgEqual(1, "del", "rm") {
					name := getNickname()
					vars := ctx.LoadPlayerVars()

					VarSetValue(ctx, "$t新角色名", &VMValue{VMTypeString, fmt.Sprintf("<%s>", name)})
					_, exists := vars.ValueMap["$ch:"+name]
					if exists {
						delete(vars.ValueMap, "$ch:"+name)

						text := DiceFormatTmpl(ctx, "核心:角色管理_删除成功")
						if name == ctx.Player.Name {
							VarSetValue(ctx, "$t新角色名", &VMValue{VMTypeString, fmt.Sprintf("<%s>", msg.Sender.Nickname)})
							text += "\n" + DiceFormatTmpl(ctx, "核心:角色管理_删除成功_当前卡")
						}

						ReplyToSender(ctx, msg, text)
					} else {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
					}
				} else {
					help := "角色指令\n"
					help += ".ch save <角色名> // 保存角色，角色名省略则为当前昵称\n.ch load <角色名> // 加载角色，角色名省略则为当前昵称\n.ch list // 列出当前角色\n.ch del <角色名> // 删除角色"
					ReplyToSender(ctx, msg, help)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}
	d.CmdMap["角色"] = cmdChar
	d.CmdMap["ch"] = cmdChar
	d.CmdMap["char"] = cmdChar
	d.CmdMap["character"] = cmdChar
	d.CmdMap["pc"] = cmdChar
}

func getDefaultDicePoints(ctx *MsgContext) int64 {
	diceSides := int64(ctx.Player.DiceSideNum)
	if diceSides == 0 {
		diceSides = ctx.Group.DiceSideNum
	}
	if diceSides <= 0 {
		diceSides = 100
	}
	return diceSides
}
