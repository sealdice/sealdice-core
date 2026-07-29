//nolint:testpackage
package dice

import (
	"testing"

	ds "github.com/sealdice/dicescript"
)

var (
	benchUint64Sink uint64
	benchRollSink   int64
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
			for i := 0; i < b.N; i++ {
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
			for i := 0; i < b.N; i++ {
				benchRollSink = int64(ds.Roll(src, 100, 0))
			}
		})
	}
}
