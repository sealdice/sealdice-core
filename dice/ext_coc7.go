package dice

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

var (
	//go:embed coc7_fear.txt
	fearListText string
	//go:embed coc7_mania.txt
	maniaListText string
)

var difficultyPrefixMap = map[string]int{
	"":    1,
	"常规":  1,
	"困难":  2,
	"极难":  3,
	"大成功": 4,
	"困難":  2,
	"極難":  3,
	"常規":  1,
}

func cardRuleCheck(mctx *MsgContext, msg *Message) *GameSystemTemplate {
	cardType := ReadCardType(mctx)
	if cardType != "" && cardType != mctx.Group.System {
		ReplyToSender(mctx, msg, fmt.Sprintf("阻止操作：当前卡规则为 %s，群规则为 %s。\n为避免损坏此人物卡，请先更换角色卡，或使用.st fmt强制转卡", cardType, mctx.Group.System))
		return nil
	}
	tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
	if tmpl == nil {
		ReplyToSender(mctx, msg, fmt.Sprintf("阻止操作：未发现人物卡使用的规则: %s，可能相关扩展已经卸载，请联系骰主", cardType))
		return nil
	}
	cmdStCharFormat(mctx, tmpl) // 转一下卡
	return tmpl
}

func RegisterBuiltinExtCoc7(self *Dice) {
	// 初始化疯狂列表
	reFear := regexp.MustCompile(`(\d+)\)\s+([^\n]+)`)
	m := reFear.FindAllStringSubmatch(fearListText, -1)
	fearMap := map[int]string{}
	for _, i := range m {
		n, _ := strconv.Atoi(i[1])
		fearMap[n] = i[2]
	}

	m = reFear.FindAllStringSubmatch(maniaListText, -1)
	maniaMap := map[int]string{}
	for _, i := range m {
		n, _ := strconv.Atoi(i[1])
		maniaMap[n] = i[2]
	}

	helpRc := "" +
		".ra/rc <属性表达式> // 属性检定指令，当前者小于等于后者，检定通过\n" +
		".ra <难度><属性> // 如 .ra 困难侦查\n" +
		".ra b <属性表达式> // 奖励骰或惩罚骰\n" +
		".ra p2 <属性表达式> // 多个奖励骰或惩罚骰\n" +
		".ra 3#p <属性表达式> // 多重检定\n" +
		".ra <属性表达式> @某人 // 对某人做检定(使用他的属性)\n" +
		".rch/rah // 暗中检定，和检定指令用法相同"

	cmdRc := &CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "rc/ra",
		ShortHelp:               helpRc,
		Help:                    "检定指令:\n" + helpRc,
		AllowDelegate:           true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if len(cmdArgs.Args) == 0 {
				ctx.DelegateText = ""
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:检定_格式错误"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			mctx.DelegateText = ctx.DelegateText
			mctx.SystemTemplate = mctx.Group.GetCharTemplate(ctx.Dice)
			restText := cmdArgs.CleanArgs

			tmpl := cardRuleCheck(mctx, msg)
			if tmpl == nil {
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			// alias resolution now relies on the active system template set above

			reBP := regexp.MustCompile(`^[bBpP(]`)
			re2 := regexp.MustCompile(`([^\d]+)\s+([\d]+)`)

			if !reBP.MatchString(restText) {
				restText = re2.ReplaceAllString(restText, "$1$2")
				restText = "D100 " + restText
			} else {
				replaced := true
				if len(restText) > 1 {
					// 为了避免一种分支情况: .ra  b 50 测试，b和50中间的空格被消除
					ch2 := restText[1]
					r := rune(ch2)
					if unicode.IsSpace(r) { // 暂不考虑太过奇葩的空格
						replaced = true
						restText = restText[:1] + " " + re2.ReplaceAllString(restText[2:], "$1$2")
					} else if restText[0] != '(' { // if !(unicode.IsNumber(r) || r == '(')
						// 将 .rab测试 切开为 "b 测试"
						// 注: 判断 ( 是为了 .ra(1)50 能够运行，除此之外还有 .rab3(1)50 等等
						for index, i := range restText[1:] {
							if i == '(' {
								break
							}

							if !unicode.IsNumber(i) {
								restText = restText[:index+1] + " " + restText[index+1:]
								break
							}
						}
					}
				}

				if !replaced {
					restText = re2.ReplaceAllString(restText, "$1$2")
				}
			}

			cocRule := mctx.Group.CocRuleIndex
			if cmdArgs.Command == "rc" || cmdArgs.Command == "rch" {
				// 强制规则书
				cocRule = 0
			}

			var reason string
			var commandInfoItems []interface{}
			rollOne := func(manyTimes bool) *CmdExecuteResult {
				difficultyRequire := 0
				// 试图读取检定表达式
				swap := false
				r1, detail1, err := DiceExprEvalBase(mctx, restText, RollExtraFlags{
					CocVarNumberMode: true, // CallbackLoadVar 替代
					CocDefaultAttrOn: true, // 弃用
					DisableBlock:     true,
				})

				if err != nil {
					ReplyToSender(mctx, msg, "解析出错: "+restText)
					return &CmdExecuteResult{Matched: true, Solved: true}
				}

				difficultyRequire2 := difficultyPrefixMap[r1.GetCocPrefix()]
				if difficultyRequire2 > difficultyRequire {
					difficultyRequire = difficultyRequire2
				}
				expr1Text := r1.GetMatched()
				expr2Text := strings.TrimSpace(r1.GetRestInput())

				// 如果读取完了，那么说明刚才读取的实际上是属性表达式
				if expr2Text == "" {
					expr2Text = "D100"
					swap = true
				}

				r2, detail2, err := DiceExprEvalBase(mctx, expr2Text, RollExtraFlags{
					CocVarNumberMode: true,
					CocDefaultAttrOn: true,
					DisableBlock:     true,
				})

				if err != nil {
					ReplyToSender(mctx, msg, "解析出错: "+expr2Text)
					return &CmdExecuteResult{Matched: true, Solved: true}
				}

				expr2Text = r2.GetMatched()
				reason = LimitCommandReasonText(r2.GetRestInput())

				difficultyRequire2 = difficultyPrefixMap[r2.GetCocPrefix()]
				if difficultyRequire2 > difficultyRequire {
					difficultyRequire = difficultyRequire2
				}

				if swap {
					r1, r2 = r2, r1
					detail1, detail2 = detail2, detail1 //nolint
					expr1Text, expr2Text = expr2Text, expr1Text
				}

				if r1.TypeId != ds.VMTypeInt || r2.TypeId != ds.VMTypeInt {
					ReplyToSender(mctx, msg, "你输入的表达式并非文本类型")
					return &CmdExecuteResult{Matched: true, Solved: true}
				}

				// 注: GetMatched()只能使用一次，因为第二次执行后就会变成新的，因此改为读取之前的值
				if expr1Text == "d100" || expr1Text == "D100" {
					// 此时没有必要
					detail1 = ""
				}

				var outcome = int64(r1.Value.(ds.IntType))
				var attrVal = int64(r2.Value.(ds.IntType))

				successRank, criticalSuccessValue := ResultCheck(mctx, cocRule, outcome, attrVal, difficultyRequire)
				// 根据难度需求，修改判定值
				checkVal := attrVal
				switch difficultyRequire {
				case 2:
					checkVal /= 2
				case 3:
					checkVal /= 5
				case 4:
					checkVal = criticalSuccessValue
				}
				VarSetValueInt64(mctx, "$tD100", outcome)
				// $tD100是为了兼容旧的模板，$t骰子出目 和 $t检定结果 是新的，coc 与 dnd 都可以用
				VarSetValueInt64(mctx, "$t骰子出目", outcome)
				VarSetValueInt64(mctx, "$t检定结果", outcome)
				VarSetValueInt64(mctx, "$t判定值", checkVal)
				VarSetValueInt64(mctx, "$tSuccessRank", int64(successRank))
				VarSetValueStr(mctx, "$t属性表达式文本", expr2Text)

				var suffix string
				var suffixFull string
				var suffixShort string
				if difficultyRequire > 1 {
					// 此时两者内容相同这样做是为了避免失败文本被计算两次
					suffixFull = GetResultTextWithRequire(mctx, successRank, difficultyRequire, false)
					suffixShort = suffixFull
				} else {
					suffixFull = GetResultTextWithRequire(mctx, successRank, difficultyRequire, false)
					suffixShort = GetResultTextWithRequire(mctx, successRank, difficultyRequire, true)
				}

				if manyTimes {
					suffix = suffixShort
				} else {
					suffix = suffixFull
				}

				VarSetValueStr(mctx, "$t判定结果", suffix)
				VarSetValueStr(mctx, "$t判定结果_详细", suffixFull)
				VarSetValueStr(mctx, "$t判定结果_简短", suffixShort)

				detailWrap := ""
				if detail1 != "" {
					detailWrap = ", (" + detail1 + ")"
				}

				// 指令信息标记
				infoItem := map[string]interface{}{
					"expr1":    expr1Text,
					"expr2":    expr2Text,
					"outcome":  outcome,
					"attrVal":  attrVal,
					"checkVal": checkVal,
					"rank":     successRank,
				}
				commandInfoItems = append(commandInfoItems, infoItem)

				VarSetValueStr(mctx, "$t检定表达式文本", expr1Text)
				VarSetValueStr(mctx, "$t检定计算过程", detailWrap)
				VarSetValueStr(mctx, "$t计算过程", detailWrap)

				SetTempVars(mctx, mctx.Player.Name) // 信息里没有QQ昵称，用这个顶一下
				return nil
			}

			var text string
			if cmdArgs.SpecialExecuteTimes > 1 {
				VarSetValueInt64(mctx, "$t次数", int64(cmdArgs.SpecialExecuteTimes))
				if cmdArgs.SpecialExecuteTimes > int(ctx.Dice.Config.MaxExecuteTime) {
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_轮数过多警告"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				texts := []string{}
				for range cmdArgs.SpecialExecuteTimes {
					ret := rollOne(true)
					if ret != nil {
						return *ret
					}
					texts = append(texts, DiceFormatTmpl(mctx, "COC:检定_单项结果文本"))
				}

				VarSetValueStr(mctx, "$t原因", reason)
				VarSetValueStr(mctx, "$t结果文本", strings.Join(texts, "\n"))
				text = DiceFormatTmpl(mctx, "COC:检定_多轮")
			} else {
				ret := rollOne(false)
				if ret != nil {
					return *ret
				}
				VarSetValueStr(mctx, "$t原因", reason)
				VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:检定_单项结果文本"))
				text = DiceFormatTmpl(mctx, "COC:检定")
			}

			isHide := cmdArgs.Command == "rah" || cmdArgs.Command == "rch"

			// 指令信息
			commandInfo := map[string]interface{}{
				"cmd":     "ra",
				"rule":    "coc7",
				"pcName":  mctx.Player.Name,
				"cocRule": cocRule,
				"items":   commandInfoItems,
			}
			if isHide {
				commandInfo["hide"] = isHide
			}
			mctx.CommandInfo = commandInfo

			if kw := cmdArgs.GetKwarg("ci"); kw != nil {
				info, err := json.Marshal(mctx.CommandInfo)
				if err == nil {
					text += "\n" + string(info)
				} else {
					text += "\n" + "指令信息无法序列化"
				}
			}

			if isHide {
				if msg.Platform == "QQ-CH" {
					ReplyToSender(mctx, msg, "QQ频道内尚不支持暗骰")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if mctx.IsPrivate {
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "核心:提示_私聊不可用"))
				} else {
					mctx.CommandHideFlag = mctx.Group.GroupID
					ReplyGroup(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_暗中_群内"))
					ReplyPerson(mctx, msg, DiceFormatTmpl(mctx, "COC:检定_暗中_私聊_前缀")+text)
				}
			} else {
				ReplyToSender(mctx, msg, text)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSetCOC := ".setcoc 0-5 // 设置常见的0-5房规\n" +
		".setcoc dg // delta green 扩展规则"
	cmdSetCOC := &CmdItemInfo{
		Name:      "setcoc",
		ShortHelp: helpSetCOC,
		Help:      "设置房规:\n" + helpSetCOC,
		HelpFunc: func(isShort bool) string {
			var help strings.Builder
			help.WriteString(".setcoc 0-5 // 设置常见的0-5房规，0为规则书，2为国内常用规则\n")
			help.WriteString(".setcoc dg // delta green 扩展规则\n")
			help.WriteString(".setcoc details // 列出所有规则及其解释文本 \n")
			// 自定义
			for _, i := range self.CocExtraRules {
				n := strings.ReplaceAll(i.Desc, "\n", " ")
				fmt.Fprintf(&help, ".setcoc %d/%s // %s\n", i.Index, i.Key, n)
			}

			helpText := help.String()
			if isShort {
				return helpText
			}
			return "设置房规:\n" + helpText
		},
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			n := cmdArgs.GetArgN(1)
			suffix := "\nCOC7规则扩展已自动开启"
			setRuleByName(ctx, "coc7")

			switch n {
			case "0":
				ctx.Group.CocRuleIndex = 0
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "1":
				ctx.Group.CocRuleIndex = 1
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "2":
				ctx.Group.CocRuleIndex = 2
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "3":
				ctx.Group.CocRuleIndex = 3
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "4":
				ctx.Group.CocRuleIndex = 4
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "5":
				ctx.Group.CocRuleIndex = 5
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "dg":
				ctx.Group.CocRuleIndex = 11
				text := fmt.Sprintf("已切换房规为%s:\n%s%s", SetCocRulePrefixText[ctx.Group.CocRuleIndex], SetCocRuleText[ctx.Group.CocRuleIndex], suffix)
				ReplyToSender(ctx, msg, text)
			case "details":
				var help strings.Builder
				help.WriteString("当前有coc7规则如下:\n")
				for i := range 6 {
					basicStr := strings.ReplaceAll(SetCocRuleText[i], "\n", " ")
					fmt.Fprintf(&help, ".setcoc %d // %s\n", i, basicStr)
				}
				// dg
				dgStr := strings.ReplaceAll(SetCocRuleText[11], "\n", " ")
				fmt.Fprintf(&help, ".setcoc dg // %s\n", dgStr)

				// 自定义
				for _, i := range self.CocExtraRules {
					ruleStr := strings.ReplaceAll(i.Desc, "\n", " ")
					fmt.Fprintf(&help, ".setcoc %d/%s // %s\n", i.Index, i.Key, ruleStr)
				}
				ReplyToSender(ctx, msg, help.String())
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			default:
				nInt, _ := strconv.ParseInt(n, 10, 64)
				for _, i := range ctx.Dice.CocExtraRules {
					if i.Key == n || nInt == int64(i.Index) {
						ctx.Group.CocRuleIndex = i.Index
						text := fmt.Sprintf("已切换房规为%s:\n%s%s", i.Name, i.Desc, suffix)
						ReplyToSender(ctx, msg, text)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
				}

				if text, ok := SetCocRuleText[ctx.Group.CocRuleIndex]; ok {
					VarSetValueStr(ctx, "$t房规文本", text)
					VarSetValueStr(ctx, "$t房规", SetCocRulePrefixText[ctx.Group.CocRuleIndex])
					VarSetValueInt64(ctx, "$t房规序号", int64(ctx.Group.CocRuleIndex))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:设置房规_当前"))
				} else {
					// TODO: 修改这种找规则的方式
					var rule *CocRuleInfo
					nInt64 := ctx.Group.CocRuleIndex
					for _, i := range ctx.Dice.CocExtraRules {
						if nInt64 == i.Index {
							rule = i
							break
						}
					}

					VarSetValueStr(ctx, "$t房规文本", rule.Desc)
					VarSetValueStr(ctx, "$t房规", rule.Name)
					VarSetValueInt64(ctx, "$t房规序号", int64(rule.Index))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:设置房规_当前"))
				}
			}

			ctx.Group.ExtActive(ctx.Dice.ExtFind("coc7", false))
			ctx.Group.System = "coc7"
			ctx.Group.MarkDirty(ctx.Dice)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpRcv := ".rav/.rcv <技能> @某人 // 自己和某人进行对抗检定\n" +
		".rav <技能1> <技能2> @某A @某B // 对A和B两人做对抗检定，分别使用输入的两个技能数值\n" +
		"// 注: <技能>写法举例: 侦查、侦查40、困难侦查、40、侦查+10"
	cmdRcv := &CmdItemInfo{
		Name:          "rcv/rav",
		ShortHelp:     helpRcv,
		Help:          "对抗检定:\n" + helpRcv,
		AllowDelegate: true, // 特殊的代骰
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			ctx.DelegateText = "" // 消除代骰文本，避免单人代骰出问题

			switch val {
			case "help", "":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			default:
				// 至少@一人，检定才成立
				if len(cmdArgs.At) == 0 {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				ctx1 := ctx
				ctx2 := ctx

				if cmdArgs.AmIBeMentionedFirst {
					// 第一个at的是骰子，不计为 at的人
					if len(cmdArgs.At) == 2 {
						// 单人
						ctx2, _ = cmdArgs.At[1].CopyCtx(ctx)
					} else if len(cmdArgs.At) == 3 {
						ctx1, _ = cmdArgs.At[1].CopyCtx(ctx)
						ctx2, _ = cmdArgs.At[2].CopyCtx(ctx)
					}
				} else {
					if len(cmdArgs.At) == 1 {
						// 单人
						ctx2, _ = cmdArgs.At[0].CopyCtx(ctx)
					} else if len(cmdArgs.At) == 2 {
						ctx1, _ = cmdArgs.At[0].CopyCtx(ctx)
						ctx2, _ = cmdArgs.At[1].CopyCtx(ctx)
					}
				}

				restText := cmdArgs.CleanArgs
				var lastMatched string
				readOneVal := func(mctx *MsgContext) (*CmdExecuteResult, int64, string, string) {
					r, _, err := DiceExprEvalBase(mctx, restText, RollExtraFlags{
						CocVarNumberMode: true,
						CocDefaultAttrOn: true,
						DisableBlock:     true,
					})

					if err != nil {
						ReplyToSender(ctx, msg, "解析出错: "+restText)
						return &CmdExecuteResult{Matched: true, Solved: true}, 0, "", ""
					}
					val, ok := r.ReadInt()
					if !ok {
						ReplyToSender(ctx, msg, "类型不是数字: "+r.GetMatched())
						return &CmdExecuteResult{Matched: true, Solved: true}, 0, "", ""
					}
					lastMatched = r.GetMatched()
					restText = r.GetRestInput()
					return nil, int64(val), r.GetCocPrefix(), r.GetMatched()
				}

				readOneOutcomeVal := func(mctx *MsgContext) (*CmdExecuteResult, int64, string) {
					restText = strings.TrimSpace(restText)
					if strings.HasPrefix(restText, ",") || strings.HasPrefix(restText, "，") {
						re := regexp.MustCompile(`[,，](.*)`)
						m := re.FindStringSubmatch(restText)
						restText = m[1]
						r, detail, err := DiceExprEvalBase(mctx, restText, RollExtraFlags{DisableBlock: true})
						if err != nil {
							ReplyToSender(ctx, msg, "解析出错: "+restText)
							return &CmdExecuteResult{Matched: true, Solved: true}, 0, ""
						}
						val, ok := r.ReadInt()
						if !ok {
							ReplyToSender(ctx, msg, "类型不是数字: "+r.GetMatched())
							return &CmdExecuteResult{Matched: true, Solved: true}, 0, ""
						}
						restText = r.GetRestInput()
						return nil, int64(val), "[" + detail + "]"
					}
					return nil, DiceRoll64(100), ""
				}

				ret, val1, difficult1, expr1 := readOneVal(ctx1)
				if ret != nil {
					return *ret
				}
				ret, outcome1, rollDetail1 := readOneOutcomeVal(ctx1)
				if ret != nil {
					return *ret
				}

				if restText == "" {
					restText = lastMatched
				}

				// lastMatched
				ret, val2, difficult2, expr2 := readOneVal(ctx2)
				if ret != nil {
					return *ret
				}
				ret, outcome2, rollDetail2 := readOneOutcomeVal(ctx2)
				if ret != nil {
					return *ret
				}

				cocRule := ctx.Group.CocRuleIndex
				if cmdArgs.Command == "rcv" {
					// 强制规则书
					cocRule = 0
				}

				successRank1, _ := ResultCheck(ctx, cocRule, outcome1, val1, 0)
				difficultyRequire1 := difficultyPrefixMap[difficult1]
				checkPass1 := successRank1 >= difficultyRequire1 // A是否通过检定

				successRank2, _ := ResultCheck(ctx, cocRule, outcome2, val2, 0)
				difficultyRequire2 := difficultyPrefixMap[difficult2]
				checkPass2 := successRank2 >= difficultyRequire2 // B是否通过检定

				winNum := 0
				switch {
				case checkPass1 && checkPass2:
					if successRank1 > successRank2 {
						// A 胜出
						winNum = -1
					} else if successRank1 < successRank2 {
						// B 胜出
						winNum = 1
					} else { //nolint:gocritic
						// 这里状况复杂，属性检定时，属性高的人胜出
						// 攻击时，成功等级相同，视为被攻击者胜出(目标选择闪避)
						// 攻击时，成功等级相同，视为攻击者胜出(目标选择反击)
						// 技能高的人胜出

						if cocRule == 11 {
							// dg规则下，似乎并不区分情况，比骰点大小即可
							if outcome1 < outcome2 {
								winNum = -1
							}
							if outcome1 > outcome2 {
								winNum = 1
							}
						} /* else {
							这段代码不能使用，因为如果是反击，那么技能是相同的，然而攻击方必胜
							reX := regexp.MustCompile("\\d+$")
							expr1X := reX.ReplaceAllString(expr1, "")
							expr2X := reX.ReplaceAllString(expr2, "")
							if expr1X != "" && expr1X == expr2X {
								if val1 > val2 {
									winNum = -1
								}
								if val1 < val2 {
									winNum = 1
								}
							}
						} */
					}
				case checkPass1 && !checkPass2:
					winNum = -1 // A胜
				case !checkPass1 && checkPass2:
					winNum = 1 // B胜
				default: /*no-op*/
				}

				suffix1 := GetResultTextWithRequire(ctx1, successRank1, difficultyRequire1, true)
				suffix2 := GetResultTextWithRequire(ctx2, successRank2, difficultyRequire2, true)

				p1Name := ctx1.Player.Name
				p2Name := ctx2.Player.Name
				if p1Name == "" {
					p1Name = "玩家A"
				}
				if p2Name == "" {
					p2Name = "玩家B"
				}

				VarSetValueStr(ctx, "$t玩家A", p1Name)
				VarSetValueStr(ctx, "$t玩家B", p2Name)

				VarSetValueStr(ctx, "$t玩家A判定式", expr1)
				VarSetValueStr(ctx, "$t玩家B判定式", expr2)

				VarSetValueInt64(ctx, "$t玩家A属性", val1) // 这个才是真正的判定值（判定线值，属性+难度影响）
				VarSetValueInt64(ctx, "$t玩家B属性", val2)

				VarSetValueInt64(ctx, "$t玩家A判定值", outcome1) // 这两个由下面的替换
				VarSetValueInt64(ctx, "$t玩家B判定值", outcome2) // 这两个由下面的替换
				VarSetValueInt64(ctx, "$t玩家A出目", outcome1)
				VarSetValueInt64(ctx, "$t玩家B出目", outcome2)

				VarSetValueStr(ctx, "$t玩家A判定过程", rollDetail1)
				VarSetValueStr(ctx, "$t玩家B判定过程", rollDetail2)

				VarSetValueStr(ctx, "$t玩家A判定结果", suffix1)
				VarSetValueStr(ctx, "$t玩家B判定结果", suffix2)

				VarSetValueInt64(ctx, "$tWinFlag", int64(winNum))

				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:对抗检定"))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdSt := getCmdStBase(CmdStOverrideInfo{})

	helpEn := `.en <技能名称>[<技能点数>] [+[<失败成长值>/]<成功成长值>] // 整体格式，可以直接看下面几个分解格式
.en <技能名称> // 骰D100，若点数大于当前值，属性成长1d10
.en <技能名称>[<技能点数>] // 骰D100，若点数大于技能点数，属性=技能点数+1d10
.en <技能名称>[<技能点数>] +<成功成长值> // 骰D100，若点数大于当前值，属性成长成功成长值点
.en <技能名称>[<技能点数>] +<失败成长值>/<成功成长值> // 骰D100，若点数大于当前值，属性成长成功成长值点，否则增加失败
.en <技能名称1> <技能名称2> // 批量技能成长，支持上述多种格式，复杂情况建议用|隔开每个技能`

	cmdEn := &CmdItemInfo{
		Name:          "en",
		ShortHelp:     helpEn,
		Help:          "成长指令:\n" + helpEn,
		AllowDelegate: false,
		Solve: func(mctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			// .en [技能名称]([技能值])+(([失败成长值]/)[成功成长值])
			// FIXME: 实在是被正则绕晕了，把多组和每组的正则分开了
			re := regexp.MustCompile(`([a-zA-Z_\p{Han}]+)\s*(\d+)?\s*(\+\s*([-+\ddD]+\s*/)?\s*([-+\ddD]+))?[^|]*?`)
			// 支持多组技能成长
			skills := re.FindAllString(cmdArgs.CleanArgs, -1)

			type enCheckResult struct {
				valid         bool
				invalidReason error

				varName     string
				varValueStr string
				successExpr string
				failExpr    string

				varValue    int64
				rollValue   int64
				successRank int
				success     bool
				resultText  string
				increment   int64
				newVarValue int64
			}
			RuleNotMatch := errors.New("rule not match")
			FormatMismatch := errors.New("format mismatch")
			SkillNotEntered := errors.New("skill not entered")
			SkillTypeError := errors.New("skill value type error")
			SuccessExprFormatError := errors.New("success expr format error")
			FailExprFormatError := errors.New("fail expr format error")
			singleRe := regexp.MustCompile(`([a-zA-Z_\p{Han}]+)\s*(\d+)?\s*(\+(([^/]+)/)?\s*(.+))?`)
			check := func(skill string) (checkResult enCheckResult) {
				checkResult.valid = true
				m := singleRe.FindStringSubmatch(skill)
				tmpl := cardRuleCheck(mctx, msg)
				if tmpl == nil {
					checkResult.valid = false
					checkResult.invalidReason = RuleNotMatch
					return checkResult
				}

				if m == nil {
					checkResult.valid = false
					checkResult.invalidReason = FormatMismatch
					return checkResult
				}

				varName := m[1]     // 技能名称
				varValueStr := m[2] // 技能值 - 字符串
				successExpr := m[6] // 成功的加值表达式
				failExpr := m[5]    // 失败的加值表达式

				var varValue int64
				checkResult.varName = varName
				checkResult.varValueStr = varValueStr

				// 首先，试图读取技能的值
				if varValueStr != "" {
					varValue, _ = strconv.ParseInt(varValueStr, 10, 64)
				} else {
					val, exists := VarGetValue(mctx, varName)
					if !exists {
						// 没找到，尝试取得默认值
						var valX *ds.VMValue
						valX, _, _, exists = tmpl.GetDefaultValueEx0(mctx, varName)
						if exists {
							val = valX
						}
					}
					if !exists {
						checkResult.valid = false
						checkResult.invalidReason = SkillNotEntered
						return checkResult
					}
					if val.TypeId != ds.VMTypeInt {
						checkResult.valid = false
						checkResult.invalidReason = SkillTypeError
						return checkResult
					}
					varValue = int64(val.MustReadInt())
				}

				d100 := DiceRoll64(100)
				// 注意一下，这里其实是，小于失败 大于成功
				successRank, _ := ResultCheck(mctx, mctx.Group.CocRuleIndex, d100, varValue, 0)
				var resultText string
				// 若玩家投出了高于当前技能值的结果，或者结果大于95，则调查员该技能获得改善：骰1D10并且立即将结果加到当前技能值上。技能可通过此方式超过100%。
				if d100 > 95 {
					successRank = -1
				}
				var success bool
				if successRank > 0 {
					resultText = "失败"
					success = false
				} else {
					resultText = "成功"
					success = true
				}

				checkResult.rollValue = d100
				checkResult.varValue = varValue
				checkResult.resultText = resultText
				checkResult.successRank = successRank
				checkResult.success = success

				if success {
					if successExpr == "" {
						successExpr = "1d10"
					}

					r, _, err := DiceExprEvalBase(mctx, successExpr, RollExtraFlags{DisableBlock: true})
					checkResult.successExpr = successExpr
					if err != nil {
						checkResult.valid = false
						checkResult.invalidReason = SuccessExprFormatError
						return checkResult
					}

					increment := int64(r.MustReadInt())
					checkResult.increment = increment
					checkResult.newVarValue = varValue + increment
				} else {
					if failExpr == "" {
						checkResult.increment = 0
						checkResult.newVarValue = varValue
					} else {
						r, _, err := DiceExprEvalBase(mctx, failExpr, RollExtraFlags{})
						checkResult.failExpr = failExpr
						if err != nil {
							checkResult.valid = false
							checkResult.invalidReason = FailExprFormatError
							return checkResult
						}

						increment := int64(r.MustReadInt())
						checkResult.increment = increment
						checkResult.newVarValue = varValue + increment
					}
				}
				return checkResult
			}

			VarSetValueInt64(mctx, "$t数量", int64(len(skills)))
			if len(skills) < 1 { //nolint:nestif
				ReplyToSender(mctx, msg, "指令格式不匹配")
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if len(skills) > 10 {
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_批量_技能过多警告"))
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if len(skills) == 1 {
				checkResult := check(skills[0])
				VarSetValueStr(mctx, "$t技能", checkResult.varName)
				VarSetValueInt64(mctx, "$tD100", checkResult.rollValue)
				VarSetValueInt64(mctx, "$t骰子出目", checkResult.rollValue)
				VarSetValueInt64(mctx, "$t检定结果", checkResult.rollValue)
				VarSetValueInt64(mctx, "$t判定值", checkResult.varValue)
				VarSetValueStr(mctx, "$t判定结果", checkResult.resultText)
				VarSetValueInt64(mctx, "$tSuccessRank", int64(checkResult.successRank))
				VarSetValueInt64(mctx, "$t旧值", checkResult.varValue)
				VarSetValueInt64(mctx, "$t增量", checkResult.increment)
				VarSetValueInt64(mctx, "$t新值", checkResult.newVarValue)
				if checkResult.valid {
					VarSetValueInt64(mctx, checkResult.varName, checkResult.newVarValue)
					if checkResult.success {
						VarSetValueStr(mctx, "$t表达式文本", checkResult.successExpr)
						VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_成功"))
					} else {
						VarSetValueStr(mctx, "$t表达式文本", checkResult.failExpr)
						if checkResult.failExpr == "" {
							VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败"))
						} else {
							VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败变更"))
						}
					}
					VarSetValueInt64(mctx, "$t数量", int64(1))

					VarSetValueStr(mctx, "$t当前绑定角色", lo.Must(mctx.Dice.AttrsManager.LoadByCtx(mctx)).Name)
					if mctx.Player.AutoSetNameTemplate != "" {
						_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
					}
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长"))
				} else {
					switch {
					case errors.Is(checkResult.invalidReason, RuleNotMatch):
						// skip
						return CmdExecuteResult{Matched: true, Solved: true}
					case errors.Is(checkResult.invalidReason, FormatMismatch):
						ReplyToSender(mctx, msg, "指令格式不匹配")
						return CmdExecuteResult{Matched: true, Solved: true}
					case errors.Is(checkResult.invalidReason, SkillNotEntered):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_属性未录入"))
					case errors.Is(checkResult.invalidReason, SkillTypeError):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的属性类型"))
					case errors.Is(checkResult.invalidReason, SuccessExprFormatError):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的成功成长值"))
					case errors.Is(checkResult.invalidReason, FailExprFormatError):
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_错误的失败成长值"))
					}
				}
			} else {
				var checkResultStrs []string
				var checkResults []enCheckResult
				for _, skill := range skills {
					checkResult := check(skill)
					checkResults = append(checkResults, checkResult)
				}
				for _, checkResult := range checkResults {
					VarSetValueStr(mctx, "$t技能", checkResult.varName)
					VarSetValueInt64(mctx, "$tD100", checkResult.rollValue)
					VarSetValueInt64(mctx, "$t骰子出目", checkResult.rollValue)
					VarSetValueInt64(mctx, "$t检定结果", checkResult.rollValue)
					VarSetValueInt64(mctx, "$t判定值", checkResult.varValue)
					VarSetValueStr(mctx, "$t判定结果", checkResult.resultText)
					VarSetValueInt64(mctx, "$tSuccessRank", int64(checkResult.successRank))
					VarSetValueInt64(mctx, "$t旧值", checkResult.varValue)
					VarSetValueInt64(mctx, "$t增量", checkResult.increment)
					VarSetValueInt64(mctx, "$t新值", checkResult.newVarValue)
					if checkResult.valid {
						VarSetValueInt64(mctx, checkResult.varName, checkResult.newVarValue)
						if checkResult.success {
							VarSetValueStr(mctx, "$t表达式文本", checkResult.successExpr)
							VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_成功_无后缀"))
						} else {
							VarSetValueStr(mctx, "$t表达式文本", checkResult.failExpr)
							if checkResult.failExpr == "" {
								VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败"))
							} else {
								VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:技能成长_结果_失败变更_无后缀"))
							}
						}
						resStr := DiceFormatTmpl(mctx, "COC:技能成长_批量_单条")
						checkResultStrs = append(checkResultStrs, resStr)
					} else {
						temp := DiceFormatTmpl(mctx, "COC:技能成长_批量_单条错误前缀")
						switch {
						case errors.Is(checkResult.invalidReason, RuleNotMatch):
							// skip
							return CmdExecuteResult{Matched: true, Solved: true}
						case errors.Is(checkResult.invalidReason, FormatMismatch):
							ReplyToSender(mctx, msg, "指令格式不匹配")
							return CmdExecuteResult{Matched: true, Solved: true}
						case errors.Is(checkResult.invalidReason, SkillNotEntered):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_属性未录入_无前缀")
						case errors.Is(checkResult.invalidReason, SkillTypeError):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_错误的属性类型_无前缀")
						case errors.Is(checkResult.invalidReason, SuccessExprFormatError):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_错误的成功成长值_无前缀")
						case errors.Is(checkResult.invalidReason, FailExprFormatError):
							temp += DiceFormatTmpl(mctx, "COC:技能成长_错误的失败成长值_无前缀")
						}
						checkResultStrs = append(checkResultStrs, temp)
					}
				}
				sep := DiceFormatTmpl(mctx, "COC:技能成长_批量_分隔符")
				resultStr := strings.Join(checkResultStrs, sep)
				VarSetValueStr(mctx, "$t总结果文本", resultStr)
				VarSetValueStr(mctx, "$t当前绑定角色", lo.Must(mctx.Dice.AttrsManager.LoadByCtx(mctx)).Name)
				if mctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
				}
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:技能成长_批量"))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdTi := &CmdItemInfo{
		Name:      "ti",
		ShortHelp: ".ti // 抽取一个临时性疯狂症状",
		Help:      "抽取临时性疯狂症状:\n.ti // 抽取一个临时性疯狂症状",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			num := DiceRoll(10)
			VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("1D10=%d", num))
			VarSetValueInt64(ctx, "$t选项值", int64(num))

			var desc string
			extraNum1 := DiceRoll(10)
			VarSetValueInt64(ctx, "$t附加值1", int64(extraNum1))
			switch num {
			case 1:
				desc += fmt.Sprintf("失忆：调查员会发现自己只记得最后身处的安全地点，却没有任何来到这里的记忆。例如，调查员前一刻还在家中吃着早饭，下一刻就已经直面着不知名的怪物。这将会持续 1D10=%d 轮。", extraNum1)
			case 2:
				desc += fmt.Sprintf("假性残疾：调查员陷入了心理性的失明，失聪以及躯体缺失感中，持续 1D10=%d 轮。", extraNum1)
			case 3:
				desc += fmt.Sprintf("暴力倾向：调查员陷入了六亲不认的暴力行为中，对周围的敌人与友方进行着无差别的攻击，持续 1D10=%d 轮。", extraNum1)
			case 4:
				desc += fmt.Sprintf("偏执：调查员陷入了严重的偏执妄想之中。有人在暗中窥视着他们，同伴中有人背叛了他们，没有人可以信任，万事皆虚。持续 1D10=%d 轮", extraNum1)
			case 5:
				desc += fmt.Sprintf("人际依赖：守秘人适当参考调查员的背景中重要之人的条目，调查员因为一些原因而将他人误认为了他重要的人并且努力的会与那个人保持那种关系，持续 1D10=%d 轮", extraNum1)
			case 6:
				desc += fmt.Sprintf("昏厥：调查员当场昏倒，并需要 1D10=%d 轮才能苏醒。", extraNum1)
			case 7:
				desc += fmt.Sprintf("逃避行为：调查员会用任何的手段试图逃离现在所处的位置，即使这意味着开走唯一一辆交通工具并将其它人抛诸脑后，调查员会试图逃离 1D10=%d 轮。", extraNum1)
			case 8:
				desc += fmt.Sprintf("歇斯底里：调查员表现出大笑，哭泣，嘶吼，害怕等的极端情绪表现，持续 1D10=%d 轮。", extraNum1)
			case 9:
				desc += fmt.Sprintf("恐惧：调查员通过一次 D100 或者由守秘人选择，来从恐惧症状表中选择一个恐惧源，就算这一恐惧的事物是并不存在的，调查员的症状会持续 1D10=%d 轮。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += fearMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			case 10:
				desc += fmt.Sprintf("躁狂：调查员通过一次 D100 或者由守秘人选择，来从躁狂症状表中选择一个躁狂的诱因，这个症状将会持续 1D10=%d 轮。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += maniaMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			}
			VarSetValueStr(ctx, "$t疯狂描述", desc)

			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:疯狂发作_即时症状"))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdLi := &CmdItemInfo{
		Name:      "li",
		ShortHelp: ".li // 抽取一个总结性疯狂症状",
		Help:      "抽取总结性疯狂症状:\n.li // 抽取一个总结性疯狂症状",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			num := DiceRoll(10)
			VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("1D10=%d", num))
			VarSetValueInt64(ctx, "$t选项值", int64(num))

			var desc string
			extraNum1 := DiceRoll(10)
			VarSetValueInt64(ctx, "$t附加值1", int64(extraNum1))
			switch num {
			case 1:
				desc += "失忆：回过神来，调查员们发现自己身处一个陌生的地方，并忘记了自己是谁。记忆会随时间恢复。"
			case 2:
				desc += fmt.Sprintf("被窃：调查员在 1D10=%d 小时后恢复清醒，发觉自己被盗，身体毫发无损。如果调查员携带着宝贵之物（见调查员背景），做幸运检定来决定其是否被盗。所有有价值的东西无需检定自动消失。", extraNum1)
			case 3:
				desc += fmt.Sprintf("遍体鳞伤：调查员在 1D10=%d 小时后恢复清醒，发现自己身上满是拳痕和瘀伤。生命值减少到疯狂前的一半，但这不会造成重伤。调查员没有被窃。这种伤害如何持续到现在由守秘人决定。", extraNum1)
			case 4:
				desc += "暴力倾向：调查员陷入强烈的暴力与破坏欲之中。调查员回过神来可能会理解自己做了什么也可能毫无印象。调查员对谁或何物施以暴力，他们是杀人还是仅仅造成了伤害，由守秘人决定。"
			case 5:
				desc += "极端信念：查看调查员背景中的思想信念，调查员会采取极端和疯狂的表现手段展示他们的思想信念之一。比如一个信教者会在地铁上高声布道。"
			case 6:
				desc += fmt.Sprintf("重要之人：考虑调查员背景中的重要之人，及其重要的原因。在 1D10=%d 小时或更久的时间中，调查员将不顾一切地接近那个人，并为他们之间的关系做出行动。", extraNum1)
			case 7:
				desc += "被收容：调查员在精神病院病房或警察局牢房中回过神来，他们可能会慢慢回想起导致自己被关在这里的事情。"
			case 8:
				desc += "逃避行为：调查员恢复清醒时发现自己在很远的地方，也许迷失在荒郊野岭，或是在驶向远方的列车或长途汽车上。"
			case 9:
				desc += fmt.Sprintf("恐惧：调查员患上一个新的恐惧症状。在恐惧症状表上骰 1 个 D100 来决定症状，或由守秘人选择一个。调查员在 1D10=%d 小时后回过神来，并开始为避开恐惧源而采取任何措施。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += fearMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			case 10:
				desc += fmt.Sprintf("狂躁：调查员患上一个新的狂躁症状。在狂躁症状表上骰 1 个 d100 来决定症状，或由守秘人选择一个。调查员会在 1D10=%d 小时后恢复理智。在这次疯狂发作中，调查员将完全沉浸于其新的狂躁症状。这症状是否会表现给旁人则取决于守秘人和此调查员。", extraNum1)
				extraNum2 := DiceRoll(100)
				desc += fmt.Sprintf("\n1D100=%d\n", extraNum2)
				desc += maniaMap[extraNum2]
				VarSetValueInt64(ctx, "$t附加值2", int64(extraNum2))
			}
			VarSetValueStr(ctx, "$t疯狂描述", desc)

			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:疯狂发作_总结症状"))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSc := ".sc <成功时掉san>/<失败时掉san> // 对理智进行一次D100检定，根据结果扣除理智\n" +
		".sc <失败时掉san> //同上，简易写法 \n" +
		".sc [b|p] [<成功时掉san>/]<失败时掉san> // 加上奖惩骰"
	cmdSc := &CmdItemInfo{
		Name:          "sc",
		ShortHelp:     helpSc,
		Help:          "理智检定:\n" + helpSc,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			// http://www.antagonistes.com/files/CoC%20CheatSheet.pdf
			// v2: (worst) FAIL — REGULAR SUCCESS — HARD SUCCESS — EXTREME SUCCESS (best)

			if len(cmdArgs.Args) == 0 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			mctx := GetCtxProxyFirst(ctx, cmdArgs)

			tmpl := cardRuleCheck(mctx, msg)
			if tmpl == nil {
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			mctx.SystemTemplate = tmpl

			// 首先读取一个值
			// 试图读取 /: 读到了，当前是成功值，转入读取单项流程，试图读取失败值
			// 试图读取 ,: 读到了，当前是失败值，试图转入下一项
			// 试图读取表达式: 读到了，当前是判定值

			defaultSuccessExpr := "0"
			argText := cmdArgs.CleanArgs
			argCap := cmdArgs.GetKwarg("cap")
			argHalf := cmdArgs.GetKwarg("half")

			diceExpr, lossSucc, lossFail, status := func() (string, string, string, int) {
				expr1 := "d100" // 先假设为常见情况，也就是D100
				expr2 := ""
				expr3 := ""

				splitSlash := func(text string) (int, string, string) {
					ret := strings.SplitN(text, "/", 2)
					if len(ret) == 1 {
						return 1, ret[0], ""
					}
					return 2, ret[0], ret[1]
				}

				parseStatus := func() int {
					var err error
					r, _, err := DiceExprEvalBase(mctx, argText, RollExtraFlags{IgnoreDiv0: true, DisableBlock: true})
					if err != nil {
						// 情况1，完全不能解析
						return 1
					}

					num, t1, t2 := splitSlash(r.GetMatched())
					if num == 2 {
						expr2 = t1
						expr3 = t2
						argText = r.GetRestInput()
						return 0
					}

					// 现在可以肯定并非是 .sc 1/1 形式，那么判断一下
					// .sc 1 或 .sc 1 1/1 或 .sc 1 1
					if strings.HasPrefix(r.GetRestInput(), ",") || r.GetRestInput() == "" {
						// 结束了，所以这是 .sc 1
						expr2 = defaultSuccessExpr
						expr3 = r.GetMatched()
						argText = r.GetRestInput()
						return 0
					}

					// 可能是 .sc 1 1 或 .sc 1 1/1
					expr1 = r.GetMatched()
					r2, _, err := DiceExprEvalBase(mctx, r.GetRestInput(), RollExtraFlags{DisableBlock: true})
					if err != nil {
						return 2
					}
					num, t1, t2 = splitSlash(r2.GetMatched())
					if num == 2 {
						// sc 1 1
						expr2 = t1
						expr3 = t2
						argText = r2.GetRestInput()
						return 0
					}

					// sc 1/1
					expr2 = defaultSuccessExpr
					expr3 = t1
					argText = r2.GetRestInput()
					return 0
				}()

				return expr1, expr2, expr3, parseStatus
			}()

			switch status {
			case 1, 2, 3:
				// 1: 这输入的是啥啊，完全不能解析
				// 2: 已经匹配了/，失败扣除血量不正确
				// 3: 第一个式子对了，第二个是啥东西？
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:理智检定_格式错误"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// 完全正确
			var d100 int64
			var san int64

			// 获取判定值
			rCond, detailCond, err := DiceExprEvalBase(mctx, diceExpr, RollExtraFlags{DisableBlock: true})
			if err == nil && rCond.TypeId == ds.VMTypeInt {
				d100 = int64(rCond.MustReadInt())
			}
			detailWrap := ""
			if detailCond != "" {
				if diceExpr != "d100" {
					detailWrap = ", (" + detailCond + ")"
				}
			}

			// 读取san值
			r, _, err := DiceExprEvalBase(mctx, "san", RollExtraFlags{DisableBlock: true})
			if err == nil && r.TypeId == ds.VMTypeInt {
				san = int64(r.MustReadInt())
			}
			_san, err := strconv.ParseInt(strings.TrimSpace(argText), 10, 64)
			if err == nil {
				san = _san
			}

			// 进行检定
			successRank, _ := ResultCheck(mctx, mctx.Group.CocRuleIndex, d100, san, 0)
			suffix := GetResultText(mctx, successRank, false)
			suffixShort := GetResultText(mctx, successRank, true)

			var textExpr string
			if successRank > 0 {
				textExpr = lossSucc
			} else {
				textExpr = lossFail
			}

			var sanLossOri, sanLoss int64
			r, _, err = DiceExprEvalBase(mctx, textExpr, RollExtraFlags{BigFailDiceOn: successRank == -2, DisableBlock: true})
			if err == nil {
				v, _ := r.ReadInt()
				sanLossOri = int64(v)
				sanLoss = sanLossOri
			}

			if argHalf != nil {
				sanLoss /= 2
			}

			if argCap != nil {
				if lossCap, errParseInt := strconv.ParseInt(argCap.Value, 10, 64); errParseInt == nil {
					if lossCap > 0 && sanLoss > lossCap {
						sanLoss = lossCap
					}
				}
			}

			sanNew := san - sanLoss
			if sanNew < 0 {
				sanNew = 0
			}

			VarSetValueStr(mctx, "$t检定表达式文本", diceExpr)
			VarSetValueStr(mctx, "$t检定计算过程", detailWrap)

			VarSetValueInt64(mctx, "$tD100", d100)
			VarSetValueInt64(mctx, "$t骰子出目", d100)
			VarSetValueInt64(mctx, "$t检定结果", d100)
			VarSetValueInt64(mctx, "$t判定值", san)
			VarSetValueStr(mctx, "$t判定结果", suffix)
			VarSetValueStr(mctx, "$t判定结果_详细", suffix)
			VarSetValueStr(mctx, "$t判定结果_简短", suffixShort)
			VarSetValueInt64(ctx, "$tSuccessRank", int64(successRank))
			VarSetValueInt64(mctx, "$t旧值", san)

			SetTempVars(mctx, mctx.Player.Name) // 信息里没有QQ昵称，用这个顶一下
			VarSetValueStr(mctx, "$t结果文本", DiceFormatTmpl(mctx, "COC:理智检定_单项结果文本"))

			VarSetValueInt64(mctx, mctx.Player.GetValueNameByAlias("理智", tmpl.Alias), sanNew)

			// 输出结果
			VarSetValueInt64(mctx, "$t新值", sanNew)
			VarSetValueStr(mctx, "$t表达式文本", textExpr)
			VarSetValueInt64(mctx, "$t表达式值", sanLoss) // For Compatibility
			VarSetValueInt64(mctx, "$t表达式原始值", sanLossOri)
			VarSetValueInt64(mctx, "$t表达式调整值", sanLoss)

			var crazyTip string
			if sanNew == 0 {
				crazyTip += DiceFormatTmpl(mctx, "COC:提示_永久疯狂") + "\n"
			} else if sanLoss >= 5 {
				crazyTip += DiceFormatTmpl(mctx, "COC:提示_临时疯狂") + "\n"
			}
			VarSetValueStr(mctx, "$t提示_角色疯狂", crazyTip)

			switch successRank {
			case -2:
				VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_大失败"))
			case -1:
				VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_失败"))
			case 1, 2, 3:
				VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_成功"))
			case 4:
				VarSetValueStr(mctx, "$t附加语", DiceFormatTmpl(ctx, "COC:理智检定_附加语_大成功"))
			default:
				VarSetValueStr(mctx, "$t附加语", "")
			}

			// 指令信息
			commandInfo := map[string]interface{}{
				"cmd":     "sc",
				"rule":    "coc7",
				"pcName":  mctx.Player.Name,
				"cocRule": mctx.Group.CocRuleIndex,
				"items": []any{
					map[string]any{
						"outcome": d100,
						"exprs":   []string{diceExpr, lossSucc, lossFail},
						"rank":    successRank,
						"sanOld":  san,
						"sanNew":  sanNew,
					},
				},
			}
			ctx.CommandInfo = commandInfo

			text := DiceFormatTmpl(mctx, "COC:理智检定")
			if kw := cmdArgs.GetKwarg("ci"); kw != nil {
				info, err := json.Marshal(ctx.CommandInfo)
				if err == nil {
					text += "\n" + string(info)
				} else {
					text += "\n" + "指令信息无法序列化"
				}
			}

			ReplyToSender(mctx, msg, text)
			if mctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdCoc := &CmdItemInfo{
		Name:      "coc",
		ShortHelp: ".coc [<数量>] // 制卡指令，返回<数量>组人物属性",
		Help:      "COC制卡指令:\n.coc [<数量>] // 制卡指令，返回<数量>组人物属性",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			n := cmdArgs.GetArgN(1)
			val, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				if n == "" {
					val = 1 // 数量不存在时，视为1次
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
			}
			if val > ctx.Dice.Config.MaxCocCardGen {
				val = ctx.Dice.Config.MaxCocCardGen
			}

			var ss []string
			for range val {
				result := ctx.EvalFString(`力量:{力量=3d6*5} 敏捷:{敏捷=3d6*5} 意志:{意志=3d6*5}\n体质:{体质=3d6*5} 外貌:{外貌=3d6*5} 教育:{教育=(2d6+6)*5}\n体型:{体型=(2d6+6)*5} 智力:{智力=(2d6+6)*5} 幸运:{幸运=3d6*5}\nHP:{(体质+体型)/10} <DB:{(力量 + 体型) < 65 ? -2, (力量 + 体型) < 85 ? -1, (力量 + 体型) < 125 ? 0, (力量 + 体型) < 165 ? '1d4', (力量 + 体型) < 205 ? '1d6'}> [{力量+敏捷+意志+体质+外貌+教育+体型+智力}/{力量+敏捷+意志+体质+外貌+教育+体型+智力+幸运}]`, nil)
				if result.vm.Error != nil {
					break
				}
				resultText := result.ToString()
				resultText = strings.ReplaceAll(resultText, `\n`, "\n")
				ss = append(ss, resultText)
			}
			sep := DiceFormatTmpl(ctx, "COC:制卡_分隔符")
			info := strings.Join(ss, sep)
			VarSetValueStr(ctx, "$t制卡结果文本", info)
			text := DiceFormatTmpl(ctx, "COC:制卡")
			// fmt.Sprintf("<%s>的七版COC人物作成:\n%s", ctx.Player.Name, info)
			if ctx.Dice.Config.CocCardMergeForward {
				title := fmt.Sprintf("<%s>的COC7制卡结果", ctx.Player.Name)
				if TryReplyToSenderMergedForward(ctx, msg, title, ss) {
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}
			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	theExt := &ExtInfo{
		Name:       "coc7",
		Version:    "1.0.0",
		Brief:      "第七版克苏鲁的呼唤TRPG跑团指令集",
		AutoActive: true,
		Author:     "木落",
		Official:   true,
		ConflictWith: []string{
			"dnd5e",
		},
		OnLoad: func() {

		},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"en":     cmdEn,
			"setcoc": cmdSetCOC,
			"ti":     cmdTi,
			"li":     cmdLi,
			"ra":     cmdRc,
			"rc":     cmdRc,
			"rch":    cmdRc,
			"rah":    cmdRc,
			"cra":    cmdRc,
			"crc":    cmdRc,
			"crch":   cmdRc,
			"crah":   cmdRc,
			"rav":    cmdRcv,
			"rcv":    cmdRcv,
			"sc":     cmdSc,
			"coc":    cmdCoc,
			"st":     cmdSt,
			"cst":    cmdSt,
		},
	}
	self.RegisterExtension(theExt)
}

func GetResultTextWithRequire(ctx *MsgContext, successRank int, difficultyRequire int, userShortVersion bool) string {
	if difficultyRequire > 1 {
		isSuccess := successRank >= difficultyRequire

		if successRank > difficultyRequire && successRank == 4 {
			// 大成功
			VarSetValueStr(ctx, "$t附加判定结果", fmt.Sprintf("(%s)", GetResultText(ctx, successRank, true)))
		} else if !isSuccess && successRank == -2 {
			// 大失败
			VarSetValueStr(ctx, "$t附加判定结果", fmt.Sprintf("(%s)", GetResultText(ctx, successRank, true)))
		} else {
			VarSetValueStr(ctx, "$t附加判定结果", "")
		}

		var suffix string
		switch difficultyRequire {
		case +2:
			if isSuccess {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_困难_成功")
			} else {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_困难_失败")
			}
		case +3:
			if isSuccess {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_极难_成功")
			} else {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_极难_失败")
			}
		case +4:
			if isSuccess {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_大成功_成功")
			} else {
				suffix = DiceFormatTmpl(ctx, "COC:判定_必须_大成功_失败")
			}
		}
		return suffix
	}
	return GetResultText(ctx, successRank, userShortVersion)
}

func GetResultText(ctx *MsgContext, successRank int, userShortVersion bool) string {
	var suffix string
	if userShortVersion {
		switch successRank {
		case -2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_大失败")
		case -1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_失败")
		case +1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_成功_普通")
		case +2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_成功_困难")
		case +3:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_成功_极难")
		case +4:
			suffix = DiceFormatTmpl(ctx, "COC:判定_简短_大成功")
		}
	} else {
		switch successRank {
		case -2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_大失败")
		case -1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_失败")
		case +1:
			suffix = DiceFormatTmpl(ctx, "COC:判定_成功_普通")
		case +2:
			suffix = DiceFormatTmpl(ctx, "COC:判定_成功_困难")
		case +3:
			suffix = DiceFormatTmpl(ctx, "COC:判定_成功_极难")
		case +4:
			suffix = DiceFormatTmpl(ctx, "COC:判定_大成功")
		}
	}
	return suffix
}

type CocRuleCheckRet struct {
	SuccessRank          int   `jsbind:"successRank"`          // 成功级别
	CriticalSuccessValue int64 `jsbind:"criticalSuccessValue"` // 大成功数值
}

type CocRuleInfo struct {
	Index int    `jsbind:"index"` // 序号
	Key   string `jsbind:"key"`   // .setcoc key
	Name  string `jsbind:"name"`  // 已切换至规则 Name: Desc
	Desc  string `jsbind:"desc"`  // 规则描述

	Check func(ctx *MsgContext, d100 int64, checkValue int64, difficultyRequired int) CocRuleCheckRet `jsbind:"check"`
}

func ResultCheck(ctx *MsgContext, cocRule int, d100 int64, attrValue int64, difficultyRequired int) (successRank int, criticalSuccessValue int64) {
	if cocRule >= 20 {
		d := ctx.Dice
		val, exists := d.CocExtraRules[cocRule]
		if !exists {
			cocRule = 0
		} else {
			ret := val.Check(ctx, d100, attrValue, difficultyRequired)
			return ret.SuccessRank, ret.CriticalSuccessValue
		}
	}
	return ResultCheckBase(cocRule, d100, attrValue, difficultyRequired)
}

/*
大失败：骰出 100。若成功需要的值低于 50，大于等于 96 的结果都是大失败 -> -2
失败：骰出大于角色技能或属性值（但不是大失败） -> -1
常规成功：骰出小于等于角色技能或属性值 -> 1
困难成功：骰出小于等于角色技能或属性值的一半 -> 2
极难成功：骰出小于等于角色技能或属性值的五分之一 -> 3
大成功：骰出1 -> 4
*/
func ResultCheckBase(cocRule int, d100 int64, attrValue int64, difficultyRequired int) (successRank int, criticalSuccessValue int64) {
	criticalSuccessValue = int64(1) // 大成功阈值
	fumbleValue := int64(100)       // 大失败阈值

	checkVal := attrValue
	switch difficultyRequired {
	case 2:
		checkVal /= 2
	case 3:
		checkVal /= 5
	case 4:
		checkVal = criticalSuccessValue
	}

	if d100 <= checkVal {
		successRank = 1
	} else {
		successRank = -1
	}

	// 分支规则设定
	switch cocRule {
	case 0:
		// 规则书规则
		// 不满50出96-100大失败，满50出100大失败
		if checkVal < 50 {
			fumbleValue = 96
		}
	case 1:
		// 不满50出1大成功，满50出1-5大成功
		// 不满50出96-100大失败，满50出100大失败
		if attrValue >= 50 {
			criticalSuccessValue = 5
		}
		if attrValue < 50 {
			fumbleValue = 96
		}
	case 2:
		// 出1-5且<=成功率大成功
		// 出100或出96-99且>成功率大失败
		criticalSuccessValue = 5
		if attrValue < criticalSuccessValue {
			criticalSuccessValue = attrValue
		}
		fumbleValue = 96
		if attrValue >= fumbleValue {
			fumbleValue = attrValue + 1
			if fumbleValue > 100 {
				fumbleValue = 100
			}
		}
	case 3:
		// 出1-5大成功
		// 出100或出96-99大失败
		criticalSuccessValue = 5
		fumbleValue = 96
	case 4:
		// 出1-5且<=成功率/10大成功
		// 不满50出>=96+成功率/10大失败，满50出100大失败
		// 规则4 -> 大成功判定值 = min(5, 判定值/10)，大失败判定值 = min(96+判定值/10, 100)
		criticalSuccessValue = attrValue / 10
		if criticalSuccessValue > 5 {
			criticalSuccessValue = 5
		}
		fumbleValue = 96 + attrValue/10
		if 100 < fumbleValue {
			fumbleValue = 100
		}
	case 5:
		// 出1-2且<成功率/5大成功
		// 不满50出96-100大失败，满50出99-100大失败
		criticalSuccessValue = attrValue / 5
		if criticalSuccessValue > 2 {
			criticalSuccessValue = 2
		}
		if attrValue < 50 {
			fumbleValue = 96
		} else {
			fumbleValue = 99
		}
	case 11: // dg
		criticalSuccessValue = 1
		fumbleValue = 100
	}

	// 成功判定
	if successRank == 1 || d100 <= criticalSuccessValue {
		// 区分大成功、困难成功、极难成功等
		if d100 <= attrValue/2 {
			// suffix = "成功(困难)"
			successRank = 2
		}
		if d100 <= attrValue/5 {
			// suffix = "成功(极难)"
			successRank = 3
		}
		if d100 <= criticalSuccessValue {
			// suffix = "大成功！"
			successRank = 4
		}
	} else if d100 >= fumbleValue {
		// suffix = "大失败！"
		successRank = -2
	}

	if cocRule == 0 || cocRule == 1 || cocRule == 2 {
		if d100 == 1 {
			// 为 1 必是大成功，即使判定线是0
			// 根据群友说法，相关描述见40周年版407页 / 89-90页
			// 保守起见，只在规则0、1、2下生效 [规则1与官方规则相似]
			successRank = 4
		}
	}

	// 默认规则改判，为100必然是大失败
	if d100 == 100 && cocRule == 0 {
		successRank = -2
	}

	// 规则3的改判，强行大成功或大失败
	if cocRule == 3 {
		if d100 <= criticalSuccessValue {
			// suffix = "大成功！"
			successRank = 4
		}
		if d100 >= fumbleValue {
			// suffix = "大失败！"
			successRank = -2
		}
	}

	// 规则DG改判，检定成功基础上，个位十位相同大成功
	// 检定失败基础上，个位十位相同大失败
	if cocRule == 11 {
		numUnits := d100 % 10
		numTens := d100 % 100 / 10
		dgCheck := numUnits == numTens

		if successRank > 0 {
			if dgCheck {
				successRank = 4
			} else {
				successRank = 1 // 抹除困难极难成功
			}
		} else {
			if dgCheck {
				successRank = -2
			} else {
				successRank = -1
			}
		}

		// 23.3 根据dg规则书修正: 为1大成功
		if d100 == 1 {
			successRank = 4
		}
	}

	return successRank, criticalSuccessValue
}
