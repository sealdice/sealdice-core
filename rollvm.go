package main

import (
	"errors"
	"fmt"
	"math"
	"regexp"
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
	TypeLoadVarname
	TypeLoadFormatString
	TypeStore
	TypeHalt
	TypeSwap
	TypeLeftValueMark
)

type ByteCode struct {
	T     Type
	Value int64
	ValueStr string
	ValueAny interface{}
}

func (code *ByteCode) String() string {
	switch code.T {
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
	case TypeDiceUnary:
		return "dice1"
	case TypeLoadVarname:
		return "ld.v " + code.ValueStr
	case TypeLoadFormatString:
		return "ld.fs"
	case TypeStore:
		return "store"
	case TypeHalt:
		return "halt"
	case TypeSwap:
		return "swap"
	case TypeLeftValueMark:
		return "mark.left"
	}
	return ""
}

type RollExpression struct {
	Code []ByteCode
	Top  int
	BigFailDiceOn bool
	error error
}

func (e *RollExpression) Init(stackLength int) {
	e.Code = make([]ByteCode, stackLength)
}

func (e *RollExpression) checkStackOverflow() bool {
	if e.error != nil {
		return true
	}
	if e.Top >= len(e.Code) {
		e.error = errors.New("E1:指令虚拟机栈溢出，请不要发送过于离谱的指令")
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


func (e *RollExpression) AddOperator(operator Type) {
	code, top := e.Code, e.Top
	if e.checkStackOverflow() {
		return
	}
	e.Top++
	code[top].T = operator
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

type vmStack struct {
	typeId int
	value interface{}
}

type vmResult struct {
	vmStack
	parser *DiceRollParser
}

func (e *RollExpression) Evaluate(d *Dice, p *PlayerInfo) (*vmStack, string, error) {
	stack, top := make([]vmStack, len(e.Code)), 0
	//lastIsDice := false
	//var lastValIndex int
	times := 0
	lastDetails := []string{}
	lastDetailsLeft := []string{}
	calcDetail := ""

	for _, code := range e.Code[0:e.Top] {
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
				str = strings.Replace(str, i, strconv.FormatInt(stack[top-len(parts)+index].value.(int64), 10), 1)
			}

			top -= len(parts)
			stack[top].typeId = 1
			stack[top].value = str
			top++
			continue
		case TypeNumber:
			stack[top].typeId = 0
			stack[top].value = code.Value
			top++
			continue
		case TypePushString:
			stack[top].typeId = 1
			stack[top].value = code.ValueStr
			top++
			continue
		case TypeLoadVarname:
			var v int64
			if p != nil {
				var exists bool
				v, exists = p.GetValueInt64(code.ValueStr, nil)
				if !exists {
					// TODO: 找不到时的处理
				}
			}

			stack[top].typeId = 0
			stack[top].value = v
			top++

			lastDetail := fmt.Sprintf("%s=%d", code.ValueStr, v)
			lastDetails = append(lastDetails, lastDetail)
			continue
		case TypeNegation:
			a := &stack[top-1]
			a.value = -a.value.(int64)
			continue
		case TypeDiceUnary:
			a := &stack[top-1]
			// dice XXX, 如 d100
			a.value = DiceRoll64(a.value.(int64))
			continue
		case TypeHalt:
			continue
		}

		a, b := &stack[top-2], &stack[top-1]
		//lastValIndex = top-3
		top--

		checkDice := func (t *ByteCode) {
			// 第一次 左然后右
			// 后 一直右
			times += 1

			checkLeft := func () {
				if calcDetail == "" {
					calcDetail += strconv.FormatInt(a.value.(int64), 10)
				}

				if len(lastDetailsLeft) > 0 {
					vLeft := "[" + strings.Join(lastDetailsLeft, ",") + "]"
					calcDetail += vLeft
				}
			}

			if t.T != TypeDice && top == 1 {
				if times == 1 {
					calcDetail += fmt.Sprintf("%d %s %d", a.value.(int64), t.String(), b.value.(int64))
				} else {
					checkLeft()
					calcDetail += fmt.Sprintf(" %s %d", t.String(), b.value.(int64))

					if len(lastDetails) > 0 {
						calcDetail += fmt.Sprintf("[%s]", strings.Join(lastDetails, ","))
						lastDetails = lastDetails[:0]
					}
				}
			}
		}

		var aInt, bInt int64
		if a.typeId == 0 {
			aInt = a.value.(int64)
		}
		if b.typeId == 0 {
			bInt = b.value.(int64)
		}

		// 二目运算符
		switch code.T {
		case TypeAdd:
			checkDice(&code)
			a.value = aInt + bInt
		case TypeSubtract:
			checkDice(&code)
			a.value = aInt - bInt
		case TypeMultiply:
			checkDice(&code)
			a.value = aInt * bInt
		case TypeDivide:
			checkDice(&code)
			a.value = aInt / bInt
		case TypeModulus:
			checkDice(&code)
			a.value = aInt % bInt
		case TypeExponentiation:
			checkDice(&code)
			a.value = int64(math.Pow(float64(aInt), float64(bInt)))
		case TypeSwap:
			a.value, b.value = bInt, aInt
			top++
		case TypeStore:
			fmt.Println(e.GetAsmText())
			top--
			if p != nil {
				p.SetValueInt64(a.value.(string), b.value.(int64), nil)
			}
			stack[top].typeId = b.typeId
			stack[top].value = b.value
			top++
			continue
		case TypeDice:
			checkDice(&code)
			// XXX dice YYY, 如 3d100
			var num int64
			for i := int64(0); i < aInt; i+=1 {
				if e.BigFailDiceOn {
					num += bInt
				} else {
					num += DiceRoll64(bInt)
				}
			}

			lastDetail := fmt.Sprintf("%dd%d=%d", aInt, bInt, num)
			lastDetails = append(lastDetails, lastDetail)
			a.value = num
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
