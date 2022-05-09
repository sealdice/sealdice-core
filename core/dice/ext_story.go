package dice

import (
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	"html"
	"math/rand"
	"strconv"
	"strings"
)

func cmdRandomName(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, cmdsList [][]string, rulesList [][]string, defaultIndex int) CmdExecuteResult {
	names := []string{}
	chops := []string{}
	for _, i := range cmdsList {
		for _, j := range i {
			chops = append(chops, j)
		}
	}
	cmdArgs.ChopPrefixToArgsWith(chops...)

	numText, exists := cmdArgs.GetArgN(2)
	var num int64
	if exists {
		num, _ = strconv.ParseInt(numText, 10, 64)
	}
	if num == 0 {
		num = 5
	}

	var rules []string
	// 如果没有参数，采用默认
	if len(cmdArgs.Args) == 0 && defaultIndex != -1 {
		rules = rulesList[defaultIndex]
	} else {
		for index, cmds := range cmdsList {
			if cmdArgs.IsArgEqual(1, cmds...) {
				rules = rulesList[index]
				break
			}
		}
	}

	// 没匹配到，显示帮助
	if len(rules) == 0 {
		return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
	}

	// 开始抽取
	for i := int64(0); i < num; i++ {
		rule := rules[rand.Int()%len(rules)]
		names = append(names, ctx.Dice.Parent.NamesGenerator.NameGenerate(rule))
	}

	ReplyToSender(ctx, msg, fmt.Sprintf("为<%s>生成以下名字：\n%s", ctx.Player.Name, strings.Join(names, "、")))
	return CmdExecuteResult{Matched: true, Solved: true}
}

func RegisterBuiltinStory(self *Dice) {
	cmdName := &CmdItemInfo{
		Name:     "name",
		Help:     ".name cn/en/jp (<数量>)",
		LongHelp: "生成随机名字:\n.name cn/en/jp (<数量>)",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				return cmdRandomName(ctx, msg, cmdArgs, [][]string{
					{"cn", "中文", "zh", "中国"},
					{"en", "英文", "英国", "美国", "us"},
					{"jp", "日文", "日本"},
				}, [][]string{
					{
						"{中文:姓氏}{中文:男性名}",
						"{中文:姓氏}{中文:女性名}",
					},
					{"{英文:名字} {英文:姓氏} ({英文:名字中文#英文:名字.index}·{英文:姓氏中文#英文:姓氏.index})"},
					{
						"{日文:姓氏} {日文:男性名}({日文:姓氏平假名#日文:姓氏.index} {日文:男性名平假名#日文:男性名.index})",
						"{日文:姓氏} {日文:女性名}({日文:姓氏平假名#日文:姓氏.index} {日文:女性名平假名#日文:女性名.index})",
					},
				}, 0)
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	cmdNameDnd := &CmdItemInfo{
		Name:     "namednd",
		Help:     ".namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族/地精",
		LongHelp: "生成随机DND名字:\n.namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族/地精",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}

				return cmdRandomName(ctx, msg, cmdArgs, [][]string{
					{"达马拉人"},
					{"卡林珊人"},
					{"莱瑟曼人"},
					{"受国人"},
					{"精灵"},
					{"矮人"},
					{"兽人"},
					{"海族"},
					{"地精"},
				}, [][]string{
					{
						"{DND达马拉人:男性英文名} {DND达马拉人:英文姓氏} ({DND达马拉人:男性中文名#DND达马拉人:男性英文名.index}·{DND达马拉人:中文姓氏#DND达马拉人:英文姓氏.index})",
						"{DND达马拉人:女性英文名} {DND达马拉人:英文姓氏} ({DND达马拉人:女性中文名#DND达马拉人:女性英文名.index}·{DND达马拉人:中文姓氏#DND达马拉人:英文姓氏.index})",
					},
					{
						"{DND卡林珊人:Calashite_名_男} {DND卡林珊人:Calashite_姓} ({DND卡林珊人:Calashite_名_男_中文#DND卡林珊人:Calashite_名_男.index}·{DND卡林珊人:Calashite_姓_中文#DND卡林珊人:Calashite_姓.index})",
						"{DND卡林珊人:Calashite_名_女} {DND卡林珊人:Calashite_姓} ({DND卡林珊人:Calashite_名_女_中文#DND卡林珊人:Calashite_名_女.index}·{DND卡林珊人:Calashite_姓_中文#DND卡林珊人:Calashite_姓.index})",
					},
					{
						"{DND莱瑟曼人:Rashemi_名_男} {DND莱瑟曼人:Rashemi_姓} ({DND莱瑟曼人:Rashemi_名_男_中文#DND莱瑟曼人:Rashemi_名_男.index}·{DND莱瑟曼人:Rashemi_姓_中文#DND莱瑟曼人:Rashemi_姓.index})",
						"{DND莱瑟曼人:Rashemi_名_女} {DND莱瑟曼人:Rashemi_姓} ({DND莱瑟曼人:Rashemi_名_女_中文#DND莱瑟曼人:Rashemi_名_女.index}·{DND莱瑟曼人:Rashemi_姓_中文#DND莱瑟曼人:Rashemi_姓.index})",
					},
					{
						"{DND受国人:男性中文名}·{DND受国人:中文姓氏} ({DND受国人:男性英文名#DND受国人:男性中文名.index} {DND受国人:英文姓氏#DND受国人:中文姓氏.index})",
						"{DND受国人:女性中文名}·{DND受国人:中文姓氏} ({DND受国人:女性英文名#DND受国人:女性中文名.index} {DND受国人:英文姓氏#DND受国人:中文姓氏.index})",
					},
					{
						"{DND精灵:精灵_名_男} {DND精灵:精灵_姓} ({DND精灵:精灵_名_男_中文#DND精灵:精灵_名_男.index}·{DND精灵:精灵_姓_中文#DND精灵:精灵_姓.index})",
						"{DND精灵:精灵_名_女} {DND精灵:精灵_姓} ({DND精灵:精灵_名_女_中文#DND精灵:精灵_名_女.index}·{DND精灵:精灵_姓_中文#DND精灵:精灵_姓.index})",
					},
					{
						"{DND矮人:矮人_名_男} {DND矮人:矮人_姓} ({DND矮人:矮人_名_男_中文#DND矮人:矮人_名_男.index}·{DND矮人:矮人_姓_中文#DND矮人:矮人_姓.index})",
						"{DND矮人:矮人_名_女} {DND矮人:矮人_姓} ({DND矮人:矮人_名_女_中文#DND矮人:矮人_名_女.index}·{DND矮人:矮人_姓_中文#DND矮人:矮人_姓.index})",
					},
					{
						"{DND兽人:兽人_名_男} \"{DND兽人:兽人_绰号}\" (“{DND兽人:兽人_绰号_中文#DND兽人:兽人_绰号.index}”{DND兽人:兽人_名_男_中文#DND兽人:兽人_名_男.index})",
						"{DND兽人:兽人_名_女} \"{DND兽人:兽人_绰号}\" (“{DND兽人:兽人_绰号_中文#DND兽人:兽人_绰号.index}”{DND兽人:兽人_名_女_中文#DND兽人:兽人_名_女.index})",
					},
					{
						"{DND海族:海族_名_男} ({DND海族:海族_名_男_中文#DND海族:海族_名_男.index})",
					},
					{
						"{DND地精:地精_名_男} ({DND地精:地精_名_男_中文#DND地精:地精_名_男.index})",
						"{DND地精:地精_名_女} ({DND地精:地精_名_女_中文#DND地精:地精_名_女.index})",
					},
				}, -1)
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	cmdWho := &CmdItemInfo{
		Name:     "who",
		Help:     ".who a b c",
		LongHelp: "顺序重排:\n.who a b c",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}
				if cmdArgs.IsArgEqual(1, "help") {
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}

				if len(cmdArgs.Args) < 2 {
					ReplyToSender(ctx, msg, "who 的对象必须多于2项")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				items := make([]string, len(cmdArgs.Args))
				copy(items, cmdArgs.Args)
				rand.Shuffle(len(items), func(i, j int) {
					items[i], items[j] = items[j], items[i]
				})

				ReplyToSender(ctx, msg, fmt.Sprintf("打乱顺序: %s", strings.Join(items, ", ")))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpCnmods := ".modu search <关键字> (<页码>) // 搜索关键字\n" +
		".modu rec <关键字> (<页码>) // 搜索编辑推荐\n" +
		".modu author <关键字> (<页码>) // 搜索指定作者\n" +
		".modu luck (<页码>) // 查看编辑推荐\n" +
		".modu get <编号> // 查看指定详情\n" +
		".modu roll // 随机抽取\n" +
		".modu help // 显示帮助"
	cmdCnmods := &CmdItemInfo{
		Name:     "modu",
		Help:     helpCnmods,
		LongHelp: "魔都查询:\n" + helpCnmods,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: false, Solved: false}
				}
				cmdArgs.ChopPrefixToArgsWith("help", "search", "find", "rec", "luck", "get", "author", "roll")

				if cmdArgs.IsArgEqual(1, "help") {
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}

				getDetail := func(keyId string) string {
					ret := CnmodsDetail(keyId)
					if ret != nil {
						item := ret.Data.Module

						opinion := item.Opinion
						opinion = html.UnescapeString(item.Opinion) // 确实仍然存在一些 有标签的，如2488
						opinion = strip.StripTags(opinion)
						ori := "是"
						if !item.Original {
							ori = "否"
						}

						var recInfo string
						//if len(ret.Data.RecommendList) > 0 {
						//	rec := ret.Data.RecommendList[0]
						//	recInfo = fmt.Sprintf("推荐语: %s - by %s\n", rec.Content, rec.LoginUser.NickName)
						//}

						text := fmt.Sprintf("[%d]%s\n作者: %s\n背景: %s,%s\n规模: %d-%d人，%d-%d时\n原创: %v\n简介: %s\n%s链接: %s",
							item.KeyId, item.Title,
							item.Article,
							item.ModuleAge, item.OccurrencePlace,
							item.MinAmount, item.MaxAmount,
							item.MinDuration, item.MaxDuration,
							ori,
							opinion,
							recInfo,
							"https://www.cnmods.net/#/moduleDetail/index?keyId="+strconv.Itoa(item.KeyId))
						return text
					}
					return ""
				}

				_val, _ := cmdArgs.GetArgN(1)
				val := strings.ToLower(_val)
				switch val {
				case "search", "find", "rec", "luck", "author":
					keyword, _ := cmdArgs.GetArgN(2)
					page, _ := cmdArgs.GetArgN(3)
					isRec := false
					if val == "luck" {
						keyword = ""
						page, _ = cmdArgs.GetArgN(2)
						isRec = true
					}
					if val == "rec" {
						isRec = true
					}
					var author = ""
					if val == "author" {
						author, _ = cmdArgs.GetArgN(2)
					}

					thePage, _ := strconv.ParseInt(page, 10, 64)
					if thePage <= 0 {
						thePage = 1
					}

					ret := CnmodsSearch(keyword, int(thePage), 7, isRec, author)
					if ret != nil {
						text := fmt.Sprintf("来自cnmods的搜索结果 - %d/%d页%d项:\n", thePage, ret.Data.TotalPages, ret.Data.TotalElements)
						for _, item := range ret.Data.List {
							ver := ""
							if item.ModuleVersion == "coc6th" {
								ver = "[coc6]"
							}
							// 魔都现在只有coc本
							//if item.ModuleVersion == "coc7th" {
							//	ver = "[coc7]"
							//}
							text += fmt.Sprintf("[%d]%s%s %s%s - by %s\n", item.KeyId, ver, item.Title, item.ModuleAge, item.OccurrencePlace, item.Article)
						}
						if len(ret.Data.List) == 0 {
							text += "什么也没发现"
						}
						ReplyToSender(ctx, msg, text)
					}
				case "roll":
					ret := CnmodsSearch("", 1, 1, false, "")
					id := rand.Int()%ret.Data.TotalElements + 1
					text := getDetail(strconv.Itoa(id))
					if text != "" {
						ReplyToSender(ctx, msg, text)
					} else {
						ReplyToSender(ctx, msg, "什么也没发现")
					}

				case "get":
					page, _ := cmdArgs.GetArgN(2)
					text := getDetail(page)
					if text != "" {
						ReplyToSender(ctx, msg, text)
					} else {
						ReplyToSender(ctx, msg, "什么也没发现")
					}
				default:
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	theExt := &ExtInfo{
		Name:       "story", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.1",
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
			"name":    cmdName,
			"namednd": cmdNameDnd,
			"who":     cmdWho,
			"cnmods":  cmdCnmods,
			"modu":    cmdCnmods,
			"魔都":      cmdCnmods,
		},
	}

	self.RegisterExtension(theExt)
}
