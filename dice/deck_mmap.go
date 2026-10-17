package dice

// The source deck is authoritative. These derived files are immutable,
// disposable compressed copies, never a second editable deck format.
import (
	"bufio"
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
	wr "github.com/mroth/weightedrand/v3"
)

const (
	deckBlockEntries = 256
	deckBlockBytes   = 16 << 10
	deckCacheBytes   = 4 << 20
)

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
		weighted := false
		for _, text := range items {
			w, _ := deckEntryWeight(text)
			if w != 1 {
				weighted = true
				break
			}
		}
		if weighted {
			g.weightsOffset, g.total = pos, 0
			choices := make([]wr.Choice[int, uint], len(items))
			weights := make([]byte, 0, len(items)*8)
			for i, text := range items {
				w, _ := deckEntryWeight(text)
				// Match weightedrand's uint64 total, including on 32-bit systems.
				if uint64(w) >= ^uint64(0)-g.total {
					return fmt.Errorf("牌组 %q 权重总和溢出", name)
				}
				choices[i] = wr.NewChoice(i, w)
				g.total += uint64(w)
				weights = binary.LittleEndian.AppendUint64(weights, g.total)
			}
			if err = write(weights); err != nil {
				return err
			}
			slices.SortFunc(choices, func(a, b wr.Choice[int, uint]) int { return cmp.Compare(a.Weight, b.Weight) })
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
