package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

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
		if base.TypeId == ds.VMTypeUndefined {
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

func DiceFormatTmplV1(ctx *MsgContext, s string) string { //nolint:revive
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

	r, _, _ := ctx.Dice.ExprText(s, ctx)
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

type VMResultV2 struct {
	*ds.VMValue
	vm        *ds.Context
	legacy    *VMResult
	cocPrefix string
}

func (r *VMResultV2) GetAsmText() string {
	if r.legacy != nil {
		return r.legacy.Parser.GetAsmText()
	}
	return r.vm.GetAsmText()
}

func (r *VMResultV2) IsCalculated() bool {
	if r.legacy != nil {
		return r.legacy.Parser.Calculated
	}
	return r.vm.IsDiceCalculateExists()
}

func (r *VMResultV2) GetRestInput() string {
	if r.legacy != nil {
		return r.legacy.restInput
	}
	return r.vm.RestInput
}

func (r *VMResultV2) GetMatched() string {
	if r.legacy != nil {
		return r.legacy.Matched
	}
	return r.vm.Matched
}

func (r *VMResultV2) GetCocPrefix() string {
	if r.legacy != nil {
		return r.legacy.Parser.CocFlagVarPrefix
	}
	return r.cocPrefix
}

func (r *VMResultV2) GetVersion() int64 {
	if r.legacy != nil {
		return 1
	}
	return 2
}

// DiceExprEvalBase 不建议用，纯兼容旧版
func DiceExprEvalBase(ctx *MsgContext, s string, flags RollExtraFlags) (*VMResultV2, string, error) {
	ctx.CreateVmIfNotExists()
	vm := ctx.vm
	vm.Ret = nil
	vm.Error = nil

	s = CompatibleReplace(ctx, s)

	vm.Config.DisableStmts = flags.DisableBlock
	vm.Config.IgnoreDiv0 = flags.IgnoreDiv0

	var cocFlagVarPrefix string
	if flags.CocVarNumberMode {
		vm.Config.CallbackLoadVar = func(name string) (string, *ds.VMValue) {
			re := regexp.MustCompile(`^(困难|极难|大成功|常规|失败|困難|極難|常規|失敗)?([^\d]+)(\d+)?$`)
			m := re.FindStringSubmatch(name)

			if len(m) > 0 {
				if m[1] != "" {
					cocFlagVarPrefix = chsS2T.Read(m[1])
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

	err := ctx.vm.Run(s)
	if err != nil || ctx.vm.Ret == nil {
		fmt.Println("脚本执行出错V2: ", s, "->", err)

		// 尝试一下V1
		val, detail, err := ctx.Dice.ExprEvalBase(s, ctx, flags)
		if err != nil {
			return nil, detail, err
		}

		return &VMResultV2{val.ConvertToV2(), ctx.vm, val, cocFlagVarPrefix}, detail, err
	} else {
		return &VMResultV2{ctx.vm.Ret, ctx.vm, nil, cocFlagVarPrefix}, ctx.vm.GetDetailText(), nil
	}
}

// DiceExprTextBase 不建议用，纯兼容旧版
func DiceExprTextBase(ctx *MsgContext, s string, flags RollExtraFlags) (*VMResultV2, string, error) {
	return DiceExprEvalBase(ctx, "\x1e"+s+"\x1e", flags)
}

type spanByEnd []ds.BufferSpan

func (a spanByEnd) Len() int           { return len(a) }
func (a spanByEnd) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a spanByEnd) Less(i, j int) bool { return a[i].End < a[j].End }

func (ctx *MsgContext) CreateVmIfNotExists() {
	if ctx.vm == nil {
		// 初始化骰子
		ctx.vm = ds.NewVM()

		// 根据当前规则开语法 - 暂时是都开
		ctx.vm.Config.EnableDiceWoD = true
		ctx.vm.Config.EnableDiceCoC = true
		ctx.vm.Config.EnableDiceFate = true
		ctx.vm.Config.EnableDiceDoubleCross = true

		ctx.vm.GlobalValueLoadOverwriteFunc = func(name string, curVal *ds.VMValue) *ds.VMValue {
			if curVal == nil {
				return ds.NewIntVal(0)
			}
			return curVal
		}

		ctx.vm.Config.CustomMakeDetailFunc = func(ctx *ds.Context, details []ds.BufferSpan, dataBuffer []byte) string {
			detailResult := dataBuffer

			curPoint := ds.IntType(-1) // nolint
			lastEnd := ds.IntType(-1)  // nolint

			var m []struct {
				begin ds.IntType
				end   ds.IntType
				spans []ds.BufferSpan
			}

			for _, i := range details {
				// fmt.Println("?", i, lastEnd)
				if i.Begin > lastEnd {
					curPoint = i.Begin
					m = append(m, struct {
						begin ds.IntType
						end   ds.IntType
						spans []ds.BufferSpan
					}{begin: curPoint, end: i.End, spans: []ds.BufferSpan{i}})
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
					detail += "=" + last.Text
				}
				subDetailsText = ""
				detail += subDetailsText + "]"

				r = append(r, ([]byte)(last.Ret.ToString()+detail)...)
				r = append(r, detailResult[item.end:]...)
				detailResult = r
			}

			return string(detailResult)
		}

		// 设置默认骰子面数
		ctx.vm.Config.DefaultDiceSideExpr = fmt.Sprintf("%d", ctx.Group.DiceSideNum)
	}
}

func DiceFormatV2(ctx *MsgContext, s string) (string, error) { //nolint:revive
	ctx.CreateVmIfNotExists()
	ctx.vm.Ret = nil
	ctx.vm.Error = nil

	s = CompatibleReplace(ctx, s)

	// 隐藏的内置字符串符号 \x1e
	err := ctx.vm.Run("\x1e" + s + "\x1e")
	if err != nil || ctx.vm.Ret == nil {
		fmt.Println("脚本执行出错V2f: ", s, "->", err)
		return "", err
	} else {
		return ctx.vm.Ret.ToString(), nil
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
