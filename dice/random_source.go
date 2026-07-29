package dice

import (
	"encoding/binary"
	"fmt"
	"io"
	randv2 "math/rand/v2"
	"reflect"
	"strings"
	"sync"

	gmrand "github.com/emmansun/gmsm/rand"
	wr "github.com/mroth/weightedrand/v3"
	ds "github.com/sealdice/dicescript"
	ctrdrbg "github.com/sixafter/aes-ctr-drbg"
)

type DiceRandomMode string

const (
	DiceRandomModePCG    DiceRandomMode = "pcg"
	DiceRandomModeGM     DiceRandomMode = "gm"
	DiceRandomModeNIST   DiceRandomMode = "nist"
	DiceRandomModeCrypto DiceRandomMode = "crypto"
)

type readerDiceSource struct {
	reader io.Reader
	mu     sync.Mutex
}

type pcgDiceSource struct {
	mu  sync.Mutex
	pcg *randv2.PCG
}

func (s *readerDiceSource) Uint64() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var data [8]byte
	if _, err := io.ReadFull(s.reader, data[:]); err != nil {
		panic(fmt.Errorf("dice random source read failed: %w", err))
	}
	return binary.BigEndian.Uint64(data[:])
}

func newPCGDiceSource(seed uint64) ds.StatefulDiceSource {
	return &pcgDiceSource{pcg: randv2.NewPCG(seed, seed)}
}

func (s *pcgDiceSource) Uint64() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcg.Uint64()
}

func (s *pcgDiceSource) MarshalBinary() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pcg.MarshalBinary()
}

func (s *pcgDiceSource) UnmarshalBinary(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pcg == nil {
		s.pcg = randv2.NewPCG(0, 0)
	}
	return s.pcg.UnmarshalBinary(data)
}

func normalizeDiceRandomMode(raw string) DiceRandomMode {
	switch DiceRandomMode(strings.ToLower(strings.TrimSpace(raw))) {
	case DiceRandomModeGM:
		return DiceRandomModeGM
	case DiceRandomModeNIST:
		return DiceRandomModeNIST
	case DiceRandomModeCrypto:
		return DiceRandomModeCrypto
	default:
		return DiceRandomModePCG
	}
}

func newDiceSourceForMode(mode DiceRandomMode) (ds.DiceSource, error) {
	switch mode {
	case DiceRandomModeGM:
		return &readerDiceSource{reader: gmrand.Reader}, nil
	case DiceRandomModeNIST:
		reader, err := ctrdrbg.NewReader(
			ctrdrbg.WithKeySize(ctrdrbg.KeySize256),
			ctrdrbg.WithPersonalization([]byte("sealdice-nist")),
		)
		if err != nil {
			return nil, err
		}
		return &readerDiceSource{reader: reader}, nil
	case DiceRandomModeCrypto:
		return ds.NewCryptoDiceSource(), nil
	case DiceRandomModePCG:
		fallthrough
	default:
		return newPCGDiceSource(generateRandSeed()), nil
	}
}

func (d *Dice) getDiceRandomMode() DiceRandomMode {
	return normalizeDiceRandomMode(d.Config.DiceRandomMode)
}

func (d *Dice) newDiceSource() ds.DiceSource {
	src, err := newDiceSourceForMode(d.getDiceRandomMode())
	if err != nil {
		panic(fmt.Errorf("dice random mode %q init failed: %w", d.getDiceRandomMode(), err))
	}
	return src
}

func (d *Dice) getSystemDiceSource() ds.DiceSource {
	mode := d.getDiceRandomMode()

	d.randomSourceMu.Lock()
	defer d.randomSourceMu.Unlock()

	if d.systemDiceSource == nil || d.systemDiceMode != mode {
		d.systemDiceSource = d.newDiceSource()
		d.systemDiceMode = mode
	}
	return d.systemDiceSource
}

func (d *Dice) Roll(points int) int {
	if points <= 0 {
		return 0
	}
	return int(ds.Roll(d.getSystemDiceSource(), ds.IntType(points), 0))
}

func (d *Dice) Roll64(points int64) int64 {
	return DiceRoll64x(d.getSystemDiceSource(), points)
}

func (ctx *MsgContext) getDiceSource() ds.DiceSource {
	if ctx.diceRandSrc != nil {
		ctx._v1Rand = ctx.diceRandSrc
		return ctx.diceRandSrc
	}
	if ctx.Dice != nil {
		ctx.diceRandSrc = ctx.Dice.newDiceSource()
	} else {
		ctx.diceRandSrc = randSource
	}
	ctx._v1Rand = ctx.diceRandSrc
	return ctx.diceRandSrc
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

func (d *Dice) RandIntn(n int) int {
	return randIntnFromSource(d.getSystemDiceSource(), n)
}

func (ctx *MsgContext) RandIntn(n int) int {
	return randIntnFromSource(ctx.getDiceSource(), n)
}

func (ctx *MsgContext) Shuffle(n int, swap func(i, j int)) {
	shuffleWithSource(ctx.getDiceSource(), n, swap)
}

func (d *Dice) Shuffle(n int, swap func(i, j int)) {
	shuffleWithSource(d.getSystemDiceSource(), n, swap)
}

type chooserWeight interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func pickChooserWithSource[T any, W chooserWeight](chooser *wr.Chooser[T, W], src ds.DiceSource) T {
	if chooser == nil {
		var zero T
		return zero
	}
	return chooser.PickWith(randv2.New(normalizeDiceSource(src)))
}

func randIntnFromSource(src ds.DiceSource, n int) int {
	if n <= 0 {
		panic("invalid bound")
	}
	return int(randUint64n(src, uint64(n)))
}

func randInt64nFromSource(src ds.DiceSource, n int64) int64 {
	if n <= 0 {
		panic("invalid bound")
	}
	return int64(randUint64n(src, uint64(n)))
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

func normalizeDiceSource(src ds.DiceSource) ds.DiceSource {
	if isNilDiceSource(src) {
		return randSource
	}
	return src
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
