package dice

import (
	"errors"
	"fmt"
	"github.com/fy0/lockfree"
	"sort"
	"strconv"
	"strings"
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

func cmdStGetItemsForShow(mctx *MsgContext, tmpl *GameSystemTemplate, pickItems map[string]int, limit int64) (items []string, droppedByLimit int, err error) {
	usePickItem := len(pickItems) > 0
	useLimit := limit > 0
	limitSkipCount := 0
	items = []string{}

	if mctx.ChVarsNumGet() > 0 {
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

		vars, _ := mctx.ChVarsGet()
		_ = vars.Iterate(func(_k interface{}, _v interface{}) error {
			key := _k.(string)
			// 只添加不存在的
			if used[key] {
				return nil
			}
			for _, n := range tmpl.AttrConfig.Ignores {
				// 跳过忽略项
				if n == key {
					return nil
				}
			}
			attrKeys2 = append(attrKeys2, key)
			return nil
		})

		// 排序、合并key
		sort.Strings(attrKeys2)
		for _, key := range attrKeys2 {
			attrKeys = append(attrKeys, key)
		}

		if len(pickItems) > 0 {
			attrKeys = []string{}
			for k, _ := range pickItems {
				attrKeys = append(attrKeys, k)
			}
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

			v, err := tmpl.GetShowAs(mctx, k)
			if err != nil {
				return nil, 0, errors.New("模板卡异常, 属性: " + k)
			}

			if index >= topNum {
				if useLimit && v.TypeId == VMTypeInt64 && v.Value.(int64) < limit {
					limitSkipCount += 1
					continue
				}
			}

			items = append(items, fmt.Sprintf("%s:%s", k, v.ToString()))
		}
	}

	return items, limitSkipCount, nil
}

func cmdStValueMod(mctx *MsgContext, tmpl *GameSystemTemplate, chVars lockfree.HashMap, commandInfo map[string]interface{}, i *stSetOrModInfoItem) {
	// TODO: 这套api是第一次尝试，之后再重新考虑
	// 获取当前值
	var curVal *VMValue
	if a, ok := chVars.Get(i.name); ok {
		curVal = a.(*VMValue)
	}
	if curVal == nil {
		curVal = tmpl.GetDefaultValueEx(mctx, i.name)
	}
	if curVal == nil {
		curVal = VMValueNew(VMTypeInt64, int64(0))
	}

	if curVal.TypeId != VMTypeInt64 {
		// 跳过非数字
		return
	}

	// 进行变更
	theOldValue, _ := curVal.ReadInt64()
	theModValue, _ := i.value.ReadInt64()
	var theNewValue int64

	signText := ""
	switch i.op {
	case "+":
		signText = "增加"
		theNewValue = theOldValue + theModValue
	case "-":
		signText = "扣除"
		theNewValue = theOldValue + theModValue
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

	//baseValue := ""
	//text := fmt.Sprintf("%s%s(%d ➯ %d)", i.name, baseValue, theOldValue, theNewValue)
	chVars.Set(i.name, VMValueNew(VMTypeInt64, theNewValue))
	//textPieces = append(textPieces, text)

	VarSetValueStr(mctx, "$t属性", i.name)
	VarSetValueInt64(mctx, "$t旧值", theOldValue)
	VarSetValueInt64(mctx, "$t新值", theNewValue)
	VarSetValueInt64(mctx, "$t变化量", theModValue)
	VarSetValueStr(mctx, "$t增加或扣除", signText)
	VarSetValueStr(mctx, "$t表达式文本", i.expr)
}

type stSetOrModInfoItem struct {
	name  string
	value *VMValue
	op    string
	expr  string
}

func cmdStReadOrMod(ctx *MsgContext, tmpl *GameSystemTemplate, text string) (r *VmResult, toSetItems []*stSetOrModInfoItem, toModItems []*stSetOrModInfoItem, err error) {
	//var texts []string
	//mctx.SystemTemplate = tmpl

	// 处理全角符号
	text = strings.ReplaceAll(text, "＋", "+")
	text = strings.ReplaceAll(text, "－", "-")
	text = strings.ReplaceAll(text, "＝", "=")
	text = strings.ReplaceAll(text, "：", ":")
	text = strings.ReplaceAll(text, "＆", "&")
	text = strings.ReplaceAll(text, "＊", "*")

	r, _, err = ctx.Dice.ExprEvalBase("^st"+text, ctx, RollExtraFlags{
		DefaultDiceSideNum: getDefaultDicePoints(ctx),
		DisableBlock:       true,
		StCallback: func(_type string, name string, val *VMValue, op string, detail string) {
			//texts = append(texts, fmt.Sprintf("[%s]%s: %s %s [%s]", _type, name, val.ToString(), op, detail))
			switch _type {
			case "set":
				newname := tmpl.GetAlias(name)
				toSetItems = append(toSetItems, &stSetOrModInfoItem{name: newname, value: val})
			case "mod":
				newname := tmpl.GetAlias(name)
				if val.TypeId != VMTypeInt64 {
					return
				}
				toModItems = append(toModItems, &stSetOrModInfoItem{name: newname, value: val, op: op, expr: detail})
			}
		},
	})

	return r, toSetItems, toModItems, err
}

func cmdStCharFormat1(mctx *MsgContext, tmpl *GameSystemTemplate, vars lockfree.HashMap) {
	if tmpl != nil {
		toRemove := map[string]interface{}{}
		toAdd := map[string]interface{}{}

		// 先转存一次的原因是后面获取默认值时，可能会死锁
		backups := map[string]interface{}{}
		_ = vars.Iterate(func(_k interface{}, _v interface{}) error {
			key := _k.(string)
			v := (_v).(*VMValue)
			backups[key] = v
			return nil
		})

		for key, _v := range backups {
			v := (_v).(*VMValue)

			newKey := tmpl.GetAlias(key)
			if v.TypeId == VMTypeInt64 {
				// val, detail, calculated, exists2
				val, _, _, exists := tmpl.GetDefaultValueEx0(mctx, newKey)
				if exists && val.TypeId == v.TypeId && val.Value == v.Value {
					// 与默认值相同，跳过
					toRemove[key] = true
					continue
				}
			}

			if key != newKey {
				toRemove[key] = true
			}

			toAdd[newKey] = _v
		}

		for k, _ := range toRemove {
			vars.Del(k)
		}
		for k, v := range toAdd {
			vars.Set(k, v)
		}
	}
}

func cmdStCharFormat(mctx *MsgContext, tmpl *GameSystemTemplate) {
	if tmpl != nil {
		vars, _ := mctx.ChVarsGet()
		cmdStCharFormat1(mctx, tmpl, vars)
	}

	SetCardType(mctx, mctx.Group.System)
	mctx.ChVarsUpdateTime()
}

func getCmdStBase() *CmdItemInfo {
	helpSt := ""
	helpSt += ".st show // 展示个人属性\n"
	helpSt += ".st show <属性1> <属性2> ... // 展示特定的属性数值\n"
	helpSt += ".st show <数字> // 展示高于<数字>的属性，如.st show 30\n"
	helpSt += ".st clr // 清除属性\n"
	helpSt += ".st fmt // 强制转卡为当前规则(改变卡片类型，转换同义词)\n"
	helpSt += ".st del <属性1> <属性2> ... // 删除属性，可多项，以空格间隔\n"
	//helpSt += ".st export // 导出\n"
	helpSt += ".st help // 帮助\n"
	helpSt += ".st <属性><值> // 例：.st 敏捷50 力量3d6*5\n"
	helpSt += ".st &<属性>=<式子> // 例：.st &手枪=1d6\n"
	helpSt += ".st <属性>±<表达式> // 例：.st 敏捷+2 hp+1d3 "

	cmdNewSt := &CmdItemInfo{
		Name:          "st",
		ShortHelp:     helpSt,
		Help:          "属性修改指令，支持分支指令如下:\n" + helpSt,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("del", "rm", "show", "list")
			dice := ctx.Dice
			val := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			tmpl := ctx.Group.GetCharTemplate(dice)

			chVars, _ := mctx.ChVarsGet()
			cardType := ReadCardType(mctx)

			switch val {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

			case "show", "list":
				pickItems, limit := cmdStGetPickItemAndLimit(tmpl, cmdArgs)
				items, droppedByLimit, err := cmdStGetItemsForShow(mctx, tmpl, pickItems, limit)
				if err != nil {
					ReplyToSender(mctx, msg, err.Error())
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 每四个一行，拼起来
				tick := 0
				info := ""
				for _, i := range items {
					tick += 1
					info += i
					if tick%4 == 0 {
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
					//info += fmt.Sprintf("\n注：%d条属性因≤%d被隐藏", limktSkipCount, limit)
				}

				VarSetValueStr(mctx, "$t属性信息", info)
				extra := ReadCardTypeEx(mctx, ctx.Group.System)
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_列出")+extra)

			case "del", "rm":
				var nums []string
				var failed []string

				for _, varname := range cmdArgs.Args[1:] {
					vname := tmpl.GetAlias(varname)

					_, ok := chVars.Get(vname)
					if ok {
						nums = append(nums, vname)
						chVars.Del(vname)
					} else {
						failed = append(failed, vname)
					}
				}

				if len(nums) > 0 {
					mctx.ChVarsUpdateTime()
				}

				//VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
				//VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
				//ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_删除"))
				VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
				VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_删除"))

			case "clr", "clear":
				num := mctx.ChVarsClear()
				VarSetValueInt64(mctx, "$t数量", int64(num))
				//text := fmt.Sprintf("<%s>的属性数据已经清除，共计%d条", p.Name, num)
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "COC:属性设置_清除"))

			case "fmt", "format":
				cmdStCharFormat(mctx, tmpl)
				ReplyToSender(mctx, msg, "角色卡片类型被强制修改为: "+ctx.Group.System)

			//case "export":
			default:
				if cardType != "" && cardType != mctx.Group.System {
					ReplyToSender(mctx, msg, fmt.Sprintf("当前卡规则为 %s，群规则为 %s。\n为避免误操作，请先换卡、或使用.st clr清除数据再录卡", cardType, mctx.Group.System))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				cmdStCharFormat(mctx, tmpl) // 转一下卡

				mctx.SystemTemplate = tmpl
				r, toSetItems, toModItems, err := cmdStReadOrMod(mctx, tmpl, cmdArgs.CleanArgs)

				if err != nil {
					//ReplyToSender(mctx, msg, "") // +err.Error()
					dice.Logger.Info(".st 格式错误: ", err.Error())
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				// 处理直接设置属性
				var text string
				//retText := fmt.Sprintf("<%s>的%s人物属性设置如下:\n", mctx.Player.Name, tmpl.Name)

				validNum := int64(0)
				if len(toSetItems) > 0 {
					// 是 set
					for _, i := range toSetItems {
						def := tmpl.GetDefaultValueEx(ctx, i.name)
						val := i.value
						var curVal *VMValue
						if a, ok := chVars.Get(i.name); ok {
							curVal = a.(*VMValue)
						}

						if def.TypeId == val.TypeId && def.Value == val.Value {
							// 如果当前有值
							if curVal != nil {
								// 不搞花的，直接赋值一次
								//if curVal.TypeId == val.TypeId {
								//	if curVal.Value == val.Value {
								//		// 如果与当前值相同，放弃
								//		continue
								//	}
								//}
							} else {
								// 与预设相同，放弃
								continue
							}
						}

						validNum += 1
						chVars.Set(i.name, i.value)
						//textPieces = append(textPieces, fmt.Sprintf("%s:%s", i.name, i.value.ToString()))
					}
					mctx.ChVarsUpdateTime()
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
					chName := mctx.ChBindCurGet()
					for _, i := range toModItems {
						cmdStValueMod(mctx, tmpl, chVars, commandInfo, i)
						VarSetValueStr(mctx, "$t当前绑定角色", chName)
						text2 := DiceFormatTmpl(mctx, "COC:属性设置_增减_单项")
						textItems = append(textItems, text2)
					}

					//text = DiceFormatTmpl(mctx, "COC:属性设置_增减")
					VarSetValueStr(mctx, "$t变更列表", strings.Join(textItems, "\n"))
					text = DiceFormatTmpl(mctx, "COC:属性设置_增减")

					ctx.CommandInfo = commandInfo
					mctx.ChVarsUpdateTime()
					SetCardType(mctx, tmpl.Name)
				}

				if r.restInput != "" {
					text += "\n解析失败: " + r.restInput
				}

				ReplyToSender(mctx, msg, text)
			}

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	return cmdNewSt
}

func RegisterBuiltinExtExp(dice *Dice) {
	//cmdNewSt := getCmdStBase()
	//
	//theExt := &ExtInfo{
	//	Name:       "exp", // 扩展的名称，需要用于开启和关闭指令中，写简短点
	//	Version:    "1.0.0",
	//	Brief:      "实验指令模块，如果不知道里面有什么，建议不要开",
	//	Author:     "木落",
	//	AutoActive: false, // 是否自动开启
	//	OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	//		//p := getPlayerInfoBySender(session, msg)
	//		//p.TempValueAlias = &ac.Alias;
	//	},
	//	GetDescText: func(i *ExtInfo) string {
	//		return GetExtensionDesc(i)
	//	},
	//	CmdMap: CmdMapCls{
	//		"st":  cmdNewSt,
	//		"nst": cmdNewSt,
	//	},
	//}
	//
	//dice.RegisterExtension(theExt)
}
