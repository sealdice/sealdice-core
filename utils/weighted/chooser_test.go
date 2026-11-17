package weighted_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	rand "math/rand/v2"
	"strconv"
	"testing"

	"sealdice-core/utils/weighted"
)

func TestChooserSeededCompatibility(t *testing.T) {
	// Recorded with weightedrand v3.0.0 before removal. Check both the selected
	// sequence and the next random word, including rejection sampling and ties.
	type sample struct {
		name    string
		weights []uint64
		digest  string
		next    uint64
	}
	tests := []sample{
		{"uniform", []uint64{1, 1, 1}, "8a2b0f94699c14e55c0b8dc49fc0474712b881220e739daee3dbeb0eec28a507", 12561962438520182476},
		{"zero-and-ties", []uint64{0, 3, 1, 9, 3, 0}, "5d16f8505674946960248582b27f85d795daaf5b8ac5e0639a7ada07e01380d6", 12561962438520182476},
		{"many-ties", []uint64{2, 1, 2, 0, 3, 1, 2, 2, 1, 0, 3, 2, 1, 2, 3, 1, 0, 2, 1, 3, 2, 1, 2, 3}, "35e06cade9639312145eb3c61df0e0ebc2cf28b11d57881495109efdc9a5b4a3", 12561962438520182476},
	}
	if strconv.IntSize == 64 {
		tests = append(tests,
			sample{"wide", []uint64{1 << 32, 1, 0, 1 << 33}, "581deb8058a132ddf4c3a36e1568c83fcdb00b23c18cb80effdea6213dd2b874", 12561962438520182476},
			sample{"rejection", []uint64{1 << 63, 1 << 62, 1}, "ba789a1d6d1039a54b7b6512ee65905d3189c1ef12da2b96648e2f016474177f", 6126256135718613777},
		)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choices := make([]weighted.Choice[int], len(tt.weights))
			for i, weight := range tt.weights {
				choices[i] = weighted.Choice[int]{Item: i, Weight: uint(weight)}
			}
			chooser, err := weighted.NewChooser(choices...)
			if err != nil {
				t.Fatal(err)
			}
			src := rand.New(rand.NewPCG(11, 29))
			hash := sha256.New()
			for range 128 {
				fmt.Fprintf(hash, "%d,", chooser.PickWith(src))
			}
			if got := hex.EncodeToString(hash.Sum(nil)); got != tt.digest {
				t.Fatalf("draw sequence changed: %s != %s", got, tt.digest)
			}
			if got := src.Uint64(); got != tt.next {
				t.Fatalf("random consumption changed: %d != %d", got, tt.next)
			}
		})
	}
}

func TestChooserWeightBoundaries(t *testing.T) {
	for _, choices := range [][]weighted.Choice[string]{nil, {{Item: "never", Weight: 0}}} {
		chooser, err := weighted.NewChooser(choices...)
		if chooser != nil || err == nil || err.Error() != "zero Choices with Weight >= 1" {
			t.Fatalf("empty weights: %v, %v", chooser, err)
		}
	}
	// The previous chooser rejected totals equal to MaxUint64 as well as overflow.
	if strconv.IntSize == 64 {
		for _, weights := range [][]uint{{^uint(0)}, {^uint(0) - 1, 1}, {^uint(0) - 1, 2}} {
			choices := make([]weighted.Choice[int], len(weights))
			for i, weight := range weights {
				choices[i] = weighted.Choice[int]{Item: i, Weight: weight}
			}
			chooser, err := weighted.NewChooser(choices...)
			if chooser != nil || err == nil || err.Error() != "sum of Choice Weights exceeds max int" {
				t.Fatalf("overflow weights %v: %v, %v", weights, chooser, err)
			}
		}
	}
	chooser, err := weighted.NewChooser(weighted.Choice[string]{Item: "never"}, weighted.Choice[string]{Item: "only", Weight: ^uint(0) - 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := chooser.Pick(); got != "only" {
		t.Fatalf("Pick() = %q", got)
	}
}
