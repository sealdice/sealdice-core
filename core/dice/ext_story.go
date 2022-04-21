package dice

import (
	"fmt"
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
				return cmdRandomName(ctx, msg, cmdArgs, [][]string{
					{"cn", "中文"},
					{"en", "英文"},
					{"jp", "日文"},
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
		Help:     ".namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族",
		LongHelp: "生成随机DND名字:\n.namednd 达马拉人/卡林珊人/莱瑟曼人/受国人/精灵/矮人/兽人/海族",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				return cmdRandomName(ctx, msg, cmdArgs, [][]string{
					{"达马拉人"},
					//{"卡林珊人"},
					//{"莱瑟曼人"},
					//{"受国人"},
					//{"精灵"},
					//{"矮人"},
					//{"兽人"},
					//{"海族"},
				}, [][]string{
					{
						"{DND达马拉人:男性英文名} {DND达马拉人:英文姓氏} ({DND达马拉人:男性中文名#DND达马拉人:男性英文名.index}·{DND达马拉人:中文姓氏#DND达马拉人:英文姓氏.index})",
						"{DND达马拉人:女性英文名} {DND达马拉人:英文姓氏} ({DND达马拉人:女性中文名#DND达马拉人:女性英文名.index}·{DND达马拉人:中文姓氏#DND达马拉人:英文姓氏.index})",
					},
				}, -1)
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	theExt := &ExtInfo{
		Name:       "story", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
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
		},
	}

	self.RegisterExtension(theExt)
}
