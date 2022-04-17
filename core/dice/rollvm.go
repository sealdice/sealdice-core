package dice

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Type uint8

const (
	TypeNumber Type = iota
	TypePushString
	TypeNegation
	TypeAdd
	TypeSubtract
	TypeMultiply
	TypeDivide
	TypeModulus
	TypeExponentiation
	TypeDiceUnary
	TypeDice
	TypeDicePenalty
	TypeDiceBonus
	TypeDiceFate
	TypeLoadVarname
	TypeLoadFormatString
	TypeStore
	TypeHalt
	TypeSwap
	TypeLeftValueMark
	TypeDiceSetK
	TypeDiceSetQ
	TypeClearDetail

	TypeCompLT
	TypeCompLE
	TypeCompEQ
	TypeCompNE
	TypeCompGE
	TypeCompGT

	TypeJmp
	TypeJe
	TypeJne

	TypeBitwiseAnd
	TypeBitwiseOr
	TypeLogicAnd
	TypeLogicOr
)

type ByteCode struct {
	T        Type
	Value    int64
	ValueStr string
	ValueAny interface{}
}

func (code *ByteCode) String() string {
	switch code.T {
	case TypeJne:
		return "->"
	case TypeJe:
		return "->"
	case TypeJmp:
		return "->"
	case TypeBitwiseAnd:
		return "&"
	case TypeBitwiseOr:
		return "|"
	case TypeLogicAnd:
		return "&&"
	case TypeLogicOr:
		return "||"
	case TypeCompLT:
		return "<"
	case TypeCompLE:
		return "<="
	case TypeCompGT:
		return ">"
	case TypeCompGE:
		return ">="
	case TypeCompEQ:
		return "=="
	case TypeCompNE:
		return "!="
	case TypeAdd:
		return "+"
	case TypeNegation, TypeSubtract:
		return "-"
	case TypeMultiply:
		return "*"
	case TypeDivide:
		return "/"
	case TypeModulus:
		return "%"
	case TypeExponentiation:
		return "**"
	case TypeDiceFate:
		return "df"
	}
	return ""
}

func (code *ByteCode) CodeString() string {
	switch code.T {
	case TypeNumber:
		return "push " + strconv.FormatInt(code.Value, 10)
	case TypePushString:
		return "push.str " + code.ValueStr
	case TypeAdd:
		return "add"
	case TypeNegation, TypeSubtract:
		return "sub"
	case TypeMultiply:
		return "mul"
	case TypeDivide:
		return "div"
	case TypeModulus:
		return "mod"
	case TypeExponentiation:
		return "pow"
	case TypeDice:
		return "dice"
	case TypeDicePenalty:
		return "dice.penalty"
	case TypeDiceBonus:
		return "dice.bonus"
	case TypeDiceFate:
		return "dice.fate"
	case TypeDiceSetK:
		return "dice.setk"
	case TypeDiceSetQ:
		return "dice.setq"
	case TypeDiceUnary:
		return "dice1"
	case TypeLoadVarname:
		return "ld.v " + code.ValueStr
	case TypeLoadFormatString:
		return fmt.Sprintf("ld.fs %s", code.ValueAny.([]string))
	case TypeStore:
		return "store"
	case TypeHalt:
		return "halt"
	case TypeSwap:
		return "swap"
	case TypeLeftValueMark:
		return "mark.left"
	case TypeJmp:
		return fmt.Sprintf("jmp %d", code.Value)
	case TypeJe:
		return fmt.Sprintf("je %d", code.Value)
	case TypeJne:
		return fmt.Sprintf("jne %d", code.Value)
	case TypeCompLT:
		return "comp.lt"
	case TypeCompLE:
		return "comp.le"
	case TypeCompEQ:
		return "comp.eq"
	case TypeCompNE:
		return "comp.ne"
	case TypeCompGE:
		return "comp.ge"
	case TypeCompGT:
		return "comp.gt"
	}
	return ""
}

type RollExtraFlags struct {
	BigFailDiceOn      bool
	DisableLoadVarname bool  // 不允许加载变量，这是为了防止遇到 .r XXX 被当做属性读取，而不是“由于XXX，骰出了”
	CocVarNumberMode   bool  // 特殊的变量模式，此时这种类型的变量“力量50”被读取为50，而解析的文本被算作“力量”，如果没有后面的数字则正常进行
	CocDefaultAttrOn   bool  // 启用COC的默认属性值，如攀爬20等
	IgnoreDiv0         bool  // 当div0时暂不报错
	DisableValueBuff   bool  // 不计算buff值
	DNDAttrReadMod     bool  // 基础属性被读取为调整值，仅在检定时使用
	DNDAttrReadDC      bool  // 将力量豁免读取为力量再计算豁免值
	DefaultDiceSideNum int64 // 默认骰子面数
}

type RollExpression struct {
	Code             []ByteCode
	Top              int
	CocFlagVarPrefix string // 解析过程中出现，当VarNumber开启时有效，可以是困难极难常规大成功

	JmpStack []int
	flags    RollExtraFlags
	Error    error
}

func (e *RollExpression) Init(stackLength int) {
	e.Code = make([]ByteCode, stackLength)
	e.JmpStack = []int{}
}

func (e *RollExpression) checkStackOverflow() bool {
	if e.Error != nil {
		return true
	}
	if e.Top >= len(e.Code) {
		e.Error = errors.New("E1:指令虚拟机栈溢出，请不要发送过于离谱的指令")
		return true
	}
	return false
}

func (e *RollExpression) AddLeftValueMark() {
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].T = TypeLeftValueMark
}

func (e *RollExpression) AddOperator(operator Type) int {
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return -1
	}
	e.Top++
	code[top].T = operator
	return e.Top
}

func (e *RollExpression) PushForOffset() {
	e.JmpStack = append(e.JmpStack, e.Top-1)
}

func (e *RollExpression) PopAndSetOffset() {
	last := len(e.JmpStack) - 1
	codeIndex := e.JmpStack[last]
	e.JmpStack = e.JmpStack[:last]
	e.Code[codeIndex].Value = int64(e.Top - codeIndex - 1)
	//fmt.Println("XXXX", e.Code[codeIndex], "|", e.Top, codeIndex)
}

func (e *RollExpression) AddOperatorWithInt64(operator Type, val int64) {
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].T = operator
	code[top].Value = val
}

func (e *RollExpression) AddLoadVarname(value string) {
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].T = TypeLoadVarname
	code[top].ValueStr = value
}

func (e *RollExpression) AddStore() {
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].T = TypeStore
}

func (e *RollExpression) AddValue(value string) {
	// 实质上的压栈命令
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].Value, _ = strconv.ParseInt(value, 10, 64)
}

func (e *RollExpression) AddValueStr(value string) {
	// 实质上的压栈命令
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].T = TypePushString
	code[top].ValueStr = value
}

func (e *RollExpression) AddFormatString(value string) {
	// 载入一个字符串并格式化
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].T = TypeLoadFormatString
	code[top].Value = 1

	re := regexp.MustCompile(`\{[^}]*?\}`)
	code[top].ValueStr = value
	code[top].ValueAny = re.FindAllString(value, -1)
}

type vmStack = VMValue

type VmResult struct {
	VMValue
	Parser    *DiceRollParser
	Matched   string
	restInput string
}

func (e *RollExpression) Evaluate(d *Dice, ctx *MsgContext) (*vmStack, string, error) {
	stack, top := make([]vmStack, len(e.Code)), 0
	//lastIsDice := false
	//var lastValIndex int
	times := 0
	lastDetails := []string{}
	lastDetailsLeft := []string{}
	calcDetail := ""

	var registerDiceK *VMValue
	var registerDiceQ *VMValue
	var kqFlag int64

	codes := e.Code[0:e.Top]
	for opIndex := 0; opIndex < len(codes); opIndex += 1 {
		code := codes[opIndex]
		//fmt.Println("!!!", code.CodeString())

		// 单目运算符
		switch code.T {
		case TypeLeftValueMark:
			if top == 1 {
				lastDetailsLeft = make([]string, len(lastDetails))
				copy(lastDetailsLeft, lastDetails)
				lastDetails = lastDetails[:0]
			}
			continue
		case TypeLoadFormatString:
			parts := code.ValueAny.([]string)
			str := code.ValueStr

			for index, i := range parts {
				var val vmStack
				if top-len(parts)+index < 0 {
					return nil, "", errors.New("E3: 无效的表达式")
					//val = vmStack{VMTypeString, ""}
				} else {
					val = stack[top-len(parts)+index]
				}
				str = strings.Replace(str, i, val.ToString(), 1)
			}

			top -= len(parts)
			stack[top].TypeId = VMTypeString
			stack[top].Value = str
			top++
			continue
		case TypeNumber:
			stack[top].TypeId = VMTypeInt64
			stack[top].Value = code.Value
			top++
			continue
		case TypeDiceSetK:
			t := stack[top-1]
			registerDiceK = &VMValue{t.TypeId, t.Value}
			kqFlag = code.Value
			top--
			continue
		case TypeJe:
			t := stack[top-1]
			top--

			if t.Value != int64(0) {
				opIndex += int(code.Value)
			}
		case TypeJne:
			t := stack[top-1]
			top--

			if t.Value == int64(0) {
				opIndex += int(code.Value)
			}
			continue
		case TypeJmp:
			opIndex += int(code.Value)
			continue
		case TypeClearDetail:
			calcDetail = ""
			lastDetails = lastDetails[:0]
			top = 0
			continue
		case TypeDiceSetQ:
			t := stack[top-1]
			registerDiceQ = &VMValue{t.TypeId, t.Value}
			kqFlag = code.Value
			top--
			continue
		case TypeDicePenalty, TypeDiceBonus:
			t := stack[top-1]
			diceResult := DiceRoll64(100)
			diceTens := diceResult / 10
			diceUnits := diceResult % 10

			nums := []string{}
			diceMin := diceTens
			diceMax := diceTens
			num10Exists := false
			for i := int64(0); i < t.Value.(int64); i++ {
				n := DiceRoll64(10)

				if n == 10 {
					num10Exists = true
					nums = append(nums, "0")
					continue
				} else {
					nums = append(nums, strconv.FormatInt(n, 10))
				}

				if n < diceMin {
					diceMin = n
				}
				if n > diceMax {
					diceMax = n
				}
			}

			var newVal int64
			if code.T == TypeDiceBonus {
				// 如果个位数不是0，那么允许十位为0
				if diceUnits != 0 && num10Exists {
					diceMin = 0
				}

				newVal = diceMin*10 + diceUnits
				lastDetail := fmt.Sprintf("D100=%d, 奖励 %s", diceResult, strings.Join(nums, " "))
				lastDetails = append(lastDetails, lastDetail)
			} else {
				// 如果个位数为0，那么允许十位为10
				if diceUnits == 0 && num10Exists {
					diceMax = 10
				}

				newVal = diceMax*10 + diceUnits
				lastDetail := fmt.Sprintf("D100=%d, 惩罚 %s", diceResult, strings.Join(nums, " "))
				lastDetails = append(lastDetails, lastDetail)
			}

			stack[top-1].Value = newVal
			stack[top-1].TypeId = VMTypeInt64
			continue
		case TypePushString:
			stack[top].TypeId = VMTypeString
			stack[top].Value = code.ValueStr
			top++
			continue
		case TypeLoadVarname:
			var v interface{}
			var vType VMValueType
			var lastDetail string

			varname := code.ValueStr

			if e.flags.DisableLoadVarname {
				return nil, calcDetail, errors.New("解析失败")
			}

			if e.flags.DNDAttrReadDC {
				// 额外调整值补正，用于检定
				switch varname {
				case "力量豁免", "敏捷豁免", "体质豁免", "智力豁免", "感知豁免", "魅力豁免":
					vName := strings.ReplaceAll(varname, "豁免", "")
					realV, _, err := ctx.Dice.ExprEvalBase(fmt.Sprintf("$豁免_%s", vName), ctx, RollExtraFlags{})
					if err == nil {
						vType = realV.TypeId
						v = realV.Value
					}
					//lastDetail = fmt.Sprintf("%s调整值%d", varname, mod)
					//lastDetail = varname + lastDetail
				}
			}

			if e.flags.CocVarNumberMode {
				re := regexp.MustCompile(`^(困难|极难|大成功|常规|失败)?([^\d]+)(\d+)?$`)
				m := re.FindStringSubmatch(code.ValueStr)
				if len(m) > 0 {
					if m[1] != "" {
						e.CocFlagVarPrefix = m[1]
						varname = varname[len(m[1]):]
					}

					// 有末值时覆盖，有初值时
					if m[3] != "" {
						vType = VMTypeInt64
						v, _ = strconv.ParseInt(m[3], 10, 64)
					}
				}
			}

			if v == nil && ctx != nil {
				var exists bool
				v2, exists := VarGetValue(ctx, varname)

				if e.flags.CocDefaultAttrOn {
					if !exists {
						if varname == "母语" {
							v2, exists = VarGetValue(ctx, "edu")
						}
					}

					if !exists {
						if varname == "闪避" {
							// 闪避默认值为敏捷的一半
							v2, exists = VarGetValue(ctx, "敏捷")
							if exists {
								if v2.TypeId == VMTypeInt64 {
									v2.Value = v2.Value.(int64) / 2
								}
							}
						}
					}

					if !exists {
						var val int64
						val, exists = Coc7DefaultAttrs[varname]
						if exists {
							v2 = &VMValue{VMTypeInt64, val}
						}
					}
				}

				if exists {
					vType = v2.TypeId
					v = v2.Value
				} else {
					textTmpl := ctx.Dice.TextMap[varname]
					if textTmpl != nil {
						vType = VMTypeString
						v = DiceFormat(ctx, textTmpl.Pick().(string))
					} else {
						if strings.Contains(varname, ":") {
							vType = VMTypeString
							v = "<%未定义值-" + varname + "%>"
						} else {
							vType = VMTypeInt64 // 这个方案不好，更多类型的时候就出事了
							v = int64(0)
						}
					}
				}
			}

			if vType == VMTypeComputedValue {
				// 解包计算属性
				vd := v.(*VMComputedValueData)
				VarSetValue(ctx, "$tVal", &vd.BaseValue)
				realV, _, err := ctx.Dice.ExprEvalBase(vd.Expr, ctx, RollExtraFlags{})
				if err != nil {
					return nil, "", errors.New("E3: 获取计算属性异常: " + vd.Expr)
				}
				vType = realV.TypeId
				v = realV.Value
			}

			if vType == VMTypeInt64 {
				lastDetail = fmt.Sprintf("%d", v)
			}

			if !e.flags.DisableValueBuff {
				_, exists := VarGetValue(ctx, "$buff_"+varname)
				if exists {
					if vType == VMTypeInt64 {
						buffV, _, err := ctx.Dice.ExprEvalBase("$buff_"+varname, ctx, RollExtraFlags{})
						if err == nil {
							if buffV.TypeId == VMTypeInt64 {
								lastDetail += fmt.Sprintf("+%d", buffV.Value.(int64))
								v = v.(int64) + buffV.Value.(int64)
							}
						}
					}
				}
			}

			detailFlag := false
			if e.flags.DNDAttrReadMod {
				// 额外调整值补正，用于检定
				if v != nil {
					switch varname {
					case "力量", "敏捷", "体质", "智力", "感知", "魅力":
						if vType == VMTypeInt64 {
							detailFlag = true
							mod := v.(int64)/2 - 5
							v = mod
							lastDetail = fmt.Sprintf("%s调整值%d", varname, mod)
							//lastDetail = varname + lastDetail
						}
					}
				}
			}

			if !detailFlag {
				lastDetail = fmt.Sprintf("%s=%s", varname, lastDetail)
			}

			stack[top].TypeId = vType
			stack[top].Value = v
			top++

			if vType == VMTypeInt64 {
				lastDetails = append(lastDetails, lastDetail)
			}
			continue
		case TypeNegation:
			a := &stack[top-1]
			a.Value = -a.Value.(int64)
			continue
		case TypeDiceUnary:
			a := &stack[top-1]
			// Dice XXX, 如 d100
			a.Value = DiceRoll64(a.Value.(int64))
			continue
		case TypeHalt:
			if len(lastDetails) > 0 {
				calcDetail += fmt.Sprintf("[%s]", strings.Join(lastDetails, ","))
				lastDetails = lastDetails[:0]
			}
			continue
		}

		a, b := &stack[top-2], &stack[top-1]
		//lastValIndex = top-3
		top--

		checkDice := func(t *ByteCode) {
			// 第一次 左然后右
			// 后 一直右
			times += 1

			checkLeft := func() {
				if calcDetail == "" {
					if a.TypeId == VMTypeNone {
						calcDetail += "0"
					} else {
						calcDetail += strconv.FormatInt(a.Value.(int64), 10)
					}
				}

				if len(lastDetailsLeft) > 0 {
					vLeft := "[" + strings.Join(lastDetailsLeft, ",") + "]"
					calcDetail += vLeft
				}
			}

			if t.T != TypeDice && top == 1 {
				if times == 1 {
					if t.T == TypeDiceFate {
						// 如果什么都不输出，反而正常了……不然会这样：
						// <木落>掷出了 f + 1=0 df 0[0-++] + 1=2
					} else {
						calcDetail += fmt.Sprintf("%d %s %d", a.Value.(int64), t.String(), b.Value.(int64))
					}
				} else {
					checkLeft()
					calcDetail += fmt.Sprintf(" %s %d", t.String(), b.Value.(int64))

					if len(lastDetails) > 0 {
						calcDetail += fmt.Sprintf("[%s]", strings.Join(lastDetails, ","))
						lastDetails = lastDetails[:0]
					}
				}
			}
		}

		var aInt, bInt int64
		if a.TypeId == 0 {
			aInt = a.Value.(int64)
		}
		if b.TypeId == 0 {
			bInt = b.Value.(int64)
		}

		boolToInt64 := func(val bool) int64 {
			if val {
				return 1
			}
			return 0
		}

		// 二目运算符
		switch code.T {
		case TypeCompLT:
			checkDice(&code)
			a.Value = boolToInt64(aInt < bInt)
		case TypeCompLE:
			checkDice(&code)
			a.Value = boolToInt64(aInt <= bInt)
		case TypeCompEQ:
			checkDice(&code)
			a.Value = boolToInt64(aInt == bInt)
		case TypeCompNE:
			checkDice(&code)
			a.Value = boolToInt64(aInt != bInt)
		case TypeCompGT:
			checkDice(&code)
			a.Value = boolToInt64(aInt > bInt)
		case TypeCompGE:
			checkDice(&code)
			a.Value = boolToInt64(aInt >= bInt)
		case TypeBitwiseAnd:
			checkDice(&code)
			a.Value = aInt & bInt
		case TypeBitwiseOr:
			checkDice(&code)
			a.Value = aInt | bInt
		case TypeAdd:
			checkDice(&code)
			a.Value = aInt + bInt
		case TypeSubtract:
			checkDice(&code)
			a.Value = aInt - bInt
		case TypeMultiply:
			checkDice(&code)
			a.Value = aInt * bInt
		case TypeDivide:
			checkDice(&code)
			if e.flags.IgnoreDiv0 {
				if bInt == 0 {
					bInt = 1 // 这种情况是为了读取 sc 1/0 的值，不是真的做运算，注意！！
				}
			} else {
				if bInt == 0 {
					return nil, "", errors.New("E2:被除数为0")
				}
			}
			a.Value = aInt / bInt
		case TypeModulus:
			checkDice(&code)
			a.Value = aInt % bInt
		case TypeExponentiation:
			checkDice(&code)
			a.Value = int64(math.Pow(float64(aInt), float64(bInt)))
		case TypeSwap:
			a.Value, b.Value = bInt, aInt
			top++
		case TypeStore:
			top--
			if ctx != nil {
				VarSetValue(ctx, a.Value.(string), b)
				//p.SetValueInt64(a.value.(string), b.value.(int64), nil)
			}
			stack[top].TypeId = b.TypeId
			stack[top].Value = b.Value
			top++
			continue
		case TypeDiceFate:
			checkDice(&code)
			text := ""
			sum := int64(0)
			for i := 0; i < 4; i++ {
				n := rand.Int63()%3 - 1
				sum += n
				switch n {
				case -1:
					text += "-"
				case 0:
					text += "0"
				case +1:
					text += "+"
				}
			}
			lastDetail := text
			lastDetails = append(lastDetails, lastDetail)
			//a := &stack[top]
			a.TypeId = VMTypeInt64
			a.Value = sum
			//top++
			//continue
		case TypeDice:
			checkDice(&code)
			if bInt == 0 {
				bInt = e.flags.DefaultDiceSideNum
				if bInt == 0 {
					bInt = 100
				}
			}

			if registerDiceK != nil || registerDiceQ != nil {
				var diceKQ int64
				isDiceK := registerDiceK != nil

				if isDiceK {
					diceKQ = registerDiceK.Value.(int64)
				} else {
					diceKQ = registerDiceQ.Value.(int64)
				}
				if kqFlag == 1 {
					diceKQ = aInt - diceKQ
				}

				var nums []int64
				for i := int64(0); i < aInt; i += 1 {
					if e.flags.BigFailDiceOn {
						nums = append(nums, bInt)
					} else {
						nums = append(nums, DiceRoll64(bInt))
					}
				}

				if isDiceK {
					sort.Slice(nums, func(i, j int) bool { return nums[i] > nums[j] })
				} else {
					sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
				}

				num := int64(0)
				for i := int64(0); i < diceKQ; i++ {
					num += nums[i]
				}

				text := "{"
				for i := int64(0); i < int64(len(nums)); i++ {
					if i == diceKQ {
						text += "| "
					}
					text += fmt.Sprintf("%d ", nums[i])
				}
				text += "}"

				lastDetail := text
				lastDetails = append(lastDetails, lastDetail)
				a.Value = num

				registerDiceK = nil
				registerDiceQ = nil
			} else {
				// XXX Dice YYY, 如 3d100
				var num int64
				text := ""
				for i := int64(0); i < aInt; i += 1 {
					var curNum int64
					if e.flags.BigFailDiceOn {
						curNum = bInt
					} else {
						curNum = DiceRoll64(bInt)
					}

					num += curNum
					text += fmt.Sprintf("+%d", curNum)
				}

				var suffix string
				if aInt > 1 {
					suffix = ", " + text[1:]
				}

				lastDetail := fmt.Sprintf("%dd%d=%d%s", aInt, bInt, num, suffix)
				lastDetails = append(lastDetails, lastDetail)
				a.Value = num
			}
		}
	}

	return &stack[0], calcDetail, nil
}

func (e *RollExpression) GetAsmText() string {
	ret := ""
	ret += "=== VM Code ===\n"
	for index, i := range e.Code {
		if index >= e.Top {
			break
		}
		s := i.CodeString()
		if s != "" {
			ret += s + "\n"
		} else {
			ret += "@raw: " + string(i.T) + "\n"
		}
	}
	ret += "=== VM Code End===\n"
	return ret
}
