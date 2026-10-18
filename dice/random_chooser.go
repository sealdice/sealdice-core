package dice

import (
	randv2 "math/rand/v2"

	ds "github.com/sealdice/dicescript"

	randcore "sealdice-core/utils/random"
	"sealdice-core/utils/weighted"
)

func newPCGDiceSource(seed uint64) ds.StatefulDiceSource {
	return randcore.NewPCGSource(seed)
}

func pickChooserWithRand[T any](chooser *weighted.Chooser[T], rand *randv2.Rand) T {
	if chooser == nil {
		var zero T
		return zero
	}
	return chooser.PickWith(rand)
}

func pickChooserWithSource[T any](chooser *weighted.Chooser[T], src ds.DiceSource) T {
	return pickChooserWithRand(chooser, randv2.New(normalizeDiceSource(src)))
}
