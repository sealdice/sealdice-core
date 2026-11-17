// Package weighted provides weighted selection using a caller-supplied random
// source or the global math/rand/v2 source.
package weighted

import (
	"cmp"
	"errors"
	rand "math/rand/v2"
	"slices"
)

// Choice pairs an item with its weight. All callers use native uint weights;
// cumulative totals remain uint64, including on 32-bit platforms.
type Choice[T any] struct {
	Item   T
	Weight uint
}

// Chooser holds items and their cumulative weights for weighted selection.
type Chooser[T any] struct {
	data   []Choice[T]
	totals []uint64
	max    uint64
}

// NewChooser sorts choices in place and retains them, as the previous chooser
// did. Equal weights must keep the same sort algorithm for seeded compatibility.
func NewChooser[T any](choices ...Choice[T]) (*Chooser[T], error) {
	slices.SortFunc(choices, func(a, b Choice[T]) int { return cmp.Compare(a.Weight, b.Weight) })
	totals := make([]uint64, len(choices))
	var total uint64
	for i, choice := range choices {
		weight := uint64(choice.Weight)
		if weight >= ^uint64(0)-total {
			return nil, errors.New("sum of Choice Weights exceeds max int")
		}
		total += weight
		totals[i] = total
	}
	if total == 0 {
		return nil, errors.New("zero Choices with Weight >= 1")
	}
	return &Chooser[T]{data: choices, totals: totals, max: total}, nil
}

// Pick preserves the global math/rand/v2 behavior of the existing public method.
func (c Chooser[T]) Pick() T {
	i, _ := slices.BinarySearch(c.totals, rand.Uint64N(c.max)+1)
	return c.data[i].Item
}

// PickWith consumes the caller's random stream using the existing Uint64N rule.
func (c Chooser[T]) PickWith(src *rand.Rand) T {
	i, _ := slices.BinarySearch(c.totals, src.Uint64N(c.max)+1)
	return c.data[i].Item
}

/*
The chooser above is adapted from weightedrand/v3 under the MIT License.

Copyright (c) 2018 Matthew Rothenberg

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
