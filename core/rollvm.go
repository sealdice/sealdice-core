package main

import (
	"math/big"
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
)

type ByteCode struct {
	T     Type
	Value *big.Int
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

func (e *RollExpression) AddValue(value string) {
	// 实质上的压栈命令
	code, top := e.Code, e.Top
	e.Top++
	code[top].Value = new(big.Int)
	code[top].Value.SetString(value, 10)
}

func (e *RollExpression) Evaluate() *big.Int {
	stack, top := make([]big.Int, len(e.Code)), 0
	for _, code := range e.Code[0:e.Top] {
		// 单目运算符
		switch code.T {
		case TypeNumber:
			stack[top].Set(code.Value)
			top++
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
		}

		a, b := &stack[top-2], &stack[top-1]
		top--

		// 二目运算符
		switch code.T {
		case TypeAdd:
			a.Add(a, b)
		case TypeSubtract:
			a.Sub(a, b)
		case TypeMultiply:
			a.Mul(a, b)
		case TypeDivide:
			a.Div(a, b)
		case TypeModulus:
			a.Mod(a, b)
		case TypeExponentiation:
			a.Exp(a, b, nil)
		case TypeDice:
			// XXX dice YYY, 如 3d100
			var num int64
			for i := int64(0); i < a.Int64(); i+=1 {
				num += DiceRoll64(b.Int64())
			}
			a.SetInt64(num)
		}
	}
	return &stack[0]
}
