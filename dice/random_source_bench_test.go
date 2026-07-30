//nolint:testpackage
package dice

import (
	"testing"

	ds "github.com/sealdice/dicescript"
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
		DiceRandomModeCrypto,
	}

	for _, mode := range modes {
		b.Run(string(mode), func(b *testing.B) {
			src, err := newDiceSourceForMode(mode)
			if err != nil {
				b.Fatalf("newDiceSourceForMode(%s) error = %v", mode, err)
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
		DiceRandomModeCrypto,
	}

	for _, mode := range modes {
		b.Run(string(mode), func(b *testing.B) {
			src, err := newDiceSourceForMode(mode)
			if err != nil {
				b.Fatalf("newDiceSourceForMode(%s) error = %v", mode, err)
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

	b.Run("reuse_system_source", func(b *testing.B) {
		benchSrcSink = d.getSystemDiceSource()
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

	b.Run("new_source_baseline", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			benchSrcSink = d.newDiceSource()
		}
	})
}
