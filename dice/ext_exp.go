package dice

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func RegisterBuiltinExtExp(dice *Dice) {
	cmdNewSt := &CmdItemInfo{
		Name:          "st",
		Help:          "一个实验型的，通用的st指令",
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("del", "rm", "show", "list")
			val := cmdArgs.GetArgN(1)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			tmpl := ctx.Group.GetCharTemplate(dice)

			chVars, _ := mctx.ChVarsGet()
			cardType := ReadCardType(mctx)

			switch val {
			case "show", "list":
				info := ""
				//p := mctx.Player

				useLimit := false
				usePickItem := false
				limktSkipCount := 0
				var limit int64

				if len(cmdArgs.Args) >= 2 {
					arg2 := cmdArgs.GetArgN(2)
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
						pickItems[tmpl.GetAlias(i)] = 1
					}
				}

				tick := 0
				if mctx.ChVarsNumGet() == 0 {
					info = DiceFormatTmpl(mctx, "COC:属性设置_列出_未发现记录")
				} else {
					// 按照配置文件排序
					var attrKeys []string
					used := map[string]bool{}
					for _, key := range tmpl.AttrSettings.Top {
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
						for _, n := range tmpl.AttrSettings.Ignores {
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
							//fmt.Println("xxx", err.Error())
							ReplyToSender(mctx, msg, "模板卡异常, 属性: "+k)
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						if index >= topNum {
							if useLimit && v.TypeId == VMTypeInt64 && v.Value.(int64) < limit {
								limktSkipCount += 1
								continue
							}
						}

						tick += 1
						info += fmt.Sprintf("%s:%s\t", k, v.ToString())
						if tick%4 == 0 {
							info = strings.TrimSpace(info) // 去除末尾空格
							info += fmt.Sprintf("\n")
						}
					}

					if info == "" {
						info = DiceFormatTmpl(mctx, "COC:属性设置_列出_未发现记录")
					}
				}

				if useLimit {
					VarSetValueInt64(mctx, "$t数量", int64(limktSkipCount))
					VarSetValueInt64(mctx, "$t判定值", limit)
					info += DiceFormatTmpl(mctx, "COC:属性设置_列出_隐藏提示")
					//info += fmt.Sprintf("\n注：%d条属性因≤%d被隐藏", limktSkipCount, limit)
				}

				VarSetValueStr(mctx, "$t属性信息", info)
				extra := ReadCardTypeEx(mctx, "coc7")
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

				VarSetValueStr(mctx, "$t属性列表", strings.Join(nums, " "))
				VarSetValueInt64(mctx, "$t失败数量", int64(len(failed)))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_删除"))

			case "clr", "clear":
				num := mctx.ChVarsClear()
				VarSetValueInt64(mctx, "$t数量", int64(num))
				ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "DND:属性设置_清除"))

			case "convert":
				SetCardType(mctx, ctx.Group.System)

			default:
				if cardType != "" && cardType != mctx.Group.System {
					ReplyToSender(mctx, msg, fmt.Sprintf("当前卡规则为 %s，群规则为 %s。\n为避免误操作，请先换卡、或.st convert强制转卡，或使用.st clr清除数据", cardType, mctx.Group.System))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// TODO: 重构抽象出这个调用
				var toSetItems []struct {
					name  string
					value *VMValue
				}
				var toModItems []struct {
					name  string
					value *VMValue
					op    string
				}

				//var texts []string
				mctx.SystemTemplate = tmpl
				r, _, err := mctx.Dice.ExprEvalBase("^st"+cmdArgs.CleanArgs, ctx, RollExtraFlags{
					DefaultDiceSideNum: getDefaultDicePoints(mctx),
					StCallback: func(_type string, name string, val *VMValue, op string, detail string) {
						//texts = append(texts, fmt.Sprintf("[%s]%s: %s %s [%s]", _type, name, val.ToString(), op, detail))
						switch _type {
						case "set":
							newname := tmpl.GetAlias(name)
							def := tmpl.GetDefaultValueEx(mctx, newname)
							if def.TypeId == val.TypeId && def.Value == val.Value {
								// 与预设相同，放弃
							} else {
								toSetItems = append(toSetItems, struct {
									name  string
									value *VMValue
								}{name: newname, value: val})
							}
						case "mod":
							newname := tmpl.GetAlias(name)
							if val.TypeId != VMTypeInt64 {
								return
							}
							toModItems = append(toModItems, struct {
								name  string
								value *VMValue
								op    string
							}{name: newname, value: val, op: op})
						}
					},
				})

				if err != nil {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				retText := fmt.Sprintf("<%s>的%s人物属性设置如下:\n", mctx.Player.Name, tmpl.KeyName)
				SetCardType(mctx, tmpl.KeyName)

				var textPieces []string
				if len(toSetItems) > 0 {
					// 是 set
					for _, i := range toSetItems {
						chVars.Set(i.name, i.value)
						textPieces = append(textPieces, fmt.Sprintf("%s:%s", i.name, i.value.ToString()))
					}
					mctx.ChVarsUpdateTime()
					retText += "读入: " + strings.Join(textPieces, ", ") + "\n"
				}

				textPieces = []string{}
				if len(toModItems) > 0 {
					// 修改
					for _, i := range toModItems {
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
							continue
						}

						// 进行变更
						theOldValue, _ := curVal.ReadInt64()
						theModValue, _ := i.value.ReadInt64()
						var theNewValue int64

						switch i.op {
						case "+":
							theNewValue = theOldValue + theModValue
						case "-":
							theNewValue = theOldValue - theModValue
						}

						baseValue := ""
						text := fmt.Sprintf("%s%s(%d ➯ %d)", i.name, baseValue, theOldValue, theNewValue)
						chVars.Set(i.name, VMValueNew(VMTypeInt64, theNewValue))
						textPieces = append(textPieces, text)
					}
					mctx.ChVarsUpdateTime()
					retText += "变更: " + strings.Join(textPieces, ", ") + "\n"
				}

				if r.restInput != "" {
					retText += "解析失败: " + r.restInput
				}

				ReplyToSender(mctx, msg, retText)
			}

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	theExt := &ExtInfo{
		Name:       "exp", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "实验指令模块，如果不知道里面有什么，建议不要开",
		Author:     "木落",
		AutoActive: false, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func(i *ExtInfo) string {
			return GetExtensionDesc(i)
		},
		CmdMap: CmdMapCls{
			"st":  cmdNewSt,
			"nst": cmdNewSt,
		},
	}

	dice.RegisterExtension(theExt)
}
