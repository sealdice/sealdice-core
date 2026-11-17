//nolint:testpackage
package dice

import (
	"testing"

	ds "github.com/sealdice/dicescript"

	randcore "sealdice-core/utils/random"
)

var (
	benchUint64Sink uint64
	benchRollSink   int64
	benchSrcSink    ds.DiceSource
)

func BenchmarkDiceRandomSourceUint64(b *testing.B) {
	modes := []DiceRandomMode{
		DiceRandomModePCG,
		DiceRandomModeGM,
		DiceRandomModeNIST,
		DiceRandomModeCRNG,
	}

	for _, mode := range modes {
		b.Run(string(mode), func(b *testing.B) {
			src, err := randcore.NewSourceForMode(mode, nil)
			if err != nil {
				b.Fatalf("NewSourceForMode(%s) error = %v", mode, err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				benchUint64Sink = src.Uint64()
			}
		})
	}
}

func BenchmarkDiceRandomSourceRollD100(b *testing.B) {
	modes := []DiceRandomMode{
		DiceRandomModePCG,
		DiceRandomModeGM,
		DiceRandomModeNIST,
		DiceRandomModeCRNG,
	}

	for _, mode := range modes {
		b.Run(string(mode), func(b *testing.B) {
			src, err := randcore.NewSourceForMode(mode, nil)
			if err != nil {
				b.Fatalf("NewSourceForMode(%s) error = %v", mode, err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				benchRollSink = int64(ds.Roll(src, 100, 0))
			}
		})
	}
}

func BenchmarkNISTContextSourceBinding(b *testing.B) {
	cfg := DefaultConfig
	cfg.DiceRandomMode = string(DiceRandomModeNIST)
	d := &Dice{Config: cfg}
	ctx := &MsgContext{Dice: d}

	b.Run("reuse_global_source", func(b *testing.B) {
		benchSrcSink = globalRandSource
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			ctx.diceRandSrc = nil
			ctx._v1Rand = nil
			ctx.chooserRand = nil
			ctx.chooserSrc = nil
			benchSrcSink = ctx.getDiceSource()
		}
	})

	b.Run("factory_baseline", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			var err error
			benchSrcSink, err = randcore.NewSourceForMode(DiceRandomModeNIST, nil)
			if err != nil {
				b.Fatalf("NewSourceForMode(nist) error = %v", err)
			}
		}
	})
}
