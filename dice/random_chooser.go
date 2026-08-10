package dice

import (
	randv2 "math/rand/v2"

	wr "github.com/mroth/weightedrand/v3"
	ds "github.com/sealdice/dicescript"

	randcore "sealdice-core/utils/random"
)

type chooserWeight interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func newPCGDiceSource(seed uint64) ds.StatefulDiceSource {
	return randcore.NewPCGSource(seed)
}

func pickChooserWithRand[T any, W chooserWeight](chooser *wr.Chooser[T, W], rand *randv2.Rand) T {
	if chooser == nil {
		var zero T
		return zero
	}
	return chooser.PickWith(rand)
}

func pickChooserWithSource[T any, W chooserWeight](chooser *wr.Chooser[T, W], src ds.DiceSource) T {
	return pickChooserWithRand(chooser, randv2.New(normalizeDiceSource(src)))
}
