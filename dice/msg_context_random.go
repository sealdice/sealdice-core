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

// ShallowCopyWithIsolatedRandomSource 返回浅拷贝，并换用独立初始化的 PCG 随机源。
// 求值会推进随机源状态：diceRandSrc 为 nil 时共享的是全局源，非 nil 时共享的是
// 原上下文的源，两者都会改变后续骰点结果。旁路求值（如跑团日志昵称）必须用本方法。
func (ctx *MsgContext) ShallowCopyWithIsolatedRandomSource() *MsgContext {
	copied := ctx.ShallowCopy()
	copied.vm = nil
	copied._v1Rand = nil
	copied.diceRandSrc = newPCGDiceSource(randv2.Uint64())
	copied.chooserRand = nil
	copied.chooserSrc = nil
	return copied
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
