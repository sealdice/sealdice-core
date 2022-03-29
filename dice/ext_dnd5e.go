package dice

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
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

var dndAttrParent = map[string]string{
	"运动": "力量",

	"特技": "敏捷",
	"巧手": "敏捷",
	"隐匿": "敏捷",

	"调查": "智力",
	"奥秘": "智力",
	"历史": "智力",
	"自然": "智力",
	"宗教": "智力",

	"察觉": "感知",
	"洞悉": "感知",
	"驯养": "感知",
	"医疗": "感知",
	"生存": "感知",

	"说服": "魅力",
	"欺诈": "魅力",
	"威吓": "魅力",
	"表演": "魅力",
}

func setupConfigDND(d *Dice) AttributeConfigs {
	attrConfigFn := d.GetExtConfigFilePath("dnd5e", "attribute.yaml")

	if _, err := os.Stat(attrConfigFn); err == nil && false {
		// 如果文件存在，那么读取
		ac := AttributeConfigs{}
		af, err := ioutil.ReadFile(attrConfigFn)
		if err == nil {
			err = yaml.Unmarshal(af, &ac)
			if err != nil {
				panic(err)
			}
		}
		return ac
	} else {
		// 如果不存在，新建

		defaultVals := AttributeConfigs{
			Alias: map[string][]string{},
			Order: AttributeOrder{
				Top:    []string{"力量", "敏捷", "体质", "智力", "感知", "魅力", "HP", "HPMax", "EXP"},
				Others: AttributeOrderOthers{SortBy: "Name"},
			},
		}

		buf, err := yaml.Marshal(defaultVals)
		if err != nil {
			fmt.Println(err)
		} else {
			ioutil.WriteFile(attrConfigFn, buf, 0644)
		}
		return defaultVals
	}
}

func RegisterBuiltinExtDnd5e(self *Dice) {
	ac := setupConfigDND(self)

	helpSt := ".st 模板 // 录卡模板"
	helpSt += ".st show // 展示个人属性\n"
	helpSt += ".st show <属性1> <属性2> ... // 展示特定的属性数值\n"
	helpSt += ".st show <数字> // 展示高于<数字>的属性，如.st show 30\n"
	helpSt += ".st clr/clear // 清除属性\n"
	helpSt += ".st del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n"
	helpSt += ".st help // 帮助\n"
	helpSt += ".st <属性>:<值> // 设置属性，技能加值会自动计算。例：.st 感知:20 洞悉:3\n"
	helpSt += ".st <属性>±<表达式> // 修改属性，例：.st 生命+1d4"
	helpSt += ".st <属性>±<表达式> @某人 // 修改他人属性，例：.st 生命+1d4"

	cmdSt := &CmdItemInfo{
		Name:     "st",
		Help:     helpSt,
		LongHelp: "DND5E 人物属性设置:\n" + helpSt,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				val, _ := cmdArgs.GetArgN(1)
				mctx := &*ctx // 复制一个ctx，用于其他用途
				if len(cmdArgs.At) > 0 {
					p, exists := ctx.Group.Players[cmdArgs.At[0].UserId]
					if exists {
						mctx.Player = p
					}
				}

				switch val {
				case "模板":
					text := "人物卡模板(第二行文本):\n"
					text += ".dst 力量:10 体质:10 敏捷:10 智力:10 感知:10 魅力:10 HP:10 HPMax: 10 熟练:2 运动:0 特技:0 巧手:0 隐匿:0 调查:0 奥秘:0 历史:0 自然:0 宗教:0 察觉:0 洞悉:0 驯养:0 医疗:0 生存:0 说服:0 欺诈:0 威吓:0 表演:0\n"
					text += "注意: 技能只填写修正值即可，属性调整值会自动计算。熟练写为“运动*:0”"
					ReplyToSender(ctx, msg, text)
					return CmdExecuteResult{Matched: true, Solved: true}
				case "del", "rm":
					var nums []string
					var failed []string

					for _, varname := range cmdArgs.Args[1:] {
						_, ok := mctx.Player.ValueMap[varname]
						if ok {
							nums = append(nums, varname)
							delete(mctx.Player.ValueMap, varname)
						} else {
							failed = append(failed, varname)
						}
					}

					VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
					VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_删除"))

				case "clr", "clear":
					p := ctx.Player
					num := len(p.ValueMap)
					p.ValueMap = map[string]*VMValue{}
					VarSetValueInt64(ctx, "$t数量", int64(num))
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_清除"))

				case "show", "list":
					info := ""
					p := mctx.Player

					useLimit := false
					usePickItem := false
					limktSkipCount := 0
					var limit int64

					if len(cmdArgs.Args) >= 2 {
						arg2, _ := cmdArgs.GetArgN(2)
						_limit, err := strconv.ParseInt(arg2, 10, 64)
						if err == nil {
							limit = _limit
							useLimit = true
						} else {
							usePickItem = true
						}
					}

					pickItems := map[string]int{}

					if usePickItem {
						for _, i := range cmdArgs.Args[1:] {
							key := p.GetValueNameByAlias(i, ac.Alias)
							pickItems[key] = 1
						}
					}

					tick := 0
					if len(p.ValueMap) == 0 {
						info = DiceFormatTmpl(ctx, "DND:属性设置_列出_未发现记录")
					} else {
						// 按照配置文件排序
						attrKeys := []string{}
						used := map[string]bool{}
						for _, i := range ac.Order.Top {
							key := p.GetValueNameByAlias(i, ac.Alias)
							if used[key] {
								continue
							}
							attrKeys = append(attrKeys, key)
							used[key] = true
						}

						// 其余按字典序
						topNum := len(attrKeys)
						attrKeys2 := []string{}
						for k := range p.ValueMap {
							attrKeys2 = append(attrKeys2, k)
						}
						sort.Strings(attrKeys2)
						for _, key := range attrKeys2 {
							if used[key] {
								continue
							}
							attrKeys = append(attrKeys, key)
						}

						// 遍历输出
						for index, k := range attrKeys {
							if strings.HasPrefix(k, "$") {
								continue
							}
							v, exists := p.ValueMap[k]
							if !exists {
								// 不存在的值，强行补0
								v = &VMValue{VMTypeInt64, int64(0)}
							}

							if index >= topNum {
								if useLimit && v.TypeId == VMTypeInt64 && v.Value.(int64) < limit {
									limktSkipCount += 1
									continue
								}
							}

							if usePickItem {
								_, ok := pickItems[k]
								if !ok {
									continue
								}
							}

							tick += 1
							vText := ""
							if v.TypeId == VMTypeComputedValue {
								vd := v.Value.(*VMComputedValueData)
								val, _, _ := ctx.Dice.ExprEvalBase(k, mctx, RollExtraFlags{})
								vText = fmt.Sprintf("%s[基础值%s]", val.ToString(), vd.BaseValue.ToString())
							} else {
								vText = v.ToString()
							}
							info += fmt.Sprintf("%s: %s\t", k, vText)
							if tick%4 == 0 {
								info += fmt.Sprintf("\n")
							}
						}

						if info == "" {
							info = DiceFormatTmpl(ctx, "DND:属性设置_列出_未发现记录")
						}
					}

					if useLimit {
						VarSetValueInt64(ctx, "$t数量", int64(limktSkipCount))
						VarSetValueInt64(ctx, "$t判定值", int64(limit))
						info += DiceFormatTmpl(ctx, "DND:属性设置_列出_隐藏提示")
					}

					VarSetValueStr(ctx, "$t属性信息", info)
					extra := ReadCardType(mctx, "dnd5e")
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:属性设置_列出")+extra)

				case "help", "":
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				default:
					text := cmdArgs.CleanArgs
					re := regexp.MustCompile(`(([^\s:0-9*][^\s:0-9*]*)\*?)\s*([:+\-])`)
					attrSeted := []string{}
					attrChanged := []string{}

					for {
						m := re.FindStringSubmatch(text)
						if len(m) == 0 {
							break
						}
						text = text[len(m[0]):]

						attrName := m[2]
						isSkilled := strings.HasSuffix(m[1], "*")
						r, _, err := ctx.Dice.ExprEvalBase(text, mctx, RollExtraFlags{})
						if err != nil {
							ReplyToSender(ctx, msg, "无法解析属性: "+attrName)
							return CmdExecuteResult{Matched: true, Solved: true}
						}
						text = r.restInput

						if r.TypeId != VMTypeInt64 {
							ReplyToSender(ctx, msg, "这个属性的值并非数字: "+attrName)
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						if m[3] == ":" {
							exprTmpl := "$tVal + %s/2 - 5"
							if isSkilled {
								exprTmpl += " + 熟练"
							}

							parent, _ := dndAttrParent[attrName]
							aText := attrName
							if parent != "" {
								if isSkilled {
									aText += "[技能, 熟练]"
								} else {
									aText += "[技能]"
								}
								VarSetValueDNDComputed(ctx, attrName, r.Value.(int64), fmt.Sprintf(exprTmpl, parent))
							} else {
								VarSetValueInt64(ctx, attrName, r.Value.(int64))
							}
							attrSeted = append(attrSeted, aText)
						}
						if m[3] == "+" || m[3] == "-" {
							v, exists := VarGetValue(ctx, attrName)
							if !exists {
								ReplyToSender(ctx, msg, "不存在的属性: "+attrName)
								return CmdExecuteResult{Matched: true, Solved: true}
							}
							if v.TypeId != VMTypeInt64 && v.TypeId != VMTypeComputedValue {
								ReplyToSender(ctx, msg, "这个属性的值并非数字: "+attrName)
								return CmdExecuteResult{Matched: true, Solved: true}
							}

							var newVal int64
							var leftValue *VMValue
							if v.TypeId == VMTypeComputedValue {
								leftValue = &v.Value.(*VMComputedValueData).BaseValue
							} else {
								leftValue = v
							}

							if m[3] == "+" {
								newVal = leftValue.Value.(int64) + r.Value.(int64)
							} else {
								newVal = leftValue.Value.(int64) - r.Value.(int64)
							}

							vOld, _, _ := ctx.Dice.ExprEvalBase(attrName, mctx, RollExtraFlags{})
							theOldValue := vOld.Value.(int64)

							leftValue.Value = newVal

							vNew, _, _ := ctx.Dice.ExprEvalBase(attrName, mctx, RollExtraFlags{})
							theNewValue := vNew.Value.(int64)

							baseValue := ""
							if v.TypeId == VMTypeComputedValue {
								baseValue = fmt.Sprintf("[%d]", newVal)
							}
							attrChanged = append(attrChanged, fmt.Sprintf("%s%s(%d ➯ %d)", attrName, baseValue, theOldValue, theNewValue))
						}
					}

					retText := "人物属性设置如下:\n"
					if len(attrSeted) > 0 {
						SetCardType(mctx, "dnd5e")
						retText += "读入: " + strings.Join(attrSeted, ", ") + "\n"
					}
					if len(attrChanged) > 0 {
						retText += "修改: " + strings.Join(attrChanged, ", ") + "\n"
					}
					if text != "" {
						retText += "解析失败: " + text
					}
					ReplyToSender(ctx, msg, retText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpRc := "" +
		".rc <属性> // .rc 力量\n" +
		".rc <表达式> // .rc 力量+3\n" +
		".rc 优势 <表达式> // .rc 优势 力量+4\n" +
		".rc 劣势 <表达式> (原因) // .rc 劣势 力量+4 推一下试试\n" +
		".rc <表达式> @某人 // 对某人做检定"

	cmdRc := &CmdItemInfo{
		Name:     "rc",
		Help:     helpRc,
		LongHelp: "DND5E 检定:\n" + helpRc,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				mctx := &*ctx // 复制一个ctx，用于其他用途
				if len(cmdArgs.At) > 0 {
					p, exists := ctx.Group.Players[cmdArgs.At[0].UserId]
					if exists {
						mctx.Player = p
					}
				}

				val, _ := cmdArgs.GetArgN(1)
				switch val {
				case "", "help":
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				default:
					restText := cmdArgs.CleanArgs
					re := regexp.MustCompile(`^优势|劣势`)
					m := re.FindString(restText)
					if m != "" {
						restText = strings.TrimSpace(restText[len(m):])
					}
					expr := fmt.Sprintf("d20%s + %s", m, restText)
					r, detail, err := mctx.Dice.ExprEvalBase(expr, mctx, RollExtraFlags{})
					if err != nil {
						ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					reason := r.restInput
					if reason == "" {
						reason = restText
					}

					text := fmt.Sprintf("<%s>的“%s”检定结果为:\n%s = %s", mctx.Player.Name, reason, detail, r.ToString())
					ReplyToSender(mctx, msg, text)
				}

				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	theExt := &ExtInfo{
		Name:       "dnd5e", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "正在努力完成的DND模块",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		ConflictWith: []string{
			"coc7",
		},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func(i *ExtInfo) string {
			return GetExtensionDesc(i)
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
			"dst": cmdSt,
			"st":  cmdSt,
			"属性":  cmdSt,
			"drc": cmdRc,
			"rc":  cmdRc,
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
