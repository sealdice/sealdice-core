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
				Top:    []string{"力量", "敏捷", "体质", "智力", "感知", "魅力", "HP", "HPMax", "EXP", "熟练"},
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

func stExport(mctx *MsgContext, whiteList map[string]bool, regexps []*regexp.Regexp) map[string]string {
	exportMap := map[string]string{}
	for k, v := range mctx.Player.ValueMap {
		doIt := whiteList[k]
		if !doIt && regexps != nil {
			for _, i := range regexps {
				if i.MatchString(k) {
					doIt = true
					break
				}
			}
		}

		if doIt {
			switch v.TypeId {
			case VMTypeInt64:
				exportMap[k] = strconv.FormatInt(v.Value.(int64), 10)
			case VMTypeString:
				exportMap[k] = v.Value.(string)
			case VMTypeComputedValue:
				vd := v.Value.(*VMComputedValueData)
				if strings.Index(vd.Expr, "熟练") != -1 {
					k = k + "*"
				}
				val, ok := vd.ReadBaseInt64()
				if ok {
					exportMap[k] = strconv.FormatInt(val, 10)
				}
			}
		}
	}
	return exportMap
}

func RegisterBuiltinExtDnd5e(self *Dice) {
	ac := setupConfigDND(self)

	helpSt := ".st 模板 // 录卡模板"
	helpSt += ".st show // 展示个人属性\n"
	helpSt += ".st show <属性1> <属性2> ... // 展示特定的属性数值\n"
	helpSt += ".st show <数字> // 展示高于<数字>的属性，如.st show 30\n"
	helpSt += ".st clr/clear // 清除属性\n"
	helpSt += ".st del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n"
	helpSt += ".st export // 导出，包括属性和法术位\n"
	helpSt += ".st help // 帮助\n"
	helpSt += ".st <属性>:<值> // 设置属性，技能加值会自动计算。例：.st 感知:20 洞悉:3\n"
	helpSt += ".st <属性>±<表达式> // 修改属性，例：.st hp+1d4\n"
	helpSt += ".st <属性>±<表达式> @某人 // 修改他人属性，例：.st hp+1d4"
	helpSt += "特别的，扣除hp时，会先将其buff值扣除到0。以及增加hp时，hp的值不会超过hpmax"

	cmdSt := &CmdItemInfo{
		Name:     "st",
		Help:     helpSt,
		LongHelp: "DND5E 人物属性设置:\n" + helpSt,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				val, _ := cmdArgs.GetArgN(1)
				mctx, _ := GetCtxStandInFirst(ctx, cmdArgs, true)

				switch val {
				case "模板":
					text := "人物卡模板(第二行文本):\n"
					text += ".dst 力量:10 体质:10 敏捷:10 智力:10 感知:10 魅力:10 hp:10 hpmax: 10 熟练:2 运动:0 特技:0 巧手:0 隐匿:0 调查:0 奥秘:0 历史:0 自然:0 宗教:0 察觉:0 洞悉:0 驯养:0 医疗:0 生存:0 说服:0 欺诈:0 威吓:0 表演:0\n"
					text += "注意: 技能只填写修正值即可，属性调整值会自动计算。熟练写为“运动*:0”"
					ReplyToSender(mctx, msg, text)
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

				case "export":
					m := stExport(mctx, map[string]bool{
						"力量": true, "敏捷": true, "体质": true, "智力": true, "感知": true, "魅力": true,
						"运动": true,
						"特技": true, "巧手": true, "隐匿": true,
						"调查": true, "奥秘": true, "历史": true, "自然": true, "宗教": true,
						"察觉": true, "洞悉": true, "驯养": true, "医疗": true, "生存": true,
						"说服": true, "欺诈": true, "威吓": true, "表演": true,

						"hd": true, "hp": true, "hpmax": true,
						//"$cardType": true,
					}, []*regexp.Regexp{
						regexp.MustCompile(`^\$法术位_[1-9]$`),
						regexp.MustCompile(`^\$法术位上限_[1-9]$`),
					})

					texts := []string{}
					for k, v := range m {
						texts = append(texts, fmt.Sprintf("%s:%s", k, v))
					}
					sort.Strings(texts)
					ReplyToSender(mctx, msg, "属性导出(注意，人物基础属性的熟练还不能支持):\n"+strings.Join(texts, " "))

				case "clr", "clear":
					p := mctx.Player
					num := len(p.ValueMap)
					p.ValueMap = map[string]*VMValue{}
					VarSetValueInt64(mctx, "$t数量", int64(num))
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
						info = DiceFormatTmpl(mctx, "DND:属性设置_列出_未发现记录")
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

							if usePickItem {
								_, ok := pickItems[k]
								if !ok {
									continue
								}
							}

							var v *VMValue
							vRaw, exists := p.ValueMap[k]
							if !exists {
								// 不存在的值，强行补0
								v = &VMValue{VMTypeInt64, int64(0)}
								vRaw = v
							} else {
								v2, _, _ := mctx.Dice.ExprEvalBase(k, mctx, RollExtraFlags{})
								v = &v2.VMValue
							}

							if index >= topNum {
								if useLimit && v.TypeId == VMTypeInt64 && v.Value.(int64) < limit {
									limktSkipCount += 1
									continue
								}
							}

							tick += 1

							vRawStr := vRaw.ToString()
							vStr := v.ToString()

							vText := ""
							if vRaw.TypeId == VMTypeComputedValue {
								vd := vRaw.Value.(*VMComputedValueData)
								b := vd.BaseValue.ToString()
								if vStr != b {
									vText = fmt.Sprintf("%s[%s]", vStr, b)
								} else {
									vText = vStr
								}
							} else {
								if vRawStr != vStr {
									vText = fmt.Sprintf("%s[%s]", vStr, vRawStr)
								} else {
									vText = v.ToString()
								}
							}
							info += fmt.Sprintf("%s: %s\t", k, vText)
							if tick%4 == 0 {
								info += fmt.Sprintf("\n")
							}
						}

						if info == "" {
							info = DiceFormatTmpl(mctx, "DND:属性设置_列出_未发现记录")
						}
					}

					if useLimit {
						VarSetValueInt64(mctx, "$t数量", int64(limktSkipCount))
						VarSetValueInt64(mctx, "$t判定值", int64(limit))
						info += DiceFormatTmpl(mctx, "DND:属性设置_列出_隐藏提示")
					}

					VarSetValueStr(mctx, "$t属性信息", info)
					extra := ReadCardType(mctx, "dnd5e")
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_列出")+extra)

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
						r, _, err := mctx.Dice.ExprEvalBase(text, mctx, RollExtraFlags{})
						if err != nil {
							ReplyToSender(mctx, msg, "无法解析属性: "+attrName)
							return CmdExecuteResult{Matched: true, Solved: true}
						}
						text = r.restInput

						if r.TypeId != VMTypeInt64 {
							ReplyToSender(mctx, msg, "这个属性的值并非数字: "+attrName)
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						if m[3] == ":" {
							exprTmpl := "$tVal + %s/2 - 5"
							if isSkilled {
								exprTmpl += " + 熟练"
							}

							parent, _ := dndAttrParent[attrName]
							aText := attrName
							aText += fmt.Sprintf(":%d", r.Value.(int64))
							if parent != "" {
								if isSkilled {
									aText += "[技能, 熟练]"
								} else {
									aText += "[技能]"
								}
								VarSetValueDNDComputed(mctx, attrName, r.Value.(int64), fmt.Sprintf(exprTmpl, parent))
							} else {
								VarSetValueInt64(mctx, attrName, r.Value.(int64))
								VarSetValueDNDComputed(mctx, fmt.Sprintf("$豁免_%s", attrName), int64(0), fmt.Sprintf(exprTmpl, attrName))
							}
							attrSeted = append(attrSeted, aText)
						}
						if m[3] == "+" || m[3] == "-" {
							v, exists := VarGetValue(mctx, attrName)
							if !exists {
								ReplyToSender(mctx, msg, "不存在的属性: "+attrName)
								return CmdExecuteResult{Matched: true, Solved: true}
							}
							if v.TypeId != VMTypeInt64 && v.TypeId != VMTypeComputedValue {
								ReplyToSender(mctx, msg, "这个属性的值并非数字: "+attrName)
								return CmdExecuteResult{Matched: true, Solved: true}
							}

							if m[3] == "-" {
								r.Value = -r.Value.(int64)
							}

							if attrName == "hp" {
								// 当扣血时，特别处理
								if r.Value.(int64) < 0 {
									vHpBuff, exists := VarGetValue(mctx, "$buff_hp")
									if exists {
										vHpBuffVal := vHpBuff.Value.(int64)
										// 正盾才做反馈
										if vHpBuffVal > 0 {
											val := vHpBuffVal + r.Value.(int64)
											if val >= 0 {
												// 有充足的盾，扣掉
												vHpBuff.Value = val
												r.Value = int64(0)
											} else {
												// 没有充足的盾，盾扣到0
												r.Value = val
												vHpBuff.Value = int64(0)
											}
										}
									}
								}
							}

							var newVal int64
							var leftValue *VMValue
							if v.TypeId == VMTypeComputedValue {
								leftValue = &v.Value.(*VMComputedValueData).BaseValue
							} else {
								leftValue = v
							}

							newVal = leftValue.Value.(int64) + r.Value.(int64)
							if attrName == "hp" {
								vHpMax, exists := VarGetValue(mctx, "hpmax")
								if exists {
									// 生命值上限限制
									if newVal > vHpMax.Value.(int64) {
										newVal = vHpMax.Value.(int64)
									}
								}
							}

							vOld, _, _ := mctx.Dice.ExprEvalBase(attrName, mctx, RollExtraFlags{})
							theOldValue := vOld.Value.(int64)

							leftValue.Value = newVal

							vNew, _, _ := mctx.Dice.ExprEvalBase(attrName, mctx, RollExtraFlags{})
							theNewValue := vNew.Value.(int64)

							baseValue := ""
							if v.TypeId == VMTypeComputedValue {
								baseValue = fmt.Sprintf("[%d]", newVal)
							}
							attrChanged = append(attrChanged, fmt.Sprintf("%s%s(%d ➯ %d)", attrName, baseValue, theOldValue, theNewValue))
						}
					}

					retText := fmt.Sprintf("<%s>的dnd5e人物属性设置如下:\n", mctx.Player.Name)
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
					ReplyToSender(mctx, msg, retText)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpRc := "" +
		".rc <属性> // .rc 力量\n" +
		".rc <属性>豁免 // .rc 力量豁免\n" +
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
				mctx, _ := GetCtxStandInFirst(ctx, cmdArgs, true)

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
					r, detail, err := mctx.Dice.ExprEvalBase(expr, mctx, RollExtraFlags{DNDAttrReadMod: true, DNDAttrReadDC: true})
					if err != nil {
						ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					reason := r.restInput
					if reason == "" {
						reason = restText
					}

					text := fmt.Sprintf("<%s>的“%s”检定(dnd5e)结果为:\n%s = %s", mctx.Player.Name, reason, detail, r.ToString())
					ReplyToSender(mctx, msg, text)
				}

				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpBuff := "" +
		".buff // 展示当前buff\n" +
		".buff clr // 清除buff\n" +
		".buff del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n" +
		".buff help // 帮助\n" +
		".buff <属性>:<值> // 设置buff属性，例：.buff 力量:4  奥秘*:0，奥秘临时熟练加成\n" +
		".buff <属性>±<表达式> // 修改属性，例：.buff hp+1d4\n" +
		".buff <属性>±<表达式> @某人 // 修改他人buff属性，例：.buff hp+1d4"

	cmdBuff := &CmdItemInfo{
		Name:     "buff",
		Help:     helpBuff,
		LongHelp: "属性临时加值:\n" + helpBuff,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				val, _ := cmdArgs.GetArgN(1)
				mctx, _ := GetCtxStandInFirst(ctx, cmdArgs, true)

				switch val {
				case "del", "rm":
					var nums []string
					var failed []string

					for _, rawVarname := range cmdArgs.Args[1:] {
						varname := "$buff_" + rawVarname
						_, ok := mctx.Player.ValueMap[varname]
						if ok {
							nums = append(nums, rawVarname)
							delete(mctx.Player.ValueMap, varname)
						} else {
							failed = append(failed, varname)
						}
					}

					VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
					VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:BUFF设置_删除"))

				case "clr", "clear":
					p := mctx.Player
					toDelete := []string{}
					for varname, _ := range p.ValueMap {
						varname = "$buff_" + varname
						if _, exists := p.ValueMap[varname]; exists {
							toDelete = append(toDelete, varname)
						}
					}

					num := len(toDelete)
					for _, varname := range toDelete {
						delete(p.ValueMap, varname)
					}
					VarSetValueInt64(mctx, "$t数量", int64(num))
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:BUFF设置_清除"))

				case "show", "list", "":
					p := mctx.Player
					var info string

					attrKeys2 := []string{}
					for k := range p.ValueMap {
						if strings.HasPrefix(k, "$buff_") {
							attrKeys2 = append(attrKeys2, k)
						}
					}
					sort.Strings(attrKeys2)

					tick := 0
					if len(p.ValueMap) == 0 {
						info = DiceFormatTmpl(mctx, "DND:属性设置_列出_未发现记录")
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
						for _, k := range attrKeys {
							if !strings.HasPrefix(k, "$buff_") {
								continue
							}
							v, exists := p.ValueMap[k]
							if !exists {
								// 不存在的值，强行补0
								v = &VMValue{VMTypeInt64, int64(0)}
							}

							tick += 1
							vText := ""
							if v.TypeId == VMTypeComputedValue {
								vd := v.Value.(*VMComputedValueData)
								val, _, _ := mctx.Dice.ExprEvalBase(k, mctx, RollExtraFlags{})
								a := val.ToString()
								b := vd.BaseValue.ToString()
								if a != b {
									vText = fmt.Sprintf("%s[%s]", a, b)
								} else {
									vText = a
								}
							} else {
								vText = v.ToString()
							}
							k = k[len("$buff_"):]
							info += fmt.Sprintf("%s: %s\t", k, vText)
							if tick%4 == 0 {
								info += fmt.Sprintf("\n")
							}
						}

						if info == "" {
							info = DiceFormatTmpl(mctx, "DND:属性设置_列出_未发现记录")
						}
					}

					VarSetValueStr(mctx, "$t属性信息", info)
					extra := ReadCardType(mctx, "dnd5e")
					ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_列出")+extra)

				case "help":
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

						attrNameRaw := m[2]
						attrNameBuff := "$buff_" + attrNameRaw
						isSkilled := strings.HasSuffix(m[1], "*")
						r, _, err := mctx.Dice.ExprEvalBase(text, mctx, RollExtraFlags{})
						if err != nil {
							ReplyToSender(mctx, msg, "无法解析属性: "+attrNameRaw)
							return CmdExecuteResult{Matched: true, Solved: true}
						}
						text = r.restInput

						if r.TypeId != VMTypeInt64 {
							ReplyToSender(mctx, msg, "这个属性的值并非数字: "+attrNameRaw)
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						if m[3] == ":" {
							exprTmpl := "$tVal"
							if isSkilled {
								exprTmpl += " + 熟练"
							}

							parent, _ := dndAttrParent[attrNameRaw]
							aText := attrNameRaw
							aText += fmt.Sprintf(":%d", r.Value.(int64))
							if parent != "" {
								if isSkilled {
									aText += "[技能, 熟练]"
								} else {
									aText += "[技能]"
								}
								VarSetValueDNDComputed(mctx, attrNameBuff, r.Value.(int64), fmt.Sprintf(exprTmpl, parent))
							} else {
								VarSetValueInt64(mctx, attrNameBuff, r.Value.(int64))
							}
							attrSeted = append(attrSeted, aText)
						}
						if m[3] == "+" || m[3] == "-" {
							v, exists := VarGetValue(mctx, attrNameBuff)
							if !exists {
								ReplyToSender(mctx, msg, "不存在的BUFF属性: "+attrNameRaw)
								return CmdExecuteResult{Matched: true, Solved: true}
							}
							if v.TypeId != VMTypeInt64 && v.TypeId != VMTypeComputedValue {
								ReplyToSender(mctx, msg, "这个属性的值并非数字: "+attrNameRaw)
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

							vOld, _, _ := mctx.Dice.ExprEvalBase(attrNameBuff, mctx, RollExtraFlags{})
							theOldValue := vOld.Value.(int64)

							leftValue.Value = newVal

							vNew, _, _ := mctx.Dice.ExprEvalBase(attrNameBuff, mctx, RollExtraFlags{})
							theNewValue := vNew.Value.(int64)

							baseValue := ""
							if v.TypeId == VMTypeComputedValue {
								baseValue = fmt.Sprintf("[%d]", newVal)
							}
							attrChanged = append(attrChanged, fmt.Sprintf("%s%s(%d ➯ %d)", attrNameRaw, baseValue, theOldValue, theNewValue))
						}
					}

					retText := fmt.Sprintf("<%s>的dnd5e人物Buff属性设置如下:\n", mctx.Player.Name)
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
					ReplyToSender(mctx, msg, retText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	spellSlotsRenew := func(mctx *MsgContext, msg *Message) int {
		num := 0
		for i := 1; i < 10; i += 1 {
			//_, _ := VarGetValueInt64(mctx, fmt.Sprintf("$法术位_%d", i))
			spellLevelMax, exists := VarGetValueInt64(mctx, fmt.Sprintf("$法术位上限_%d", i))
			if exists {
				num += 1
				VarSetValueInt64(mctx, fmt.Sprintf("$法术位_%d", i), spellLevelMax)
			}
		}
		return num
	}

	spellSlotsChange := func(mctx *MsgContext, msg *Message, spellLevel int64, num int64) *CmdExecuteResult {
		spellLevelCur, _ := VarGetValueInt64(mctx, fmt.Sprintf("$法术位_%d", spellLevel))
		spellLevelMax, _ := VarGetValueInt64(mctx, fmt.Sprintf("$法术位上限_%d", spellLevel))

		newLevel := spellLevelCur + num
		if newLevel < 0 {
			ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>无法消耗%d个%d环法术位，当前%d个`, mctx.Player.Name, -num, spellLevel, spellLevelCur))
			return &CmdExecuteResult{Matched: true, Solved: true}
		}
		if newLevel > spellLevelMax {
			newLevel = spellLevelMax
		}
		VarSetValueInt64(mctx, fmt.Sprintf("$法术位_%d", spellLevel), newLevel)
		if num < 0 {
			ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>的%d环法术位消耗至%d个，上限%d个`, mctx.Player.Name, spellLevel, newLevel, spellLevelMax))
		} else {
			ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>的%d环法术位恢复至%d个，上限%d个`, mctx.Player.Name, spellLevel, newLevel, spellLevelMax))
		}
		return nil
	}

	helpSS := "" +
		".ss // 查看当前法术位状况\n" +
		".ss init 4 3 2 // 设置1 2 3环的法术位上限，以此类推到9环\n" +
		".ss set 2环 4 // 单独设置某一环的法术位上限，可连写多组，逗号分隔\n" +
		".ss clr // 清除法术位设置\n" +
		".ss rest // 恢复所有法术位(不回复hp)\n" +
		".ss 3环 +1 // 增加一个3环法术位（不会超过上限）\n" +
		".ss lv3 +1 // 增加一个3环法术位 - 另一种写法\n" +
		".ss 3环 -1 // 消耗一个3环法术位，也可以用.cast 3"

	cmdSpellSlot := &CmdItemInfo{
		Name:     "ss",
		Help:     helpSS,
		LongHelp: "DND5E 法术位(.ss .法术位):\n" + helpSS,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				val, _ := cmdArgs.GetArgN(1)
				mctx, _ := GetCtxStandInFirst(ctx, cmdArgs, true)

				switch val {
				case "init":
					reSlot := regexp.MustCompile(`\d+`)
					slots := reSlot.FindAllString(cmdArgs.CleanArgs, -1)
					if len(slots) > 0 {
						texts := []string{}
						for index, levelVal := range slots {
							val, _ := strconv.ParseInt(levelVal, 10, 64)
							VarSetValueInt64(mctx, fmt.Sprintf("$法术位_%d", index+1), val)
							VarSetValueInt64(mctx, fmt.Sprintf("$法术位上限_%d", index+1), val)
							texts = append(texts, fmt.Sprintf("%d环%d个", index+1, val))
						}
						ReplyToSender(mctx, msg, "为<"+mctx.Player.Name+">设置法术位: "+strings.Join(texts, ", "))
					} else {
						return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
					}

				case "clr":
					vm := mctx.Player.ValueMap
					for i := 1; i < 10; i += 1 {
						delete(vm, fmt.Sprintf("$法术位_%d", i))
						delete(vm, fmt.Sprintf("$法术位上限_%d", i))
					}
					ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>法术位数据已清除`, mctx.Player.Name))

				case "rest":
					n := spellSlotsRenew(mctx, msg)
					if n > 0 {
						ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>的法术位已经完全恢复`, mctx.Player.Name))
					} else {
						ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>并没有设置过法术位`, mctx.Player.Name))
					}

				case "set":
					reSlot := regexp.MustCompile(`(\d+)[环cC]\s*(\d+)|[lL][vV](\d+)\s+(\d+)`)
					slots := reSlot.FindAllStringSubmatch(cmdArgs.CleanArgs, -1)
					if len(slots) > 0 {
						texts := []string{}
						for _, oneSlot := range slots {
							level := oneSlot[1]
							if level == "" {
								level = oneSlot[3]
							}
							levelVal := oneSlot[2]
							if levelVal == "" {
								levelVal = oneSlot[4]
							}
							iLevel, _ := strconv.ParseInt(level, 10, 64)
							iLevelVal, _ := strconv.ParseInt(levelVal, 10, 64)

							VarSetValueInt64(mctx, fmt.Sprintf("$法术位_%d", iLevel), iLevelVal)
							VarSetValueInt64(mctx, fmt.Sprintf("$法术位上限_%d", iLevel), iLevelVal)
							texts = append(texts, fmt.Sprintf("%d环%d个", iLevel, iLevelVal))
						}
						ReplyToSender(mctx, msg, "为<"+mctx.Player.Name+">设置法术位: "+strings.Join(texts, ", "))
					} else {
						return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
					}
				case "":
					texts := []string{}
					for i := 1; i < 10; i += 1 {
						spellLevelCur, _ := VarGetValueInt64(mctx, fmt.Sprintf("$法术位_%d", i))
						spellLevelMax, exists := VarGetValueInt64(mctx, fmt.Sprintf("$法术位上限_%d", i))
						if exists {
							texts = append(texts, fmt.Sprintf("%d环:%d/%d", i, spellLevelCur, spellLevelMax))
						}
					}
					summary := strings.Join(texts, ", ")
					if summary == "" {
						summary = "没有设置过法术位"
					}
					ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>的法术位状况: %s`, mctx.Player.Name, summary))
				case "help":
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				default:
					reSlot := regexp.MustCompile(`(\d+)[环cC]\s*([+-])(\d+)|[lL][vV](\d+)\s*([+-])(\d+)`)
					slots := reSlot.FindAllStringSubmatch(cmdArgs.CleanArgs, -1)
					if len(slots) > 0 {
						for _, oneSlot := range slots {
							level := oneSlot[1]
							if level == "" {
								level = oneSlot[4]
							}
							op := oneSlot[2]
							if op == "" {
								op = oneSlot[5]
							}
							levelVal := oneSlot[3]
							if levelVal == "" {
								levelVal = oneSlot[6]
							}
							iLevel, _ := strconv.ParseInt(level, 10, 64)
							iLevelVal, _ := strconv.ParseInt(levelVal, 10, 64)
							if op == "-" {
								iLevelVal = -iLevelVal
							}

							ret := spellSlotsChange(mctx, msg, iLevel, iLevelVal)
							if ret != nil {
								return *ret
							}
						}
					} else {
						return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
					}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpCast := "" +
		".cast 1 // 消耗1个1环法术位\n" +
		".cast 1 2 // 消耗2个1环法术位"

	cmdCast := &CmdItemInfo{
		Name:     "cast",
		Help:     helpCast,
		LongHelp: "DND5E 法术位使用(.cast):\n" + helpCast,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				val, _ := cmdArgs.GetArgN(1)
				mctx, _ := GetCtxStandInFirst(ctx, cmdArgs, true)

				switch val {
				default:
					// 该正则匹配: 2 1, 2环1, 2环 1, 2c1, lv2 1
					reSlot := regexp.MustCompile(`(\d+)(?:[环cC]?|\s)\s*(\d+)?|[lL][vV](\d+)(?:\s+(\d+))?`)

					slots := reSlot.FindAllStringSubmatch(cmdArgs.CleanArgs, -1)
					if len(slots) > 0 {
						for _, oneSlot := range slots {
							level := oneSlot[1]
							if level == "" {
								level = oneSlot[3]
							}
							levelVal := oneSlot[2]
							if levelVal == "" {
								levelVal = oneSlot[4]
							}
							if levelVal == "" {
								levelVal = "1"
							}
							iLevel, _ := strconv.ParseInt(level, 10, 64)
							iLevelVal, _ := strconv.ParseInt(levelVal, 10, 64)

							ret := spellSlotsChange(mctx, msg, iLevel, -iLevelVal)
							if ret != nil {
								return *ret
							}
						}
					} else {
						return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
					}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpLongRest := "" +
		".长休 // 恢复生命值(必须设置hpmax且hp>0)和法术位 \n" +
		".longrest // 另一种写法"

	cmdLongRest := &CmdItemInfo{
		Name:     "长休",
		Help:     helpLongRest,
		LongHelp: "DND5E 长休:\n" + helpLongRest,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsCurGroupBotOn || ctx.IsPrivate {
				val, _ := cmdArgs.GetArgN(1)
				mctx, _ := GetCtxStandInFirst(ctx, cmdArgs, true)

				switch val {
				case "":
					hpText := "没有设置hpmax，无法回复hp"
					hpMax, exists := VarGetValueInt64(mctx, "hpmax")
					if exists {
						hpText = fmt.Sprintf("hp得到了恢复，现为%d", hpMax)
						VarSetValueInt64(mctx, "hp", hpMax)
					}

					n := spellSlotsRenew(mctx, msg)
					ssText := ""
					if n > 0 {
						ssText = "。法术位得到了恢复"
					}
					ReplyToSender(mctx, msg, fmt.Sprintf(`<%s>的长休: `+hpText+ssText, mctx.Player.Name))
				default:
					return CmdExecuteResult{Matched: true, Solved: true, ShowLongHelp: true}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	theExt := &ExtInfo{
		Name:       "dnd5e", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "提供DND5E规则TRPG支持",
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
							result, _, err := self.ExprText(`力量:{$t1=4d6k3} 体质:{$t2=4d6k3} 敏捷:{$t3=4d6k3} 智力:{$t4=4d6k3} 感知:{$t5=4d6k3} 魅力:{$t6=4d6k3} 共计:{$t1+$t2+$t3+$t4+$t5+$t6}`, ctx)
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
						mctx, _ := GetCtxStandInFirst(ctx, cmdArgs, true)

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
			//"属性":    cmdSt,
			"st":         cmdSt,
			"dst":        cmdSt,
			"rc":         cmdRc,
			"drc":        cmdRc,
			"buff":       cmdBuff,
			"dbuff":      cmdBuff,
			"spellslots": cmdSpellSlot,
			"ss":         cmdSpellSlot,
			"dss":        cmdSpellSlot,
			"法术位":        cmdSpellSlot,
			"cast":       cmdCast,
			"dcast":      cmdCast,
			"长休":         cmdLongRest,
			"longrest":   cmdLongRest,
			"dlongrest":  cmdLongRest,
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
