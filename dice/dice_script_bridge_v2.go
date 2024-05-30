package dice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

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

// 不建议用，纯兼容旧版
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
						//fmt.Println("COC值:", name, cocFlagVarPrefix)
						return name, ds.VMValueNewInt(ds.IntType(v))
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
		return &VMResultV2{ctx.vm.Ret, ctx.vm, nil, cocFlagVarPrefix}, ctx.vm.Detail, nil
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
		ctx.vm.Config.EnableDiceWoD = true
		ctx.vm.Config.EnableDiceCoC = true
		ctx.vm.Config.EnableDiceFate = true
		ctx.vm.Config.EnableDiceDoubleCross = true

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
