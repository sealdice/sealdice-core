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

	"github.com/samber/lo"
	rand2 "golang.org/x/exp/rand"
)

type Type uint8

const (
	TypePushNumber Type = iota
	TypePushString
	TypePushComputed
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
	TypeDiceWod
	TypeWodSetInit       // 重置参数
	TypeWodSetPool       // 设置骰池(骰数)
	TypeWodSetPoints     // 面数
	TypeWodSetThreshold  // 阈值 >=
	TypeWodSetThresholdQ // 阈值 <=
	TypeDiceDC
	TypeDCSetInit
	TypeDCSetPool   // 骰池
	TypeDCSetPoints // 面数
	TypeLoadVarname
	TypeLoadFormatString
	TypeStore
	TypeHalt
	TypeSwap
	TypeLeftValueMark
	TypeDiceSetK
	TypeDiceSetQ
	TypeClearDetail

	TypePop

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

	TypeStSetName
	TypeStModify
	TypeNop

	TypeLoadVarnameForThis
	TypeConvertInt
	TypeConvertStr
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
	default:
		return ""
	}
}

func (code *ByteCode) CodeString() string {
	switch code.T {
	case TypePushNumber:
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
	case TypeDiceSetK:
		return "dice.setk"
	case TypeDiceSetQ:
		return "dice.setq"
	case TypeDiceUnary:
		return "dice1"
	case TypeDiceFate:
		return "dice.fate"
	case TypeWodSetInit:
		return "wod.init"
	case TypeWodSetPool:
		return "wod.pool"
	case TypeWodSetPoints:
		return "wod.points"
	case TypeWodSetThreshold:
		return "wod.threshold"
	case TypeWodSetThresholdQ:
		return "wod.thresholdQ"
	case TypeDiceDC:
		return "dice.dc"
	case TypeDCSetInit:
		return "dice.setInit"
	case TypeDCSetPool:
		return "dice.setPool"
	case TypeDCSetPoints:
		return "dice.setPoints"
	case TypeDiceWod:
		return "dice.wod"
	case TypeLoadVarname:
		return "ld.v " + code.ValueStr
	case TypeLoadFormatString:
		return fmt.Sprintf("ld.fs %d, %s", code.Value, code.ValueStr)
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
	case TypePop:
		return "pop"
	case TypeClearDetail:
		return "reset"
	case TypeStSetName:
		return "st.set"
	case TypeStModify:
		return "st.mod"
	case TypePushComputed:
		return "push.computed " + code.ValueStr
	case TypeNop:
		return "nop"
	case TypeLoadVarnameForThis:
		return "ld.v.this " + code.ValueStr
	case TypeConvertInt:
		return "tmp.conv.int"
	case TypeConvertStr:
		return "tmp.conv.str"
	default:
		return ""
	}
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
	DisableBitwiseOp   bool  // 禁用位运算，用于st
	DisableBlock       bool  // 禁用语句块(实际还禁用赋值语句，暂定名如此)
	DisableNumDice     bool  // 禁用r2d语法
	DisableBPDice      bool  // 禁用bp语法
	DisableCrossDice   bool  // 禁用双十字骰
	DisableDicePool    bool  // 禁用骰池

	V2Only     bool                                                                    // 仅使用v2
	V1Only     bool                                                                    // 仅使用v1，如果v1 v2两个选项同开，会走v1
	vmDepth    int64                                                                   // 层数
	StCallback func(_type string, name string, val *VMValue, op string, detail string) // st回调
}

type RollExpression struct {
	Code             []ByteCode
	Top              int
	CocFlagVarPrefix string // 解析过程中出现，当VarNumber开启时有效，可以是困难极难常规大成功

	NumOpCount int64 // 算力计数

	JmpStack     []int
	CounterStack []int64
	flags        RollExtraFlags
	Error        error
	TmpCodeStack []int // 目前专用于computed，处理codepush
	Calculated   bool  // 是否经过计算，如骰点和符号运算
}

func (e *RollExpression) Init(stackLength int) {
	e.Code = make([]ByteCode, stackLength)
	e.JmpStack = []int{}
	e.CounterStack = []int64{}
	e.TmpCodeStack = []int{}
}

func (e *RollExpression) checkStackOverflow() bool {
	if e.Error != nil {
		return true
	}
	if e.Top >= len(e.Code) {
		need := len(e.Code) * 2
		if need <= 8192 {
			newCode := make([]ByteCode, need)
			copy(newCode, e.Code)
			e.Code = newCode
		} else {
			e.Error = errors.New("E1:指令虚拟机栈溢出，请不要发送过于离谱的指令")
			return true
		}
	}
	return false
}

func (e *RollExpression) AddLeftValueMark() {
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeLeftValueMark
}

func (e *RollExpression) AddOperator(operator Type) int {
	if e.checkStackOverflow() {
		return -1
	}
	code, top := e.Code, e.Top
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
	// fmt.Println("XXXX", e.Code[codeIndex], "|", e.Top, codeIndex)
}

func (e *RollExpression) CounterPush() {
	e.CounterStack = append(e.CounterStack, 0)
}

func (e *RollExpression) CounterAdd(offset int64) {
	last := len(e.CounterStack) - 1
	if last != -1 {
		e.CounterStack[last] += offset
	}
}

func (e *RollExpression) CounterPop() int64 {
	last := len(e.CounterStack) - 1
	num := e.CounterStack[last]
	e.CounterStack = e.CounterStack[:last]
	return num
}

func (e *RollExpression) AddOperatorWithInt64(operator Type, val int64) {
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = operator
	code[top].Value = val
}

func (e *RollExpression) AddLoadVarname(value string) {
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeLoadVarname
	code[top].ValueStr = value
}

func (e *RollExpression) AddLoadVarnameForThis(value string) {
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeLoadVarnameForThis
	code[top].ValueStr = value
}

func (e *RollExpression) AddStore() {
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeStore
}

func (e *RollExpression) AddValue(value string) {
	// 实质上的压栈命令
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].Value, _ = strconv.ParseInt(value, 10, 64)
}

func (e *RollExpression) AddValueStr(value string) {
	// 实质上的压栈命令
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypePushString
	code[top].ValueStr = value
}

func (e *RollExpression) AddStName() {
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeStSetName
}

func (e *RollExpression) AddStModify(op string, text string) {
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeStModify
	code[top].ValueAny = text
	code[top].ValueStr = op
}

func (e *RollExpression) WriteCode(t Type, value int64, valueStr string) {
	if e.checkStackOverflow() {
		return
	}

	code, top := e.Code, e.Top
	code[top].T = t
	code[top].Value = value
	code[top].ValueStr = valueStr
	e.Top++
}

func (e *RollExpression) CodePush() {
	e.TmpCodeStack = append(e.TmpCodeStack, e.Top)
}

func (e *RollExpression) CodePop() {
	if len(e.TmpCodeStack) > 0 {
		// 先这样凑合一下，因为好像删除中间的指令会引起别的问题
		for i := e.TmpCodeStack[len(e.TmpCodeStack)-1]; i < e.Top; i++ {
			e.Code[i].T = TypeNop
		}
		e.TmpCodeStack = e.TmpCodeStack[:len(e.TmpCodeStack)-1]
	}
}

func (e *RollExpression) AddStoreComputed(text string) {
	// 相比DiceScript被迫少了很多机制，创建对象的位置也变了
	e.WriteCode(TypePushComputed, 0, text)
}

func (e *RollExpression) AddFormatString(value string, num int64) {
	// 载入一个字符串并格式化
	if e.checkStackOverflow() {
		return
	}
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeLoadFormatString
	code[top].Value = num      // 需要合并的字符串数量
	code[top].ValueStr = value // 仅象征性意义
}

type VMStack = VMValue

type VMResult struct {
	VMValue
	Parser    *DiceRollParser
	Matched   string
	restInput string
}

func (e *RollExpression) Evaluate(_ *Dice, ctx *MsgContext) (*VMStack, string, error) {
	stack, top := make([]VMStack, len(e.Code)), 0
	times := 0
	lastDetails := []string{}
	lastDetailsLeft := []string{}
	calcDetail := ""

	var registerDiceK *VMValue
	var registerDiceQ *VMValue
	var kqFlag int64

	var wodState struct {
		pool      *VMValue
		points    *VMValue
		threshold *VMValue
		isGE      bool
	}

	wodInit := func() {
		wodState.pool = &VMValue{TypeID: VMTypeInt64, Value: int64(1)}
		wodState.points = &VMValue{TypeID: VMTypeInt64, Value: int64(10)}   // 面数，默认d10
		wodState.threshold = &VMValue{TypeID: VMTypeInt64, Value: int64(8)} // 成功线，默认9
		wodState.isGE = true

		am := ctx.Dice.AttrsManager
		groupAttrs := lo.Must(am.LoadById(ctx.Group.GroupID))

		if threshold, exists := groupAttrs.LoadX("wodThreshold"); exists {
			wodState.threshold = dsValueToRollVMv1(threshold)
		}
		if wodPoints, exists := groupAttrs.LoadX("wodPoints"); exists {
			wodState.points = dsValueToRollVMv1(wodPoints)
		}
	}

	var dcState struct {
		pool   *VMValue
		points *VMValue
	}

	dcInit := func() {
		dcState.pool = &VMValue{TypeID: VMTypeInt64, Value: int64(1)}    // 骰数，默认1
		dcState.points = &VMValue{TypeID: VMTypeInt64, Value: int64(10)} // 面数，默认d10
	}

	numOpCountAdd := func(count int64) bool {
		e.NumOpCount += count
		return e.NumOpCount > 30000
	}

	getE5 := func() error {
		return errors.New("e5: 超出单指令允许算力，不予计算")
	}

	codes := e.Code[0:e.Top]
	// fmt.Println("!!!!!", e.GetAsmText())

	for opIndex := 0; opIndex < len(codes); opIndex++ {
		code := codes[opIndex]
		// fmt.Println("!!!", code.CodeString(), time.Now().UnixMilli())

		// 单目运算符
		switch code.T {
		case TypeLeftValueMark:
			if top == 1 {
				lastDetailsLeft = make([]string, len(lastDetails))
				copy(lastDetailsLeft, lastDetails)
				lastDetails = lastDetails[:0]
			}
			continue
		case TypePop:
			top--
			continue
		case TypeLoadFormatString:
			num := int(code.Value)

			outStr := ""
			for index := 0; index < num; index++ {
				var val VMStack
				if top-num+index < 0 {
					return nil, "", errors.New("E3:无效的表达式")
					// val = vmStack{VMTypeString, ""}
				}
				val = stack[top-num+index]
				outStr += val.ToString()
			}
			outStr = strings.ReplaceAll(outStr, "\\n", "\n")
			// for index, i := range parts {
			//	var val vmStack
			//	if top-len(parts)+index < 0 {
			//		return nil, "", errors.New("E3: 无效的表达式")
			//		//val = vmStack{VMTypeString, ""}
			//	} else {
			//		val = stack[top-len(parts)+index]
			//	}
			//	str = strings.Replace(str, i, val.ToString(), 1)
			// }

			top -= num
			stack[top].TypeID = VMTypeString
			stack[top].Value = outStr
			top++
			continue
		case TypePushNumber:
			stack[top].TypeID = VMTypeInt64
			stack[top].Value = code.Value
			top++
			continue
		case TypeDiceSetK:
			t := stack[top-1]
			registerDiceK = &VMValue{TypeID: t.TypeID, Value: t.Value}
			kqFlag = code.Value
			top--
			continue
		case TypeJe:
			t := stack[top-1]
			top--

			if t.AsBool() {
				opIndex += int(code.Value)
			}
			e.Calculated = true
		case TypeJne:
			t := stack[top-1]
			top--

			if !t.AsBool() {
				opIndex += int(code.Value)
			}
			e.Calculated = true
			continue
		case TypeJmp:
			opIndex += int(code.Value)
			e.Calculated = true
			continue
		case TypeClearDetail:
			calcDetail = ""
			lastDetails = lastDetails[:0]
			top = 0
			continue
		case TypeNop:
			continue
		case TypeDiceSetQ:
			t := stack[top-1]
			registerDiceQ = &VMValue{TypeID: t.TypeID, Value: t.Value}
			kqFlag = code.Value
			top--
			continue
		case TypeWodSetInit:
			wodInit()
			continue
		case TypeWodSetPoints:
			t := stack[top-1]
			wodState.points = &VMValue{TypeID: t.TypeID, Value: t.Value}
			top--
			continue
		case TypeWodSetThreshold:
			t := stack[top-1]
			wodState.threshold = &VMValue{TypeID: t.TypeID, Value: t.Value}
			wodState.isGE = true
			top--
			continue
		case TypeWodSetThresholdQ:
			t := stack[top-1]
			wodState.threshold = &VMValue{TypeID: t.TypeID, Value: t.Value}
			wodState.isGE = false
			top--
			continue
		case TypeWodSetPool:
			t := stack[top-1]
			wodState.pool = &VMValue{TypeID: t.TypeID, Value: t.Value}
			top--
			continue
		case TypeDiceWod:
			t := &stack[top-1] // 加骰线
			ret, nums, rounds, details := DiceWodRollVM(ctx._v1Rand, e, t, wodState.pool, wodState.points, wodState.threshold, wodState.isGE)
			if e.Error != nil {
				return nil, "", e.Error
			}
			stack[top-1].Value = ret.Value
			stack[top-1].TypeID = ret.TypeID

			roundsText := ""
			if rounds > 1 {
				roundsText = fmt.Sprintf(" 轮数:%d", rounds)
			}

			detailText := ""
			if len(details) > 0 {
				detailText = " " + strings.Join(details, ",")
			}
			lastDetail := fmt.Sprintf("成功%d/%d%s%s", ret.Value, nums, roundsText, detailText)
			lastDetails = append(lastDetails, lastDetail)
			e.Calculated = true
			continue
		case TypeDCSetPool:
			t := stack[top-1]
			dcState.pool = &VMValue{TypeID: t.TypeID, Value: t.Value}
			top--
			continue
		case TypeDCSetPoints:
			t := stack[top-1]
			dcState.points = &VMValue{TypeID: t.TypeID, Value: t.Value}
			top--
			continue
		case TypeDCSetInit:
			dcInit()
			continue
		case TypeDiceDC:
			t := &stack[top-1] // 暴击值 / 也可以理解为加骰线
			ret, nums, rounds, details := DiceDCRollVM(ctx._v1Rand, e, t, dcState.pool, dcState.points)
			if e.Error != nil {
				return nil, "", e.Error
			}
			stack[top-1].Value = ret.Value
			stack[top-1].TypeID = ret.TypeID

			detailText := ""
			if len(details) > 0 {
				detailText = " " + strings.Join(details, ",")
			}

			roundsText := ""
			if rounds > 1 {
				roundsText = fmt.Sprintf(" 轮数:%d", rounds)
			}

			if ret.Value == 1 {
				lastDetail := fmt.Sprintf("大失败 出目%d/%d%s%s", ret.Value, nums, roundsText, detailText)
				lastDetails = append(lastDetails, lastDetail)
			} else {
				lastDetail := fmt.Sprintf("出目%d/%d%s%s", ret.Value, nums, roundsText, detailText)
				lastDetails = append(lastDetails, lastDetail)
			}
			e.Calculated = true
			continue
		case TypeDicePenalty, TypeDiceBonus:
			t := stack[top-1]
			diceResult := DiceRoll64x(ctx._v1Rand, 100)
			diceTens := diceResult / 10
			diceUnits := diceResult % 10

			nums := []string{}
			diceMin := diceTens
			diceMax := diceTens
			num10Exists := false

			if numOpCountAdd(t.Value.(int64)) {
				return nil, "", getE5()
			}

			for i := int64(0); i < t.Value.(int64); i++ {
				n := DiceRoll64x(ctx._v1Rand, 10)

				if n == 10 {
					num10Exists = true
					nums = append(nums, "0")
					continue
				}
				nums = append(nums, strconv.FormatInt(n, 10))

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
			stack[top-1].TypeID = VMTypeInt64
			e.Calculated = true
			continue
		case TypePushString:
			unquote, err := strconv.Unquote(`"` + strings.ReplaceAll(code.ValueStr, `"`, `\"`) + `"`)
			if err != nil {
				unquote = code.ValueStr
			}
			stack[top].TypeID = VMTypeString
			stack[top].Value = unquote
			top++
			continue
		case TypeStSetName:
			stVal := stack[top-1]
			stName := stack[top-2]

			if e.flags.StCallback != nil {
				name, _ := stName.ReadString()
				e.flags.StCallback("set", name, &stVal, "", "")
			}
			// 这样子栈好像是对的，但是为啥我也不懂……
			// 之前的栈管理实在是太神秘了
			continue
		case TypeStModify:
			stVal := stack[top-1]
			stName := stack[top-2]

			if e.flags.StCallback != nil {
				name, _ := stName.ReadString()
				e.flags.StCallback("mod", name, &stVal, code.ValueStr, code.ValueAny.(string))
			}
			continue
		case TypePushComputed:
			val := VMValueNewComputedRaw(&ComputedData{
				Expr: code.ValueStr,
			})
			stack[top].Value = val.Value
			stack[top].TypeID = VMTypeComputedValue
			top++
			continue

		case TypeConvertInt:
			e.Calculated = true
			val := stack[top-1]
			if val.TypeID == VMTypeString {
				v, _ := val.ReadString()
				r, err := strconv.ParseInt(v, 10, 64)
				if err == nil {
					stack[top-1] = VMValue{VMTypeInt64, r, 0}
				} else {
					return nil, "", errors.New("错误: int() 无法转换为数字: " + v)
				}
				continue
			} else if val.TypeID == VMTypeInt64 {
				// 啥都不做
				continue
			} else {
				return nil, "", errors.New("错误: int() 只能对字符串使用")
			}
		case TypeConvertStr:
			e.Calculated = true
			val := stack[top-1]
			stack[top-1] = VMValue{VMTypeString, val.ToString(), 0}
			continue

		case TypeLoadVarname, TypeLoadVarnameForThis:
			var v interface{}
			var vType VMValueType
			var lastDetail string

			varname := code.ValueStr
			// 如果变量名以_开头，那么忽略所有的_
			for {
				if strings.HasPrefix(varname, "_") {
					varname = varname[len("_"):]
				} else {
					break
				}
			}

			if e.flags.DisableLoadVarname {
				return nil, calcDetail, errors.New("解析失败")
			}

			if e.flags.DNDAttrReadDC {
				// 额外调整值补正，用于检定
				if ctx.SystemTemplate != nil && strings.HasSuffix(varname, "豁免") {
					// NOTE: 1.4.4 版本新增，此处逻辑是为了使 "XX豁免" 中的 XX 能被同义词所替换
					name := strings.TrimSuffix(varname, "豁免")
					name = ctx.SystemTemplate.GetAlias(name)
					varname = name + "豁免"
				}

				switch varname {
				case "力量豁免", "敏捷豁免", "体质豁免", "智力豁免", "感知豁免", "魅力豁免":
					vName := strings.TrimSuffix(varname, "豁免")
					realV, _, err := ctx.Dice._ExprEvalBaseV1(fmt.Sprintf("$豁免_%s", vName), ctx, RollExtraFlags{})
					if err == nil {
						vType = realV.TypeID
						v = realV.Value
					}
				}
			}

			if e.flags.CocVarNumberMode {
				re := regexp.MustCompile(`^(困难|极难|大成功|常规|失败|困難|極難|常規|失敗)?([^\d]+)(\d+)?$`)
				m := re.FindStringSubmatch(code.ValueStr)
				if len(m) > 0 {
					if m[1] != "" {
						e.CocFlagVarPrefix = chsS2T.Read(m[1])
						varname = varname[len(m[1]):]
					}

					// 有末值时覆盖，有初值时
					if m[3] != "" {
						vType = VMTypeInt64
						v, _ = strconv.ParseInt(m[3], 10, 64)
					}
				}
			}

			if v == nil && ctx != nil { //nolint:nestif
				name2 := varname
				if ctx.SystemTemplate != nil {
					name2 = ctx.SystemTemplate.GetAlias(varname)
				}
				v2, exists := _VarGetValueV1(ctx, name2)

				if !exists {
					if ctx.SystemTemplate != nil {
						v2n, detail, calculated, exists2 := ctx.SystemTemplate.GetDefaultValueEx0V1(ctx, varname)
						if exists2 {
							v2 = dsValueToRollVMv1(v2n)
							if calculated && !e.Calculated {
								e.Calculated = calculated
							}
							lastDetail = detail
							exists = true
						}
					}
				}

				if exists {
					vType = v2.TypeID
					v = v2.Value
				} else {
					textTmpl := ctx.Dice.TextMap[varname]
					if textTmpl != nil {
						vType = VMTypeString
						v = DiceFormat(ctx, textTmpl.Pick().(string))
					} else if strings.Contains(varname, ":") {
						vType = VMTypeString
						v = "<%未定义值-" + varname + "%>"
					}
				}
			}

			if vType == VMTypeDNDComputedValue {
				// 解包计算属性
				vd := v.(*VMDndComputedValueData)
				_VarSetValueV1(ctx, "$tVal", &vd.BaseValue)
				realV, _, err := ctx.Dice._ExprEvalBaseV1(vd.Expr, ctx, RollExtraFlags{vmDepth: e.flags.vmDepth + 1})
				if err != nil {
					return nil, "", errors.New("E3: 获取计算属性异常: " + vd.Expr)
				}
				vType = realV.TypeID
				v = realV.Value
			}

			if vType == VMTypeComputedValue {
				// 解包计算属性
				v2 := VMValue{vType, v, 0}
				ret, detail, err := v2.ComputedExecute(ctx, e.flags.vmDepth)
				if err != nil {
					return nil, "", err
					// errors.New("E3: 获取计算属性异常: " + err.Error())
				}
				if ret == nil {
					cd, _ := v2.ReadComputed()
					return nil, "", errors.New("E3: 获取计算属性异常: " + cd.Expr)
				}
				calculated := ret.Parser.Calculated
				if calculated && !e.Calculated {
					e.Calculated = calculated
				}

				lastDetail = detail
				vType = ret.TypeID
				v = ret.Value
			}

			if vType == VMTypeInt64 && lastDetail == "" {
				lastDetail = fmt.Sprintf("%d", v)
			}

			if !e.flags.DisableValueBuff {
				_, exists := _VarGetValueV1(ctx, "$buff_"+varname)
				if exists {
					if vType == VMTypeInt64 {
						buffV, _, err := ctx.Dice._ExprEvalBaseV1("$buff_"+varname, ctx, RollExtraFlags{})
						if err == nil {
							if buffV.TypeID == VMTypeInt64 {
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
							// lastDetail = varname + lastDetail
						}
					}
				}
			}

			if v == nil {
				if ctx.SystemTemplate != nil {
					_v2, detail, calculated, _ := ctx.SystemTemplate.GetDefaultValueEx0V1(ctx, varname)
					v2 := dsValueToRollVMv1(_v2)
					if calculated && !e.Calculated {
						e.Calculated = calculated
					}
					lastDetail = detail

					if v2 != nil {
						vType = v2.TypeID
						v = v2.Value
						lastDetail = v2.ToString()
					}
				}
			}
			if v == nil {
				// 默认值为0
				vType = VMTypeInt64
				v = int64(0)
				lastDetail = "0"
			}

			if !detailFlag {
				lastDetail = fmt.Sprintf("%s=%s", varname, lastDetail)
			}

			stack[top].TypeID = vType
			stack[top].Value = v
			top++

			if vType == VMTypeInt64 {
				lastDetails = append(lastDetails, lastDetail)
			}
			continue
		case TypeNegation:
			a := &stack[top-1]
			a.Value = -a.Value.(int64)
			e.Calculated = true
			continue
		case TypeDiceUnary:
			a := &stack[top-1]
			// Dice XXX, 如 d100
			a.Value = DiceRoll64x(ctx._v1Rand, a.Value.(int64))
			e.Calculated = true
			continue
		case TypeHalt:
			if len(lastDetails) > 0 {
				calcDetail += fmt.Sprintf("[%s]", strings.Join(lastDetails, ","))
				lastDetails = lastDetails[:0]
			}
			continue
		default: /*no-op*/
		}

		a, b := &stack[top-2], &stack[top-1]
		// lastValIndex = top-3
		top--

		checkDice := func(t *ByteCode) {
			// 第一次 左然后右
			// 后 一直右
			times++
			if a.TypeID != VMTypeInt64 || b.TypeID != VMTypeInt64 {
				return
			}

			checkLeft := func() {
				if calcDetail == "" {
					if a.TypeID == VMTypeNone {
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
					if t.T != TypeDiceFate {
						if len(lastDetailsLeft) > 0 {
							// .r 3c2+1
							// .r b2+1
							vLeft := "[" + strings.Join(lastDetailsLeft, ",") + "]"
							calcDetail += fmt.Sprintf("%d%s %s %d", a.Value.(int64), vLeft, t.String(), b.Value.(int64))
						} else {
							calcDetail += fmt.Sprintf("%d %s %d", a.Value.(int64), t.String(), b.Value.(int64))
						}
					} /* else {
						如果什么都不输出，反而正常了……不然会这样：
						<木落>掷出了 f + 1=0 df 0[0-++] + 1=2
					} */
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
		if a.TypeID == 0 {
			aInt = a.Value.(int64)
		}
		if b.TypeID == 0 {
			bInt = b.Value.(int64)
		}

		boolToInt64 := func(val bool) int64 {
			if val {
				return 1
			}
			return 0
		}

		e4check := func() error {
			if a.TypeID != b.TypeID {
				return errors.New("E4:符号运算类型不匹配")
			}
			return nil
		}

		// 二目运算符
		switch code.T {
		case TypeCompLT:
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.Value = boolToInt64(aInt < bInt)
		case TypeCompLE:
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.Value = boolToInt64(aInt <= bInt)
		case TypeCompEQ:
			checkDice(&code)
			if a.TypeID != b.TypeID {
				a.TypeID = VMTypeInt64
				a.Value = int64(0)
			} else {
				a.TypeID = VMTypeInt64
				a.Value = boolToInt64(a.Value == b.Value)
			}
		case TypeCompNE:
			checkDice(&code)
			if a.TypeID != b.TypeID {
				a.TypeID = VMTypeInt64
				a.Value = int64(1)
			} else {
				a.TypeID = VMTypeInt64
				a.Value = boolToInt64(a.Value != b.Value)
			}
			// a.Value = boolToInt64(aInt != bInt)
		case TypeCompGT:
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.TypeID = VMTypeInt64
			a.Value = boolToInt64(aInt > bInt)
		case TypeCompGE:
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.TypeID = VMTypeInt64
			a.Value = boolToInt64(aInt >= bInt)
		case TypeBitwiseAnd:
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.TypeID = VMTypeInt64
			a.Value = aInt & bInt
		case TypeBitwiseOr:
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.TypeID = VMTypeInt64
			a.Value = aInt | bInt
		case TypeAdd:
			e.Calculated = true
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			if a.TypeID == VMTypeString {
				a.Value = a.Value.(string) + b.Value.(string)
			} else {
				a.Value = aInt + bInt
			}
		case TypeSubtract:
			e.Calculated = true
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.Value = aInt - bInt
		case TypeMultiply:
			e.Calculated = true
			if a.TypeID != b.TypeID {
				return nil, "", errors.New("E4:符号运算类型不匹配")
			}
			checkDice(&code)
			a.Value = aInt * bInt
		case TypeDivide:
			e.Calculated = true
			if err := e4check(); err != nil {
				return nil, "", err
			}
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
			e.Calculated = true
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			if bInt == 0 {
				return nil, "", errors.New("E2:被除数为0")
			}
			a.Value = aInt % bInt
		case TypeExponentiation:
			e.Calculated = true
			if err := e4check(); err != nil {
				return nil, "", err
			}
			checkDice(&code)
			a.Value = int64(math.Pow(float64(aInt), float64(bInt)))
		case TypeSwap:
			a.Value, b.Value = bInt, aInt
			top++
		case TypeStore:
			e.Calculated = true
			top--
			if ctx != nil {
				_VarSetValueV1(ctx, a.Value.(string), b)
				// p.SetValueInt64(a.value.(string), b.value.(int64), nil)
			}
			stack[top].TypeID = b.TypeID
			stack[top].Value = b.Value
			top++
			continue
		case TypeDiceFate:
			e.Calculated = true
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
			a.TypeID = VMTypeInt64
			a.Value = sum
		case TypeDice:
			e.Calculated = true
			checkDice(&code)
			if bInt == 0 {
				bInt = e.flags.DefaultDiceSideNum
				if bInt == 0 {
					bInt = 100
				}
			}

			if numOpCountAdd(aInt) {
				return nil, "", getE5()
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
				for i := int64(0); i < aInt; i++ {
					if e.flags.BigFailDiceOn {
						nums = append(nums, bInt)
					} else {
						nums = append(nums, DiceRoll64x(ctx._v1Rand, bInt))
					}
				}

				if isDiceK {
					sort.Slice(nums, func(i, j int) bool { return nums[i] > nums[j] })
				} else {
					sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
				}

				num := int64(0)
				for i := int64(0); i < diceKQ; i++ {
					// 当取数大于上限 跳过
					if i >= int64(len(nums)) {
						continue
					}
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
				for i := int64(0); i < aInt; i++ {
					var curNum int64
					if e.flags.BigFailDiceOn {
						curNum = bInt
					} else {
						curNum = DiceRoll64x(ctx._v1Rand, bInt)
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
		default: /*no-op*/
		}
	}

	if len(calcDetail) > 500 {
		calcDetail = "[略]"
	}

	return &stack[0], calcDetail, nil
}

func DiceDCRollVM(randSrc *rand2.PCGSource, e *RollExpression, addLine *VMValue, pool *VMValue, points *VMValue) (*VMValue, int64, int64, []string) { //nolint:revive
	makeE6 := func() {
		e.Error = errors.New("E6: 类型错误")
	}

	if addLine.TypeID != VMTypeInt64 {
		makeE6()
	}
	if pool.TypeID != VMTypeInt64 {
		makeE6()
	}
	if points.TypeID != VMTypeInt64 {
		makeE6()
	}
	if e.Error != nil {
		return nil, 0, 0, nil
	}

	var valPool, valAddLine, valPoints int64
	if valPool, _ = pool.ReadInt64(); valPool < 1 || valPool > 20000 {
		e.Error = errors.New("E7: 非法数值, 骰池范围是1到20000")
		return nil, 0, 0, nil
	}

	if valAddLine, _ = addLine.ReadInt64(); valAddLine < 2 {
		e.Error = errors.New("E7: 非法数值, 加骰线必须大于等于2")
		return nil, 0, 0, nil
	}

	if valPoints, _ = points.ReadInt64(); valPoints < 1 || valPoints > 2000 {
		e.Error = errors.New("E7: 非法数值, 面数至少为1，最多为2000")
		return nil, 0, 0, nil
	}

	ret1, ret2, ret3, details := DiceDCRoll(randSrc, valAddLine, valPool, valPoints)
	return &VMValue{TypeID: VMTypeInt64, Value: ret1}, ret2, ret3, details
}

func DiceDCRoll(randSrc *rand2.PCGSource, addLine int64, pool int64, points int64) (int64, int64, int64, []string) { //nolint:revive
	var details []string
	addTimes := 1

	isShowDetails := pool < 15
	allRollCount := pool
	resultDice := int64(0)

	for times := 0; times < addTimes; times++ {
		addCount := int64(0)
		var detailsOne []string
		maxDice := int64(0)

		for i := int64(0); i < pool; i++ {
			one := DiceRoll64x(randSrc, points)
			if one > maxDice {
				maxDice = one
			}
			reachAddRound := one >= addLine

			if reachAddRound {
				addCount++
				maxDice = 10
			}

			if isShowDetails {
				baseText := strconv.FormatInt(one, 10)
				if reachAddRound {
					baseText = "<" + baseText + ">"
				}
				detailsOne = append(detailsOne, baseText)
			}
		}

		resultDice += maxDice
		allRollCount += addCount

		// 有加骰，再骰一次
		if addCount > 0 {
			addTimes++
			pool = addCount
		}

		if allRollCount > 100 {
			// 多于100，清空
			isShowDetails = false
			details = details[:0]
		}

		if isShowDetails {
			details = append(details, "{"+strings.Join(detailsOne, ",")+"}")
		}
	}

	// 成功数，总骰数，轮数，细节
	return resultDice, allRollCount, int64(addTimes), details
}

func DiceWodRoll(randSrc *rand2.PCGSource, addLine int64, pool int64, points int64, threshold int64, isGE bool) (int64, int64, int64, []string) { //nolint:revive
	var details []string
	addTimes := 1

	isShowDetails := pool < 15
	allRollCount := pool
	successCount := int64(0)

	for times := 0; times < addTimes; times++ {
		addCount := int64(0)
		var detailsOne []string

		for i := int64(0); i < pool; i++ {
			var reachSuccess bool
			var reachAddRound bool
			one := DiceRoll64x(randSrc, points)

			if addLine != 0 {
				reachAddRound = one >= addLine
			}

			if isGE {
				reachSuccess = one >= threshold
			} else {
				reachSuccess = one <= threshold
			}

			if reachSuccess {
				successCount++
			}
			if reachAddRound {
				addCount++
			}

			if isShowDetails {
				baseText := strconv.FormatInt(one, 10)
				if reachSuccess {
					baseText += "*"
				}
				if reachAddRound {
					baseText = "<" + baseText + ">"
				}
				detailsOne = append(detailsOne, baseText)
			}
		}

		allRollCount += addCount
		// 有加骰，再骰一次
		if addCount > 0 {
			addTimes++
			pool = addCount
		}

		if allRollCount > 100 {
			// 多于100，清空
			isShowDetails = false
			details = details[:0]
		}

		if isShowDetails {
			details = append(details, "{"+strings.Join(detailsOne, ",")+"}")
		}
	}

	// 成功数，总骰数，轮数，细节
	return successCount, allRollCount, int64(addTimes), details
}

func DiceWodRollVM(randSrc *rand2.PCGSource, e *RollExpression, addLine *VMStack, pool *VMValue, points *VMValue, threshold *VMValue, isGE bool) (*VMValue, int64, int64, []string) { //nolint:revive
	makeE6 := func() {
		e.Error = errors.New("E6: 类型错误")
	}

	if addLine.TypeID != VMTypeInt64 {
		makeE6()
	}
	if pool.TypeID != VMTypeInt64 {
		makeE6()
	}
	if points.TypeID != VMTypeInt64 {
		makeE6()
	}
	if threshold.TypeID != VMTypeInt64 {
		makeE6()
	}
	if e.Error != nil {
		return nil, 0, 0, nil
	}

	var valPool, valAddLine, valPoints, valThreshold int64
	if valPool, _ = pool.ReadInt64(); valPool < 1 || valPool > 20000 {
		e.Error = errors.New("E7: 非法数值, 骰池范围是1到20000")
		return nil, 0, 0, nil
	}

	if valAddLine, _ = addLine.ReadInt64(); valAddLine != 0 && valAddLine < 2 {
		e.Error = errors.New("E7: 非法数值, 加骰线必须为0[不加骰]，或≥2")
		return nil, 0, 0, nil
	}

	if valPoints, _ = points.ReadInt64(); valPoints < 1 {
		e.Error = errors.New("E7: 非法数值, 面数至少为1")
		return nil, 0, 0, nil
	}

	if valThreshold, _ = threshold.ReadInt64(); valThreshold < 1 {
		e.Error = errors.New("E7: 非法数值, 成功线至少为1")
		return nil, 0, 0, nil
	}

	ret1, ret2, ret3, details := DiceWodRoll(randSrc, valAddLine, valPool, valPoints, valThreshold, isGE)
	return &VMValue{TypeID: VMTypeInt64, Value: ret1}, ret2, ret3, details
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
			ret += "@raw: " + strconv.FormatInt(int64(i.T), 10) + "\n"
		}
	}
	ret += "=== VM Code End===\n"
	return ret
}
