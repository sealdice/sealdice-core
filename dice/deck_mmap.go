package dice

// The source deck is authoritative. These derived files are immutable,
// disposable compressed copies, never a second editable deck format.
import (
	"bufio"
	"bytes"
	"cmp"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	mmap "github.com/blevesearch/mmap-go"
	"github.com/cespare/xxhash/v2"
	"github.com/klauspost/compress/zstd"

	"sealdice-core/utils/weighted"
)

const (
	deckBlockEntries = 256
	deckBlockBytes   = 16 << 10
	deckCacheBytes   = 4 << 20
)

// Exactly 16 retained Go bytes per group. Equal hashes are resolved by comparing
// the original key in the mapping; neither hash equality nor offset is identity.
type deckHashEntry struct{ hash, offset uint64 }

type deckBlock struct {
	first, count   int
	offset, length int
	rawSize        int
}

type deckGroupIndex struct {
	nilItems bool // Preserve nil-vs-empty behavior for recursive references.
	count    int
	blocks   []deckBlock
	// Zero means uniform weight 1. Otherwise the file contains uint64
	// cumulative weights, read directly from the immutable mapping.
	weightsOffset    int
	replaceOffset    int // (cumulative uint64, original index uint64), sorted like weightedrand.
	total            uint64
	diskBlocksOffset int
	diskBlockCount   int
}

type deckBlockStore struct {
	data       mmap.MMap
	filename   string
	references int
	closed     bool
	hashes     []deckHashEntry
	cache      *deckDiskCache
	persistent bool
}

type deckCacheKey struct {
	file   *deckBlockStore
	offset int
}
type deckCacheEntry struct {
	key  deckCacheKey
	data []byte
}

// The budget counts retained decoded byte-slice capacities, not map/list nodes,
// the decoder, active result strings, mapped pages or per-context shuffle state.
// Block reads and the LRU use this lock; weight mmap reads need only the
// enclosing catalog lease because mappings are immutable. No
// formatting, recursion, random selection or network IO runs under it.
var deckBlockReader = struct {
	sync.Mutex
	decoder *zstd.Decoder
	entries map[deckCacheKey]*list.Element
	lru     list.List
	bytes   int
}{entries: make(map[deckCacheKey]*list.Element)}

func buildMappedDeck(dir string, groups map[string][]string) (_ *deckBlockStore, retErr error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, "deck-*.blocks")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
		if retErr != nil {
			_ = os.Remove(f.Name())
		}
	}()
	s := &deckBlockStore{filename: f.Name(), references: 1}
	writer := bufio.NewWriterSize(f, deckBlockBytes)
	// Nonempty header also permits an entirely empty deck to be mapped.
	if _, err = writer.WriteString("SEALDECK3"); err != nil {
		return nil, err
	}
	if err = encodeDeckBlocks(s, groups, func(_ int, b []byte) error { _, e := writer.Write(b); return e }); err != nil {
		return nil, err
	}
	if err = writer.Flush(); err != nil {
		return nil, err
	}
	s.data, err = mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		return nil, err
	}
	if err = s.loadMappedIndex(); err != nil {
		_ = s.data.Unmap()
		return nil, err
	}
	return s, nil
}

// The parser still holds a whole deck; this is not a streaming source parser.
func encodeDeckBlocks(s *deckBlockStore, groups map[string][]string, appendBlock func(int, []byte) error) error {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	slices.Sort(names)
	enc, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1), zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithWindowSize(128<<10))
	if err != nil {
		return err
	}
	defer enc.Close()
	pos := 9
	write := func(b []byte) error { e := appendBlock(pos, b); pos += len(b); return e }
	var raw, compressed []byte
	var directory []deckHashEntry
	for _, name := range names {
		items := groups[name]
		g := &deckGroupIndex{count: len(items), total: uint64(len(items)), nilItems: items == nil}
		nameOffset := pos + 8
		key := binary.LittleEndian.AppendUint64(nil, uint64(len(name)))
		key = append(key, name...)
		if err = write(key); err != nil {
			return err
		}
		hasWeights := false
		for _, text := range items {
			w, _ := deckEntryWeight(text)
			if w != 1 {
				hasWeights = true
				break
			}
		}
		if hasWeights {
			g.weightsOffset, g.total = pos, 0
			choices := make([]weighted.Choice[int], len(items))
			weights := make([]byte, 0, len(items)*8)
			for i, text := range items {
				w, _ := deckEntryWeight(text)
				// Match weightedrand's uint64 total, including on 32-bit systems.
				if uint64(w) >= ^uint64(0)-g.total {
					return fmt.Errorf("牌组 %q 权重总和溢出", name)
				}
				choices[i] = weighted.Choice[int]{Item: i, Weight: w}
				g.total += uint64(w)
				weights = binary.LittleEndian.AppendUint64(weights, g.total)
			}
			if err = write(weights); err != nil {
				return err
			}
			slices.SortFunc(choices, func(a, b weighted.Choice[int]) int { return cmp.Compare(a.Weight, b.Weight) })
			g.replaceOffset = pos
			weights = make([]byte, 0, len(items)*16)
			var total uint64
			for _, choice := range choices {
				total += uint64(choice.Weight)
				weights = binary.LittleEndian.AppendUint64(weights, total)
				weights = binary.LittleEndian.AppendUint64(weights, uint64(choice.Item))
			}
			if err = write(weights); err != nil {
				return err
			}
		}
		for first := 0; first < len(items); {
			raw = raw[:0]
			end := first
			for end < len(items) && end-first < deckBlockEntries {
				text := items[end]
				if end > first && len(raw)+8+len(text) > deckBlockBytes {
					break
				}
				raw = binary.LittleEndian.AppendUint64(raw, uint64(len(text)))
				raw = append(raw, text...)
				end++
			}
			// A single oversized entry gets its own block, without a size limit.
			compressed = enc.EncodeAll(raw, compressed[:0])
			g.blocks = append(g.blocks, deckBlock{first: first, count: end - first, offset: pos, length: len(compressed), rawSize: len(raw)})
			if err = write(compressed); err != nil {
				return err
			}
			first = end
			if cap(raw) > deckBlockBytes*2 {
				raw = nil
			}
			if cap(compressed) > deckBlockBytes*2 {
				compressed = nil
			}
		}
		directory = append(directory, deckHashEntry{hash: xxhash.Sum64String(name), offset: uint64(pos)})
		record := encodeDeckGroup(nameOffset, len(name), g)
		if err = write(record); err != nil {
			return err
		}
	}
	slices.SortFunc(directory, func(a, b deckHashEntry) int {
		if c := cmp.Compare(a.hash, b.hash); c != 0 {
			return c
		}
		return cmp.Compare(a.offset, b.offset)
	})
	start := pos
	for _, entry := range directory {
		b := binary.LittleEndian.AppendUint64(nil, entry.hash)
		b = binary.LittleEndian.AppendUint64(b, entry.offset)
		if err = write(b); err != nil {
			return err
		}
	}
	footer := append([]byte{}, "SEALIDX3"...)
	footer = binary.LittleEndian.AppendUint64(footer, uint64(start))
	footer = binary.LittleEndian.AppendUint64(footer, uint64(len(directory)))
	if err = write(footer); err != nil {
		return err
	}
	return nil
}

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

func (s *deckBlockStore) close() error {
	deckBlockReader.Lock()
	defer deckBlockReader.Unlock()
	s.references--
	if s.references > 0 {
		return nil
	}
	for key, element := range deckBlockReader.entries {
		if key.file == s {
			deckBlockReader.bytes -= cap(element.Value.(deckCacheEntry).data)
			deckBlockReader.lru.Remove(element)
			delete(deckBlockReader.entries, key)
		}
	}
	s.closed = true
	if s.data == nil {
		return nil
	}
	err := s.data.Unmap()
	if !s.persistent {
		err = errors.Join(err, os.Remove(s.filename))
	}
	if s.cache != nil {
		err = errors.Join(err, s.cache.releaseFile(s.filename))
	}
	return err
}

func (s *deckBlockStore) retain() {
	deckBlockReader.Lock()
	s.references++
	deckBlockReader.Unlock()
}

func (s *deckBlockStore) cumulative(g *deckGroupIndex, i int) (uint64, error) {
	if i < 0 {
		return 0, nil
	}
	if g.weightsOffset == 0 {
		return uint64(i + 1), nil
	}
	b, err := s.weightBytes(g.weightsOffset, i*8, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

func (s *deckBlockStore) item(g *deckGroupIndex, index int) (string, error) {
	if index < 0 || index >= g.count {
		return "", errors.New("牌组下标越界")
	}
	deckBlockReader.Lock()
	defer deckBlockReader.Unlock()
	if s.closed {
		return "", errors.New("牌堆存储已关闭")
	}
	blockIndex := sort.Search(g.blockCount(), func(i int) bool { b := s.blockAt(g, i); return b.first+b.count > index })
	b := s.blockAt(g, blockIndex)
	key := deckCacheKey{s, b.offset}
	var raw []byte
	if hit := deckBlockReader.entries[key]; hit != nil {
		deckBlockReader.lru.MoveToFront(hit)
		raw = hit.Value.(deckCacheEntry).data
	} else {
		compressed, err := s.blockBytes(b)
		if err != nil {
			return "", err
		}
		raw, err = s.decodeFrame(compressed, b.rawSize)
		if err != nil {
			return "", fmt.Errorf("解压牌堆: %w", err)
		}
		if len(raw) != b.rawSize {
			return "", errors.New("牌堆块长度不一致")
		}
		cacheDeckBlock(key, raw)
	}
	for n := b.first; n <= index; n++ {
		if len(raw) < 8 {
			return "", errors.New("牌堆块截断")
		}
		length := binary.LittleEndian.Uint64(raw)
		raw = raw[8:]
		if length > uint64(len(raw)) {
			return "", errors.New("牌堆条目截断")
		}
		if n == index {
			return string(raw[:int(length)]), nil
		}
		raw = raw[int(length):]
	}
	return "", errors.New("牌堆条目不存在")
}

// Called under deckBlockReader. One decoder is shared across all mapped decks.
func (s *deckBlockStore) decodeFrame(compressed []byte, rawSize int) ([]byte, error) {
	if deckBlockReader.decoder == nil {
		var err error
		deckBlockReader.decoder, err = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true), zstd.WithDecoderMaxMemory(1<<63))
		if err != nil {
			return nil, err
		}
	}
	return deckBlockReader.decoder.DecodeAll(compressed, make([]byte, 0, rawSize))
}

// The catalog lease keeps these immutable mapped bytes alive until the draw ends.
func (s *deckBlockStore) blockBytes(b deckBlock) ([]byte, error) {
	return s.data[b.offset : b.offset+b.length], nil
}

func (s *deckBlockStore) weightBytes(table, offset, length int) ([]byte, error) {
	return s.data[table+offset : table+offset+length], nil
}

// Caller holds deckBlockReader. Oversized single entries are decoded normally
// but are not retained by the bounded cache.
func cacheDeckBlock(key deckCacheKey, data []byte) {
	if cap(data) > deckCacheBytes {
		return
	}
	for deckBlockReader.bytes+cap(data) > deckCacheBytes {
		last := deckBlockReader.lru.Back()
		entry := last.Value.(deckCacheEntry)
		deckBlockReader.bytes -= cap(entry.data)
		delete(deckBlockReader.entries, entry.key)
		deckBlockReader.lru.Remove(last)
	}
	deckBlockReader.entries[key] = deckBlockReader.lru.PushFront(deckCacheEntry{key, data})
	deckBlockReader.bytes += cap(data)
}

// Only the local base is materialized for legacy cloud merge/deduplication.
// It is transient and is never attached to the persistent deck metadata.
func (s *deckBlockStore) materialize(name string) ([]string, error) {
	g, exists := s.group(name)
	if !exists || g.nilItems {
		return nil, nil
	}
	items := make([]string, g.count)
	for i := range items {
		text, err := s.item(&g, i)
		if err != nil {
			return nil, err
		}
		items[i] = text
	}
	return items, nil
}

// Equivalent to extractWeight, including ParseInt's overflow behavior. Avoid
// recompiling its regular expression for every imported record.
func deckEntryWeight(text string) (uint, string) {
	if strings.HasPrefix(text, "::") {
		for i := 2; i < len(text); i++ {
			if text[i] >= '0' && text[i] <= '9' {
				continue
			}
			if i > 2 && strings.HasPrefix(text[i:], "::") {
				weight, _ := strconv.ParseInt(text[2:i], 10, 64)
				return uint(weight), text[i+2:]
			}
			break
		}
	}
	return 1, text
}
