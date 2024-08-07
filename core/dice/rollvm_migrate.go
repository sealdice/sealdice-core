package dice

import (
	"bytes"
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
	errV2     error
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
	return r.vm.IsCalculateExists() || r.vm.IsComputedLoaded
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
	vm.Config.DiceMaxMode = flags.BigFailDiceOn

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
		errV2 := err // 某种情况下没有这个值，很奇怪

		// 尝试一下V1
		val, detail, err := ctx.Dice._ExprEvalBaseV1(s, ctx, flags)
		if err != nil {
			// 我们不关心 v1 的报错
			return nil, detail, errV2
		}

		return &VMResultV2m{val.ConvertToV2(), ctx.vm, val, cocFlagVarPrefix, errV2}, detail, err
	} else {
		return &VMResultV2m{ctx.vm.Ret, ctx.vm, nil, cocFlagVarPrefix, nil}, ctx.vm.GetDetailText(), nil
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
	ctx.vm.Config.HookFuncValueLoad = func(ctx *ds.Context, name string) (string, *ds.VMValue) {
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

func tryLoadByBuff(ctx *MsgContext, varname string, curVal *ds.VMValue, computedOnly bool, detail *ds.BufferSpan) (*ds.VMValue, bool) {
	buffName := "$buff_" + varname
	replaced := false

	am := ctx.Dice.AttrsManager
	if attrs, _ := am.LoadByCtx(ctx); attrs != nil {
		buffVal := attrs.Load(buffName)
		if buffVal != nil {
			if computedOnly {
				if curVal.TypeId == ds.VMTypeComputedValue && buffVal.TypeId == ds.VMTypeComputedValue {
					// 当buff值也是computed的情况下，进行叠加
					x, _ := curVal.ToJSON()
					newVal := curVal.Clone() // 注: Clone的实现有问题，computed没被正确复制，此处用反序列化绕过
					_ = newVal.UnmarshalJSON(x)
					cdCur, _ := newVal.ReadComputed()
					cdBuff, _ := buffVal.ReadComputed()

					// 将computed的内部值进行相加
					cdBuff.Attrs.Range(func(key string, value *ds.VMValue) bool {
						if v, ok := cdCur.Attrs.Load(key); ok {
							vAddRet := v.OpAdd(ctx.vm, value)
							ctx.vm.Error = nil
							if vAddRet != nil {
								cdCur.Attrs.Store(key, vAddRet)
							}
						} else {
							cdCur.Attrs.Store(key, value.Clone())
						}
						return true
					})

					return newVal, true // 读取完成后使用新的值，对这个值的修改不会反馈到原值
				}

				return curVal, false
			}

			detail.Text += fmt.Sprintf("%s+buff%s", curVal.ToString(), buffVal.ToString())
			newVal := curVal.OpAdd(ctx.vm, buffVal)
			ctx.vm.Error = nil
			if newVal != nil {
				curVal = newVal
				detail.Ret = newVal
				replaced = true
			}
		}
	}
	return curVal, replaced
}

// setDndReadForVM 主要是为rc设定属性豁免，暂时没有写在规则模板的原因是可以自定义detail输出。
// 更新: 属性豁免已经能被规则模板描述，现在是为了buff机制、属性和技能检定，希望能逐渐移动到规则模板
func (ctx *MsgContext) setDndReadForVM(rcMode bool) {
	var skip bool
	loadBuff := true

	ctx.vm.Config.HookFuncValueLoadOverwrite = func(vm *ds.Context, varname string, curVal *ds.VMValue, doCompute func(curVal *ds.VMValue) *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue {
		if ctx.SystemTemplate == nil {
			curVal = doCompute(curVal)
			return curVal
		}

		if strings.HasPrefix(varname, "$org_") {
			varname, _ = strings.CutPrefix(varname, "$org_")
			curVal = vm.LoadName(varname, true, false)
			return curVal
		}

		if loadBuff {
			// 这里只处理一种情况：原值是computed，buff也是computed
			curVal, _ = tryLoadByBuff(ctx, varname, curVal, true, detail)
		}

		curVal = doCompute(curVal)
		if curVal == nil {
			return nil
		}

		if loadBuff {
			curVal, _ = tryLoadByBuff(ctx, varname, curVal, false, detail)
		}

		if !skip && rcMode {
			// rc时将属性替换为调整值，只在0级起作用，避免在函数调用等地方造成影响
			if isAbilityScores(varname) && vm.Depth() == 0 && vm.UpCtx == nil {
				if curVal != nil && curVal.TypeId == ds.VMTypeInt {
					mod := curVal.MustReadInt()/2 - 5
					v := ds.NewIntVal(mod)
					if detail != nil {
						detail.Tag = "dnd-rc"
						detail.Text = fmt.Sprintf("%s调整值%d", varname, mod)
						detail.Ret = v
					}
					return v
				}
			} else if dndAttrParent[varname] != "" && curVal.TypeId == ds.VMTypeInt {
				name := dndAttrParent[varname]
				base, err := ctx.SystemTemplate.GetRealValue(ctx, name)
				v := curVal.MustReadInt()
				if err == nil {
					// ab := tryLoadByBuff(ctx, name, base)
					ab := base
					mod := ab.MustReadInt()/2 - 5

					detail.Tag = "dnd-rc"
					detail.Text = fmt.Sprintf("%s调整值%d", name, mod)
					v -= mod

					exprProficiency := fmt.Sprintf("&%s.factor * 熟练", varname)
					skip = true
					if ret2, _ := ctx.vm.RunExpr(exprProficiency, false); ret2 != nil {
						// 注意: 这个值未必总能读到，如果ret2为nil，那么我们可以忽略这个加值的存在
						detail.Text += fmt.Sprintf("+熟练%s", ret2.ToString())
						if ret2.TypeId == ds.VMTypeInt {
							v -= ret2.MustReadInt()
						}
					}
					ctx.vm.Error = nil
					skip = false
					detail.Text += fmt.Sprintf("+%s%d", varname, v)
				}
			}
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

		var v *ds.VMValue
		if ctx.SystemTemplate != nil {
			// 从模板取值，模板中的设定是如果取不到获得0
			// TODO: 目前没有好的方法去复制vm，实际上这个行为应当类似于ds中的函数调用
			ctx2 := *ctx
			ctx2.vm = nil
			ctx2.CreateVmIfNotExists()
			ctx2.vm.UpCtx = ctx.vm
			ctx2.vm.Attrs = ctx.vm.Attrs
			ctx2.vm.Config = ctx.vm.Config

			name = ctx.SystemTemplate.GetAlias(name)
			v, _ = ctx.SystemTemplate.GetRealValueBase(&ctx2, name)
		} else {
			playerAttrs := lo.Must(am.LoadById(ctx.Player.UserID))
			v = playerAttrs.Load(name)
		}

		// 注: 如果已经有值，就不再覆盖，因此v要判空
		if v == nil && strings.Contains(name, ":") {
			textTmpl := ctx.Dice.TextMap[name]
			if textTmpl != nil {
				if v2, err := DiceFormatV2(ctx, textTmpl.Pick().(string)); err == nil {
					return ds.NewStrVal(v2)
				}
			} else {
				return ds.NewStrVal("<%未定义值-" + name + "%>")
			}
		}

		if v != nil {
			return v
		}

		if curVal == nil {
			return ds.NewIntVal(0)
		}

		return curVal
	}

	reSimpleBP := regexp.MustCompile(`^[bpBP]\d*$`)

	mctx := ctx
	ctx.vm.Config.CustomMakeDetailFunc = func(ctx *ds.Context, details []ds.BufferSpan, dataBuffer []byte) string {
		detailResult := dataBuffer[:len(ctx.Matched)]

		var curPoint ds.IntType
		lastEnd := ds.IntType(-1) //nolint:ineffassign

		type Group struct {
			begin ds.IntType
			end   ds.IntType
			tag   string
			spans []ds.BufferSpan
			val   *ds.VMValue
		}

		// 特殊机制: 从模板读取detail进行覆盖
		for index, i := range details {
			if i.Tag == "load" && mctx.SystemTemplate != nil && ctx.UpCtx == nil {
				expr := string(detailResult[i.Begin:i.End])
				detailExpr := mctx.SystemTemplate.DetailOverwrite[expr]
				if detailExpr == "" {
					// 如果没有，尝试使用通配
					detailExpr = mctx.SystemTemplate.DetailOverwrite["*"]
					if detailExpr != "" {
						// key 应该是等于expr的
						ctx.StoreNameLocal("name", ds.NewStrVal(expr))
					}
				}
				if detailExpr != "" {
					v, err := ctx.RunExpr(detailExpr, true)
					if v != nil {
						details[index].Text = v.ToString()
					}
					if err != nil {
						details[index].Text = err.Error()
					}
				}
			}
		}

		var m []Group
		for _, i := range details {
			// fmt.Println("?", i, lastEnd)
			if i.Begin > lastEnd {
				curPoint = i.Begin
				m = append(m, Group{begin: curPoint, end: i.End, tag: i.Tag, spans: []ds.BufferSpan{i}, val: i.Ret})
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
			buf := bytes.Buffer{}
			writeBuf := func(p []byte) {
				buf.Write(p)
			}
			writeBufStr := func(s string) {
				buf.WriteString(s)
			}

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

			exprText := last.Expr
			baseExprText := string(detailResult[item.begin:item.end])
			if last.Expr == "" {
				exprText = baseExprText
			}

			writeBuf(detailResult[:item.begin])

			// 主体结果部分，如 (10d3)d5=63[(10d3)d5=2+2+2+5+2+5+5+4+1+3+4+1+4+5+4+3+4+5+2,10d3=19]
			partRet := last.Ret.ToString()

			detail := "[" + exprText
			if last.Text != "" && partRet != last.Text { // 规则1.1
				detail += "=" + last.Text
			}

			switch item.tag {
			case "dnd-rc":
				detail = "[" + last.Text
			case "load":
				detail = "[" + exprText
				if last.Text != "" {
					detail += "," + last.Text
				}
			case "dice-coc-bonus", "dice-coc-penalty":
				// 对简单式子进行结果简化，未来或许可以做成通配规则(给左式加个规则进行消除)
				if reSimpleBP.MatchString(exprText) {
					detail = "[" + last.Text[1:len(last.Text)-1]
				}
			}

			detail += subDetailsText + "]"
			if len(m) == 1 && detail == "["+baseExprText+"]" {
				detail = "" // 规则1.3
			}
			if len(detail) > 400 {
				detail = "[略]"
			}
			writeBufStr(partRet + detail)
			writeBuf(detailResult[item.end:])
			detailResult = buf.Bytes()

			d := ds.NewDictValWithArrayMust(
				ds.NewStrVal("tag"), ds.NewStrVal(item.tag),
				ds.NewStrVal("expr"), ds.NewStrVal(exprText),
				ds.NewStrVal("val"), item.val,
			)
			detailArr = append(detailArr, d.V())
		}

		detailStr := string(detailResult)
		if detailStr == ctx.Ret.ToString() {
			detailStr = "" // 如果detail和结果值完全一致，那么将其置空
		}
		ctx.StoreNameLocal("details", ds.NewArrayValRaw(lo.Reverse(detailArr)))
		return detailStr
	}

	// 设置默认骰子面数
	if ctx.Group != nil {
		// 情况不明，在sealchat的第一次测试中出现Group为nil
		ctx.vm.Config.DefaultDiceSideExpr = fmt.Sprintf("%d", ctx.Group.DiceSideNum)
	} else {
		ctx.vm.Config.DefaultDiceSideExpr = "100"
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
