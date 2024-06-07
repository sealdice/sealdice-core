package dice

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
	"gopkg.in/yaml.v3"
)

type RIListItem struct {
	name   string
	val    int64
	detail string
	uid    string
}

type ByRIListValue []*RIListItem

func (lst ByRIListValue) Len() int {
	return len(lst)
}
func (lst ByRIListValue) Swap(i, j int) {
	lst[i], lst[j] = lst[j], lst[i]
}
func (lst ByRIListValue) Less(i, j int) bool {
	if lst[i].val == lst[j].val {
		return lst[i].name > lst[j].name
	}
	return lst[i].val > lst[j].val
}

var dndAttrParent = map[string]string{
	"运动": "力量",

	"体操": "敏捷",
	"巧手": "敏捷",
	"隐匿": "敏捷",

	"调查": "智力",
	"奥秘": "智力",
	"历史": "智力",
	"自然": "智力",
	"宗教": "智力",

	"察觉": "感知",
	"洞悉": "感知",
	"驯兽": "感知",
	"医药": "感知",
	"求生": "感知",

	"游说": "魅力",
	"欺瞒": "魅力",
	"威吓": "魅力",
	"表演": "魅力",
}

func setupConfigDND(d *Dice) AttributeConfigs {
	attrConfigFn := d.GetExtConfigFilePath("dnd5e", "attribute.yaml")

	_, _ = os.Stat(attrConfigFn)
	// 如果不存在，新建
	defaultVals := AttributeConfigs{
		Alias: map[string][]string{
			"力量": {"str", "Strength"},
			"敏捷": {"dex", "Dexterity"},
			"体质": {"con", "Constitution", "體質", "體魄", "体魄"},
			"智力": {"int", "Intelligence"},
			"感知": {"wis", "Wisdom"},
			"魅力": {"cha", "Charisma"},

			"ac":    {"AC", "护甲等级", "护甲值", "护甲", "護甲等級", "護甲值", "護甲", "装甲", "裝甲"},
			"hp":    {"HP", "生命值", "生命", "血量", "体力", "體力", "耐久值"},
			"hpmax": {"HPMAX", "生命值上限", "生命上限", "血量上限", "耐久上限"},
			"dc":    {"DC", "难度等级", "法术豁免", "難度等級", "法術豁免"},
			"hd":    {"HD", "生命骰"},
			"pp":    {"PP", "被动察觉", "被动感知", "被動察覺", "被动感知", "PW"},

			"熟练": {"熟练加值", "熟練", "熟練加值"},
			"体型": {"siz", "size", "體型", "体型", "体形", "體形"},

			// 技能
			"运动": {"Athletics", "運動"},

			"体操": {"Acrobatics", "杂技", "特技", "體操", "雜技", "特技動作", "特技动作"},
			"巧手": {"Sleight of Hand", "上手把戲", "上手把戏"},
			"隐匿": {"Stealth", "隱匿", "潜行", "潛行"},

			"调查": {"Investigation", "調查"},
			"奥秘": {"Arcana", "奧秘"},
			"历史": {"History", "歷史"},
			"自然": {"Nature"},
			"宗教": {"Religion"},

			"察觉": {"Perception", "察覺", "觉察", "覺察"},
			"洞悉": {"Insight", "洞察", "察言觀色", "察言观色"},
			"驯兽": {"Animal Handling", "馴獸", "驯养", "馴養", "動物馴服", "動物馴養", "动物驯服", "动物驯养"},
			"医药": {"Medicine", "醫藥", "医疗", "醫療"},
			"求生": {"Survival", "生存"},

			"游说": {"Persuasion", "说服", "话术", "遊說", "說服", "話術"},
			"欺瞒": {"Deception", "唬骗", "欺诈", "欺骗", "诈骗", "欺瞞", "唬騙", "欺詐", "欺騙", "詐騙"},
			"威吓": {"Intimidation", "恐吓", "威嚇", "恐嚇"},
			"表演": {"Performance"},
		},
		Order: AttributeOrder{
			Top:    []string{"力量", "敏捷", "体质", "体型", "魅力", "智力", "感知", "hp", "ac", "熟练"},
			Others: AttributeOrderOthers{SortBy: "Name"},
		},
	}
	buf, err2 := yaml.Marshal(defaultVals)
	if err2 != nil {
		fmt.Println(err2)
	} else {
		_ = os.WriteFile(attrConfigFn, buf, 0644)
	}
	return defaultVals
}

func getPlayerNameTempFunc(mctx *MsgContext) string {
	if mctx.Dice.PlayerNameWrapEnable {
		return fmt.Sprintf("<%s>", mctx.Player.Name)
	}
	return mctx.Player.Name
}

func RegisterBuiltinExtDnd5e(self *Dice) {
	ac := setupConfigDND(self)

	deathSavingStable := func(ctx *MsgContext) {
		VarDelValue(ctx, "DSS")
		VarDelValue(ctx, "DSF")
		if ctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
		}
	}

	deathSaving := func(ctx *MsgContext, successPlus int64, failurePlus int64) (int64, int64) {
		readAndAssign := func(name string) int64 {
			var val int64
			v, exists := _VarGetValueV1(ctx, name)

			if !exists {
				VarSetValueInt64(ctx, name, int64(0))
			} else {
				val, _ = v.ReadInt64()
			}
			return val
		}

		val1 := readAndAssign("DSS")
		val2 := readAndAssign("DSF")

		if successPlus != 0 {
			val1 += successPlus
			VarSetValueInt64(ctx, "DSS", val1)
		}

		if failurePlus != 0 {
			val2 += failurePlus
			VarSetValueInt64(ctx, "DSF", val2)
		}

		if ctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
		}

		return val1, val2
	}

	deathSavingResultCheck := func(ctx *MsgContext, a int64, b int64) string {
		text := ""
		if a >= 3 {
			text = DiceFormatTmpl(ctx, "DND:死亡豁免_结局_伤势稳定")
			deathSavingStable(ctx)
		}
		if b >= 3 {
			text = DiceFormatTmpl(ctx, "DND:死亡豁免_结局_角色死亡")
			deathSavingStable(ctx)
		}
		return text
	}

	helpSt := ".st 模板 // 录卡模板\n"
	helpSt += ".st show // 展示个人属性\n"
	helpSt += ".st show <属性1> <属性2> ... // 展示特定的属性数值\n"
	helpSt += ".st show <数字> // 展示高于<数字>的属性，如.st show 30\n"
	helpSt += ".st clr/clear // 清除属性\n"
	helpSt += ".st del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n"
	helpSt += ".st export // 导出，包括属性和法术位\n"
	helpSt += ".st help // 帮助\n"
	helpSt += ".st <属性>:<值> // 设置属性，技能加值会自动计算。例：.st 感知:20 洞悉:3\n"
	helpSt += ".st <属性>=<值> // 设置属性，等号效果完全相同\n"
	helpSt += ".st <属性>±<表达式> // 修改属性，例：.st hp+1d4\n"
	helpSt += ".st <属性>±<表达式> @某人 // 修改他人属性，例：.st hp+1d4\n"
	helpSt += ".st hp-1d6 --over // 不计算临时生命扣血\n"
	helpSt += "特别的，扣除hp时，会先将其buff值扣除到0。以及增加hp时，hp的值不会超过hpmax\n"
	helpSt += "需要使用coc版本st，请执行.set coc"

	cmdSt := getCmdStBase(CmdStOverrideInfo{
		HelpSt: helpSt,
		CommandSolve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) *CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			switch val {
			case "模板":
				text := "人物卡模板(第二行文本):\n"
				text += ".dst 力量:10 体质:10 敏捷:10 智力:10 感知:10 魅力:10 hp:10 hpmax:10 熟练:2 运动:0 体操:0 巧手:0 隐匿:0 调查:0 奥秘:0 历史:0 自然:0 宗教:0 察觉:0 洞悉:0 驯兽:0 医药:0 求生:0 游说:0 欺瞒:0 威吓:0 表演:0\n"
				text += "注意: 技能只写修正值，调整值会自动计算。\n熟练写为“运动*:0”，半个熟练“运动*0.5:0”，录卡也可写为.dst 力量=10"
				ReplyToSender(ctx, msg, text)
				return &CmdExecuteResult{Matched: true, Solved: true}
			}
			return nil
		},
	})

	helpRc := "" +
		".rc <属性> // .rc 力量\n" +
		".rc <属性>豁免 // .rc 力量豁免\n" +
		".rc <表达式> // .rc 力量+3\n" +
		".rc 优势 <表达式> // .rc 优势 力量+4\n" +
		".rc 劣势 <表达式> (原因) // .rc 劣势 力量+4 推一下试试\n" +
		".rc <表达式> @某人 // 对某人做检定"

	cmdRc := &CmdItemInfo{
		Name:          "rc",
		ShortHelp:     helpRc,
		Help:          "DND5E 检定:\n" + helpRc,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			mctx.DelegateText = ctx.DelegateText
			mctx.Player.TempValueAlias = &ac.Alias

			val := cmdArgs.GetArgN(1)
			switch val {
			case "", "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			default:
				restText := cmdArgs.CleanArgs
				re := regexp.MustCompile(`^优势|劣势|優勢|劣勢`)
				m := re.FindString(restText)
				if m != "" {
					m = strings.Replace(m, "優勢", "优势", 1)
					m = strings.Replace(m, "劣勢", "劣势", 1)
					restText = strings.TrimSpace(restText[len(m):])
				}
				expr := fmt.Sprintf("D20%s + %s", m, restText)
				r, detail, err := mctx.Dice.ExprEvalBase(expr, mctx, RollExtraFlags{DNDAttrReadMod: true, DNDAttrReadDC: true})
				if err != nil {
					ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				reason := r.restInput
				if reason == "" {
					reason = restText
				}

				text := fmt.Sprintf("%s的“%s”检定(dnd5e)结果为:\n%s = %s", getPlayerNameTempFunc(mctx), reason, detail, r.ToString())

				// 指令信息
				commandInfo := map[string]interface{}{
					"cmd":    "rc",
					"rule":   "dnd5e",
					"pcName": mctx.Player.Name,
					"items": []interface{}{
						map[string]interface{}{
							"expr":   expr,
							"reason": reason,
							"result": r.Value,
						},
					},
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
				ReplyToSender(mctx, msg, text)
			}

			return CmdExecuteResult{Matched: true, Solved: true}
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
		Name:          "buff",
		ShortHelp:     helpBuff,
		Help:          "属性临时加值:\n" + helpBuff,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("del", "rm", "show", "list")
			val := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)

			attrs, _ := mctx.Dice.AttrsManager.LoadByCtx(mctx)

			switch val {
			case "del", "rm":
				var nums []string
				var failed []string

				for _, rawVarname := range cmdArgs.Args[1:] {
					varname := "$buff_" + rawVarname
					_, ok := attrs.LoadX(varname)
					if ok {
						nums = append(nums, rawVarname)
						attrs.Delete(varname)
					} else {
						failed = append(failed, varname)
					}
				}

				VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
				VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:BUFF设置_删除"))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}

			case "clr", "clear":
				var varNames []string
				attrs.Range(func(key string, value *ds.VMValue) bool {
					// 嵌套中不能再调用自己 会死锁，所以分开两步
					varname := "$buff_" + key
					varNames = append(varNames, varname)
					return true
				})

				var toDelete []string
				for _, varname := range varNames {
					if _, exists := attrs.LoadX(varname); exists {
						toDelete = append(toDelete, varname)
					}
				}

				num := len(toDelete)
				for _, varname := range toDelete {
					attrs.Delete(varname)
				}

				VarSetValueInt64(mctx, "$t数量", int64(num))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:BUFF设置_清除"))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}

			case "show", "list", "":
				p := mctx.Player
				var info string

				var attrKeys2 []string
				attrs.Range(func(k string, value *ds.VMValue) bool {
					if strings.HasPrefix(k, "$buff_") {
						attrKeys2 = append(attrKeys2, k)
					}
					return true
				})
				sort.Strings(attrKeys2)

				tick := 0
				if attrs.Len() == 0 {
					info = DiceFormatTmpl(mctx, "DND:属性设置_列出_未发现记录")
				} else {
					// 按照配置文件排序
					var attrKeys []string
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
					var attrKeys2 []string
					attrs.Range(func(key string, value *ds.VMValue) bool {
						attrKeys2 = append(attrKeys2, key)
						return true
					})
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
						v, exists := attrs.LoadX(k)
						if !exists {
							// 不存在的值，强行补0
							v = ds.NewIntVal(0)
						}

						tick++
						// var vText string
						// TODO: 重新弄一下
						// if v.TypeID == VMTypeDNDComputedValue {
						// 	vd := v.Value.(*VMDndComputedValueData)
						// 	val, _, _ := mctx.Dice.ExprEvalBase(k, mctx, RollExtraFlags{})
						// 	a := val.ToString()
						// 	b := vd.BaseValue.ToString()
						// 	if a != b {
						// 		vText = fmt.Sprintf("%s[%s]", a, b)
						// 	} else {
						// 		vText = a
						// 	}
						// } else {
						// 	vText = v.ToString()
						// }
						vText := v.ToString()
						k = k[len("$buff_"):]
						info += fmt.Sprintf("%s:%s\t", k, vText) // 单个文本
						if tick%4 == 0 {
							info += "\n"
						}
					}

					if info == "" {
						info = DiceFormatTmpl(mctx, "DND:属性设置_列出_未发现记录")
					}
				}

				VarSetValueStr(mctx, "$t属性信息", info)
				extra := ReadCardTypeEx(mctx, "dnd5e")
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_列出")+extra)

			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			default:
				text := cmdArgs.CleanArgs
				// \*(?:\d+(?:\.\d+)?)? // 这一段是熟练度
				re := regexp.MustCompile(`(?:([^\s:$0-9*]+)(\*(?:\d+(?:\.\d+)?)?)?)\s*([:：=＝+\-＋－])`)
				var attrSeted []string
				var attrChanged []string

				for {
					m := re.FindStringSubmatch(text)
					if len(m) == 0 {
						break
					}
					text = text[len(m[0]):]

					attrNameRaw := m[1]
					attrNameBuff := "$buff_" + attrNameRaw
					isSkilled := strings.HasPrefix(m[2], "*")
					var skilledFactor float64 = 1
					if isSkilled {
						val, err := strconv.ParseFloat(m[2][len("*"):], 64)
						if err == nil {
							skilledFactor = val
						}
					}

					if m[3] == "+" || m[3] == "-" || m[3] == "＋" || m[3] == "－" {
						text = m[3] + text
					}
					r, _, err := mctx.Dice.ExprEvalBase(text, mctx, RollExtraFlags{DisableNumDice: true, DisableBPDice: true, DisableCrossDice: true, DisableDicePool: true, DisableBlock: true, DisableBitwiseOp: true})
					if err != nil {
						ReplyToSender(mctx, msg, "无法解析属性: "+attrNameRaw)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					text = r.restInput

					if r.TypeID != VMTypeInt64 {
						ReplyToSender(mctx, msg, "这个属性的值并非数字: "+attrNameRaw)
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					attrNameRaw = ctx.Player.GetValueNameByAlias(attrNameRaw, ac.Alias)
					if m[3] == ":" || m[3] == "：" || m[3] == "=" || m[3] == "＝" {
						exprTmpl := "$tVal"
						if isSkilled {
							var factorText string
							if skilledFactor != math.Trunc(skilledFactor) {
								n := int64(skilledFactor * 100)
								factorText = fmt.Sprintf("熟练*%d/100", n)
							} else {
								factorText = "熟练*" + strconv.FormatInt(int64(skilledFactor), 10)
							}
							exprTmpl += " + " + factorText
						}

						parent := dndAttrParent[attrNameRaw]
						aText := attrNameRaw
						aText += fmt.Sprintf(":%d", r.Value.(int64))
						if parent != "" {
							if isSkilled {
								aText += fmt.Sprintf("[技能, 熟练%s]", m[2])
							} else {
								aText += "[技能]"
							}
							_VarSetValueDNDComputedV1(mctx, attrNameBuff, r.Value.(int64), fmt.Sprintf(exprTmpl, parent))
						} else {
							VarSetValueInt64(mctx, attrNameBuff, r.Value.(int64))
						}
						attrSeted = append(attrSeted, aText)
					}
					if m[3] == "+" || m[3] == "-" || m[3] == "＋" || m[3] == "－" {
						v, exists := _VarGetValueV1(mctx, attrNameBuff)
						if !exists {
							ReplyToSender(mctx, msg, "不存在的BUFF属性: "+attrNameRaw)
							return CmdExecuteResult{Matched: true, Solved: true}
						}
						if v.TypeID != VMTypeInt64 && v.TypeID != VMTypeDNDComputedValue {
							ReplyToSender(mctx, msg, "这个属性的值并非数字: "+attrNameRaw)
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						var newVal int64
						var leftValue *VMValue
						if v.TypeID == VMTypeDNDComputedValue {
							leftValue = &v.Value.(*VMDndComputedValueData).BaseValue
						} else {
							leftValue = v
						}

						newVal = leftValue.Value.(int64) + r.Value.(int64)

						vOld, _, _ := mctx.Dice.ExprEvalBase(attrNameBuff, mctx, RollExtraFlags{DisableNumDice: true, DisableBPDice: true, DisableCrossDice: true, DisableDicePool: true, DisableBlock: true, DisableBitwiseOp: true})
						theOldValue := vOld.Value.(int64)

						leftValue.Value = newVal

						vNew, _, _ := mctx.Dice.ExprEvalBase(attrNameBuff, mctx, RollExtraFlags{DisableNumDice: true, DisableBPDice: true, DisableCrossDice: true, DisableDicePool: true, DisableBlock: true, DisableBitwiseOp: true})
						theNewValue := vNew.Value.(int64)

						baseValue := ""
						if v.TypeID == VMTypeDNDComputedValue {
							baseValue = fmt.Sprintf("[%d]", newVal)
						}
						attrChanged = append(attrChanged, fmt.Sprintf("%s%s(%d ➯ %d)", attrNameRaw, baseValue, theOldValue, theNewValue))
					}
				}

				retText := fmt.Sprintf("%s的dnd5e人物Buff属性设置如下:\n", getPlayerNameTempFunc(mctx))
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
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if ctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	spellSlotsRenew := func(mctx *MsgContext, _ *Message) int {
		num := 0
		for i := 1; i < 10; i++ {
			// _, _ := VarGetValueInt64(mctx, fmt.Sprintf("$法术位_%d", i))
			spellLevelMax, exists := VarGetValueInt64(mctx, fmt.Sprintf("$法术位上限_%d", i))
			if exists {
				num++
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
			ReplyToSender(mctx, msg, fmt.Sprintf(`%s无法消耗%d个%d环法术位，当前%d个`, getPlayerNameTempFunc(mctx), -num, spellLevel, spellLevelCur))
			return &CmdExecuteResult{Matched: true, Solved: true}
		}
		if newLevel > spellLevelMax {
			newLevel = spellLevelMax
		}
		VarSetValueInt64(mctx, fmt.Sprintf("$法术位_%d", spellLevel), newLevel)
		if num < 0 {
			ReplyToSender(mctx, msg, fmt.Sprintf(`%s的%d环法术位消耗至%d个，上限%d个`, getPlayerNameTempFunc(mctx), spellLevel, newLevel, spellLevelMax))
		} else {
			ReplyToSender(mctx, msg, fmt.Sprintf(`%s的%d环法术位恢复至%d个，上限%d个`, getPlayerNameTempFunc(mctx), spellLevel, newLevel, spellLevelMax))
		}
		if mctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
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
		Name:          "ss",
		ShortHelp:     helpSS,
		Help:          "DND5E 法术位(.ss .法术位):\n" + helpSS,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("init", "set")

			val := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)

			switch val {
			case "init":
				reSlot := regexp.MustCompile(`\d+`)
				slots := reSlot.FindAllString(cmdArgs.CleanArgs, -1)
				if len(slots) > 0 {
					var texts []string
					for index, levelVal := range slots {
						val, _ := strconv.ParseInt(levelVal, 10, 64)
						VarSetValueInt64(mctx, fmt.Sprintf("$法术位_%d", index+1), val)
						VarSetValueInt64(mctx, fmt.Sprintf("$法术位上限_%d", index+1), val)
						texts = append(texts, fmt.Sprintf("%d环%d个", index+1, val))
					}
					ReplyToSender(mctx, msg, "为"+getPlayerNameTempFunc(mctx)+"设置法术位: "+strings.Join(texts, ", "))
					if ctx.Player.AutoSetNameTemplate != "" {
						_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
					}
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

			case "clr":
				attrs, _ := mctx.Dice.AttrsManager.LoadByCtx(mctx)
				for i := 1; i < 10; i++ {
					attrs.Delete(fmt.Sprintf("$法术位_%d", i))
					attrs.Delete(fmt.Sprintf("$法术位上限_%d", i))
				}
				ReplyToSender(mctx, msg, fmt.Sprintf(`%s法术位数据已清除`, getPlayerNameTempFunc(mctx)))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}

			case "rest":
				n := spellSlotsRenew(mctx, msg)
				if n > 0 {
					ReplyToSender(mctx, msg, fmt.Sprintf(`%s的法术位已经完全恢复`, getPlayerNameTempFunc(mctx)))
				} else {
					ReplyToSender(mctx, msg, fmt.Sprintf(`%s并没有设置过法术位`, getPlayerNameTempFunc(mctx)))
				}
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}

			case "set":
				reSlot := regexp.MustCompile(`(\d+)[环cC]\s*(\d+)|[lL][vV](\d+)\s+(\d+)`)
				slots := reSlot.FindAllStringSubmatch(cmdArgs.CleanArgs, -1)
				if len(slots) > 0 {
					var texts []string
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
					ReplyToSender(mctx, msg, "为"+getPlayerNameTempFunc(mctx)+"设置法术位: "+strings.Join(texts, ", "))
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
			case "":
				var texts []string
				for i := 1; i < 10; i++ {
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
				ReplyToSender(mctx, msg, fmt.Sprintf(`%s的法术位状况: %s`, getPlayerNameTempFunc(mctx), summary))
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			default:
				reSlot := regexp.MustCompile(`(\d+)[环cC]\s*([+-＋－])(\d+)|[lL][vV](\d+)\s*([+-＋－])(\d+)`)
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
						if op == "-" || op == "－" {
							iLevelVal = -iLevelVal
						}

						ret := spellSlotsChange(mctx, msg, iLevel, iLevelVal)
						if ret != nil {
							return *ret
						}
					}
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: false}
		},
	}

	helpCast := "" +
		".cast 1 // 消耗1个1环法术位\n" +
		".cast 1 2 // 消耗2个1环法术位"

	cmdCast := &CmdItemInfo{
		Name:          "cast",
		ShortHelp:     helpCast,
		Help:          "DND5E 法术位使用(.cast):\n" + helpCast,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)

			switch val { //nolint:gocritic
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
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpLongRest := "" +
		".长休 // 恢复生命值(必须设置hpmax且hp>0)和法术位 \n" +
		".longrest // 另一种写法"

	cmdLongRest := &CmdItemInfo{
		Name:          "长休",
		ShortHelp:     helpLongRest,
		Help:          "DND5E 长休:\n" + helpLongRest,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			mctx.Player.TempValueAlias = &ac.Alias // 防止找不到hpmax

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
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
				ReplyToSender(mctx, msg, fmt.Sprintf(`%s的长休: `+hpText+ssText, getPlayerNameTempFunc(mctx)))
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpDeathSavingThrow := "" +
		".死亡豁免 // 进行死亡豁免检定 \n" +
		".ds // 另一种写法\n" +
		".ds +1d4 // 检定时添加1d4的加值\n" +
		".ds 成功±1 // 死亡豁免成功±1，可简写为.ds s±1\n" +
		".ds 失败±1 // 死亡豁免失败±1，可简写为.ds f±1\n" +
		".ds stat // 查看当前死亡豁免情况\n" +
		".ds help // 查看帮助"

	cmdDeathSavingThrow := &CmdItemInfo{
		Name:          "死亡豁免",
		ShortHelp:     helpDeathSavingThrow,
		Help:          "DND5E 死亡豁免:\n" + helpDeathSavingThrow,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			mctx.Player.TempValueAlias = &ac.Alias

			restText := cmdArgs.CleanArgs
			re := regexp.MustCompile(`^(s|S|成功|f|F|失败)([+-＋－])`)
			m := re.FindStringSubmatch(restText)
			if len(m) > 0 {
				restText = strings.TrimSpace(restText[len(m[0]):])
				isNeg := m[2] == "-" || m[2] == "－"
				r, _, err := ctx.Dice.ExprEvalBase(restText, mctx, RollExtraFlags{})
				if err != nil {
					ReplyToSender(mctx, msg, "错误: 无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				v, _ := r.ReadInt64()
				if isNeg {
					v = -v
				}

				var a, b int64
				switch m[1] {
				case "s", "S", "成功":
					a, b = deathSaving(mctx, v, 0)
				case "f", "F", "失败":
					a, b = deathSaving(mctx, 0, v)
				}
				text := fmt.Sprintf("%s当前的死亡豁免情况: 成功%d 失败%d", getPlayerNameTempFunc(mctx), a, b)
				exText := deathSavingResultCheck(mctx, a, b)
				if exText != "" {
					text += "\n" + exText
				}

				ReplyToSender(mctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			val := cmdArgs.GetArgN(1)
			switch val {
			case "stat":
				a, b := deathSaving(mctx, 0, 0)
				text := fmt.Sprintf("%s当前的死亡豁免情况: 成功%d 失败%d", getPlayerNameTempFunc(mctx), a, b)
				ReplyToSender(mctx, msg, text)
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "":
				fallthrough
			default:
				hp, exists := VarGetValueInt64(mctx, "hp")
				if !exists {
					ReplyToSender(mctx, msg, fmt.Sprintf(`%s未设置生命值，无法进行死亡豁免检定。`, getPlayerNameTempFunc(mctx)))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if hp > 0 {
					ReplyToSender(mctx, msg, fmt.Sprintf(`%s生命值大于0(当前为%d)，无法进行死亡豁免检定。`, getPlayerNameTempFunc(mctx), hp))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				restText := cmdArgs.CleanArgs
				re := regexp.MustCompile(`^优势|劣势`)
				m := re.FindString(restText)
				if m != "" {
					restText = strings.TrimSpace(restText[len(m):])
				}
				expr := fmt.Sprintf("D20%s%s", m, restText)
				r, detail, err := mctx.Dice.ExprEvalBase(expr, mctx, RollExtraFlags{DNDAttrReadMod: true, DNDAttrReadDC: true})
				if err != nil {
					ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				d20, ok := r.ReadInt64()
				if !ok {
					ReplyToSender(mctx, msg, "并非数值类型: "+r.Matched)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if d20 == 20 {
					deathSavingStable(mctx)
					VarSetValueInt64(mctx, "hp", 1)
					suffix := DiceFormatTmpl(mctx, "DND:死亡豁免_D20_附加语")
					ReplyToSender(mctx, msg, fmt.Sprintf(`%s的死亡豁免检定: %s=%d %s`, getPlayerNameTempFunc(mctx), detail, d20, suffix))
				} else if d20 == 1 {
					suffix := DiceFormatTmpl(mctx, "DND:死亡豁免_D1_附加语")
					text := fmt.Sprintf(`%s的死亡豁免检定: %s=%d %s`, getPlayerNameTempFunc(mctx), detail, d20, suffix)
					a, b := deathSaving(mctx, 0, 2)
					exText := deathSavingResultCheck(mctx, a, b)
					if exText != "" {
						text += "\n" + exText
					}
					text += fmt.Sprintf("\n当前情况: 成功%d 失败%d", a, b)
					ReplyToSender(mctx, msg, text)
				} else if d20 >= 10 {
					suffix := DiceFormatTmpl(mctx, "DND:死亡豁免_成功_附加语")
					text := fmt.Sprintf(`%s的死亡豁免检定: %s=%d %s`, getPlayerNameTempFunc(mctx), detail, d20, suffix)
					a, b := deathSaving(mctx, 1, 0)
					exText := deathSavingResultCheck(mctx, a, b)
					if exText != "" {
						text += "\n" + exText
					}
					text += fmt.Sprintf("\n当前情况: 成功%d 失败%d", a, b)
					ReplyToSender(mctx, msg, text)
				} else {
					suffix := DiceFormatTmpl(mctx, "DND:死亡豁免_失败_附加语")
					text := fmt.Sprintf(`%s的死亡豁免检定: %s=%d %s`, getPlayerNameTempFunc(mctx), detail, d20, suffix)
					a, b := deathSaving(mctx, 0, 1)
					exText := deathSavingResultCheck(mctx, a, b)
					if exText != "" {
						text += "\n" + exText
					}
					text += fmt.Sprintf("\n当前情况: 成功%d 失败%d", a, b)
					ReplyToSender(mctx, msg, text)
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpDnd := ".dnd (<数量>) // 制卡指令，返回<数量>组人物属性，最高为10次\n" +
		".dndx (<数量>) // 制卡指令，但带有属性名，最高为10次"

	cmdDnd := &CmdItemInfo{
		Name:      "dnd",
		ShortHelp: helpDnd,
		Help:      "DND5E制卡指令:\n" + helpDnd,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			isMode2 := cmdArgs.Command == "dndx"
			n := cmdArgs.GetArgN(1)
			val, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				if n == "" {
					val = 1 // 数量不存在时，视为1次
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
			}
			if val > 10 {
				val = 10
			}
			var i int64

			var ss []string
			for i = 0; i < val; i++ {
				if isMode2 {
					result, _, err := self.ExprText(`力量:{$t1=4d6k3} 体质:{$t2=4d6k3} 敏捷:{$t3=4d6k3} 智力:{$t4=4d6k3} 感知:{$t5=4d6k3} 魅力:{$t6=4d6k3} 共计:{$tT=$t1+$t2+$t3+$t4+$t5+$t6}`, ctx)
					if err != nil {
						break
					}
					result = strings.ReplaceAll(result, `\n`, "\n")
					ss = append(ss, result)
				} else {
					result, _, err := self.ExprText(`{4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}`, ctx)
					if err != nil {
						break
					}

					var nums Int64SliceDesc
					total := int64(0)
					for _, i := range strings.Split(result, ", ") {
						val, _ := strconv.ParseInt(i, 10, 64)
						nums = append(nums, val)
						total += val
					}
					sort.Sort(nums)

					var items []string
					for _, i := range nums {
						items = append(items, strconv.FormatInt(i, 10))
					}

					ret := fmt.Sprintf("[%s] = %d", strings.Join(items, ", "), total)
					ss = append(ss, ret)
				}
			}
			sep := DiceFormatTmpl(ctx, "DND:制卡_分隔符")
			info := strings.Join(ss, sep)
			if isMode2 {
				ReplyToSender(ctx, msg, fmt.Sprintf("%s的DnD5e人物作成(预设模式):\n%s\n自由分配模式请用.dnd", getPlayerNameTempFunc(ctx), info))
			} else {
				ReplyToSender(ctx, msg, fmt.Sprintf("%s的DnD5e人物作成(自由分配模式):\n%s\n获取带属性名的预设请用.dndx", getPlayerNameTempFunc(ctx), info))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpRi := `.ri 小明 // 格式1，值为D20
.ri 12 张三 // 格式2，值12(只能写数字)
.ri +2 李四 // 格式3，值为D20+2
.ri =D10+3 王五 // 格式4，值为D10+3
.ri 张三, +2 李四, =D10+3 王五 // 设置全部
.ri 优势 张三, 劣势-1 李四 // 支持优势劣势`
	cmdRi := &CmdItemInfo{
		Name:          "ri",
		ShortHelp:     helpRi,
		Help:          "先攻设置:\n" + helpRi,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			text := cmdArgs.CleanArgs
			mctx := GetCtxProxyFirst(ctx, cmdArgs)

			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			readOne := func() (int, string, int64, string, string) {
				text = strings.TrimSpace(text)
				var name string
				var val int64
				var detail string
				var exprExists bool
				var uid string

				if strings.HasPrefix(text, "+") {
					// 加值情况1，D20+
					r, _detail, err := ctx.Dice.ExprEvalBase("D20"+text, mctx, RollExtraFlags{})
					if err != nil {
						// 情况1，加值输入错误
						return 1, name, val, detail, ""
					}
					detail = _detail
					val = r.Value.(int64)
					text = r.restInput
					exprExists = true
				} else if strings.HasPrefix(text, "-") {
					// 加值情况1.1，D20-
					r, _detail, err := ctx.Dice.ExprEvalBase("D20"+text, mctx, RollExtraFlags{})
					if err != nil {
						// 情况1，加值输入错误
						return 1, name, val, detail, ""
					}
					detail = _detail
					val = r.Value.(int64)
					text = r.restInput
					exprExists = true
				} else if strings.HasPrefix(text, "=") {
					// 加值情况1，=表达式
					r, _, err := ctx.Dice.ExprEvalBase(text[1:], mctx, RollExtraFlags{})
					if err != nil {
						// 情况1，加值输入错误
						return 1, name, val, detail, ""
					}
					val = r.Value.(int64)
					text = r.restInput
					exprExists = true
				} else if strings.HasPrefix(text, "优势") || strings.HasPrefix(text, "劣势") {
					// 优势/劣势
					r, _detail, err := ctx.Dice.ExprEvalBase("D20"+text, mctx, RollExtraFlags{})
					if err != nil {
						// 优势劣势输入错误
						return 2, name, val, detail, ""
					}
					detail = _detail
					val = r.Value.(int64)
					text = r.restInput
					exprExists = true
				} else {
					// 加值情况3，数字
					reNum := regexp.MustCompile(`^(\d+)`)
					m := reNum.FindStringSubmatch(text)
					if len(m) > 0 {
						val, _ = strconv.ParseInt(m[0], 10, 64)
						text = text[len(m[0]):]
						exprExists = true
					}
				}

				// 清理读取了第一项文本之后的空格
				text = strings.TrimSpace(text)

				if strings.HasPrefix(text, ",") || strings.HasPrefix(text, "，") || text == "" {
					// 句首有,的话，吃掉
					text = strings.TrimPrefix(text, ",")
					text = strings.TrimPrefix(text, "，")
					// 情况1，名字是自己
					name = mctx.Player.Name
					// 情况2，名字是自己，没有加值
					if !exprExists {
						val = DiceRoll64(20)
					}
					uid = mctx.Player.UserID
					return 0, name, val, detail, uid
				}

				// 情况3: 是名字
				reName := regexp.MustCompile(`^([^\s\d,，][^\s,，]*)\s*[,，]?`)
				m := reName.FindStringSubmatch(text)
				if len(m) > 0 {
					name = m[1]
					text = text[len(m[0]):]
					if !exprExists {
						val = DiceRoll64(20)
					}
				} else {
					// 不知道是啥，报错
					return 2, name, val, detail, ""
				}

				return 0, name, val, detail, ""
			}

			solved := true
			tryOnce := true
			var items ByRIListValue

			for tryOnce || text != "" {
				code, name, val, detail, uid := readOne()
				items = append(items, &RIListItem{name, val, detail, uid})

				if code != 0 {
					solved = false
					break
				}
				tryOnce = false
			}

			if solved {
				riMap, uidMap := dndGetRiMapList(ctx)
				textOut := DiceFormatTmpl(mctx, "DND:先攻_设置_前缀")
				sort.Sort(items)
				for order, i := range items {
					var detail string
					riMap[i.name] = i.val
					uidMap[i.name] = i.uid
					if i.detail != "" {
						detail = i.detail + "="
					}
					textOut += fmt.Sprintf("%2d. %s: %s%d\n", order+1, i.name, detail, i.val)
				}

				dndSetRiMapList(mctx, riMap, uidMap)
				ReplyToSender(ctx, msg, textOut)
			} else {
				ReplyToSender(ctx, msg, DiceFormatTmpl(mctx, "DND:先攻_设置_格式错误"))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	cmdInit := &CmdItemInfo{
		Name: "init",
		ShortHelp: ".init // 查看先攻列表\n" +
			".init del <单位1> <单位2> ... // 从先攻列表中删除\n" +
			".init set <单位名称> <先攻表达式> // 设置单位的先攻\n" +
			".init clr // 清除先攻列表\n" +
			".init end // 结束一回合" +
			".init help // 显示本帮助",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("del", "set", "rm", "ed")
			n := cmdArgs.GetArgN(1)
			switch n {
			case "", "list":
				textOut := DiceFormatTmpl(ctx, "DND:先攻_查看_前缀")
				riMap, uidMap := dndGetRiMapList(ctx)
				round, _ := VarGetValueInt64(ctx, "$g回合数")
				lst := dndRiMapToList(riMap, uidMap)

				for order, i := range lst {
					textOut += fmt.Sprintf("%2d. %s: %d\n", order+1, i.name, i.val)
				}

				if len(lst) == 0 {
					textOut += "- 没有找到任何单位"
				} else {
					if len(lst) <= int(round) || round < 0 {
						round = 0
					}
					rounder := lst[round]
					textOut += fmt.Sprintf("当前回合：%s", rounder.name)
				}

				ReplyToSender(ctx, msg, textOut)
			case "ed", "end":
				riMap, uidMap := dndGetRiMapList(ctx)
				round, _ := VarGetValueInt64(ctx, "$g回合数")
				lst := dndRiMapToList(riMap, uidMap)
				if len(lst) == 0 {
					ReplyToSender(ctx, msg, "先攻列表为空")
					break
				}
				round++
				l := len(lst)
				if l <= int(round) || round < 0 {
					round = 0
				}
				if round == 0 {
					VarSetValueStr(ctx, "$t当前回合角色名", lst[l-1].name)
					VarSetValueStr(ctx, "$t当前回合at", AtBuild(lst[l-1].uid))
				} else {
					VarSetValueStr(ctx, "$t当前回合角色名", lst[round-1].name)
					VarSetValueStr(ctx, "$t当前回合at", AtBuild(lst[round-1].uid))
				}
				VarSetValueStr(ctx, "$t下一回合角色名", lst[round].name)
				VarSetValueStr(ctx, "$t下一回合at", AtBuild(lst[round].uid))

				nextRound := round + 1
				if l <= int(nextRound) || nextRound < 0 {
					nextRound = 0
				}
				VarSetValueStr(ctx, "$t下下一回合角色名", lst[nextRound].name)
				VarSetValueStr(ctx, "$t下下一回合at", AtBuild(lst[nextRound].uid))

				VarSetValueInt64(ctx, "$g回合数", round)

				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:先攻_下一回合"))
			case "del", "rm":
				names := cmdArgs.Args[1:]
				riMap, uidMap := dndGetRiMapList(ctx)
				riList := dndRiMapToList(riMap, uidMap)

				deleted := map[string]bool{}
				for _, i := range names {
					deleted[i] = false
				}

				round, _ := VarGetValueInt64(ctx, "$g回合数")
				round %= int64(len(riList))

				preCurrent := 0 // 每有一个在当前单位前面的单位被删除, 当前单位下标需要减 1
				for i, v := range riList {
					if _, exist := deleted[v.name]; exist {
						deleted[v.name] = true
						if int64(i) < round {
							preCurrent++
						}
						delete(riMap, v.name)
					}
				}

				round -= int64(preCurrent)
				VarSetValueInt64(ctx, "$g回合数", round)

				textOut := DiceFormatTmpl(ctx, "DND:先攻_移除_前缀")
				delCounter := 0
				for _, name := range names {
					if deleted[name] {
						delCounter++
						textOut += fmt.Sprintf("%2d. %s\n", delCounter, name)
					}
				}
				if delCounter == 0 {
					textOut += "- 没有找到任何单位"
				}

				dndSetRiMapList(ctx, riMap, uidMap)
				ReplyToSender(ctx, msg, textOut)
			case "set":
				name := cmdArgs.GetArgN(2)
				exists := name != ""
				arg3 := cmdArgs.GetArgN(3)
				exists2 := arg3 != ""
				if !exists || !exists2 {
					ReplyToSender(ctx, msg, "错误的格式，应为: .init set <单位名称> <先攻表达式>")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				expr := strings.Join(cmdArgs.Args[2:], "")
				r, _detail, err := ctx.Dice.ExprEvalBase(expr, ctx, RollExtraFlags{})
				if err != nil || r.TypeID != VMTypeInt64 {
					ReplyToSender(ctx, msg, "错误的格式，应为: .init set <单位名称> <先攻表达式>")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				riMap, uidMap := dndGetRiMapList(ctx)
				riMap[name] = r.Value.(int64)

				VarSetValueStr(ctx, "$t表达式", expr)
				VarSetValueStr(ctx, "$t目标", name)
				VarSetValueStr(ctx, "$t计算过程", _detail)
				_VarSetValueV1(ctx, "$t点数", &r.VMValue)
				textOut := DiceFormatTmpl(ctx, "DND:先攻_设置_指定单位")

				dndSetRiMapList(ctx, riMap, uidMap)
				ReplyToSender(ctx, msg, textOut)
			case "clr", "clear":
				dndClearRiMapList(ctx)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "DND:先攻_清除列表"))
				VarSetValueInt64(ctx, "$g回合数", 0)
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	theExt := &ExtInfo{
		Name:       "dnd5e", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "提供DND5E规则TRPG支持",
		Author:     "木落",
		AutoActive: true, // 是否自动开启
		Official:   true,
		ConflictWith: []string{
			"coc7",
		},
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"dnd":  cmdDnd,
			"dndx": cmdDnd,
			"ri":   cmdRi,
			"init": cmdInit,
			// "属性":    cmdSt,
			"st":         cmdSt,
			"dst":        cmdSt,
			"rc":         cmdRc,
			"ra":         cmdRc,
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
			"ds":         cmdDeathSavingThrow,
			"死亡豁免":       cmdDeathSavingThrow,
		},
	}

	self.RegisterExtension(theExt)
}

func dndGetRiMapList(ctx *MsgContext) (map[string]int64, map[string]string) {
	// TODO: 类型对不上，先屏蔽了，等待重写
	return map[string]int64{}, map[string]string{}
	// am := ctx.Dice.AttrsManager
	// attrs, _ := am.LoadById(ctx.Group.GroupID)
	//
	// riMapName := "riMapList"
	// _, exists := attrs.LoadX(riMapName)
	// if !exists {
	//	attrs.Store(riMapName, &VMValue{TypeID: -1, Value: map[string]int64{}})
	// } else {
	//	a, _ := attrs.LoadX(riMapName)
	//	attrs.Set(riMapName, VMValueConvert(a.(*VMValue), nil, ""))
	// }
	// uidMapName := "uidMapList"
	// _, exists = attrs.LoadX(uidMapName)
	// if !exists {
	//	attrs.Set(uidMapName, &VMValue{TypeID: -2, Value: map[string]string{}})
	// } else {
	//	b, _ := attrs.LoadX(uidMapName)
	//	attrs.Set(uidMapName, VMValueConvert(b.(*VMValue), nil, ""))
	// }
	//
	// var riList, uidList *VMValue
	// v, e := attrs.Get(riMapName)
	// if e {
	//	riList = v.(*VMValue)
	// }
	// v2, e := attrs.LoadX(uidMapName)
	// if e {
	//	uidList = v2.(*VMValue)
	// }
	// return riList.Value.(map[string]int64), uidList.Value.(map[string]string)
}

func dndSetRiMapList(ctx *MsgContext, riMap map[string]int64, uidMap map[string]string) {
	// riMapName := "riMapList"
	// attrs.Set(riMapName, &VMValue{TypeID: -1, Value: riMap})
	//
	// uidMapName := "uidMapList"
	// attrs.Set(uidMapName, &VMValue{TypeID: -2, Value: uidMap})
}

func dndClearRiMapList(ctx *MsgContext) {
	dndSetRiMapList(ctx, map[string]int64{}, map[string]string{})
}

func dndRiMapToList(riMap map[string]int64, uidMap map[string]string) ByRIListValue {
	var lst ByRIListValue
	for k, v := range riMap {
		lst = append(lst, &RIListItem{name: k, val: v, uid: uidMap[k]})
	}
	sort.Sort(lst)
	return lst
}
