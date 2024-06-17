package dice

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

func cmdStGetPickItemAndLimit(tmpl *GameSystemTemplate, cmdArgs *CmdArgs) (pickItems map[string]int, limit int64) {
	var usePickItem bool
	if len(cmdArgs.Args) >= 2 {
		arg2 := cmdArgs.GetArgN(2)
		_limit, err := strconv.ParseInt(arg2, 10, 64)
		if err == nil {
			limit = _limit
		} else {
			usePickItem = true
		}
	}

	pickItems = map[string]int{}

	if usePickItem {
		for _, i := range cmdArgs.Args[1:] {
			if tmpl != nil {
				pickItems[tmpl.GetAlias(i)] = 1
			} else {
				pickItems[i] = 1
			}
		}
	}

	return pickItems, limit
}

func cmdStSortNamesByTmpl(mctx *MsgContext, tmpl *GameSystemTemplate, pickItems map[string]int, _ int64) (topNum int, items []string) {
	attrs, _ := mctx.Dice.AttrsManager.LoadByCtx(mctx)
	// 或者有pickItems，或者当前的变量数量大于0
	if len(pickItems) > 0 || attrs.Len() > 0 {
		// 按照配置文件排序
		var attrKeys []string
		used := map[string]bool{}
		for _, key := range tmpl.AttrConfig.Top {
			if used[key] {
				continue
			}
			attrKeys = append(attrKeys, key)
			used[key] = true
		}

		// 其余按字典序
		topNum := len(attrKeys)
		var attrKeys2 []string
		var attrKeys2v []*ds.VMValue

		attrs.Range(func(key string, value *ds.VMValue) bool {
			// 只添加不存在的
			if used[key] {
				return true
			}
			for _, n := range tmpl.AttrConfig.Ignores {
				// 跳过忽略项
				if n == key {
					return true
				}
			}
			attrKeys2 = append(attrKeys2, key)
			attrKeys2v = append(attrKeys2v, value)
			return true
		})

		// 没有pickItem时，按照配置文件排序
		if len(pickItems) == 0 {
			switch tmpl.AttrConfig.SortBy {
			case "value", "value desc":
				isDesc := tmpl.AttrConfig.SortBy == "value desc"
				// 首先变换为可排序形式
				var vals []struct {
					Key string
					Val *ds.VMValue
				}
				for i := range attrKeys2 {
					vals = append(vals, struct {
						Key string
						Val *ds.VMValue
					}{
						Key: attrKeys2[i],
						Val: attrKeys2v[i],
					})
				}

				// Define a custom sorting function
				sortByValue := func(i, j int) bool {
					a := vals[i].Val
					b := vals[j].Val
					if a.TypeId != b.TypeId {
						return a.TypeId < b.TypeId
					}
					if isDesc {
						if v := a.OpCompGT(mctx.vm, b); v != nil {
							return v.MustReadInt() == ds.IntType(1)
						}
					} else {
						if v := a.OpCompLT(mctx.vm, b); v != nil {
							return v.MustReadInt() == ds.IntType(1)
						}
					}
					return true
				}

				sort.Slice(vals, sortByValue)
				for _, i := range vals {
					attrKeys = append(attrKeys, i.Key)
				}
			case "name":
				fallthrough
			default:
				// 排序、合并key
				sort.Strings(attrKeys2)
				attrKeys = append(attrKeys, attrKeys2...)
			}
		}

		if len(pickItems) > 0 {
			attrKeys = []string{}
			for k := range pickItems {
				attrKeys = append(attrKeys, k)
			}
		}

		// 排序完成，返回
		return topNum, attrKeys
	}

	return -1, []string{}
}

func cmdStGetItemsForShow(mctx *MsgContext, tmpl *GameSystemTemplate, pickItems map[string]int, limit int64, stInfo *CmdStOverrideInfo) (items []string, droppedByLimit int, err error) {
	usePickItem := len(pickItems) > 0
	useLimit := limit > 0
	limitSkipCount := 0
	items = []string{}

	topNum, attrKeys := cmdStSortNamesByTmpl(mctx, tmpl, pickItems, limit)

	// 或者有pickItems，或者当前的变量数量大于0
	if len(attrKeys) > 0 {
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

			var v *ds.VMValue
			k, v, err = tmpl.GetShowAs(mctx, k)
			if err != nil {
				return nil, 0, errors.New("模板卡异常, 属性: " + k + "\n报错: " + err.Error())
			}

			if index >= topNum {
				if useLimit && v.TypeId == ds.VMTypeInt && int64(v.MustReadInt()) < limit {
					limitSkipCount++
					continue
				}
			}

			var text string
			if stInfo.ToShow != nil {
				text = stInfo.ToShow(mctx, k, v, tmpl)
			}

			if text != "" {
				items = append(items, text)
			} else {
				items = append(items, fmt.Sprintf("%s:%s", k, v.ToString()))
			}
		}
	}

	return items, limitSkipCount, nil
}

func cmdStGetItemsForExport(mctx *MsgContext, tmpl *GameSystemTemplate, stInfo *CmdStOverrideInfo) (items []string, droppedByLimit int, err error) {
	// 修改自 cmdStGetItemsForShow
	limitSkipCount := 0
	items = []string{}
	_, attrKeys := cmdStSortNamesByTmpl(mctx, tmpl, map[string]int{}, 0)

	// 或者有pickItems，或者当前的变量数量大于0
	if len(attrKeys) > 0 {
		// 遍历输出
		for _, k := range attrKeys {
			if strings.HasPrefix(k, "$") {
				continue
			}

			v, err := tmpl.GetRealValue(mctx, k)
			if err != nil {
				return nil, 0, errors.New("模板卡异常, 属性: " + k)
			}

			var text string
			if stInfo != nil && stInfo.ToExport != nil {
				text = stInfo.ToExport(mctx, k, v, tmpl)
			}

			if text != "" {
				items = append(items, text)
			} else {
				if v.TypeId == ds.VMTypeComputedValue {
					items = append(items, fmt.Sprintf("&%s:%s", k, v.ToString()))
				} else {
					items = append(items, fmt.Sprintf("%s:%s", k, v.ToString()))
				}
			}
		}
	}

	return items, limitSkipCount, nil
}

func cmdStValueMod(mctx *MsgContext, tmpl *GameSystemTemplate, attrs *ds.ValueMap, commandInfo map[string]any, i *stSetOrModInfoItem) {
	// 获取当前值
	curVal, _ := attrs.Load(i.name)
	if curVal == nil {
		curVal = tmpl.GetDefaultValueEx(mctx, i.name)
	}
	if curVal == nil {
		curVal = ds.NewIntVal(0)
	}

	if curVal.TypeId != ds.VMTypeInt {
		// 跳过非数字
		return
	}

	// 进行变更
	theOldValue, _ := curVal.ReadInt()
	theModValue, _ := i.value.ReadInt()
	var theNewValue ds.IntType

	signText := ""
	switch i.op {
	case "+":
		signText = "增加"
		theNewValue = theOldValue + theModValue
	case "-":
		signText = "扣除"
		theNewValue = theOldValue - theModValue
	}

	// 指令信息
	commandInfo["items"] = append(commandInfo["items"].([]interface{}), map[string]interface{}{
		"type":    "mod",
		"attr":    i.name,
		"modExpr": i.expr,
		"valOld":  theOldValue,
		"valNew":  theNewValue,
		"isInc":   signText == "增加", // 增加还是扣除
		"op":      i.op,
	})

	attrs.Store(i.name, ds.NewIntVal(theNewValue))

	VarSetValueStr(mctx, "$t属性", i.name)
	VarSetValueInt64(mctx, "$t旧值", int64(theOldValue))
	VarSetValueInt64(mctx, "$t新值", int64(theNewValue))
	VarSetValueInt64(mctx, "$t变化量", int64(theModValue))
	VarSetValueStr(mctx, "$t增加或扣除", signText)
	VarSetValueStr(mctx, "$t表达式文本", i.expr)
}

type stSetOrModInfoItem struct {
	name  string
	value *ds.VMValue
	extra *ds.VMValue
	op    string
	expr  string
}

func cmdStReadOrMod(ctx *MsgContext, tmpl *GameSystemTemplate, text string) (r *ds.VMValue, toSetItems []*stSetOrModInfoItem, toModItems []*stSetOrModInfoItem, err error) {
	// 处理全角符号
	text = strings.ReplaceAll(text, "＋", "+")
	text = strings.ReplaceAll(text, "－", "-")
	text = strings.ReplaceAll(text, "＝", "=")
	text = strings.ReplaceAll(text, "：", ":")
	text = strings.ReplaceAll(text, "＆", "&")
	text = strings.ReplaceAll(text, "＊", "*")

	ctx.CreateVmIfNotExists()
	vm := ctx.vm
	vm.Config.DisableStmts = true
	vm.Config.DefaultDiceSideExpr = "d" + strconv.FormatInt(getDefaultDicePoints(ctx), 10)

	vm.Config.CallbackSt = func(_type string, name string, val *ds.VMValue, extra *ds.VMValue, op string, detail string) {
		// fmt.Println("!!", _type, name, val, extra, op, detail)
		switch _type {
		case "set":
			newname := tmpl.GetAlias(name)
			toSetItems = append(toSetItems, &stSetOrModInfoItem{name: newname, value: val})
		case "set.x1":
			newname := tmpl.GetAlias(name)
			toSetItems = append(toSetItems, &stSetOrModInfoItem{name: newname, value: val, extra: extra})
		case "set.x0":
			newname := tmpl.GetAlias(name)
			toSetItems = append(toSetItems, &stSetOrModInfoItem{name: newname, value: val, extra: ds.NewIntVal(1)})
		case "mod":
			newname := tmpl.GetAlias(name)
			if val.TypeId != ds.VMTypeInt {
				return
			}
			toModItems = append(toModItems, &stSetOrModInfoItem{name: newname, value: val, op: op, expr: detail})
		}
	}
	err = vm.Run("^st" + text)
	return vm.Ret, toSetItems, toModItems, err
}

func cmdStCharFormat1(mctx *MsgContext, tmpl *GameSystemTemplate, vars *ds.ValueMap) {
	if tmpl != nil {
		toRemove := map[string]bool{}
		toAdd := map[string]*ds.VMValue{}

		// 先转存一次的原因是后面获取默认值时，可能会死锁
		// TODO: 对新的map来说这个问题是否也存在？
		backups := map[string]*ds.VMValue{}
		vars.Range(func(key string, value *ds.VMValue) bool {
			backups[key] = value
			return true
		})

		for key, v := range backups {
			newKey := tmpl.GetAlias(key)
			if v.TypeId == ds.VMTypeInt {
				// val, detail, calculated, exists2
				val, _, _, exists := tmpl.GetDefaultValueEx0(mctx, newKey)
				if exists && ds.ValueEqual(val, v, false) {
					// 与默认值相同，可以从数据中移除
					toRemove[key] = true
					continue
				}
			}

			if key != newKey {
				toRemove[key] = true
			}

			toAdd[newKey] = v
		}

		for k := range toRemove {
			vars.Delete(k)
		}
		for k, v := range toAdd {
			vars.Store(k, v)
		}
	}
}

func cmdStCharFormat(mctx *MsgContext, tmpl *GameSystemTemplate) {
	attrs := lo.Must(mctx.Dice.AttrsManager.LoadByCtx(mctx))

	if tmpl != nil {
		cmdStCharFormat1(mctx, tmpl, attrs.valueMap) // 这里不标记值改动，因为SetSheetType会做
		mctx.Dice.AttrsManager.CheckAndFreeUnused()
	}

	attrs.SetSheetType(mctx.Group.System)
}

type CmdStOverrideInfo struct {
	ToSet        func(ctx *MsgContext, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate)
	ToShow       func(ctx *MsgContext, key string, val *ds.VMValue, tmpl *GameSystemTemplate) string
	ToExport     func(ctx *MsgContext, key string, val *ds.VMValue, tmpl *GameSystemTemplate) string
	CommandSolve func(ctx *MsgContext, msg *Message, args *CmdArgs) *CmdExecuteResult
	HelpSt       string
	TemplateName string // 如果存在，使用此名字加载规则模板。一个用途是coc模式下可以调用dst
}

func getCmdStBase(soi CmdStOverrideInfo) *CmdItemInfo {
	helpSt := soi.HelpSt
	if helpSt == "" {
		helpSt += ".st show // 展示个人属性\n"
		helpSt += ".st show <属性1> <属性2> ... // 展示特定的属性数值\n"
		helpSt += ".st show <数字> // 展示高于<数字>的属性，如.st show 30\n"
		helpSt += ".st clr // 清除属性\n"
		helpSt += ".st fmt // 强制转卡为当前规则(改变卡片类型，转换同义词)\n"
		helpSt += ".st del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n"
		helpSt += ".st export // 导出\n"
		helpSt += ".st help // 帮助\n"
		helpSt += ".st <属性><值> // 例：.st 敏捷50 力量3d6*5\n"
		helpSt += ".st &<属性>=<式子> // 例：.st &手枪=1d6\n"
		helpSt += ".st <属性>±<表达式> // 例：.st 敏捷+2 hp+1d3 "
	}

	cmdNewSt := &CmdItemInfo{
		Name:          "st",
		ShortHelp:     helpSt,
		Help:          "属性修改指令，支持分支指令如下:\n" + helpSt,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("del", "rm", "show", "list", "export")
			dice := ctx.Dice
			val := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)

			attrs := lo.Must(dice.AttrsManager.LoadByCtx(mctx))
			cardType := ReadCardType(mctx)

			tmpl := ctx.Group.GetCharTemplate(dice)
			tmplShow := tmpl // 用于st show的模板，如果show不同规则的模板，可以以其他规则格式显示
			if cardType != tmplShow.Name {
				if tmpl2, _ := dice.GameSystemMap.Load(cardType); tmpl2 != nil {
					tmplShow = tmpl2
				}
			}

			if soi.TemplateName != "" {
				if tmpl2, _ := dice.GameSystemMap.Load(soi.TemplateName); tmpl2 != nil {
					tmpl = tmpl2
					tmplShow = tmpl2
				}
			}

			mctx.Eval(tmpl.PreloadCode, nil)
			if tmplShow != tmpl {
				mctx.Eval(tmplShow.PreloadCode, nil)
			}

			if soi.CommandSolve != nil {
				ret := soi.CommandSolve(ctx, msg, cmdArgs)
				if ret != nil {
					return *ret
				}
			}

			switch val {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

			case "show", "list":
				pickItems, limit := cmdStGetPickItemAndLimit(tmplShow, cmdArgs)
				items, droppedByLimit, err := cmdStGetItemsForShow(mctx, tmplShow, pickItems, limit, &soi)
				if err != nil {
					ReplyToSender(mctx, msg, err.Error())
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 每四个一行，拼起来
				itemsPerLine := tmplShow.AttrConfig.ItemsPerLine
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
					info = DiceFormatTmpl(mctx, "COC:属性设置_列出_未发现记录")
				}

				if limit > 0 {
					VarSetValueInt64(mctx, "$t数量", int64(droppedByLimit))
					VarSetValueInt64(mctx, "$t判定值", limit)
					info += DiceFormatTmpl(mctx, "COC:属性设置_列出_隐藏提示")
					// info += fmt.Sprintf("\n注：%d条属性因≤%d被隐藏", limktSkipCount, limit)
				}

				VarSetValueStr(mctx, "$t属性信息", info)
				extra := ReadCardTypeEx(mctx, tmpl.Name)
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_列出")+extra)

			case "export":
				items, _, err := cmdStGetItemsForExport(mctx, tmpl, &soi)
				if err != nil {
					ReplyToSender(mctx, msg, err.Error())
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				info := "导出结果：\n.st clr\n.st "
				for _, i := range items {
					info += i
					info += " "
				}
				playerName := DiceFormat(mctx, "{$t玩家_RAW}")
				if playerName != "" {
					info += "\n.nn " + playerName
				}

				if len(items) == 0 {
					info = DiceFormatTmpl(mctx, "COC:属性设置_列出_未发现记录")
				}

				ReplyToSender(mctx, msg, info)

			case "del", "rm":
				var nums []string
				var failed []string

				for _, varname := range cmdArgs.Args[1:] {
					vname := tmpl.GetAlias(varname)

					val := attrs.Load(vname)
					if val != nil {
						nums = append(nums, vname)
						attrs.Delete(vname)
					} else {
						failed = append(failed, vname)
					}
				}

				VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
				VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_删除"))
			case "clr", "clear":
				num := attrs.Clear()
				VarSetValueInt64(mctx, "$t数量", int64(num))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_清除"))

			case "fmt", "format":
				cmdStCharFormat(mctx, tmpl)
				ReplyToSender(mctx, msg, "角色卡片类型被强制修改为: "+ctx.Group.System)

			default:
				if cardType != "" && cardType != mctx.Group.System {
					ReplyToSender(mctx, msg, fmt.Sprintf("阻止操作：当前卡规则为 %s，群规则为 %s。\n为避免损坏此人物卡，请先更换角色卡，或使用.st fmt强制转卡", cardType, mctx.Group.System))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				cmdStCharFormat(mctx, tmpl) // 转一下卡

				mctx.SystemTemplate = tmpl

				// 进行简化卡的尝试解析
				input := cmdArgs.CleanArgs
				re := regexp.MustCompile(`^(([^\s\-#]{1,25})([-#]))([^\s\d]+\d+)`)
				matches := re.FindStringSubmatch(input)
				if len(matches) > 0 {
					flag := matches[3]
					name := matches[2]
					val := matches[4]

					isName := flag == "#"
					if !isName {
						// 先尝试作为名字处理
						isName = true

						// 如果"-"后面跟的句子进行了计算(骰点、符号、变量)，且不剩余文本，此时为值(如d4)，而不是名字
						r, _, err := DiceExprEvalBase(ctx, val, RollExtraFlags{})
						if err == nil && r.GetRestInput() == "" && r.IsCalculated() {
							isName = false
						}

						if isName {
							// 好像是个名字了，先再看看是不是带默认值的属性
							valName := tmpl.GetAlias(name)
							if _, _, _, exists := tmpl.GetDefaultValueEx0(mctx, valName); exists {
								// 有默认值，不能作为名字
								isName = false
								name = valName
							}
						}
					}

					if isName {
						// 确定不是属性了，现在作为角色名处理
						input = input[len(matches[1]):]

						p := mctx.Player
						VarSetValueStr(mctx, "$t旧昵称", fmt.Sprintf("<%s>", mctx.Player.Name))
						VarSetValueStr(mctx, "$t旧昵称_RAW", mctx.Player.Name)
						p.Name = name
						VarSetValueStr(mctx, "$t玩家", fmt.Sprintf("<%s>", mctx.Player.Name))
						VarSetValueStr(mctx, "$t玩家_RAW", mctx.Player.Name)
						p.UpdatedAtTime = time.Now().Unix()

						if mctx.Player.AutoSetNameTemplate != "" {
							_, _ = SetPlayerGroupCardByTemplate(mctx, mctx.Player.AutoSetNameTemplate)
						}
					}
				}

				_, toSetItems, toModItems, err := cmdStReadOrMod(mctx, tmpl, input)

				if err != nil {
					dice.Logger.Info(".st 格式错误: ", err.Error())
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				rRestIput := ctx.vm.RestInput

				// 处理直接设置属性
				var text string
				validNum := int64(0)
				if len(toSetItems) > 0 {
					// 是 set
					for _, i := range toSetItems {
						if soi.ToSet != nil {
							soi.ToSet(ctx, i, attrs, tmpl)
						}

						def := tmpl.GetDefaultValueEx(ctx, i.name)
						if ds.ValueEqual(i.value, def, true) {
							curVal := attrs.Load(i.name)
							// 如果当前有值
							if curVal == nil {
								// 与预设相同，放弃
								continue
							}
						}

						validNum++
						attrs.Store(i.name, i.value)
					}
					VarSetValueInt64(mctx, "$t数量", int64(len(toSetItems))) // 废弃
					VarSetValueInt64(mctx, "$t有效数量", validNum)
					VarSetValueInt64(mctx, "$t同义词数量", int64(0)) // 废弃
					text = DiceFormatTmpl(mctx, "COC:属性设置")
					SetCardType(mctx, tmpl.Name)
				}

				// 处理变更属性
				if len(toModItems) > 0 {
					// 修改

					// 指令信息
					commandInfo := map[string]interface{}{
						"cmd":    "st",
						"rule":   cardType,
						"pcName": mctx.Player.Name,
						"items":  []interface{}{},
					}

					var textItems []string
					chName := lo.Must(mctx.Dice.AttrsManager.LoadByCtx(mctx)).Name
					for _, i := range toModItems {
						cmdStValueMod(mctx, tmpl, attrs.valueMap, commandInfo, i)
						VarSetValueStr(mctx, "$t当前绑定角色", chName)
						text2 := DiceFormatTmpl(mctx, "COC:属性设置_增减_单项")
						textItems = append(textItems, text2)
					}

					// text = DiceFormatTmpl(mctx, "COC:属性设置_增减")
					VarSetValueStr(mctx, "$t变更列表", strings.Join(textItems, "\n"))
					text = DiceFormatTmpl(mctx, "COC:属性设置_增减")

					ctx.CommandInfo = commandInfo
					attrs.SetModified()
					SetCardType(mctx, tmpl.Name)
				}

				if rRestIput != "" {
					text += "\n解析失败: " + rRestIput
				}

				ReplyToSender(mctx, msg, text)
			}

			if ctx.Player.AutoSetNameTemplate != "" {
				_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
			}

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	return cmdNewSt
}

func RegisterBuiltinExtExp(_ *Dice) {
	// cmdNewSt := getCmdStBase()

	// theExt := &ExtInfo{
	// 	Name:       "exp", // 扩展的名称，需要用于开启和关闭指令中，写简短点
	// 	Version:    "1.0.0",
	// 	Brief:      "实验指令模块，如果不知道里面有什么，建议不要开",
	// 	Author:     "木落",
	// 	AutoActive: false, // 是否自动开启
	// 	OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	// 		//p := getPlayerInfoBySender(session, msg)
	// 		//p.TempValueAlias = &ac.Alias;
	// 	},
	// 	GetDescText: func(i *ExtInfo) string {
	// 		return GetExtensionDesc(i)
	// 	},
	// 	CmdMap: CmdMapCls{
	// 		"st":  cmdNewSt,
	// 		"nst": cmdNewSt,
	// 	},
	// }

	// dice.RegisterExtension(theExt)
}
