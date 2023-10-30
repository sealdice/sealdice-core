package dice

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
		//遇到异常，尝试一下V1
		return DiceFormatV1(ctx, s)
	}
	return ret
}

func DiceFormatTmpl(ctx *MsgContext, s string) string {
	ret, err := DiceFormatTmplV2(ctx, s)
	if err != nil {
		//遇到异常，尝试一下V1
		return DiceFormatTmplV1(ctx, s)
	}
	return ret
}
