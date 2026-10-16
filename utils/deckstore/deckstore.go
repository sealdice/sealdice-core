// Package deckstore 提供"分组字符串列表 + 每条权重"的分块压缩存储（SQLite）。
//
// 文本以 256 条 / 128KiB 为单位切块，msgpack 序列化后 zstd 压缩存入 SQLite；
// 对外提供按 idx 读取、权重读取、物化整组、通用 KV 元数据与原子换索引。
//
// 本包是通用存储层（L0），不解析任何牌堆格式，也不依赖 dice / dicescript。
// 随机源经 Rand 接口、日志经 Logger 接口由上层注入。
package deckstore

import "errors"

const (
	// SchemaVersion 是当前 SQLite schema 版本，存于 PRAGMA user_version。
	SchemaVersion = 1
	// ChunkMaxItems 是每块最多的条目数。
	ChunkMaxItems = 256
	// ChunkMaxBytes 是每块未压缩（msgpack）载荷的最大字节数。
	ChunkMaxBytes = 128 * 1024
	// ChunkCacheCap 是块缓存（解压后的 []string）的字节上限。
	ChunkCacheCap = 4 << 20
	// WeightCacheCap 是权重缓存（解压后的 []uint32）的字节上限。
	WeightCacheCap = 2 << 20
	// MetaKeyContentHash 是 deck_meta 表中存放牌堆内容指纹的 KV 键。
	// L1 导入器写入该键；L0 用其值（hex）派生缓存代号。双方必须引用同一常量，
	// 避免键名漂移导致 L0 读到空代号、重导后缓存失效静默失效并返回过期内容。
	MetaKeyContentHash = "content_hash"
)

// ErrNotFound 表示请求的条目 / 组 / 元数据不存在。
var ErrNotFound = errors.New("deckstore: not found")

// Rand 是随机源接口，由上层（dice 的 DiceSource）适配注入。
type Rand interface {
	Int63n(n int64) int64
}

// Logger 是可选日志接口；为 nil 时静默。
type Logger interface {
	Errorf(format string, args ...any)
}

// GroupMeta 是一个牌组的内存索引元数据。
type GroupMeta struct {
	ItemCount   int
	TotalWeight int64
	Weighted    bool
	ChunkStarts []int // 每块首条全局下标，升序
}

// Store 是牌堆分块存储的只读接口 + 导入 / 生命周期入口。
//
// 内存缓存提示：实现用 github.com/maypok86/otter 按字节记账缓存解压后的块与权重。
// otter 的单条成本上限约为容量的 1/10（ChunkCacheCap/10≈410KB、WeightCacheCap/10≈205KB）；
// 超过上限的条目会被丢弃、每次读取都回源读库+解压（性能退化，非正确性错误，被丢弃的次数会被计数并限频告警）。
// 常规块目标 ≤128KB（msgpack 载荷）不受影响；仅当存在超大单条（>≈410KB）或超大权重数组（>≈5.2 万条/组）时退化。
type Store interface {
	// BeginImport 开启一个写事务，先 DELETE 该 deck 的全部旧行。
	BeginImport(deckID string) (Import, error)
	// GroupMeta 返回组的内存索引元数据；未命中时 ok=false。
	GroupMeta(deckID, group string) (GroupMeta, bool)
	// Item 按全局下标读取一条；未命中返回 ErrNotFound。
	Item(deckID, group string, idx int) (string, error)
	// Weights 返回整组权重；非加权组返回 (nil, nil)。
	Weights(deckID, group string) ([]uint32, error)
	// GroupItems 物化整组（云基底用）。
	GroupItems(deckID, group string) []string
	// Meta 读取通用 KV 元数据；未命中返回 false。
	Meta(deckID, key string) ([]byte, bool)
	// Groups 返回 deck 的全部组名（未变牌堆重建内存索引用）。
	Groups(deckID string) ([]string, error)
	// DeckIDs 返回所有已存在 deck 的标识。
	DeckIDs() ([]string, error)
	// DeleteDeck 删除一个 deck 的全部行。
	DeleteDeck(deckID string) error
	// ReplaceIndex 原子替换内存索引；键 = Key(deckID, group)。
	ReplaceIndex(m map[string]GroupMeta)
	// Close 关闭底层连接。
	Close() error
}

// Import 是一次牌堆导入事务。
type Import interface {
	// BeginGroup 开始写一个组。
	BeginGroup(name string) GroupWriter
	// SetMeta 写入一条 KV 元数据。
	SetMeta(key string, value []byte)
	// Commit 提交事务并更新内存索引。
	Commit() error
	// Abort 回滚事务。
	Abort()
}

// GroupWriter 向一个组内追加条目并落块。
type GroupWriter interface {
	// Add 追加一条；weight==0 归一为 1；满 256 条或 128KB 即落一块。
	Add(text string, weight uint32)
	// Finish 落完剩余块、写入组元数据行并返回 GroupMeta。
	Finish() (GroupMeta, error)
}

// Open 打开文件库：读连接 deferred / 写连接 immediate。
func Open(path string, log Logger) (Store, error) {
	return openSQLite(path, log)
}

// OpenMemory 打开内存库（:memory:，单连接，读写同实例）。
func OpenMemory(log Logger) (Store, error) {
	return openSQLiteMemory(log)
}
