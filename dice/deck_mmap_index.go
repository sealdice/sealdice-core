package dice

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"github.com/cespare/xxhash/v2"
)

// Exactly 16 retained Go bytes per group. Equal hashes are resolved by comparing
// the original key in the mapping; neither hash equality nor offset is identity.
type deckHashEntry struct{ hash, offset uint64 }

func encodeDeckGroup(keyOffset, keyLength int, g *deckGroupIndex) []byte {
	flags := uint64(0)
	if g.nilItems {
		flags = 1
	}
	b := make([]byte, 0, 64+len(g.blocks)*40)
	for _, n := range []uint64{uint64(keyOffset), uint64(keyLength), uint64(g.count), flags,
		uint64(g.weightsOffset), uint64(g.replaceOffset), g.total, uint64(len(g.blocks))} {
		b = binary.LittleEndian.AppendUint64(b, n)
	}
	for _, block := range g.blocks {
		for _, n := range []int{block.first, block.count, block.offset, block.length, block.rawSize} {
			b = binary.LittleEndian.AppendUint64(b, uint64(n))
		}
	}
	return b
}

func (s *deckBlockStore) rangeValid(offset, length uint64) bool {
	return offset <= uint64(len(s.data)) && length <= uint64(len(s.data))-offset
}

func (s *deckBlockStore) mappedGroup(offset uint64) (deckGroupIndex, []byte) {
	b := s.data[int(offset):]
	u := func(i int) uint64 { return binary.LittleEndian.Uint64(b[i*8:]) }
	g := deckGroupIndex{count: int(u(2)), nilItems: u(3) != 0, weightsOffset: int(u(4)),
		replaceOffset: int(u(5)), total: u(6), diskBlocksOffset: int(offset) + 64, diskBlockCount: int(u(7))}
	return g, s.data[int(u(0)):int(u(0)+u(1))]
}

func (g *deckGroupIndex) blockCount() int {
	if g.blocks != nil {
		return len(g.blocks)
	}
	return g.diskBlockCount
}

func (s *deckBlockStore) blockAt(g *deckGroupIndex, index int) deckBlock {
	if g.blocks != nil {
		return g.blocks[index]
	}
	b := s.data[g.diskBlocksOffset+index*40:]
	u := func(i int) int { return int(binary.LittleEndian.Uint64(b[i*8:])) }
	return deckBlock{first: u(0), count: u(1), offset: u(2), length: u(3), rawSize: u(4)}
}

func (s *deckBlockStore) validateMappedGroup(offset, directory uint64) error {
	bad := errors.New("invalid mmap group index")
	if !s.rangeValid(offset, 64) || offset+64 > directory {
		return bad
	}
	b := s.data[int(offset):]
	u := func(i int) uint64 { return binary.LittleEndian.Uint64(b[i*8:]) }
	maxInt := uint64(^uint(0) >> 1)
	for _, field := range []int{0, 1, 2, 4, 5, 7} {
		if u(field) > maxInt {
			return bad
		}
	}
	if u(0) < 17 || !s.rangeValid(u(0), u(1)) || u(0) > offset || u(1) > offset-u(0) || u(3) > 1 {
		return bad
	}
	if binary.LittleEndian.Uint64(s.data[int(u(0))-8:]) != u(1) {
		return bad
	}
	if u(7) > (directory-offset-64)/40 {
		return bad
	}
	g, _ := s.mappedGroup(offset)
	if g.nilItems && g.count != 0 {
		return bad
	}
	keyEnd := u(0) + u(1)
	if g.weightsOffset == 0 {
		if g.replaceOffset != 0 || g.total != uint64(g.count) {
			return bad
		}
	} else {
		if g.count == 0 || uint64(g.count) > uint64(len(s.data))/16 {
			return bad
		}
		for _, table := range []struct{ offset, width uint64 }{{u(4), 8}, {u(5), 16}} {
			length := uint64(g.count) * table.width
			if table.offset < keyEnd || !s.rangeValid(table.offset, length) || table.offset+length > offset {
				return bad
			}
			var previous uint64
			for i := range g.count {
				entry := s.data[int(table.offset)+i*int(table.width):]
				current := binary.LittleEndian.Uint64(entry)
				if current < previous || current > g.total {
					return bad
				}
				if table.width == 16 && binary.LittleEndian.Uint64(entry[8:]) >= uint64(g.count) {
					return bad
				}
				previous = current
			}
			if previous != g.total {
				return bad
			}
		}
	}
	next := 0
	for i := range g.diskBlockCount {
		raw := s.data[g.diskBlocksOffset+i*40:]
		for field := range 5 {
			if binary.LittleEndian.Uint64(raw[field*8:]) > maxInt {
				return bad
			}
		}
		block := s.blockAt(&g, i)
		if block.first != next || block.count <= 0 || block.count > g.count-next || block.count > deckBlockEntries {
			return bad
		}
		if block.length <= 0 || block.offset < int(keyEnd) || uint64(block.offset) > offset || uint64(block.length) > offset-uint64(block.offset) {
			return bad
		}
		if block.rawSize/8 < block.count {
			return bad
		}
		next += block.count
	}
	if next != g.count {
		return bad
	}
	return nil
}

// Group descriptors, original keys and block tables stay in the mapping.
func (s *deckBlockStore) loadMappedIndex() error {
	if len(s.data) < 33 || string(s.data[:9]) != "SEALDECK3" {
		return errors.New("unknown mmap cache header")
	}
	footer := s.data[len(s.data)-24:]
	if string(footer[:8]) != "SEALIDX3" {
		return errors.New("missing mmap directory footer")
	}
	start, count := binary.LittleEndian.Uint64(footer[8:]), binary.LittleEndian.Uint64(footer[16:])
	if start < 9 || start > uint64(len(s.data)-24) || count > uint64(len(s.data))/16 || start+count*16 != uint64(len(s.data)-24) {
		return errors.New("invalid mmap directory bounds")
	}
	s.hashes = make([]deckHashEntry, int(count))
	var previousHash uint64
	var previousKey []byte
	for i := range int(count) {
		b := s.data[int(start)+i*16:]
		hash, offset := binary.LittleEndian.Uint64(b), binary.LittleEndian.Uint64(b[8:])
		if err := s.validateMappedGroup(offset, start); err != nil {
			return fmt.Errorf("group %d: %w", i, err)
		}
		_, key := s.mappedGroup(offset)
		if xxhash.Sum64(key) != hash || (i > 0 && (hash < previousHash || (hash == previousHash && bytes.Compare(previousKey, key) >= 0))) {
			return errors.New("invalid mmap hash directory")
		}
		previousHash, previousKey = hash, key
		s.hashes[i] = deckHashEntry{hash: hash, offset: offset}
	}
	return nil
}

func (s *deckBlockStore) group(name string) (deckGroupIndex, bool) {
	return s.groupWithHash(name, xxhash.Sum64String(name))
}

func (s *deckBlockStore) groupWithHash(name string, hash uint64) (deckGroupIndex, bool) {
	first := sort.Search(len(s.hashes), func(i int) bool { return s.hashes[i].hash >= hash })
	for i := first; i < len(s.hashes) && s.hashes[i].hash == hash; i++ {
		g, key := s.mappedGroup(s.hashes[i].offset)
		if string(key) == name {
			return g, true
		}
	}
	return deckGroupIndex{}, false
}

func (s *deckBlockStore) groupCount() int {
	return len(s.hashes)
}
