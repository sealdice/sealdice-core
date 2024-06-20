package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
)

func (v *VMValue) ConvertToV2() *ds.VMValue {
	switch v.TypeID {
	case VMTypeInt64:
		return ds.NewIntVal(ds.IntType(v.Value.(int64)))
	case VMTypeString:
		return ds.NewStrVal(v.Value.(string))
	case VMTypeNone:
		return ds.NewNullVal()
	case VMTypeDNDComputedValue:
		oldCD := v.Value.(*VMDndComputedValueData)
		m := &ds.ValueMap{}
		base := oldCD.BaseValue.ConvertToV2()
		if base.TypeId == ds.VMTypeNull {
			base = ds.NewIntVal(0)
		}
		m.Store("base", base)
		expr := strings.ReplaceAll(oldCD.Expr, "$tVal", "this.base")
		expr = strings.ReplaceAll(expr, "熟练", "(熟练||0)")
		cd := &ds.ComputedData{
			Expr:  expr,
			Attrs: m,
		}
		return ds.NewComputedValRaw(cd)
	case VMTypeComputedValue:
		oldCd, _ := v.ReadComputed()

		m := &ds.ValueMap{}
		if oldCd.Attrs != nil {
			oldCd.Attrs.Range(func(key string, value *VMValue) bool {
				m.Store(key, value.ConvertToV2())
				return true
			})
		}
		cd := &ds.ComputedData{
			Expr:  oldCd.Expr,
			Attrs: m,
		}
		return ds.NewComputedValRaw(cd)
	default:
		return ds.NewNullVal()
	}
}

func dsValueToRollVMv1(v *ds.VMValue) *VMValue {
	var v2 *VMValue
	switch v.TypeId {
	case ds.VMTypeInt:
		v2 = &VMValue{TypeID: VMTypeInt64, Value: int64(v.MustReadInt())}
	case ds.VMTypeFloat:
		v2 = &VMValue{TypeID: VMTypeInt64, Value: int64(v.MustReadFloat())}
	default:
		v2 = &VMValue{TypeID: VMTypeString, Value: v.ToString()}
	}
	return v2
}

func DiceFormatTmplV1(ctx *MsgContext, s string) string {
	var text string
	a := ctx.Dice.TextMap[s]
	if a == nil {
		text = "<%未知项-" + s + "%>"
	} else {
		text = ctx.Dice.TextMap[s].Pick().(string)
	}
	return DiceFormat(ctx, text)
}

func DiceFormatV1(ctx *MsgContext, s string) string { //nolint:revive
	s = CompatibleReplace(ctx, s)

	r, _, _ := ctx.Dice._ExprTextV1(s, ctx)
	return r
}

func DiceFormat(ctx *MsgContext, s string) string {
	ret, err := DiceFormatV2(ctx, s)
	if err != nil {
		// 遇到异常，尝试一下V1
		return DiceFormatV1(ctx, s)
	}
	return ret
}

func DiceFormatTmpl(ctx *MsgContext, s string) string {
	ret, err := DiceFormatTmplV2(ctx, s)
	if err != nil {
		// 遇到异常，尝试一下V1
		return DiceFormatTmplV1(ctx, s)
	}
	return ret
}

func (ctx *MsgContext) Eval(expr string, flags *ds.RollConfig) *VMResultV2 {
	ctx.CreateVmIfNotExists()
	vm := ctx.vm
	if flags != nil {
		vm.Config = *flags
	}
	err := vm.Run(expr)

	if err != nil {
		return &VMResultV2{vm: vm}
	}
	return &VMResultV2{VMValue: *vm.Ret, vm: vm}
}

// EvalFString TODO: 这个名字得换一个
func (ctx *MsgContext) EvalFString(expr string, flags *ds.RollConfig) *VMResultV2 {
	expr = CompatibleReplace(ctx, expr)

	// 隐藏的内置字符串符号 \x1e
	r := ctx.Eval("\x1e"+expr+"\x1e", flags)
	if r.vm.Error != nil {
		fmt.Println("脚本执行出错: ", expr, "->", r.vm.Error)
	}
	return r
}

type VMResultV2m struct {
	*ds.VMValue
	vm        *ds.Context
	legacy    *VMResult
	cocPrefix string
}

func (r *VMResultV2m) GetAsmText() string {
	if r.legacy != nil {
		return r.legacy.Parser.GetAsmText()
	}
	return r.vm.GetAsmText()
}

func (r *VMResultV2m) IsCalculated() bool {
	if r.legacy != nil {
		return r.legacy.Parser.Calculated
	}
	return r.vm.IsCalculateExists()
}

func (r *VMResultV2m) GetRestInput() string {
	if r.legacy != nil {
		return r.legacy.restInput
	}
	return r.vm.RestInput
}

func (r *VMResultV2m) GetMatched() string {
	if r.legacy != nil {
		return r.legacy.Matched
	}
	return r.vm.Matched
}

func (r *VMResultV2m) GetCocPrefix() string {
	if r.legacy != nil {
		return r.legacy.Parser.CocFlagVarPrefix
	}
	return r.cocPrefix
}

func (r *VMResultV2m) GetVersion() int64 {
	if r.legacy != nil {
		return 1
	}
	return 2
}

func (r *VMResultV2m) ToString() string {
	if r.legacy != nil {
		return r.legacy.ToString()
	}
	return r.VMValue.ToString()
}

// DiceExprEvalBase 向下兼容执行，首先尝试使用V2执行表达式，如果V2失败，fallback到V1
//
// Deprecated: 不建议用，纯兼容旧版
func DiceExprEvalBase(ctx *MsgContext, s string, flags RollExtraFlags) (*VMResultV2m, string, error) {
	ctx.CreateVmIfNotExists()
	vm := ctx.vm
	vm.Ret = nil
	vm.Error = nil

	vm.Config.DisableStmts = flags.DisableBlock
	vm.Config.IgnoreDiv0 = flags.IgnoreDiv0

	var cocFlagVarPrefix string
	if flags.CocVarNumberMode {
		ctx.setCocPrefixReadForVM(func(val string) {
			cocFlagVarPrefix = val
		})
	}

	s = CompatibleReplace(ctx, s)

	err := ctx.vm.Run(s)
	if err != nil || ctx.vm.Ret == nil {
		if flags.V2Only {
			return nil, "", err
		}
		fmt.Println("脚本执行出错V2: ", strings.ReplaceAll(s, "\x1e", "`"), "->", err)

		// 尝试一下V1
		val, detail, err := ctx.Dice._ExprEvalBaseV1(s, ctx, flags)
		if err != nil {
			return nil, detail, err
		}

		return &VMResultV2m{val.ConvertToV2(), ctx.vm, val, cocFlagVarPrefix}, detail, err
	} else {
		return &VMResultV2m{ctx.vm.Ret, ctx.vm, nil, cocFlagVarPrefix}, ctx.vm.GetDetailText(), nil
	}
}

// DiceExprTextBase
//
// Deprecated: 不建议用，纯兼容旧版
func DiceExprTextBase(ctx *MsgContext, s string, flags RollExtraFlags) (*VMResultV2m, string, error) {
	return DiceExprEvalBase(ctx, "\x1e"+s+"\x1e", flags)
}

type spanByEnd []ds.BufferSpan

func (a spanByEnd) Len() int           { return len(a) }
func (a spanByEnd) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a spanByEnd) Less(i, j int) bool { return a[i].End < a[j].End }

func (ctx *MsgContext) setCocPrefixReadForVM(cb func(cocFlagVarPrefix string)) {
	ctx.vm.Config.HookFuncValueLoad = func(name string) (string, *ds.VMValue) {
		re := regexp.MustCompile(`^(困难|极难|大成功|常规|失败|困難|極難|常規|失敗)?([^\d]+)(\d+)?$`)
		m := re.FindStringSubmatch(name)

		if len(m) > 0 {
			if m[1] != "" {
				if cb != nil {
					cb(chsS2T.Read(m[1]))
				}
				name = name[len(m[1]):]
			}

			if !strings.HasPrefix(name, "$") {
				// 有末值时覆盖，有初值时
				if m[3] != "" {
					v, _ := strconv.ParseInt(m[3], 10, 64)
					// fmt.Println("COC值:", name, cocFlagVarPrefix)
					return name, ds.NewIntVal(ds.IntType(v))
				}
			}
		}

		return name, nil
	}
}

func tryLoadByBuff(ctx *MsgContext, varname string, curVal *ds.VMValue) *ds.VMValue {
	buffName := "$buff_" + varname
	am := ctx.Dice.AttrsManager
	if attrs, _ := am.LoadByCtx(ctx); attrs != nil {
		v := attrs.Load(buffName)
		if v != nil {
			newVal := curVal.OpAdd(ctx.vm, v)
			if newVal != nil {
				curVal = newVal
			}
		}
	}
	return curVal
}

// setDndReadForVM 主要是为rc设定属性豁免，暂时没有写在规则模板的原因是可以自定义detail输出
func (ctx *MsgContext) setDndReadForVM(rcMode bool) {
	var skip bool
	ctx.vm.Config.HookFuncValueLoadOverwrite = func(varname string, curVal *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue {
		// if !skip {
		// curVal = tryLoadByBuff(ctx, varname, curVal)
		// }
		if !skip && rcMode && isAbilityScores(varname) {
			if curVal != nil && curVal.TypeId == ds.VMTypeInt {
				curVal = tryLoadByBuff(ctx, varname, curVal)
				mod := curVal.MustReadInt()/2 - 5
				if detail != nil {
					detail.Tag = "dnd-rc"
					detail.Text = fmt.Sprintf("%s调整值%d", varname, mod)
				}
				return ds.NewIntVal(mod)
			}
		}

		switch varname {
		case "力量豁免", "敏捷豁免", "体质豁免", "智力豁免", "感知豁免", "魅力豁免":
			vName := strings.TrimSuffix(varname, "豁免")
			if ctx.SystemTemplate != nil {
				// NOTE: 1.4.4 版本新增，此处逻辑是为了使 "XX豁免" 中的 XX 能被同义词所替换
				vName = ctx.SystemTemplate.GetAlias(vName)
			}
			stpName := stpFormat(vName) // saving throw proficiency

			expr := fmt.Sprintf("pbCalc(0, %s ?? 0, %s ?? 0 + %s ?? 0)", stpName, vName, "$buff_"+vName)
			skip = true
			ret, err := ctx.vm.RunExpr(expr, false)
			skip = false
			if err != nil {
				return curVal
			}

			if detail != nil && detail.Tag != "" {
				detail.Ret = ret
				if ret2, _ := ctx.vm.RunExpr(stpName+" * (熟练??0)", false); ret2 != nil {
					if ret2.TypeId == ds.VMTypeInt {
						v := ret2.MustReadInt()
						if v != 0 {
							detail.Text = fmt.Sprintf("熟练+%d", v)
						}
					} else if ret2.TypeId == ds.VMTypeFloat {
						v := ret2.MustReadFloat()
						if v != 0 {
							// 这里用toStr的原因是%f会打出末尾一大串0
							detail.Text = fmt.Sprintf("熟练+%s", ret2.ToString())
						}
					}
				}
			}
			return ret
		}
		return curVal
	}
}

func (ctx *MsgContext) CreateVmIfNotExists() {
	if ctx.vm != nil {
		return
	}
	// 初始化骰子
	ctx.vm = ds.NewVM()

	// 根据当前规则开语法 - 暂时是都开
	ctx.vm.Config.EnableDiceWoD = true
	ctx.vm.Config.EnableDiceCoC = true
	ctx.vm.Config.EnableDiceFate = true
	ctx.vm.Config.EnableDiceDoubleCross = true
	ctx.vm.Config.EnableV1IfCompatible = true
	ctx.vm.Config.OpCountLimit = 30000

	am := ctx.Dice.AttrsManager
	ctx.vm.Config.HookFuncValueStore = func(vm *ds.Context, name string, v *ds.VMValue) (overwrite *ds.VMValue, solved bool) {
		// 临时变量
		if strings.HasPrefix(name, "$t") {
			if ctx.Player.ValueMapTemp == nil {
				ctx.Player.ValueMapTemp = &ds.ValueMap{}
			}
			ctx.Player.ValueMapTemp.Store(name, v)
			// 继续存入local 因此solved为false
			return nil, false
		}

		// 个人变量
		if strings.HasPrefix(name, "$m") {
			if ctx.Session != nil && ctx.Player != nil {
				playerAttrs := lo.Must(am.LoadById(ctx.Player.UserID))
				playerAttrs.Store(name, v)
			}
			return nil, true
		}

		// 群变量
		if ctx.Group != nil && strings.HasPrefix(name, "$g") {
			groupAttrs := lo.Must(am.LoadById(ctx.Group.GroupID))
			groupAttrs.Store(name, v)
			return nil, true
		}
		return nil, false
	}

	ctx.vm.GlobalValueLoadOverwriteFunc = func(name string, curVal *ds.VMValue) *ds.VMValue {
		if curVal == nil {
			// 临时变量
			if strings.HasPrefix(name, "$t") {
				if ctx.Player.ValueMapTemp == nil {
					ctx.Player.ValueMapTemp = &ds.ValueMap{}
				}
				if v, ok := ctx.Player.ValueMapTemp.Load(name); ok {
					return v
				}
			}

			if strings.HasPrefix(name, "$") {
				am := ctx.Dice.AttrsManager
				// 个人变量
				if strings.HasPrefix(name, "$m") {
					if ctx.Session != nil && ctx.Player != nil {
						playerAttrs := lo.Must(am.LoadById(ctx.Player.UserID))
						v := playerAttrs.Load(name)
						if v == nil {
							return ds.NewIntVal(0)
						}
						return v
					}
				}

				// 群变量
				if ctx.Group != nil && strings.HasPrefix(name, "$g") {
					groupAttrs := lo.Must(am.LoadById(ctx.Group.GroupID))
					v := groupAttrs.Load(name)
					if v == nil {
						return ds.NewIntVal(0)
					}
					return v
				}
			}

			// 从模板取值，模板中的设定是如果取不到获得0
			// TODO: 目前没有好的方法去复制ctx，实际上这个行为应当类似于ds中的函数调用
			ctx.CreateVmIfNotExists()
			ctx2 := *ctx
			ctx2.vm = nil
			ctx2.CreateVmIfNotExists()
			ctx2.vm.UpCtx = ctx.vm
			ctx2.vm.Attrs = ctx.vm.Attrs

			name = ctx.SystemTemplate.GetAlias(name)
			v, err := ctx.SystemTemplate.GetRealValue(&ctx2, name)
			if err != nil {
				return ds.NewNullVal()
			}

			if strings.Contains(name, ":") {
				textTmpl := ctx.Dice.TextMap[name]
				if textTmpl != nil {
					if v2, err := DiceFormatV2(ctx, textTmpl.Pick().(string)); err == nil {
						return ds.NewStrVal(v2)
					}
				} else {
					return ds.NewStrVal("<%未定义值-" + name + "%>")
				}
			}

			return v
		}
		return curVal
	}

	ctx.vm.Config.CustomMakeDetailFunc = func(ctx *ds.Context, details []ds.BufferSpan, dataBuffer []byte) string {
		detailResult := dataBuffer[:len(ctx.Matched)]

		var curPoint ds.IntType
		lastEnd := ds.IntType(-1) //nolint:ineffassign

		var m []struct {
			begin ds.IntType
			end   ds.IntType
			tag   string
			spans []ds.BufferSpan
			val   *ds.VMValue
		}

		for _, i := range details {
			// fmt.Println("?", i, lastEnd)
			if i.Begin > lastEnd {
				curPoint = i.Begin
				m = append(m, struct {
					begin ds.IntType
					end   ds.IntType
					tag   string
					spans []ds.BufferSpan
					val   *ds.VMValue
				}{begin: curPoint, end: i.End, tag: i.Tag, spans: []ds.BufferSpan{i}, val: i.Ret})
			} else {
				m[len(m)-1].spans = append(m[len(m)-1].spans, i)
				if i.End > m[len(m)-1].end {
					m[len(m)-1].end = i.End
				}
			}

			if i.End > lastEnd {
				lastEnd = i.End
			}
		}

		var detailArr []*ds.VMValue
		for i := len(m) - 1; i >= 0; i-- {
			// for i := 0; i < len(m); i++ {
			item := m[i]
			size := len(item.spans)
			sort.Sort(spanByEnd(item.spans))
			last := item.spans[size-1]

			subDetailsText := ""
			if size > 1 {
				// 次级结果，如 (10d3)d5 中，此处为10d3的结果
				// 例如 (10d3)d5=63[(10d3)d5=...,10d3=19]
				for j := 0; j < len(item.spans)-1; j++ {
					span := item.spans[j]
					subDetailsText += "," + string(detailResult[span.Begin:span.End]) + "=" + span.Ret.ToString()
				}
			}

			exprText := string(detailResult[item.begin:item.end])

			var r []byte
			r = append(r, detailResult[:item.begin]...)

			part1 := last.Ret.ToString()
			// 主体结果部分，如 (10d3)d5=63[(10d3)d5=63=2+2+2+5+2+5+5+4+1+3+4+1+4+5+4+3+4+5+2,10d3=19]
			detail := "[" + exprText + "=" + part1
			if last.Text != "" && part1 != last.Text {
				// 如果 part1 和相关文本完全相同，直接跳过
				if item.tag == "load" {
					detail += "," + last.Text
				} else if item.tag == "dnd-rc" {
					detail = "[" + last.Text
				} else {
					detail += "=" + last.Text
				}
			}
			subDetailsText = ""
			detail += subDetailsText + "]"

			r = append(r, ([]byte)(last.Ret.ToString()+detail)...)
			r = append(r, detailResult[item.end:]...)
			detailResult = r

			d := ds.NewDictValWithArrayMust(
				ds.NewStrVal("tag"), ds.NewStrVal(item.tag),
				ds.NewStrVal("expr"), ds.NewStrVal(string(detailResult[item.begin:item.end])),
				ds.NewStrVal("val"), item.val,
			)
			detailArr = append(detailArr, d.V())
		}

		ctx.StoreNameLocal("details", ds.NewArrayValRaw(detailArr))
		return string(detailResult)
	}

	// 设置默认骰子面数
	if ctx.Group != nil {
		// 情况不明，在sealchat的第一次测试中出现Group为nil
		ctx.vm.Config.DefaultDiceSideExpr = fmt.Sprintf("%d", ctx.Group.DiceSideNum)
	} else {
		ctx.vm.Config.DefaultDiceSideExpr = "d100"
	}
}

func DiceFormatV2(ctx *MsgContext, s string) (string, error) { //nolint:revive
	ctx.CreateVmIfNotExists()
	ctx.vm.Ret = nil
	ctx.vm.Error = nil
	ctx.vm.Config.DisableStmts = false

	s = CompatibleReplace(ctx, s)

	// 隐藏的内置字符串符号 \x1e
	// err := ctx.vm.Run("\x1e" + s + "\x1e")
	if strings.Contains(s, "处于开启状") {
		ctx.vm.Config.PrintBytecode = true
	}
	v, err := ctx.vm.RunExpr("\x1e"+s+"\x1e", true)
	if err != nil || v == nil {
		fmt.Println("脚本执行出错V2f: ", s, "->", err)
		return "", err
	} else {
		return v.ToString(), nil
	}
}

func DiceFormatTmplV2(ctx *MsgContext, s string) (string, error) { //nolint:revive
	var text string
	a := ctx.Dice.TextMap[s]
	if a == nil {
		text = "<%未知项-" + s + "%>"
	} else {
		text = ctx.Dice.TextMap[s].Pick().(string)
	}

	return DiceFormatV2(ctx, text)
}
