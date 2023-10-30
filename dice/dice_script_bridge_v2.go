package dice

import (
	"fmt"
	ds "github.com/sealdice/dicescript"
)

type VMResultV2 struct {
	*ds.VMValue
	vm     *ds.Context
	legacy *VMResult
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
	return true
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

func (r *VMResultV2) GetVersion() int64 {
	if r.legacy != nil {
		return 1
	}
	return 2
}

// 不建议用，纯兼容旧版
func DiceExprEvalBase(ctx *MsgContext, s string, flags RollExtraFlags) (*VMResultV2, string, error) {
	ctx.CreateVmIfNotExists()
	ctx.vm.Ret = nil
	ctx.vm.Error = nil

	s = CompatibleReplace(ctx, s)

	err := ctx.vm.Run(s)
	if err != nil || ctx.vm.Ret == nil {
		fmt.Println("脚本执行出错V2: ", s, "->", err)

		// 尝试一下V1
		val, detail, err := ctx.Dice.ExprEvalBase(s, ctx, flags)
		if err != nil {
			return nil, detail, err
		}

		return &VMResultV2{val.ConvertToDiceScriptValue(), ctx.vm, val}, detail, err
	} else {
		return &VMResultV2{ctx.vm.Ret, ctx.vm, nil}, ctx.vm.Detail, nil
	}
}

// 不建议用，纯兼容旧版
func DiceExprTextBase(ctx *MsgContext, s string, flags RollExtraFlags) (*VMResultV2, string, error) {
	return DiceExprEvalBase(ctx, "\x1e"+s+"\x1e", flags)
}

func (ctx *MsgContext) CreateVmIfNotExists() {
	if ctx.vm == nil {
		// 初始化骰子
		ctx.vm = ds.NewVM()

		// 根据当前规则开语法 - 暂时是都开
		ctx.vm.Flags.EnableDiceWoD = true
		ctx.vm.Flags.EnableDiceCoC = true
		ctx.vm.Flags.EnableDiceFate = true
		ctx.vm.Flags.EnableDiceDoubleCross = true

		// 设置默认骰子面数
		ctx.vm.Flags.DefaultDiceSideExpr = fmt.Sprint("d%d", ctx.Group.DiceSideNum)
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
		fmt.Println("脚本执行出错: ", s, "->", err)
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
