package dice

import (
	"reflect"

	ds "github.com/sealdice/dicescript"
)

func normalizeDiceSource(src ds.DiceSource) ds.DiceSource {
	if isNilDiceSource(src) {
		return globalRandSource
	}
	return src
}

func randIntnFromSource(src ds.DiceSource, n int) int {
	if n <= 0 {
		panic("invalid bound")
	}
	return int(randUint64n(src, uint64(n)))
}

func randUint64n(src ds.DiceSource, n uint64) uint64 {
	if n == 0 {
		panic("invalid bound")
	}
	src = normalizeDiceSource(src)
	if n&(n-1) == 0 {
		return src.Uint64() & (n - 1)
	}
	ceiling := ^uint64(0) - ^uint64(0)%n
	for {
		v := src.Uint64()
		if v < ceiling {
			return v % n
		}
	}
}

func shuffleWithSource(src ds.DiceSource, n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := randIntnFromSource(src, i+1)
		swap(i, j)
	}
}

func isNilDiceSource(src ds.DiceSource) bool {
	if src == nil {
		return true
	}
	v := reflect.ValueOf(src)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
