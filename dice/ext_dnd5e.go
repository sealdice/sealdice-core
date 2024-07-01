package dice

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	ds "github.com/sealdice/dicescript"
)

type RIListItem struct {
	name   string
	val    int64
	detail string
	uid    string
}

type RIList []*RIListItem

func (lst RIList) Len() int {
	return len(lst)
}
func (lst RIList) Swap(i, j int) {
	lst[i], lst[j] = lst[j], lst[i]
}
func (lst RIList) Less(i, j int) bool {
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
	// 如果不存在，新建
	defaultVals := AttributeConfigs{
		Alias: map[string][]string{},
		Order: AttributeOrder{
			Top:    []string{"力量", "敏捷", "体质", "体型", "魅力", "智力", "感知", "hp", "ac", "熟练"},
			Others: AttributeOrderOthers{SortBy: "Name"},
		},
	}
	return defaultVals
}

func getPlayerNameTempFunc(mctx *MsgContext) string {
	if mctx.Dice.PlayerNameWrapEnable {
		return fmt.Sprintf("<%s>", mctx.Player.Name)
	}
	return mctx.Player.Name
}

func isAbilityScores(name string) bool {
	for _, i := range []string{"力量", "敏捷", "体质", "智力", "感知", "魅力"} {
		if i == name {
			return true
		}
	}
	return false
}

func stpFormat(attrName string) string {
	return "$stp_" + attrName
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

	toExport := func(ctx *MsgContext, key string, val *ds.VMValue, tmpl *GameSystemTemplate) string {
		if dndAttrParent[key] != "" && val.TypeId == ds.VMTypeComputedValue {
			cd, _ := val.ReadComputed()
			base, _ := cd.Attrs.Load("base")
			factor, _ := cd.Attrs.Load("factor")
			if base != nil {
				if factor != nil {
					if ds.ValueEqual(factor, ds.NewIntVal(1), true) {
						return fmt.Sprintf("%s*:%s", key, base.ToRepr())
					} else {
						return fmt.Sprintf("%s*%s:%s", key, factor.ToRepr(), base.ToRepr())
					}
				} else {
					return fmt.Sprintf("%s:%s", key, base.ToRepr())
				}
			}
		}

		if isAbilityScores(key) {
			// 如果为主要属性，同时读取豁免值
			attrs, _ := ctx.Dice.AttrsManager.LoadByCtx(ctx)
			stpKey := stpFormat(key)
			// 注: 如果这里改成 eval，是不是即使原始值为computed也可以？
			if v, exists := attrs.LoadX(stpKey); exists && (v.TypeId == ds.VMTypeInt || v.TypeId == ds.VMTypeFloat) {
				if ds.ValueEqual(v, ds.NewIntVal(1), true) {
					return fmt.Sprintf("%s*:%s", key, val.ToRepr())
				} else {
					return fmt.Sprintf("%s*%s:%s", key, v.ToRepr(), val.ToRepr())
				}
			}
		}
		return ""
	}

	cmdSt := getCmdStBase(CmdStOverrideInfo{
		Help:         helpSt,
		TemplateName: "dnd5e",
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
			ctx.setDndReadForVM(false)
			return nil
		},
		ToMod: func(ctx *MsgContext, args *CmdArgs, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate) bool {
			over := args.GetKwarg("over")
			attrName := tmpl.GetAlias(i.name)
			if attrName == "hp" && over != nil {
				hpBuff := attrs.Load("$buff_hp")
				if hpBuff == nil {
					hpBuff = ds.NewIntVal(0)
				}

				// 如果是生命值，先试图扣虚血
				vHpBuffVal := hpBuff.MustReadInt()
				// 正盾才做反馈
				if vHpBuffVal > 0 {
					val := vHpBuffVal - i.value.MustReadInt()
					if val >= 0 {
						// 有充足的盾，扣掉，当前伤害改为0
						attrs.Store("$buff_hp", ds.NewIntVal(val))
						i.value = ds.NewIntVal(0)
					} else {
						// 没有充足的盾，盾扣到0，剩下的继续造成伤害
						attrs.Delete("$buff_hp")
						i.value = ds.NewIntVal(val)
					}
				}
			}

			// 处理技能
			parent := dndAttrParent[attrName]
			if parent != "" {
				val := attrs.Load(attrName)
				if val.TypeId == ds.VMTypeComputedValue {
					cd, _ := val.ReadComputed()
					base, _ := cd.Attrs.Load("base")
					if base == nil {
						base = ds.NewIntVal(0)
					}
					vNew := base.OpAdd(ctx.vm, i.value)
					if vNew != nil {
						cd.Attrs.Store("base", vNew)
						return true
					}
				}
			}

			return false
		},
		ToSet: func(ctx *MsgContext, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate) bool {
			attrName := tmpl.GetAlias(i.name)
			parent := dndAttrParent[attrName]
			if parent != "" {
				m := ds.ValueMap{}
				m.Store("base", i.value)

				if i.extra != nil {
					m.Store("factor", i.extra)
				} else {
					m.Delete("factor")
				}
				i.value = ds.NewComputedValRaw(&ds.ComputedData{
					// Expr: fmt.Sprintf("this.base + ((%s)??0)/2 - 5 + (熟练??0) * this.factor", parent)
					Expr:  fmt.Sprintf("pbCalc(this.base, this.factor, %s)", parent),
					Attrs: &m,
				})
				attrs.Store(attrName, i.value)
				return true
			} else if isAbilityScores(attrName) {
				// 如果为主要属性，同时读取豁免值
				if i.extra != nil {
					attrs.Store(stpFormat(attrName), i.extra)
				} else {
					attrs.Delete(stpFormat(attrName))
				}
				attrs.Store(attrName, i.value)
				return true
			}
			return false
		},
	})

	helpRc := "" +
		".rc <属性> // .rc 力量\n" +
		".rc <属性>豁免 // .rc 力量豁免\n" +
		".rc <表达式> // .rc 力量+3\n" +
		".rc 优势 <表达式> // .rc 优势 力量+4\n" +
		".rc 劣势 <表达式> [<原因>] // .rc 劣势 力量+4 推一下试试\n" +
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
				ctx.CreateVmIfNotExists()
				tmpl := ctx.Group.GetCharTemplate(ctx.Dice)
				mctx.Eval(tmpl.PreloadCode, nil)
				ctx.setDndReadForVM(true)

				r := ctx.Eval(expr, nil)
				if r.vm.Error != nil {
					fmt.Println("xxx", restText)
					ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				reason := r.vm.RestInput
				if reason == "" {
					reason = restText
				}
				detail := r.vm.GetDetailText()

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

	cmdBuff := getCmdStBase(CmdStOverrideInfo{
		Help:         helpBuff,
		HelpPrefix:   "属性临时加值，语法同st一致:\n",
		TemplateName: "dnd5e",
		CommandSolve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) *CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			var tmpl *GameSystemTemplate
			if tmpl2, _ := ctx.Dice.GameSystemMap.Load("dnd5e"); tmpl2 != nil {
				tmpl = tmpl2
			}
			attrs, _ := ctx.Dice.AttrsManager.LoadByCtx(ctx)

			switch val {
			case "export":
				return &CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

			case "del", "rm":
				var nums []string
				var failed []string

				for _, varname := range cmdArgs.Args[1:] {
					vname := tmpl.GetAlias(varname)
					realname := "$buff_" + vname

					if _, exists := attrs.LoadX(realname); exists {
						nums = append(nums, vname)
						attrs.Delete(realname)
					} else {
						failed = append(failed, vname)
					}
				}

				VarSetValueStr(ctx, "$t属性列表", strings.Join(nums, " "))
				VarSetValueInt64(ctx, "$t失败数量", int64(len(failed)))
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:属性设置_删除"))
				return &CmdExecuteResult{Matched: true, Solved: true}

			case "clr", "clear":
				var toDelete []string
				attrs.Range(func(key string, value *ds.VMValue) bool {
					if strings.HasPrefix(key, "$buff_") {
						toDelete = append(toDelete, key)
					}
					return true
				})
				for _, i := range toDelete {
					attrs.Delete(i)
				}
				VarSetValueInt64(ctx, "$t数量", int64(len(toDelete)))
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:属性设置_清除"))
				return &CmdExecuteResult{Matched: true, Solved: true}

			case "show", "list":
				pickItems, _ := cmdStGetPickItemAndLimit(tmpl, cmdArgs)

				var items []string
				// 或者有pickItems，或者当前的变量数量大于0
				if len(pickItems) > 0 {
					for key := range pickItems {
						if value, exists := attrs.LoadX("$buff_" + key); exists {
							items = append(items, fmt.Sprintf("%s:%s", key, value.ToString()))
						} else {
							items = append(items, fmt.Sprintf("%s:0", key))
						}
					}
				} else if attrs.Len() > 0 {
					attrs.Range(func(key string, value *ds.VMValue) bool {
						if strings.HasPrefix(key, "$buff_") {
							items = append(items, fmt.Sprintf("%s:%s", strings.TrimPrefix(key, "$buff_"), value.ToString()))
						}
						return true
					})
				}

				// 每四个一行，拼起来
				itemsPerLine := tmpl.AttrConfig.ItemsPerLine
				if itemsPerLine <= 1 {
					itemsPerLine = 4
				}

				tick := 0
				info := ""
				for _, i := range items {
					tick++
					info += i
					if tick%itemsPerLine == 0 {
						info += "\n"
					} else {
						info += "\t"
					}
				}

				// 再拼点附加信息，然后输出
				if info == "" {
					info = DiceFormatTmpl(ctx, "COC:属性设置_列出_未发现记录")
				}

				VarSetValueStr(ctx, "$t属性信息", info)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "COC:属性设置_列出"))
				return &CmdExecuteResult{Matched: true, Solved: true}
			}

			ctx.setDndReadForVM(false)
			return nil
		},
		ToExport: toExport,
		ToSet: func(ctx *MsgContext, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate) bool {
			attrName := tmpl.GetAlias(i.name)
			i.name = "$buff_" + attrName
			return false
		},
		ToMod: func(ctx *MsgContext, args *CmdArgs, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate) bool {
			attrName := tmpl.GetAlias(i.name)
			i.name = "$buff_" + attrName
			return false
		},
	})

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
			tmpl := ctx.Group.GetCharTemplate(ctx.Dice)
			mctx.Player.TempValueAlias = &tmpl.Alias // 防止找不到hpmax

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
			tmpl := ctx.Group.GetCharTemplate(ctx.Dice)
			mctx.Player.TempValueAlias = &tmpl.Alias

			restText := cmdArgs.CleanArgs
			re := regexp.MustCompile(`^(s|S|成功|f|F|失败)([+-＋－])`)
			m := re.FindStringSubmatch(restText)
			if len(m) > 0 {
				restText = strings.TrimSpace(restText[len(m[0]):])
				isNeg := m[2] == "-" || m[2] == "－"
				r := ctx.Eval(restText, nil)
				if r.vm.Error != nil {
					ReplyToSender(mctx, msg, "错误: 无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				_v, _ := r.ReadInt()
				v := int64(_v)
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
				mctx.CreateVmIfNotExists()
				mctx.setDndReadForVM(true)
				r := mctx.Eval(expr, nil)
				if r.vm.Error != nil {
					ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				d20, ok := r.ReadInt()
				if !ok {
					ReplyToSender(mctx, msg, "并非数值类型: "+r.vm.Matched)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				detail := r.vm.GetDetailText()
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

	helpDnd := ".dnd [<数量>] // 制卡指令，返回<数量>组人物属性，最高为10次\n" +
		".dndx [<数量>] // 制卡指令，但带有属性名，最高为10次"

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
					r := ctx.EvalFString(`力量:{$t1=4d6k3} 体质:{$t2=4d6k3} 敏捷:{$t3=4d6k3} 智力:{$t4=4d6k3} 感知:{$t5=4d6k3} 魅力:{$t6=4d6k3} 共计:{$tT=$t1+$t2+$t3+$t4+$t5+$t6}`, nil)
					if r.vm.Error != nil {
						break
					}
					result := r.ToString() + "\n"
					ss = append(ss, result)
				} else {
					r := ctx.EvalFString(`{4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}, {4d6k3}`, nil)
					if r.vm.Error != nil {
						break
					}
					result := r.ToString()

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

					ret := fmt.Sprintf("[%s] = %d\n", strings.Join(items, ", "), total)
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
					r := ctx.Eval("D20"+text, nil)
					if r.vm.Error != nil {
						// 情况1，加值输入错误
						return 1, name, val, detail, ""
					}
					detail = r.vm.GetDetailText()
					val = int64(r.MustReadInt())
					text = r.vm.RestInput
					exprExists = true
				} else if strings.HasPrefix(text, "-") {
					// 加值情况1.1，D20-
					r := ctx.Eval("D20"+text, nil)
					if r.vm.Error != nil {
						// 情况1，加值输入错误
						return 1, name, val, detail, ""
					}
					detail = r.vm.GetDetailText()
					val = int64(r.MustReadInt())
					text = r.vm.RestInput
					exprExists = true
				} else if strings.HasPrefix(text, "=") {
					// 加值情况1，=表达式
					r := ctx.Eval(text[1:], nil)
					if r.vm.Error != nil {
						// 情况1，加值输入错误
						return 1, name, val, detail, ""
					}
					val = int64(r.MustReadInt())
					text = r.vm.GetDetailText()
					exprExists = true
				} else if strings.HasPrefix(text, "优势") || strings.HasPrefix(text, "劣势") {
					// 优势/劣势
					r := ctx.Eval("D20"+text, nil)
					if r.vm.Error != nil {
						// 优势劣势输入错误
						return 2, name, val, detail, ""
					}
					detail = r.vm.GetDetailText()
					val = int64(r.MustReadInt())
					text = r.vm.RestInput
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
						val = int64(ds.Roll(nil, 20))
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
						val = int64(ds.Roll(nil, 20))
					}
				} else {
					// 不知道是啥，报错
					return 2, name, val, detail, ""
				}

				return 0, name, val, detail, ""
			}

			solved := true
			tryOnce := true
			var items RIList

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
				riList := (RIList{}).LoadByCurGroup(ctx)

				textOut := DiceFormatTmpl(mctx, "DND:先攻_设置_前缀")
				sort.Sort(items)
				for order, i := range items {
					var detail string
					if i.detail != "" {
						detail = i.detail + "="
					}
					textOut += fmt.Sprintf("%2d. %s: %s%d\n", order+1, i.name, detail, i.val)

					item := riList.GetExists(i.name)
					if item == nil {
						riList = append(riList, i)
					} else {
						item.val = i.val
					}
				}

				sort.Sort(riList)
				riList.SaveToGroup(ctx)
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
				riList := (RIList{}).LoadByCurGroup(ctx)

				round, _ := VarGetValueInt64(ctx, "$g回合数")

				for order, i := range riList {
					textOut += fmt.Sprintf("%2d. %s: %d\n", order+1, i.name, i.val)
				}

				if len(riList) == 0 {
					textOut += "- 没有找到任何单位"
				} else {
					if len(riList) <= int(round) || round < 0 {
						round = 0
					}
					rounder := riList[round]
					textOut += fmt.Sprintf("当前回合：%s", rounder.name)
				}

				ReplyToSender(ctx, msg, textOut)
			case "ed", "end":
				lst := (RIList{}).LoadByCurGroup(ctx)
				round, _ := VarGetValueInt64(ctx, "$g回合数")
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
				riList := (RIList{}).LoadByCurGroup(ctx)
				newList := RIList{}

				round, _ := VarGetValueInt64(ctx, "$g回合数")
				round %= int64(len(riList))

				toDeleted := map[string]bool{}
				for _, i := range names {
					toDeleted[i] = true
				}

				textOut := DiceFormatTmpl(ctx, "DND:先攻_移除_前缀")
				delCounter := 0

				preCurrent := 0 // 每有一个在当前单位前面的单位被删除, 当前单位下标需要减 1
				for index, i := range riList {
					if !toDeleted[i.name] {
						newList = append(newList, i)
					} else {
						delCounter++
						textOut += fmt.Sprintf("%2d. %s\n", delCounter, i.name)

						if int64(index) < round {
							preCurrent++
						}
					}
				}

				round -= int64(preCurrent)
				VarSetValueInt64(ctx, "$g回合数", round)

				if delCounter == 0 {
					textOut += "- 没有找到任何单位"
				}

				newList.SaveToGroup(ctx)
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
				r := ctx.Eval(expr, nil)
				if r.vm.Error != nil || r.TypeId != ds.VMTypeInt {
					ReplyToSender(ctx, msg, "错误的格式，应为: .init set <单位名称> <先攻表达式>")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				riList := (RIList{}).LoadByCurGroup(ctx)
				for _, i := range riList {
					if i.name == name {
						i.val = int64(r.MustReadInt())
						break
					}
				}

				VarSetValueStr(ctx, "$t表达式", expr)
				VarSetValueStr(ctx, "$t目标", name)
				VarSetValueStr(ctx, "$t计算过程", r.vm.GetDetailText())
				VarSetValue(ctx, "$t点数", &r.VMValue)
				textOut := DiceFormatTmpl(ctx, "DND:先攻_设置_指定单位")

				riList.SaveToGroup(ctx)
				ReplyToSender(ctx, msg, textOut)
			case "clr", "clear":
				(RIList{}).SaveToGroup(ctx)
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

var dndRiLock sync.Mutex

// LoadByCurGroup 从群信息中加载
func (lst RIList) LoadByCurGroup(ctx *MsgContext) RIList {
	am := ctx.Dice.AttrsManager
	attrs, _ := am.LoadById(ctx.Group.GroupID)

	dndRiLock.Lock()
	riList := attrs.Load("riList")
	if riList == nil || riList.TypeId != ds.VMTypeArray {
		riList = ds.NewArrayVal()
		attrs.Store("riList", riList)
	}
	dndRiLock.Unlock()

	ret := RIList{}
	for _, i := range riList.MustReadArray().List {
		if i.TypeId != ds.VMTypeDict {
			continue
		}

		dd := i.MustReadDictData()
		readStr := func(key string) string {
			v, ok := dd.Dict.Load(key)
			if !ok {
				return ""
			}
			return v.ToString()
		}
		readInt := func(key string) ds.IntType {
			v, ok := dd.Dict.Load(key)
			if !ok {
				return 0
			}
			ret, _ := v.ReadInt()
			return ret
		}

		ret = append(ret, &RIListItem{
			name:   readStr("name"),
			val:    int64(readInt("val")),
			uid:    readStr("uid"),
			detail: readStr("detail"),
		})
	}

	return ret
}

// SaveToGroup 写入群信息中
func (lst RIList) SaveToGroup(ctx *MsgContext) {
	am := ctx.Dice.AttrsManager
	attrs, _ := am.LoadById(ctx.Group.GroupID)
	riList := ds.NewArrayVal()

	ad := riList.MustReadArray()
	for _, i := range lst {
		v := ds.NewDictValWithArrayMust(
			ds.NewStrVal("name"), ds.NewStrVal(i.name),
			ds.NewStrVal("val"), ds.NewIntVal(ds.IntType(i.val)),
			ds.NewStrVal("uid"), ds.NewStrVal(i.uid),
			ds.NewStrVal("detail"), ds.NewStrVal(i.detail),
		)
		ad.List = append(ad.List, v.V())
	}

	dndRiLock.Lock()
	attrs.Store("riList", riList)
	dndRiLock.Unlock()
}

func (lst RIList) GetExists(name string) *RIListItem {
	for _, i := range lst {
		if i.name == name {
			return i
		}
	}
	return nil
}
