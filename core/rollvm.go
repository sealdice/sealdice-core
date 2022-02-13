package main

import (
	"fmt"
	"math/big"
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
	TypeHalt
	TypeSwap
)

type ByteCode struct {
	T     Type
	Value *big.Int
	ValueStr string
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

func (e *RollExpression) Evaluate(d *Dice, p *PlayerInfo) (*big.Int, string, error) {
	stack, top := make([]big.Int, len(e.Code)), 0
	//lastIsDice := false
	//var lastValIndex int
	times := 0
	lastDetails := []string{}
	calcDetail := ""


	for _, code := range e.Code[0:e.Top] {
		// 单目运算符
		switch code.T {
		case TypeNumber:
			stack[top].Set(code.Value)
			top++
			continue
		case TypeLoadVarname:
			var v int64
			if p != nil {
				var exists bool
				v, exists = p.ValueNumMap[code.ValueStr]
				if !exists {
					// TODO: 找不到时的处理
				}
			}
			stack[top].Set(big.NewInt(v))
			top++

			lastDetail := fmt.Sprintf("%s=%d", code.ValueStr, v)
			lastDetails = append(lastDetails, lastDetail)
			continue
		case TypeNegation:
			a := &stack[top-1]
			a.Neg(a)
			continue
		case TypeDiceUnary:
			a := &stack[top-1]
			// dice XXX, 如 d100
			a.SetInt64(DiceRoll64(a.Int64()))
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
					calcDetail += a.String()

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

		// 二目运算符
		switch code.T {
		case TypeAdd:
			checkDice(&code)
			a.Add(a, b)
		case TypeSubtract:
			checkDice(&code)
			a.Sub(a, b)
		case TypeMultiply:
			checkDice(&code)
			a.Mul(a, b)
		case TypeDivide:
			checkDice(&code)
			a.Div(a, b)
		case TypeModulus:
			checkDice(&code)
			a.Mod(a, b)
		case TypeExponentiation:
			checkDice(&code)
			a.Exp(a, b, nil)
		case TypeSwap:
			tmp := big.NewInt(0)
			tmp.Set(a)
			a.Set(b)
			b.Set(tmp)
			top++
		case TypeDice:
			checkDice(&code)
			// XXX dice YYY, 如 3d100
			var num int64
			for i := int64(0); i < a.Int64(); i+=1 {
				num += DiceRoll64(b.Int64())
			}

			lastDetail := fmt.Sprintf("%dd%d=%d", a.Int64(), b.Int64(), num)
			lastDetails = append(lastDetails, lastDetail)
			a.SetInt64(num)
		}
	}

	return &stack[0], calcDetail, nil
}
