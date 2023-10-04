package dice

import (
	"fmt"
	strip "github.com/grokify/html-strip-tags-go"
	wr "github.com/mroth/weightedrand"
	"html"
	"math/rand"
	"strconv"
	"strings"
)

func randSlice(s []string) string {
	return s[rand.Intn(len(s))]
}

func cmdRandomName(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, rule string) CmdExecuteResult {
	cmdArgs.ChopPrefixToArgsWith(
		"cn", "中文", "zh", "中国",
		"en", "英文", "英国", "美国", "us",
		"jp", "日文", "日本",
		"达马拉人", "卡林珊人", "莱瑟曼人", "受国人", "精灵", "矮人", "兽人", "海族", "地精",
	)
	str2num := cmdArgs.GetArgN(1) // 快捷方式 .name 10
	num, err := strconv.ParseInt(str2num, 10, 64)
	s := "cn"
	if err != nil {
		// 正常 .name cn 10
		s = cmdArgs.GetArgN(1)
		str2num = cmdArgs.GetArgN(2)
		num, _ = strconv.ParseInt(str2num, 10, 64)
	}
	if num == 0 {
		num = 5
	}
	if num > 10 {
		num = 10
	}
	switch s {
	case "cn", "中文", "zh", "中国":
		s = "中文"
	case "en", "英文", "英国", "美国", "us":
		s = "英文"
	case "jp", "日文", "日本":
		s = "日文"
	}
	s = rule + s
	s = strings.ToUpper(s) // name dnd兽人
	ng := ctx.Dice.Parent.NamesGenerator
	if l, ok := ng.names[s]; ok {
		one := func() string {
			var surname, firstname, snAs, fnAs string
			// 地精没有姓
			if len(l.surname) > 0 {
				var choices []wr.Choice
				for text, w := range l.surname {
					if cmdArgs.GetKwarg("nw") != nil {
						w = 1
					}
					choices = append(choices, wr.Choice{Item: text, Weight: uint(w)})
				}
				pool, _ := wr.NewChooser(choices...)
				surname = pool.Pick().(string)
			}
			// 有些不分男女
			if len(l.firstName) > 0 {
				firstname = randSlice(l.firstName)
			} else {
				if cmdArgs.GetKwarg("m") != nil {
					firstname = randSlice(l.maleName)
				} else if cmdArgs.GetKwarg("f") != nil {
					firstname = randSlice(l.femaleName)
				} else {
					sl := append(l.maleName, l.femaleName...)
					firstname = randSlice(sl)
				}
			}
			snAs = ng.aliasNames[surname]
			fnAs = ng.aliasNames[firstname]
			VarSetValueAuto(ctx, "$t姓", surname)
			VarSetValueAuto(ctx, "$t名", firstname)
			VarSetValueAuto(ctx, "$t姓_as", snAs)
			VarSetValueAuto(ctx, "$t名_as", fnAs)
			var t string
			switch s {
			case "中文":
				t = DiceFormatTmpl(ctx, "其它:随机名字_模版_中文")
			case "日文":
				t = DiceFormatTmpl(ctx, "其它:随机名字_模版_日文")
			default:
				t = DiceFormatTmpl(ctx, "其它:随机名字_模版_英文")
			}
			return t
		}

		var res []string
		for i := 0; i < int(num); i++ {
			res = append(res, one())
		}

		VarSetValueAuto(ctx, "$t随机名字文本", strings.Join(res, DiceFormatTmpl(ctx, "其它:随机名字_分隔符")))
		ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "其它:随机名字"))
	} else {
		return CmdExecuteResult{ShowHelp: true, Solved: true, Matched: true}
	}

	return CmdExecuteResult{Matched: true, Solved: true}
}

func RegisterBuiltinStory(self *Dice) {
	cmdName := &CmdItemInfo{
		Name:      "name",
		ShortHelp: ".name cn/en/jp (<数量>)",
		Help:      "生成随机名字:\n.name cn/en/jp (<数量>)\n--m 指定性别为男\n--f 指定性别为女\n--nw 不使用姓氏权重",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdRandomName(ctx, msg, cmdArgs, "")
		},
	}

	cmdNameDnd := &CmdItemInfo{
		Name:      "namednd",
		ShortHelp: ".namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族/地精",
		Help:      "生成随机DND名字:\n.namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族/地精\n--m 指定性别为男\n--f 指定性别为女",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			return cmdRandomName(ctx, msg, cmdArgs, "DND")
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

	helpCnmods := ".modu search <关键字> (<页码>) // 搜索关键字\n" +
		".modu rec <关键字> (<页码>) // 搜索编辑推荐\n" +
		".modu author <关键字> (<页码>) // 搜索指定作者\n" +
		".modu luck (<页码>) // 查看编辑推荐\n" +
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

			getDetail := func(keyId string) string {
				ret := CnmodsDetail(keyId)
				if ret != nil {
					item := ret.Data.Module

					//opinion := item.Opinion
					opinion := html.UnescapeString(item.Opinion) // 确实仍然存在一些 有标签的，如2488
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

					text := fmt.Sprintf("[%d]%s\n作者: %s\n背景: %s,%s\n规模: %d-%d人，%d-%d时\n原创: %v\n简介: %s\n%sPC端链接：%s\n移动端链接：%s",
						item.KeyId, item.Title,
						item.Article,
						item.ModuleAge, item.OccurrencePlace,
						item.MinAmount, item.MaxAmount,
						item.MinDuration, item.MaxDuration,
						ori,
						opinion,
						recInfo,
						"https://www.cnmods.net/web/moduleDetail?keyId="+strconv.Itoa(item.KeyId),
						"https://www.cnmods.net/mobile/moduleDetail?keyId="+strconv.Itoa(item.KeyId),
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
