package dice

import (
	"fmt"
	"html"
	"math/rand"
	"strconv"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
)

func cmdRandomName(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, cmdsList [][]string, rulesCallback func(gender string) [][]string, defaultIndex int) CmdExecuteResult {
	var names []string
	var chops []string
	for _, i := range cmdsList {
		chops = append(chops, i...)
	}
	cmdArgs.ChopPrefixToArgsWith(chops...)

	numText := cmdArgs.GetArgN(2)
	var num int64
	if numText != "" {
		num, _ = strconv.ParseInt(numText, 10, 64)
	}
	if num == 0 {
		num = 5
	}
	if num > 10 {
		num = 10
	}

	var genderText string
	checkGender := func(newText string) bool {
		matchOne := func(text string, list []string) bool {
			for _, i := range list {
				if strings.EqualFold(text, i) {
					return true
				}
			}
			return false
		}
		if matchOne(newText, []string{"男", "男性", "Male", "M"}) {
			genderText = "M"
		}
		if matchOne(newText, []string{"女", "女性", "Female", "F"}) {
			genderText = "F"
		}

		if genderText != "M" && genderText != "F" {
			genderText = ""
			return false
		}
		return true
	}

	if !checkGender(cmdArgs.GetArgN(2)) {
		checkGender(cmdArgs.GetArgN(3))
	}

	rulesList := rulesCallback(genderText)

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
		return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
	}

	// 开始抽取
	for i := int64(0); i < num; i++ {
		rule := rules[rand.Int()%len(rules)]
		names = append(names, ctx.Dice.Parent.NamesGenerator.NameGenerate(rule))
	}

	sep := DiceFormatTmpl(ctx, "其它:随机名字_分隔符")
	namesTxt := strings.Join(names, sep)
	VarSetValueStr(ctx, "$t随机名字文本", namesTxt)
	text := DiceFormatTmpl(ctx, "其它:随机名字")
	ReplyToSender(ctx, msg, text)
	return CmdExecuteResult{Matched: true, Solved: true}
}

func RegisterBuiltinStory(self *Dice) {
	cmdName := &CmdItemInfo{
		Name:      "name",
		ShortHelp: ".name cn/en/jp [<数量>] [<性别>]",
		Help:      "生成随机名字:\n.name cn/en/jp [<数量>] [<性别>]",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdRandomName(ctx, msg, cmdArgs, [][]string{
				{"cn", "中文", "zh", "中国"},
				{"en", "英文", "英国", "美国", "us"},
				{"jp", "日文", "日本"},
			}, func(gender string) [][]string {
				// 写两遍似乎不太好，但有什么其他好的办法？
				switch gender {
				case "M":
					return [][]string{
						{"{中文:姓氏@中文:姓氏权重}{中文:男性名}"},
						{"{英文:男性名} {英文:姓氏} ({英文:男性名中文#英文:男性名.index}·{英文:姓氏中文#英文:姓氏.index})"},
						{"{日文:姓氏} {日文:男性名}({日文:姓氏平假名#日文:姓氏.index} {日文:男性名平假名#日文:男性名.index})"},
					}
				case "F":
					return [][]string{
						{"{中文:姓氏@中文:姓氏权重}{中文:女性名}"},
						{"{英文:女性名} {英文:姓氏} ({英文:女性名中文#英文:女性名.index}·{英文:姓氏中文#英文:姓氏.index})"},
						{"{日文:姓氏} {日文:女性名}({日文:姓氏平假名#日文:姓氏.index} {日文:女性名平假名#日文:女性名.index})"},
					}
				default:
					return [][]string{
						{
							"{中文:姓氏@中文:姓氏权重}{中文:男性名}",
							"{中文:姓氏@中文:姓氏权重}{中文:女性名}",
						},
						{
							"{英文:男性名} {英文:姓氏} ({英文:男性名中文#英文:男性名.index}·{英文:姓氏中文#英文:姓氏.index})",
							"{英文:女性名} {英文:姓氏} ({英文:女性名中文#英文:女性名.index}·{英文:姓氏中文#英文:姓氏.index})",
						},
						{
							"{日文:姓氏} {日文:男性名}({日文:姓氏平假名#日文:姓氏.index} {日文:男性名平假名#日文:男性名.index})",
							"{日文:姓氏} {日文:女性名}({日文:姓氏平假名#日文:姓氏.index} {日文:女性名平假名#日文:女性名.index})",
						},
					}
				}
			}, 0)
		},
	}

	cmdNameDnd := &CmdItemInfo{
		Name:      "namednd",
		ShortHelp: ".namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族/地精",
		Help:      "生成随机DND名字:\n.namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族/地精",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
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
			}, func(gender string) [][]string {
				switch gender {
				case "M":
					return [][]string{
						{"{DND达马拉人:男性英文名} {DND达马拉人:英文姓氏} ({DND达马拉人:男性中文名#DND达马拉人:男性英文名.index}·{DND达马拉人:中文姓氏#DND达马拉人:英文姓氏.index})"},
						{"{DND卡林珊人:Calashite_名_男} {DND卡林珊人:Calashite_姓} ({DND卡林珊人:Calashite_名_男_中文#DND卡林珊人:Calashite_名_男.index}·{DND卡林珊人:Calashite_姓_中文#DND卡林珊人:Calashite_姓.index})"},
						{"{DND莱瑟曼人:Rashemi_名_男} {DND莱瑟曼人:Rashemi_姓} ({DND莱瑟曼人:Rashemi_名_男_中文#DND莱瑟曼人:Rashemi_名_男.index}·{DND莱瑟曼人:Rashemi_姓_中文#DND莱瑟曼人:Rashemi_姓.index})"},
						{"{DND受国人:男性中文名}·{DND受国人:中文姓氏} ({DND受国人:男性英文名#DND受国人:男性中文名.index} {DND受国人:英文姓氏#DND受国人:中文姓氏.index})"},
						{"{DND精灵:精灵_名_男} {DND精灵:精灵_姓} ({DND精灵:精灵_名_男_中文#DND精灵:精灵_名_男.index}·{DND精灵:精灵_姓_中文#DND精灵:精灵_姓.index})"},
						{"{DND矮人:矮人_名_男} {DND矮人:矮人_姓} ({DND矮人:矮人_名_男_中文#DND矮人:矮人_名_男.index}·{DND矮人:矮人_姓_中文#DND矮人:矮人_姓.index})"},
						{"{DND兽人:兽人_名_男} \"{DND兽人:兽人_绰号}\" (“{DND兽人:兽人_绰号_中文#DND兽人:兽人_绰号.index}”{DND兽人:兽人_名_男_中文#DND兽人:兽人_名_男.index})"},
						{"{DND海族:海族_名_男} ({DND海族:海族_名_男_中文#DND海族:海族_名_男.index})"},
						{"{DND地精:地精_名_男} ({DND地精:地精_名_男_中文#DND地精:地精_名_男.index})"},
					}
				case "F":
					return [][]string{
						{"{DND达马拉人:女性英文名} {DND达马拉人:英文姓氏} ({DND达马拉人:女性中文名#DND达马拉人:女性英文名.index}·{DND达马拉人:中文姓氏#DND达马拉人:英文姓氏.index})"},
						{"{DND卡林珊人:Calashite_名_女} {DND卡林珊人:Calashite_姓} ({DND卡林珊人:Calashite_名_女_中文#DND卡林珊人:Calashite_名_女.index}·{DND卡林珊人:Calashite_姓_中文#DND卡林珊人:Calashite_姓.index})"},
						{"{DND莱瑟曼人:Rashemi_名_女} {DND莱瑟曼人:Rashemi_姓} ({DND莱瑟曼人:Rashemi_名_女_中文#DND莱瑟曼人:Rashemi_名_女.index}·{DND莱瑟曼人:Rashemi_姓_中文#DND莱瑟曼人:Rashemi_姓.index})"},
						{"{DND受国人:女性中文名}·{DND受国人:中文姓氏} ({DND受国人:女性英文名#DND受国人:女性中文名.index} {DND受国人:英文姓氏#DND受国人:中文姓氏.index})"},
						{"{DND精灵:精灵_名_女} {DND精灵:精灵_姓} ({DND精灵:精灵_名_女_中文#DND精灵:精灵_名_女.index}·{DND精灵:精灵_姓_中文#DND精灵:精灵_姓.index})"},
						{"{DND矮人:矮人_名_女} {DND矮人:矮人_姓} ({DND矮人:矮人_名_女_中文#DND矮人:矮人_名_女.index}·{DND矮人:矮人_姓_中文#DND矮人:矮人_姓.index})"},
						{"{DND兽人:兽人_名_女} \"{DND兽人:兽人_绰号}\" (“{DND兽人:兽人_绰号_中文#DND兽人:兽人_绰号.index}”{DND兽人:兽人_名_女_中文#DND兽人:兽人_名_女.index})"},
						{"仅有男性"},
						{"{DND地精:地精_名_女} ({DND地精:地精_名_女_中文#DND地精:地精_名_女.index})"},
					}
				default:
					return [][]string{
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
					}
				}
			}, -1)
		},
	}

	cmdWho := &CmdItemInfo{
		Name:      "who",
		ShortHelp: ".who a b c",
		Help:      "顺序重排:\n.who a b c",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
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
		},
	}

	helpCnmods := ".modu search <关键字> [<页码>] // 搜索关键字\n" +
		".modu rec <关键字> [<页码>] // 搜索编辑推荐\n" +
		".modu author <关键字> [<页码>] // 搜索指定作者\n" +
		".modu luck [<页码>] // 查看编辑推荐\n" +
		".modu get <编号> // 查看指定详情\n" +
		".modu roll // 随机抽取\n" +
		".modu help // 显示帮助"
	cmdCnmods := &CmdItemInfo{
		Name:      "modu",
		ShortHelp: helpCnmods,
		Help:      "魔都查询:\n" + helpCnmods,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("help", "search", "find", "rec", "luck", "get", "author", "roll")

			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			logAndReply := func(err error) {
				ctx.Dice.Logger.Errorf("cnmods search error: %v", err)
				ReplyToSender(ctx, msg, "魔都查询出错，请稍后再试")
			}

			getDetail := func(keyId string) string {
				ret := CnmodsDetail(keyId)
				if ret != nil {
					item := ret.Data.Module

					// opinion := item.Opinion
					opinion := html.UnescapeString(item.Opinion) // 确实仍然存在一些 有标签的，如2488
					opinion = strip.StripTags(opinion)
					ori := "是"
					if !item.Original {
						ori = "否"
					}

					var recInfo string
					// if len(ret.Data.RecommendList) > 0 {
					//	rec := ret.Data.RecommendList[0]
					//	recInfo = fmt.Sprintf("推荐语: %s - by %s\n", rec.Content, rec.LoginUser.NickName)
					// }

					text := fmt.Sprintf("[%d]%s\n作者: %s\n背景: %s,%s\n规模: %d-%d人，%d-%d时\n原创: %v\n简介: %s\n%sPC端链接：%s\n移动端链接：%s",
						item.KeyID, item.Title,
						item.Article,
						item.ModuleAge, item.OccurrencePlace,
						item.MinAmount, item.MaxAmount,
						item.MinDuration, item.MaxDuration,
						ori,
						opinion,
						recInfo,
						"https://www.cnmods.net/web/moduleDetail?keyId="+strconv.Itoa(item.KeyID),
						"https://www.cnmods.net/mobile/moduleDetail?keyId="+strconv.Itoa(item.KeyID),
					)
					return text
				}
				return ""
			}

			_val := cmdArgs.GetArgN(1)
			val := strings.ToLower(_val)
			switch val {
			case "search", "find", "rec", "luck", "author":
				keyword := cmdArgs.GetArgN(2)
				page := cmdArgs.GetArgN(3)
				isRec := false
				if val == "luck" {
					keyword = ""
					page = cmdArgs.GetArgN(2)
					isRec = true
				}
				if val == "rec" {
					isRec = true
				}
				var author = ""
				if val == "author" {
					author = cmdArgs.GetArgN(2)
				}

				thePage, _ := strconv.ParseInt(page, 10, 64)
				if thePage <= 0 {
					thePage = 1
				}

				ret, err := CnmodsSearch(keyword, int(thePage), 7, isRec, author)
				if err != nil {
					logAndReply(err)
					break
				}

				text := fmt.Sprintf("来自cnmods的搜索结果 - %d/%d页%d项:\n", thePage, ret.Data.TotalPages, ret.Data.TotalElements)
				for _, item := range ret.Data.List {
					ver := ""
					if item.ModuleVersion == "coc6th" {
						ver = "[coc6]"
					}
					// 魔都现在只有coc本
					// if item.ModuleVersion == "coc7th" {
					//	ver = "[coc7]"
					// }
					text += fmt.Sprintf("[%d]%s%s %s%s - by %s\n", item.KeyID, ver, item.Title, item.ModuleAge, item.OccurrencePlace, item.Article)
				}
				if len(ret.Data.List) == 0 {
					text += "什么也没发现"
				}
				ReplyToSender(ctx, msg, text)
			case "roll":
				ret, err := CnmodsSearch("", 1, 1, false, "")
				if err != nil {
					logAndReply(err)
					break
				}
				id := rand.Int()%ret.Data.TotalElements + 1
				text := getDetail(strconv.Itoa(id))
				if text != "" {
					ReplyToSender(ctx, msg, text)
				} else {
					ReplyToSender(ctx, msg, "什么也没发现")
				}

			case "get":
				page := cmdArgs.GetArgN(2)
				text := getDetail(page)
				if text != "" {
					ReplyToSender(ctx, msg, text)
				} else {
					ReplyToSender(ctx, msg, "什么也没发现")
				}
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	theExt := &ExtInfo{
		Name:       "story", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.1",
		Brief:      "提供随机姓名、排序、魔都查询等功能",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		Official:   true,
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		OnLoad:      func() {},
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
