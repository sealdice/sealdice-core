package dice

import (
	randv2 "math/rand/v2"

	ds "github.com/sealdice/dicescript"
)

func (ctx *MsgContext) getDiceSource() ds.DiceSource {
	if ctx.diceRandSrc != nil {
		ctx._v1Rand = ctx.diceRandSrc
		return ctx.diceRandSrc
	}
	// 默认随机源直接走进程内唯一的全局 owner。
	ctx._v1Rand = globalRandSource
	return globalRandSource
}

func (ctx *MsgContext) getChooserRand() *randv2.Rand {
	src := normalizeDiceSource(ctx.getDiceSource())
	if ctx.chooserRand == nil || ctx.chooserSrc != src {
		ctx.chooserSrc = src
		ctx.chooserRand = randv2.New(src)
	}
	return ctx.chooserRand
}

func (ctx *MsgContext) Roll(points int) int {
	if points <= 0 {
		return 0
	}
	return int(ds.Roll(ctx.getDiceSource(), ds.IntType(points), 0))
}

func (ctx *MsgContext) Roll64(points int64) int64 {
	return DiceRoll64x(ctx.getDiceSource(), points)
}

func (ctx *MsgContext) RandIntn(n int) int {
	return randIntnFromSource(ctx.getDiceSource(), n)
}

func (ctx *MsgContext) Shuffle(n int, swap func(i, j int)) {
	shuffleWithSource(ctx.getDiceSource(), n, swap)
}
