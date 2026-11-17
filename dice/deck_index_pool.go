package dice

import (
	"encoding/binary"
	"errors"
	"sort"
)

// Only indices and cumulative weights are retained, never card strings or a
// mapping. The enclosing MsgContext is request-owned, as with legacy DeckPools.
type deckIndexPool struct {
	version uint64
	indices []int
	totals  []uint64
	total   uint64
}

func drawStoredDeck(ctx *MsgContext, deck *DeckInfo, name string, shuffle bool) (string, error) {
	s := deck.storage
	g, exists := s.group(name)
	if !exists || g.count == 0 {
		return "", errors.New("牌组为空，请检查格式是否正确")
	}
	if g.total == 0 {
		return "", errors.New("牌组无有效条目，请检查权重配置")
	}
	var index int
	if !shuffle {
		r := ctx.getChooserRand().Uint64N(g.total) + 1
		if g.replaceOffset == 0 {
			index = int(r - 1)
		} else {
			var readErr error
			selected := sort.Search(g.count, func(i int) bool {
				if readErr != nil {
					return true
				}
				b, err := s.weightBytes(g.replaceOffset, i*16, 16)
				readErr = err
				return err != nil || binary.LittleEndian.Uint64(b) >= r
			})
			if readErr != nil {
				return "", readErr
			}
			b, err := s.weightBytes(g.replaceOffset, selected*16, 16)
			if err != nil {
				return "", err
			}
			index = int(binary.LittleEndian.Uint64(b[8:]))
		}
	} else {
		if ctx.deckIndexPools == nil {
			ctx.deckIndexPools = map[string]*deckIndexPool{}
		}
		key := deck.Filename + "\x00" + name
		pool := ctx.deckIndexPools[key]
		if pool == nil || pool.version != deck.contentVersion || pool.total == 0 {
			pool = &deckIndexPool{version: deck.contentVersion, indices: make([]int, g.count), totals: make([]uint64, g.count)}
			for i := range pool.indices {
				pool.indices[i] = i
			}
			ctx.Shuffle(g.count, func(i, j int) { pool.indices[i], pool.indices[j] = pool.indices[j], pool.indices[i] })
			for i, n := range pool.indices {
				current, err := s.cumulative(&g, n)
				if err != nil {
					return "", err
				}
				previous, err := s.cumulative(&g, n-1)
				if err != nil {
					return "", err
				}
				pool.total += current - previous
				pool.totals[i] = pool.total
			}
			ctx.deckIndexPools[key] = pool
		}
		r := randUint64n(ctx.getDiceSource(), pool.total) + 1
		selected := sort.Search(len(pool.totals), func(i int) bool { return pool.totals[i] >= r })
		index = pool.indices[selected]
		weight := pool.totals[selected]
		if selected > 0 {
			weight -= pool.totals[selected-1]
		}
		pool.total -= weight
		for i := selected + 1; i < len(pool.totals); i++ {
			pool.totals[i] -= weight
		}
		pool.indices = append(pool.indices[:selected], pool.indices[selected+1:]...)
		pool.totals = append(pool.totals[:selected], pool.totals[selected+1:]...)
	}
	text, err := s.item(&g, index)
	if err != nil {
		return "", err
	}
	_, text = deckEntryWeight(text)
	return text, nil
}
