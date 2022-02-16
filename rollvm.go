package main

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

type Type uint8

const (
	TypeNumber Type = iota
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
	TypeHalt
	TypeSwap
)

type ByteCode struct {
	T     Type
	Value *big.Int
	ValueStr string
	ValueAny interface{}
}

func (code *ByteCode) String() string {
	switch code.T {
	case TypeNumber:
		return code.Value.String()
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
		return "^"
	case TypeDice:
		return "d"
	case TypeDiceUnary:
		return "d"
	case TypeLoadVarname:
		return "ldv"
	}
	return ""
}

type RollExpression struct {
	Code []ByteCode
	Top  int
	BigFailDiceOn bool
}

func (e *RollExpression) Init(stackLength int) {
	e.Code = make([]ByteCode, stackLength)
}

func (e *RollExpression) AddOperator(operator Type) {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = operator
}

func (e *RollExpression) AddLoadVarname(value string) {
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeLoadVarname
	code[top].ValueStr = value
}

func (e *RollExpression) AddValue(value string) {
	// 实质上的压栈命令
	code, top := e.Code, e.Top
	e.Top++
	code[top].Value = new(big.Int)
	code[top].Value.SetString(value, 10)
}

func (e *RollExpression) AddFormatString(value string) {
	// 载入一个字符串并格式化
	code, top := e.Code, e.Top
	e.Top++
	code[top].T = TypeLoadFormatString
	code[top].Value = big.NewInt(1)

	re := regexp.MustCompile(`\{[^}]*?\}`)
	code[top].ValueStr = value
	code[top].ValueAny = re.FindAllString(value, -1)
}

type vmStack struct {
	typeId int
	value interface{}
}

func (e *RollExpression) Evaluate(d *Dice, p *PlayerInfo) (*vmStack, string, error) {
	stack, top := make([]vmStack, len(e.Code)), 0
	//lastIsDice := false
	//var lastValIndex int
	times := 0
	lastDetails := []string{}
	calcDetail := ""


	for _, code := range e.Code[0:e.Top] {
		// 单目运算符
		switch code.T {
		case TypeLoadFormatString:
			parts := code.ValueAny.([]string)
			str := code.ValueStr

			for index, i := range parts {
				str = strings.Replace(str, i, stack[top-len(parts)+index].value.(*big.Int).String(), 1)
			}

			top -= len(parts)
			stack[top].typeId = 1
			stack[top].value = str
			top++
			continue
		case TypeNumber:
			stack[top].typeId = 0
			stack[top].value = &big.Int{}
			stack[top].value.(*big.Int).Set(code.Value)
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
			stack[top].value = &big.Int{}
			stack[top].value.(*big.Int).SetInt64(v)
			top++

			lastDetail := fmt.Sprintf("%s=%d", code.ValueStr, v)
			lastDetails = append(lastDetails, lastDetail)
			continue
		case TypeNegation:
			a := &stack[top-1]
			a.value.(*big.Int).Neg(a.value.(*big.Int))
			continue
		case TypeDiceUnary:
			a := &stack[top-1]
			// dice XXX, 如 d100
			a.value.(*big.Int).SetInt64(DiceRoll64(a.value.(*big.Int).Int64()))
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
					calcDetail += a.value.(*big.Int).String()

					if len(lastDetails) > 0 {
						calcDetail += fmt.Sprintf("[%s]", strings.Join(lastDetails, ","))
						lastDetails = lastDetails[:0]
					}
				}
			}

			if t.T != TypeDice && top == 1 {
				if times == 1 {
					calcDetail += fmt.Sprintf("%d %s %d", a, t.String(), b)
				} else {
					checkLeft()
					calcDetail += fmt.Sprintf(" %s %d", t.String(), b)

					if len(lastDetails) > 0 {
						calcDetail += fmt.Sprintf("[%s]", strings.Join(lastDetails, ","))
						lastDetails = lastDetails[:0]
					}
				}
			}
		}

		aInt := a.value.(*big.Int)
		bInt := b.value.(*big.Int)

		// 二目运算符
		switch code.T {
		case TypeAdd:
			checkDice(&code)
			// a.value.(*big.Int)
			aInt.Add(aInt, bInt)
		case TypeSubtract:
			checkDice(&code)
			aInt.Sub(aInt, bInt)
		case TypeMultiply:
			checkDice(&code)
			aInt.Mul(aInt, bInt)
		case TypeDivide:
			checkDice(&code)
			aInt.Div(aInt, bInt)
		case TypeModulus:
			checkDice(&code)
			aInt.Mod(aInt, bInt)
		case TypeExponentiation:
			checkDice(&code)
			aInt.Exp(aInt, bInt, nil)
		case TypeSwap:
			tmp := big.NewInt(0)
			tmp.Set(aInt)
			aInt.Set(bInt)
			bInt.Set(tmp)
			top++
		case TypeDice:
			checkDice(&code)
			// XXX dice YYY, 如 3d100
			var num int64
			for i := int64(0); i < aInt.Int64(); i+=1 {
				if e.BigFailDiceOn {
					num += bInt.Int64()
				} else {
					num += DiceRoll64(bInt.Int64())
				}
			}

			lastDetail := fmt.Sprintf("%dd%d=%d", aInt.Int64(), bInt.Int64(), num)
			lastDetails = append(lastDetails, lastDetail)
			aInt.SetInt64(num)
		}
	}

	return &stack[0], calcDetail, nil
}
