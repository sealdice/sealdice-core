package deckstore

import (
	"strconv"

	"github.com/klauspost/compress/zstd"
	"github.com/vmihailenco/msgpack"
)

// Key 生成 (deckID, group) 的组合键，供内存索引与缓存使用。
func Key(deckID, group string) string {
	return deckID + "\x00" + group
}

// chunkCacheKey 生成块缓存键。
func chunkCacheKey(deckID, group string, chunkNo int) string {
	return Key(deckID, group) + "\x00" + strconv.Itoa(chunkNo)
}

// ChunkForIndex 在升序的块首下标里二分，返回包含 idx 的块号；越界返回 -1。
func ChunkForIndex(starts []int, idx int) int {
	lo, hi := 0, len(starts)-1
	ans := -1
	for lo <= hi {
		mid := (lo + hi) / 2
		if starts[mid] <= idx {
			ans = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return ans
}

// msgpackStrHeaderLen 返回 msgpack 字符串头部的字节数（不含内容）。
func msgpackStrHeaderLen(n int) int {
	switch {
	case n < 32:
		return 1 // fixstr
	case n < 256:
		return 2 // str8
	case n < 65536:
		return 3 // str16
	default:
		return 5 // str32
	}
}

// msgpackArrayHeaderLen 返回 msgpack 数组头部的字节数（不含元素）。
func msgpackArrayHeaderLen(n int) int {
	switch {
	case n < 16:
		return 1 // fixarray
	case n < 65536:
		return 3 // array16
	default:
		return 5 // array32
	}
}

var (
	// 并发=1：zstd 默认按 GOMAXPROCS 预分配并发缓冲（12 核 ≈ 21.8MB），直接吃掉内存预算。
	// EncodeAll/DecodeAll 在并发=1 下仍可被并发调用，只是单次调用单 goroutine 执行。
	// 窗口=256KB：zstd 编码器默认窗口 8MB，会为历史表分配约 16MB；本库单块通常 ≤128KB
	// （极端单条可略超），256KB 窗口足够，常规数据不损失压缩比；超长输入会自动多块编码，无正确性问题。
	zstdEnc, _ = zstd.NewWriter(nil,
		zstd.WithEncoderConcurrency(1),
		zstd.WithWindowSize(1<<18), // 256KB
	) // 默认 SpeedDefault
	zstdDec, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
)

func zstdEncode(b []byte) []byte {
	return zstdEnc.EncodeAll(b, nil)
}

func zstdDecode(b []byte) ([]byte, error) {
	return zstdDec.DecodeAll(b, nil)
}

// encodeChunk 把一块条目序列化并压缩：zstd(msgpack([]string))。
func encodeChunk(items []string) ([]byte, error) {
	raw, err := msgpack.Marshal(items)
	if err != nil {
		return nil, err
	}
	return zstdEncode(raw), nil
}

// decodeChunk 解压并反序列化一块条目。
func decodeChunk(blob []byte) ([]string, error) {
	raw, err := zstdDecode(blob)
	if err != nil {
		return nil, err
	}
	var items []string
	if err := msgpack.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// encodeWeights 压缩权重数组：zstd(msgpack([]uint32))。
func encodeWeights(w []uint32) ([]byte, error) {
	raw, err := msgpack.Marshal(w)
	if err != nil {
		return nil, err
	}
	return zstdEncode(raw), nil
}

// decodeWeights 解压并反序列化权重数组。
func decodeWeights(blob []byte) ([]uint32, error) {
	raw, err := zstdDecode(blob)
	if err != nil {
		return nil, err
	}
	var w []uint32
	if err := msgpack.Unmarshal(raw, &w); err != nil {
		return nil, err
	}
	return w, nil
}

// chunkByteSize 估算一块解压后条目占用的字节数（用于 LRU 记账）。
func chunkByteSize(items []string) int64 {
	var n int64
	for _, s := range items {
		n += int64(len(s)) + 16
	}
	return n
}

// weightByteSize 计算权重数组占用的字节数。
func weightByteSize(w []uint32) int64 {
	return int64(len(w)) * 4
}

// chunkCacheCost 是块缓存的 otter 成本函数：估算一块解压后条目占用的字节数。
// 与旧 byteLRU 的 chunkByteSize 记账一致（Σ(len(s)+16)）。
// otter 要求成本 ≥1（0 会破坏 S3-FIFO 的代价记账），空块也返回 1。
func chunkCacheCost(_ string, items []string) uint32 {
	n := chunkByteSize(items)
	if n <= 0 {
		return 1
	}
	return uint32(n)
}

// weightCacheCost 是权重缓存的 otter 成本函数：len(w)*4 字节；空数组返回 1。
func weightCacheCost(_ string, w []uint32) uint32 {
	n := weightByteSize(w)
	if n <= 0 {
		return 1
	}
	return uint32(n)
}

// normalizeWeight 把 weight==0 归一为默认权重 1。
func normalizeWeight(w uint32) uint32 {
	if w == 0 {
		return 1
	}
	return w
}

// groupWriter 缓冲一个组的条目并流式落块。非线程安全。
type groupWriter struct {
	imp  *sqliteImport
	name string

	items   []string // 当前块条目缓冲
	weights []uint32 // 全组权重（仅 weighted 时落库）
	sum     int      // 当前块 msgpack 载荷（不含数组头）字节数
	count   int      // 当前块条数
	chunkNo int      // 下一块编号
	starts  []int    // 每块首条全局下标

	totalCount  int
	totalWeight int64
	weighted    bool
	err         error
}

func newGroupWriter(imp *sqliteImport, name string) *groupWriter {
	return &groupWriter{imp: imp, name: name}
}

func (w *groupWriter) Add(text string, weight uint32) {
	if w.err != nil {
		return
	}
	nw := normalizeWeight(weight)
	w.items = append(w.items, text)
	w.weights = append(w.weights, nw)
	w.sum += msgpackStrHeaderLen(len(text)) + len(text)
	w.count++
	w.totalCount++
	w.totalWeight += int64(nw)
	if nw > 1 {
		w.weighted = true
	}
	if w.count >= ChunkMaxItems || w.curChunkSize() >= ChunkMaxBytes {
		w.err = w.flush()
	}
}

// curChunkSize 返回当前块未压缩 msgpack 的精确字节数。
func (w *groupWriter) curChunkSize() int {
	return msgpackArrayHeaderLen(w.count) + w.sum
}

func (w *groupWriter) flush() error {
	if w.count == 0 {
		return nil
	}
	blob, err := encodeChunk(w.items)
	if err != nil {
		return err
	}
	if err := w.imp.tx.Exec(
		"INSERT INTO deck_chunk (deck_key, group_name, chunk_no, blob) VALUES (?, ?, ?, ?)",
		w.imp.deckID, w.name, w.chunkNo, blob,
	).Error; err != nil {
		return err
	}
	w.starts = append(w.starts, w.totalCount-w.count)
	w.chunkNo++
	w.items = w.items[:0]
	w.sum = 0
	w.count = 0
	return nil
}

// Finish 落完剩余块、写入组元数据行并把 GroupMeta 登记到导入事务。
func (w *groupWriter) Finish() (GroupMeta, error) {
	if w.err != nil {
		return GroupMeta{}, w.err
	}
	if w.count > 0 {
		if err := w.flush(); err != nil {
			w.err = err
			return GroupMeta{}, err
		}
	}
	meta := GroupMeta{
		ItemCount:   w.totalCount,
		TotalWeight: w.totalWeight,
		Weighted:    w.weighted,
		ChunkStarts: w.starts,
	}
	if err := w.writeGroupRow(meta); err != nil {
		w.err = err
		return GroupMeta{}, err
	}
	w.imp.groups[Key(w.imp.deckID, w.name)] = meta
	return meta, nil
}

func (w *groupWriter) writeGroupRow(meta GroupMeta) error {
	startsBlob, err := msgpack.Marshal(meta.ChunkStarts)
	if err != nil {
		return err
	}
	var weightBlob []byte
	if meta.Weighted {
		weightBlob, err = encodeWeights(w.weights)
		if err != nil {
			return err
		}
	}
	weighted := 0
	if meta.Weighted {
		weighted = 1
	}
	return w.imp.tx.Exec(
		"INSERT INTO deck_group (deck_key, group_name, item_count, total_weight, weighted, chunk_count, chunk_starts, weight_blob) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		w.imp.deckID, w.name, meta.ItemCount, meta.TotalWeight, weighted, len(meta.ChunkStarts), startsBlob, weightBlob,
	).Error
}
