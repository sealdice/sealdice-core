package dice

import (
	"bytes"
	"encoding/json"
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

func DiceFormatV1(ctx *MsgContext, s string) (string, error) { //nolint:revive
	s = CompatibleReplace(ctx, s)

	r, _, err := ctx.Dice._ExprTextV1(s, ctx)
	return r, err
}

func DiceFormat(ctx *MsgContext, s string) string {
	ret, err := DiceFormatV2(ctx, s)
	if err != nil {
		// 遇到异常，尝试一下V1
		ret, _ = DiceFormatV1(ctx, s)
		return ret
	}
	return ret
}

func DiceFormatTmpl(ctx *MsgContext, s string) string {
	var text string
	a := ctx.Dice.TextMap[s]
	if a == nil {
		text = "<%未知项-" + s + "%>"
	} else {
		text = ctx.Dice.TextMap[s].Pick().(string)

		// 找出其兼容情况，以决定使用什么版本的引擎
		engineVersion := "v2"
		if items, exists := ctx.Dice.TextMapCompatible.Load(s); exists {
			if info, exists := items.Load(text); exists {
				if info.Version == "v1" {
					engineVersion = "v1"
				}
			}
		}

		if engineVersion == "v2" {
			ret, _ := DiceFormatV2(ctx, text)
			return ret
		} else if engineVersion == "v1" {
			ret, _ := DiceFormatV1(ctx, text)
			return ret
		}
	}

	return text
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
					if detail != nil {
						detail.Tag = "dnd-rc"
						detail.Text = fmt.Sprintf("%s调整值%d", varname, mod)
					}
					return ds.NewIntVal(mod)
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

func _MsgCreate(messageType string, message string) *Message {
	if messageType == "" {
		messageType = "private"
	}

	userID := "UI:1001"
	groupID := ""
	groupName := ""
	groupRole := ""
	if messageType == "group" {
		userID = "UI:1002"
		messageType = "group"
		groupID = "UI-Group:2001"
		groupName = "UI-Group 2001"
		groupRole = "owner"
	}

	msg := &Message{
		MessageType: messageType,
		Message:     message,
		Platform:    "UI",
		Sender: SenderBase{
			Nickname:  "User",
			UserID:    userID,
			GroupRole: groupRole,
		},
		GroupID:   groupID,
		GroupName: groupName,
	}

	return msg
}

// TextMapCompatibleCheck 新旧预设文本兼容性检测
func TextMapCompatibleCheck(d *Dice, category, k string, textItems []TextTemplateItem) {
	key := fmt.Sprintf("%s:%s", category, k)
	x, _ := d.TextMapCompatible.LoadOrStore(key, &SyncMap[string, TextItemCompatibleInfo]{})

	for _, textItem := range textItems {
		formatExpr := textItem[0].(string)

		msg := _MsgCreate("group", "")

		// 注: 由于选择了真正执行一遍的方式，可能会有部分影响溢出导致修改到测试用户的数据
		// 但是这个测试用户是 UI:1001 所以姑且认为没有问题
		ctx := CreateTempCtx(d.UIEndpoint, msg)
		ctx.CreateVmIfNotExists()
		if a, exists := _textMapTestData2[key]; exists {
			if x, err := a.ToJSON(); err != nil {
				_ = json.Unmarshal(x, ctx.vm.Attrs) // TODO: 性能好一点的clone
			}
			for k, v := range _textMapBuiltin {
				ctx.vm.Attrs.Store(k, v.Clone())
			}
		}
		text2, err2 := DiceFormatV2(ctx, formatExpr)

		ctx = CreateTempCtx(d.UIEndpoint, msg)
		_, presetExists := _textMapTestData2[key]
		if a, exists := _textMapTestData2[key]; exists {
			if x, err := a.ToJSON(); err != nil {
				_ = json.Unmarshal(x, ctx.vm.Attrs) // TODO: 性能好一点的clone
			}
			for k, v := range _textMapBuiltin {
				ctx.vm.Attrs.Store(k, v.Clone())
			}
		}
		text1, _ := DiceFormatV1(ctx, formatExpr)

		// 情况分支: v2无语法错
		ver := "v2"
		extra := ""

		if err2 == nil {
			if text1 == text2 {
				// 情况1: 两版输出完全相同
				ver = "v2"
			} else {
				// 情况2: 两版输出不同，但没报错，v1
				ver = "v1"
			}
		} else {
			// 情况3: v2编译不通过
			ver = "v1"
			extra = "v2error"
		}

		x.Store(formatExpr, TextItemCompatibleInfo{Version: ver, TextV2: text2, TextV1: text1, PresetExists: presetExists, Extra: extra})
	}
}

func TextMapCompatibleCheckAll(d *Dice) {
	for k, v := range _textMapTestData {
		attrs := ds.ValueMap{}
		if err := json.Unmarshal([]byte(v), &attrs); err == nil {
			_textMapTestData2[k] = &attrs
		}
	}

	for category, item := range d.TextMapRaw {
		for k, v := range item {
			TextMapCompatibleCheck(d, category, k, v)
		}
	}
}

var _textMapTestData2 = map[string]*ds.ValueMap{}

var _textMapTestData = map[string]string{
	"COC:判定_大失败":          "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_大成功":          "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_失败":           "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_必须_困难_失败":     "{\"$tD100\":{\"t\":0,\"v\":80},\"$tSuccessRank\":{\"t\":0,\"v\":-1},\"$t判定值\":{\"t\":0,\"v\":40},\"$t附加判定结果\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_必须_困难_成功":     "{\"$tD100\":{\"t\":0,\"v\":40},\"$tSuccessRank\":{\"t\":0,\"v\":2},\"$t判定值\":{\"t\":0,\"v\":40},\"$t附加判定结果\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_必须_大成功_失败":    "{\"$tD100\":{\"t\":0,\"v\":80},\"$tSuccessRank\":{\"t\":0,\"v\":-1},\"$t判定值\":{\"t\":0,\"v\":2},\"$t附加判定结果\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_必须_大成功_成功":    "{\"$tD100\":{\"t\":0,\"v\":1},\"$tSuccessRank\":{\"t\":0,\"v\":4},\"$t判定值\":{\"t\":0,\"v\":2},\"$t附加判定结果\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_必须_极难_失败":     "{\"$tD100\":{\"t\":0,\"v\":80},\"$tSuccessRank\":{\"t\":0,\"v\":-1},\"$t判定值\":{\"t\":0,\"v\":16},\"$t附加判定结果\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_必须_极难_成功":     "{\"$tD100\":{\"t\":0,\"v\":10},\"$tSuccessRank\":{\"t\":0,\"v\":3},\"$t判定值\":{\"t\":0,\"v\":16},\"$t附加判定结果\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_成功_困难":        "{\"$tD100\":{\"t\":0,\"v\":40},\"$tSuccessRank\":{\"t\":0,\"v\":2},\"$t判定值\":{\"t\":0,\"v\":80},\"$t判定结果\":{\"t\":2,\"v\":\"困难成功\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"困难成功\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"困难成功\"},\"$t属性表达式文本\":{\"t\":2,\"v\":\"力量80\"},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(40 )\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":2},\"$t结果文本\":{\"t\":2,\"v\":\"(40 )=40/80 困难成功\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_成功_普通":        "{\"$tD100\":{\"t\":0,\"v\":79},\"$tSuccessRank\":{\"t\":0,\"v\":1},\"$t判定值\":{\"t\":0,\"v\":80},\"$t判定结果\":{\"t\":2,\"v\":\"成功\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"成功\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"成功\"},\"$t属性表达式文本\":{\"t\":2,\"v\":\"力量80\"},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(79 )\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":2},\"$t结果文本\":{\"t\":2,\"v\":\"(79 )=79/80 成功\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_成功_极难":        "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_简短_大失败":       "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_简短_大成功":       "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_简短_失败":        "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_简短_成功_困难":     "{\"$tD100\":{\"t\":0,\"v\":40},\"$tSuccessRank\":{\"t\":0,\"v\":2},\"$t判定值\":{\"t\":0,\"v\":80},\"$t判定结果\":{\"t\":2,\"v\":\"困难成功\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"困难成功\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"困难成功\"},\"$t属性表达式文本\":{\"t\":2,\"v\":\"力量80\"},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(40 )\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":2},\"$t结果文本\":{\"t\":2,\"v\":\"(40 )=40/80 困难成功\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_简短_成功_普通":     "{\"$tD100\":{\"t\":0,\"v\":79},\"$tSuccessRank\":{\"t\":0,\"v\":1},\"$t判定值\":{\"t\":0,\"v\":80},\"$t判定结果\":{\"t\":2,\"v\":\"成功\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"成功\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"成功\"},\"$t属性表达式文本\":{\"t\":2,\"v\":\"力量80\"},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(79 )\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":2},\"$t结果文本\":{\"t\":2,\"v\":\"(79 )=79/80 成功\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:判定_简短_成功_极难":     "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:制卡":              "{\"$t1\":{\"t\":0,\"v\":55},\"$t2\":{\"t\":0,\"v\":45},\"$t3\":{\"t\":0,\"v\":25},\"$t4\":{\"t\":0,\"v\":40},\"$t5\":{\"t\":0,\"v\":60},\"$t6\":{\"t\":0,\"v\":45},\"$t7\":{\"t\":0,\"v\":50},\"$t8\":{\"t\":0,\"v\":75},\"$t9\":{\"t\":0,\"v\":45},\"$t制卡结果文本\":{\"t\":2,\"v\":\"力量:35 敏捷:45 意志:40\\n体质:25 外貌:55 教育:65\\n体型:70 智力:65\\nHP:9 幸运:40 [400/440]\\n\\n力量:55 敏捷:45 意志:25\\n体质:40 外貌:60 教育:45\\n体型:50 智力:75\\nHP:9 幸运:45 [395/440]\"}}",
	"COC:制卡_分隔符":          "{\"$t1\":{\"t\":0,\"v\":55},\"$t2\":{\"t\":0,\"v\":45},\"$t3\":{\"t\":0,\"v\":25},\"$t4\":{\"t\":0,\"v\":40},\"$t5\":{\"t\":0,\"v\":60},\"$t6\":{\"t\":0,\"v\":45},\"$t7\":{\"t\":0,\"v\":50},\"$t8\":{\"t\":0,\"v\":75},\"$t9\":{\"t\":0,\"v\":45}}",
	"COC:属性设置":            "{\"$t同义词数量\":{\"t\":0,\"v\":0},\"$t数量\":{\"t\":0,\"v\":1},\"$t有效数量\":{\"t\":0,\"v\":1}}",
	"COC:属性设置_列出":         "{\"$t属性信息\":{\"t\":2,\"v\":\"力量:60\\t敏捷:80\\t体质:0\\t体型:0\\n外貌:0\\t智力:0\\t意志:0\\t教育:0\\n理智:90\\tdb:-2\\t克苏鲁神话:0\\t生命值:10/0\\n魔法值:0/0\\t锁匠:3\\t\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}},{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:属性设置_列出_未发现记录":   "{}",
	"COC:属性设置_删除":         "{\"$t失败数量\":{\"t\":0,\"v\":0},\"$t属性列表\":{\"t\":2,\"v\":\"敏捷\"}}",
	"COC:属性设置_增减":         "{\"$t伤害点数\":{\"t\":0,\"v\":0},\"$t变化量\":{\"t\":0,\"v\":1},\"$t变更列表\":{\"t\":2,\"v\":\"hp: 1 ➯ 0 (扣除-1=1)\"},\"$t增加或扣除\":{\"t\":2,\"v\":\"扣除\"},\"$t属性\":{\"t\":2,\"v\":\"hp\"},\"$t当前绑定角色\":{\"t\":2,\"v\":\"测试角色\"},\"$t新值\":{\"t\":0,\"v\":0},\"$t旧值\":{\"t\":0,\"v\":1},\"$t表达式文本\":{\"t\":2,\"v\":\"-1\"}}",
	"COC:属性设置_增减_单项":      "{\"$t伤害点数\":{\"t\":0,\"v\":0},\"$t变化量\":{\"t\":0,\"v\":1},\"$t增加或扣除\":{\"t\":2,\"v\":\"扣除\"},\"$t属性\":{\"t\":2,\"v\":\"hp\"},\"$t当前绑定角色\":{\"t\":2,\"v\":\"测试角色\"},\"$t新值\":{\"t\":0,\"v\":0},\"$t旧值\":{\"t\":0,\"v\":1},\"$t表达式文本\":{\"t\":2,\"v\":\"-1\"}}",
	"COC:属性设置_清除":         "{\"$t数量\":{\"t\":0,\"v\":0}}",
	"COC:技能成长":            "{\"$tD100\":{\"t\":0,\"v\":93},\"$tSuccessRank\":{\"t\":0,\"v\":3},\"$t判定值\":{\"t\":0,\"v\":2000},\"$t判定结果\":{\"t\":2,\"v\":\"失败\"},\"$t增量\":{\"t\":0,\"v\":0},\"$t当前绑定角色\":{\"t\":2,\"v\":\"测试角色\"},\"$t技能\":{\"t\":2,\"v\":\"斗殴\"},\"$t数量\":{\"t\":0,\"v\":1},\"$t新值\":{\"t\":0,\"v\":2000},\"$t旧值\":{\"t\":0,\"v\":2000},\"$t结果文本\":{\"t\":2,\"v\":\"“斗殴”成长失败了！\"},\"$t表达式文本\":{\"t\":2,\"v\":\"\"}}",
	"COC:技能成长_属性未录入":      "{\"$tD100\":{\"t\":0,\"v\":0},\"$tSuccessRank\":{\"t\":0,\"v\":0},\"$t判定值\":{\"t\":0,\"v\":0},\"$t判定结果\":{\"t\":2,\"v\":\"\"},\"$t增量\":{\"t\":0,\"v\":0},\"$t技能\":{\"t\":2,\"v\":\"ASD\"},\"$t数量\":{\"t\":0,\"v\":1},\"$t新值\":{\"t\":0,\"v\":0},\"$t旧值\":{\"t\":0,\"v\":0}}",
	"COC:技能成长_批量":         "{\"$tD100\":{\"t\":0,\"v\":46},\"$tSuccessRank\":{\"t\":0,\"v\":1},\"$t判定值\":{\"t\":0,\"v\":50},\"$t判定结果\":{\"t\":2,\"v\":\"失败\"},\"$t增量\":{\"t\":0,\"v\":0},\"$t当前绑定角色\":{\"t\":2,\"v\":\"测试角色\"},\"$t总结果文本\":{\"t\":2,\"v\":\"“力量”：D100=46/60 失败\\n“力量”成长失败了！\\n\\n“敏捷”：D100=46/50 失败\\n“敏捷”成长失败了！\"},\"$t技能\":{\"t\":2,\"v\":\"敏捷\"},\"$t数量\":{\"t\":0,\"v\":2},\"$t新值\":{\"t\":0,\"v\":50},\"$t旧值\":{\"t\":0,\"v\":50},\"$t结果文本\":{\"t\":2,\"v\":\"“敏捷”成长失败了！\"},\"$t表达式文本\":{\"t\":2,\"v\":\"\"}}",
	"COC:技能成长_批量_分隔符":     "{\"$tD100\":{\"t\":0,\"v\":46},\"$tSuccessRank\":{\"t\":0,\"v\":1},\"$t判定值\":{\"t\":0,\"v\":50},\"$t判定结果\":{\"t\":2,\"v\":\"失败\"},\"$t增量\":{\"t\":0,\"v\":0},\"$t技能\":{\"t\":2,\"v\":\"敏捷\"},\"$t数量\":{\"t\":0,\"v\":2},\"$t新值\":{\"t\":0,\"v\":50},\"$t旧值\":{\"t\":0,\"v\":50},\"$t结果文本\":{\"t\":2,\"v\":\"“敏捷”成长失败了！\"},\"$t表达式文本\":{\"t\":2,\"v\":\"\"}}",
	"COC:技能成长_批量_单条":      "{\"$tD100\":{\"t\":0,\"v\":46},\"$tSuccessRank\":{\"t\":0,\"v\":1},\"$t判定值\":{\"t\":0,\"v\":50},\"$t判定结果\":{\"t\":2,\"v\":\"失败\"},\"$t增量\":{\"t\":0,\"v\":0},\"$t技能\":{\"t\":2,\"v\":\"敏捷\"},\"$t数量\":{\"t\":0,\"v\":2},\"$t新值\":{\"t\":0,\"v\":50},\"$t旧值\":{\"t\":0,\"v\":50},\"$t结果文本\":{\"t\":2,\"v\":\"“敏捷”成长失败了！\"},\"$t表达式文本\":{\"t\":2,\"v\":\"\"}}",
	"COC:技能成长_批量_技能过多警告":  "{\"$t数量\":{\"t\":0,\"v\":12}}",
	"COC:技能成长_结果_失败":      "{\"$tD100\":{\"t\":0,\"v\":93},\"$tSuccessRank\":{\"t\":0,\"v\":3},\"$t判定值\":{\"t\":0,\"v\":2000},\"$t判定结果\":{\"t\":2,\"v\":\"失败\"},\"$t增量\":{\"t\":0,\"v\":0},\"$t技能\":{\"t\":2,\"v\":\"斗殴\"},\"$t数量\":{\"t\":0,\"v\":1},\"$t新值\":{\"t\":0,\"v\":2000},\"$t旧值\":{\"t\":0,\"v\":2000},\"$t表达式文本\":{\"t\":2,\"v\":\"\"}}",
	"COC:技能成长_结果_成功":      "{\"$tD100\":{\"t\":0,\"v\":90},\"$tSuccessRank\":{\"t\":0,\"v\":-1},\"$t判定值\":{\"t\":0,\"v\":0},\"$t判定结果\":{\"t\":2,\"v\":\"成功\"},\"$t增量\":{\"t\":0,\"v\":4},\"$t技能\":{\"t\":2,\"v\":\"斗殴\"},\"$t数量\":{\"t\":0,\"v\":1},\"$t新值\":{\"t\":0,\"v\":4},\"$t旧值\":{\"t\":0,\"v\":0},\"$t表达式文本\":{\"t\":2,\"v\":\"1d4\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:提示_临时疯狂":         "{\"$tD100\":{\"t\":0,\"v\":100},\"$tSuccessRank\":{\"t\":0,\"v\":-2},\"$t判定值\":{\"t\":0,\"v\":99},\"$t判定结果\":{\"t\":2,\"v\":\"大失败！\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"大失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"大失败！\"},\"$t新值\":{\"t\":0,\"v\":89},\"$t旧值\":{\"t\":0,\"v\":99},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(100)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(100)=100/99 大失败！\"},\"$t表达式值\":{\"t\":0,\"v\":10},\"$t表达式文本\":{\"t\":2,\"v\":\" 10d1\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:提示_永久疯狂":         "{\"$tD100\":{\"t\":0,\"v\":100},\"$tSuccessRank\":{\"t\":0,\"v\":-2},\"$t判定值\":{\"t\":0,\"v\":88},\"$t判定结果\":{\"t\":2,\"v\":\"大失败！\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"大失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"大失败！\"},\"$t新值\":{\"t\":0,\"v\":0},\"$t旧值\":{\"t\":0,\"v\":88},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(100)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(100)=100/88 大失败！\"},\"$t表达式值\":{\"t\":0,\"v\":88},\"$t表达式文本\":{\"t\":2,\"v\":\" 9999\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:检定":              "{\"$tD100\":{\"t\":0,\"v\":1},\"$tSuccessRank\":{\"t\":0,\"v\":4},\"$t判定值\":{\"t\":0,\"v\":2},\"$t判定结果\":{\"t\":2,\"v\":\"大成功！这一定是命运石之门的选择！\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"大成功！这一定是命运石之门的选择！\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"大成功！这一定是命运石之门的选择！\"},\"$t原因\":{\"t\":2,\"v\":\"\"},\"$t属性表达式文本\":{\"t\":2,\"v\":\"大成功力量80\"},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(1 )\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(1 )=1/2 大成功！这一定是命运石之门的选择！\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"$t附加判定结果\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:检定_单项结果文本":       "{\"$tD100\":{\"t\":0,\"v\":81},\"$tSuccessRank\":{\"t\":0,\"v\":-1},\"$t判定值\":{\"t\":0,\"v\":80},\"$t判定结果\":{\"t\":2,\"v\":\"失败\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"失败！\"},\"$t属性表达式文本\":{\"t\":2,\"v\":\"力量80\"},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(81 )\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":2},\"$t结果文本\":{\"t\":2,\"v\":\"(81 )=81/80 失败\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:检定_多轮":           "{\"$tD100\":{\"t\":0,\"v\":81},\"$tSuccessRank\":{\"t\":0,\"v\":-1},\"$t判定值\":{\"t\":0,\"v\":80},\"$t判定结果\":{\"t\":2,\"v\":\"失败\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"失败！\"},\"$t原因\":{\"t\":2,\"v\":\"\"},\"$t属性表达式文本\":{\"t\":2,\"v\":\"力量80\"},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(81 )\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":2},\"$t结果文本\":{\"t\":2,\"v\":\"(81 )=81/80 失败\\\\n(81 )=81/80 失败\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:检定_格式错误":         "{}",
	"COC:理智检定":            "{\"$tD100\":{\"t\":0,\"v\":100},\"$tSuccessRank\":{\"t\":0,\"v\":-2},\"$t判定值\":{\"t\":0,\"v\":88},\"$t判定结果\":{\"t\":2,\"v\":\"大失败！\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"大失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"大失败！\"},\"$t提示_角色疯狂\":{\"t\":2,\"v\":\"提示：理智归零，已永久疯狂(可用.ti或.li抽取症状)\\n\"},\"$t新值\":{\"t\":0,\"v\":0},\"$t旧值\":{\"t\":0,\"v\":88},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(100)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(100)=100/88 大失败！\"},\"$t表达式值\":{\"t\":0,\"v\":88},\"$t表达式文本\":{\"t\":2,\"v\":\" 9999\"},\"$t附加语\":{\"t\":2,\"v\":\"\\n你很快就能洞悉一切\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:理智检定_单项结果文本":     "{\"$tD100\":{\"t\":0,\"v\":100},\"$tSuccessRank\":{\"t\":0,\"v\":-2},\"$t判定值\":{\"t\":0,\"v\":88},\"$t判定结果\":{\"t\":2,\"v\":\"大失败！\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"大失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"大失败！\"},\"$t旧值\":{\"t\":0,\"v\":88},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(100)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:理智检定_格式错误":       "{}",
	"COC:理智检定_附加语_大失败":    "{\"$tD100\":{\"t\":0,\"v\":100},\"$tSuccessRank\":{\"t\":0,\"v\":-2},\"$t判定值\":{\"t\":0,\"v\":88},\"$t判定结果\":{\"t\":2,\"v\":\"大失败！\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"大失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"大失败！\"},\"$t提示_角色疯狂\":{\"t\":2,\"v\":\"提示：理智归零，已永久疯狂(可用.ti或.li抽取症状)\\n\"},\"$t新值\":{\"t\":0,\"v\":0},\"$t旧值\":{\"t\":0,\"v\":88},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(100)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(100)=100/88 大失败！\"},\"$t表达式值\":{\"t\":0,\"v\":88},\"$t表达式文本\":{\"t\":2,\"v\":\" 9999\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:理智检定_附加语_大成功":    "{\"$tD100\":{\"t\":0,\"v\":1},\"$tSuccessRank\":{\"t\":0,\"v\":4},\"$t判定值\":{\"t\":0,\"v\":99},\"$t判定结果\":{\"t\":2,\"v\":\"大成功!\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"大成功\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"大成功!\"},\"$t提示_角色疯狂\":{\"t\":2,\"v\":\"\"},\"$t新值\":{\"t\":0,\"v\":99},\"$t旧值\":{\"t\":0,\"v\":99},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(1)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(1)=1/99 大成功!\"},\"$t表达式值\":{\"t\":0,\"v\":0},\"$t表达式文本\":{\"t\":2,\"v\":\"0\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:理智检定_附加语_失败":     "{\"$tD100\":{\"t\":0,\"v\":90},\"$tSuccessRank\":{\"t\":0,\"v\":-1},\"$t判定值\":{\"t\":0,\"v\":89},\"$t判定结果\":{\"t\":2,\"v\":\"失败！\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"失败\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"失败！\"},\"$t提示_角色疯狂\":{\"t\":2,\"v\":\"\"},\"$t新值\":{\"t\":0,\"v\":88},\"$t旧值\":{\"t\":0,\"v\":89},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(90)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(90)=90/89 失败！\"},\"$t表达式值\":{\"t\":0,\"v\":1},\"$t表达式文本\":{\"t\":2,\"v\":\" 1\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:理智检定_附加语_成功":     "{\"$tD100\":{\"t\":0,\"v\":6},\"$tSuccessRank\":{\"t\":0,\"v\":3},\"$t判定值\":{\"t\":0,\"v\":89},\"$t判定结果\":{\"t\":2,\"v\":\"极难成功\"},\"$t判定结果_简短\":{\"t\":2,\"v\":\"极难成功\"},\"$t判定结果_详细\":{\"t\":2,\"v\":\"极难成功\"},\"$t提示_角色疯狂\":{\"t\":2,\"v\":\"\"},\"$t新值\":{\"t\":0,\"v\":89},\"$t旧值\":{\"t\":0,\"v\":89},\"$t检定表达式文本\":{\"t\":2,\"v\":\"(6)\"},\"$t检定计算过程\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"(6)=6/89 极难成功\"},\"$t表达式值\":{\"t\":0,\"v\":0},\"$t表达式文本\":{\"t\":2,\"v\":\"0\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"COC:疯狂发作_即时症状":       "{\"$t疯狂描述\":{\"t\":2,\"v\":\"假性残疾：调查员陷入了心理性的失明，失聪以及躯体缺失感中，持续 1D10=1 轮。\"},\"$t表达式文本\":{\"t\":2,\"v\":\"1D10=2\"},\"$t选项值\":{\"t\":0,\"v\":2},\"$t附加值1\":{\"t\":0,\"v\":1}}",
	"COC:疯狂发作_总结症状":       "{\"$t疯狂描述\":{\"t\":2,\"v\":\"失忆：回过神来，调查员们发现自己身处一个陌生的地方，并忘记了自己是谁。记忆会随时间恢复。\"},\"$t表达式文本\":{\"t\":2,\"v\":\"1D10=1\"},\"$t选项值\":{\"t\":0,\"v\":1},\"$t附加值1\":{\"t\":0,\"v\":7}}",
	"COC:设置房规_当前":         "{\"$t房规\":{\"t\":2,\"v\":\"5\"},\"$t房规序号\":{\"t\":0,\"v\":5},\"$t房规文本\":{\"t\":2,\"v\":\"出1-2且≤(成功率/5)为大成功\\n不满50出96-100大失败，满50出99-100大失败\"}}",
	"DND:先攻_下一回合":         "{\"$t下一回合at\":{\"t\":2,\"v\":\"[At:UI:1001]\"},\"$t下一回合角色名\":{\"t\":2,\"v\":\"测试角色\"},\"$t下下一回合at\":{\"t\":2,\"v\":\"[At:UI:1001]\"},\"$t下下一回合角色名\":{\"t\":2,\"v\":\"测试角色\"},\"$t当前回合at\":{\"t\":2,\"v\":\"[At:UI:1001]\"},\"$t当前回合角色名\":{\"t\":2,\"v\":\"测试角色\"},\"$t新轮开始提示\":{\"t\":2,\"v\":\"新的一轮开始了！\\n\"}}",
	"DND:先攻_新轮开始提示":       "{}",
	"DND:先攻_查看_前缀":        "{}",
	"DND:先攻_清除列表":         "{}",
	"DND:先攻_移除_前缀":        "{}",
	"DND:先攻_设置_前缀":        "{}",
	"DND:先攻_设置_指定单位":      "{\"$t点数\":{\"t\":0,\"v\":13},\"$t目标\":{\"t\":2,\"v\":\"牛头人\"},\"$t表达式\":{\"t\":2,\"v\":\"d20\"},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"DND:受到伤害_昏迷中_附加语":    "{\"$t伤害点数\":{\"t\":0,\"v\":5}}",
	"DND:受到伤害_超过HP上限_附加语": "{\"$t伤害点数\":{\"t\":0,\"v\":49}}",
	"DND:受到伤害_进入昏迷_附加语":   "{\"$t伤害点数\":{\"t\":0,\"v\":0}}",
	"DND:死亡豁免_D1_附加语":     "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"DND:死亡豁免_D20_附加语":    "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"DND:死亡豁免_失败_附加语":     "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"DND:死亡豁免_成功_附加语":     "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"DND:死亡豁免_结局_角色死亡":    "{\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"其它:ping响应":           "{}",
	"其它:抽牌_列表":            "{\"$t原始列表\":{\"t\":2,\"v\":\"导出牌组/可见牌组/击中部位/煤气灯/调查员/克苏鲁神话/万象无常牌/魔豆之袋/狂野魔法浪涌/杂货法袍/人偶依恋/人偶暗示/人偶记忆碎片/人偶宝物/人偶双记忆碎片/青果寻访/随机骰娘寻访/溯洄寻访/惠惠寻访/海豹寻访\"}}",
	"其它:抽牌_找不到牌组":         "{\"$t牌组\":{\"t\":2,\"v\":\"不存在的牌组\"}}",
	"其它:抽牌_找不到牌组_存在类似":    "{\"$t牌组\":{\"t\":2,\"v\":\"煤气\"}}",
	"其它:随机名字":             "{\"$t随机名字文本\":{\"t\":2,\"v\":\"张茜、陈雪芬、空灿飞、孙实、周碧侠\"}}",
	"其它:随机名字_分隔符":         "{}",
	"娱乐:今日人品":             "{}",
	"娱乐:鸽子理由":             "{}",
	"日志:OB_关闭":            "{}",
	"日志:OB_开启":            "{}",
	"日志:记录_上传_失败":         "{\"$t错误原因\":{\"t\":2,\"v\":\"此log不存在，或条目数为空，名字是否正确？\"}}",
	"日志:记录_关闭_失败":         "{}",
	"日志:记录_关闭_成功":         "{\"$t当前记录条数\":{\"t\":0,\"v\":2},\"$t记录名称\":{\"t\":2,\"v\":\"BBB\"}}",
	"日志:记录_列出_导入语":        "{}",
	"日志:记录_删除_失败_找不到":     "{\"$t记录名称\":{\"t\":2,\"v\":\"CCC\"}}",
	"日志:记录_删除_失败_正在进行":    "{\"$t记录名称\":{\"t\":2,\"v\":\"BBB\"}}",
	"日志:记录_删除_成功":         "{\"$t记录名称\":{\"t\":2,\"v\":\"BBB\"}}",
	"日志:记录_开启_失败_无此记录":    "{\"$t记录名称\":{\"t\":2,\"v\":\"ZZZXXXCCC\"}}",
	"日志:记录_开启_成功":         "{\"$t当前记录条数\":{\"t\":0,\"v\":2},\"$t记录名称\":{\"t\":2,\"v\":\"BBB\"}}",
	"日志:记录_新建":            "{\"$t上一记录名称\":{\"t\":2,\"v\":\"AAA\"},\"$t存在开启记录\":{\"t\":0,\"v\":1},\"$t记录名称\":{\"t\":2,\"v\":\"BBB\"}}",
	"日志:记录_结束":            "{\"$t记录名称\":{\"t\":2,\"v\":\"BBB\"}}",
	"核心:快捷指令_列表":          "{\"$t列表内容\":{\"t\":2,\"v\":\"[群].&x => .r d20\"},\"$t快捷指令名\":{\"t\":2,\"v\":\"x\"},\"$t指令\":{\"t\":2,\"v\":\".r d20\"},\"$t指令来源\":{\"t\":2,\"v\":\"群\"}}",
	"核心:快捷指令_列表_分隔符":      "{}",
	"核心:快捷指令_列表_单行":       "{\"$t快捷指令名\":{\"t\":2,\"v\":\"x\"},\"$t指令\":{\"t\":2,\"v\":\".r d20\"},\"$t指令来源\":{\"t\":2,\"v\":\"群\"}}",
	"核心:快捷指令_列表_空":        "{}",
	"核心:快捷指令_删除_未定义":      "{\"$t快捷指令名\":{\"t\":2,\"v\":\"x\"},\"$t指令来源\":{\"t\":2,\"v\":\"个人\"}}",
	"核心:快捷指令_替换":          "{\"$t快捷指令名\":{\"t\":2,\"v\":\"x\"},\"$t指令\":{\"t\":2,\"v\":\".r d20\"},\"$t指令来源\":{\"t\":2,\"v\":\"群\"},\"$t旧指令\":{\"t\":2,\"v\":\".r d10\"}}",
	"核心:提示_私聊不可用":         "{}",
	"核心:昵称_当前":            "{}",
	"核心:昵称_改名":            "{\"$t旧昵称\":{\"t\":2,\"v\":\"<User>\"},\"$t旧昵称_RAW\":{\"t\":2,\"v\":\"User\"}}",
	"核心:昵称_重置":            "{\"$t旧昵称\":{\"t\":2,\"v\":\"<测试角色>\"},\"$t旧昵称_RAW\":{\"t\":2,\"v\":\"测试角色\"}}",
	"核心:暗骰_私聊_前缀":         "{\"$t原因\":{\"t\":2,\"v\":\"\"},\"$t原因句子\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"D30=63\"},\"$t表达式文本\":{\"t\":2,\"v\":\"D30\"},\"$t计算结果\":{\"t\":0,\"v\":63},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"核心:暗骰_群内":            "{\"$t原因\":{\"t\":2,\"v\":\"\"},\"$t原因句子\":{\"t\":2,\"v\":\"\"},\"$t结果文本\":{\"t\":2,\"v\":\"D30=63\"},\"$t表达式文本\":{\"t\":2,\"v\":\"D30\"},\"$t计算结果\":{\"t\":0,\"v\":63},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"核心:留言_已记录":           "{}",
	"核心:角色管理_删除失败_已绑定":    "{\"$t角色名\":{\"t\":2,\"v\":\"测试角色\"}}",
	"核心:角色管理_删除成功":        "{\"$t新角色名\":{\"t\":2,\"v\":\"<测试角色>\"},\"$t角色名\":{\"t\":2,\"v\":\"测试角色\"}}",
	"核心:角色管理_加载成功":        "{\"$t角色名\":{\"t\":2,\"v\":\"测试角色\"}}",
	"核心:角色管理_新建_已存在":      "{\"$t角色名\":{\"t\":2,\"v\":\"测试角色\"}}",
	"核心:角色管理_绑定_失败":       "{\"$t角色名\":{\"t\":2,\"v\":\"ASD\"}}",
	"核心:角色管理_绑定_并未绑定":     "{\"$t角色名\":{\"t\":2,\"v\":\"\"}}",
	"核心:角色管理_绑定_成功":       "{\"$t角色名\":{\"t\":2,\"v\":\"测试角色\"}}",
	"核心:设定默认群组骰子面数":       "{}",
	"核心:设定默认骰子面数":         "{}",
	"核心:设定默认骰子面数_重置":      "{}",
	"核心:设定默认骰子面数_错误":      "{}",
	"核心:骰子保存设置":           "{}",
	"核心:骰子关闭":             "{}",
	"核心:骰子名字":             "{}",
	"核心:骰子帮助文本_其他":        "{}",
	"核心:骰子帮助文本_协议":        "{}",
	"核心:骰子帮助文本_附加说明":      "{}",
	"核心:骰子帮助文本_骰主":        "{}",
	"核心:骰子开启":             "{}",
	"核心:骰子状态附加文本":         "{\"$t供职群数\":{\"t\":0,\"v\":1},\"$t启用群数\":{\"t\":0,\"v\":1},\"$t群内工作状态\":{\"t\":2,\"v\":\"\\n群内工作状态: 开启\"},\"$t群内工作状态_仅状态\":{\"t\":2,\"v\":\"开启\"}}",
	"核心:骰点":               "{\"$t原因\":{\"t\":2,\"v\":\"原因\"},\"$t原因句子\":{\"t\":2,\"v\":\"由于原因，\"},\"$t结果文本\":{\"t\":2,\"v\":\"d=89[D100]=89\"},\"$t表达式文本\":{\"t\":2,\"v\":\"d\"},\"$t计算结果\":{\"t\":0,\"v\":89},\"$t计算过程\":{\"t\":2,\"v\":\"=89[D100]\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"核心:骰点_单项结果文本":        "{\"$t原因\":{\"t\":2,\"v\":\"\"},\"$t原因句子\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":3},\"$t表达式文本\":{\"t\":2,\"v\":\"D30\"},\"$t计算结果\":{\"t\":0,\"v\":32},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"核心:骰点_原因":            "{\"$t原因\":{\"t\":2,\"v\":\"原因\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"核心:骰点_多轮":            "{\"$t原因\":{\"t\":2,\"v\":\"\"},\"$t原因句子\":{\"t\":2,\"v\":\"\"},\"$t次数\":{\"t\":0,\"v\":3},\"$t结果文本\":{\"t\":2,\"v\":\"D30=23\\\\nD30=74\\\\nD30=32\"},\"$t表达式文本\":{\"t\":2,\"v\":\"D30\"},\"$t计算结果\":{\"t\":0,\"v\":32},\"$t计算过程\":{\"t\":2,\"v\":\"\"},\"details\":{\"t\":6,\"v\":{\"List\":[{\"t\":7,\"v\":{\"Dict\":{}}}]}}}",
	"核心:骰点_轮数过多警告":        "{\"$t次数\":{\"t\":0,\"v\":30}}",
}

var _textMapBuiltin = map[string]*ds.VMValue{
	"$t帐号ID":      ds.NewStrVal("UI:1001"),
	"$t骰子帐号":      ds.NewStrVal("SEALCHAT:dtil43G2Y9lh62gduN0Hz"),
	"$tDay":       ds.NewIntVal(6),
	"$t骰子账号":      ds.NewStrVal("SEALCHAT:dtil43G2Y9lh62gduN0Hz"),
	"$t账号ID_RAW":  ds.NewStrVal("1001"),
	"$tDate":      ds.NewIntVal(20240806),
	"$tQQ":        ds.NewStrVal("UI:1001"),
	"$t平台":        ds.NewStrVal("SEALCHAT"),
	"$tWeekday":   ds.NewIntVal(2),
	"$tMinute":    ds.NewIntVal(51),
	"$tTimestamp": ds.NewIntVal(1722880283),
	"$t人品":        ds.NewIntVal(35),
	"$tSecond":    ds.NewIntVal(23),
	"$t骰子昵称":      ds.NewStrVal("海豹核心"),
	"$tHour":      ds.NewIntVal(1),
	"$t帐号ID_RAW":  ds.NewStrVal("1001"),
	"$tMonth":     ds.NewIntVal(8),
	"$t消息类型":      ds.NewStrVal("group"),
	"$t玩家":        ds.NewStrVal("<氪豹>"),
	"$t账号昵称":      ds.NewStrVal("<氪豹>"),
	"$t玩家_RAW":    ds.NewStrVal("氪豹"),
	"$tQQ昵称":      ds.NewStrVal("<User>"),
	"$t帐号昵称":      ds.NewStrVal("<User>"),
	"$t账号ID":      ds.NewStrVal("UI:1001"),
	"$t个人骰子面数":    ds.NewIntVal(0),
	"$tYear":      ds.NewIntVal(2024),

	"$t游戏模式":   ds.NewStrVal("coc7"),
	"$t日志开启":   ds.NewIntVal(0),
	"$t群组骰子面数": ds.NewIntVal(100),
	"$t规则模板":   ds.NewStrVal("coc7"),
	"$tSystem": ds.NewStrVal("coc7"),
	"$t当前记录":   ds.NewStrVal(""),
	"$tMsgID":  ds.NewStrVal("<nil>"),
	"$t当前骰子面数": ds.NewIntVal(100),
	"$t轮数":     ds.NewIntVal(0),

	"$t群号":     ds.NewStrVal("UI-Group:2001"),
	"$t群号_RAW": ds.NewStrVal("2001"),
	"$t群名":     ds.NewStrVal("豹群"),
}
