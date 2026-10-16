package dice

import (
	"path/filepath"
	"strings"
	"sync"

	ds "github.com/sealdice/dicescript"
	"go.uber.org/zap"

	"sealdice-core/utils/deckstore"
)

// deckBackend 是 dice 侧统一的牌堆内容后端（含 legacy），全部以 deckKey 寻址。
type deckBackend interface {
	GroupMeta(deckKey, group string) (deckstore.GroupMeta, bool)
	Item(deckKey, group string, idx int) (string, error)
	Weights(deckKey, group string) ([]uint32, error)
	GroupItems(deckKey, group string) []string
	Close() error
}

// deckStoreBackend 是 deckstore.Store 的薄封装。
type deckStoreBackend struct{ st deckstore.Store }

func (b *deckStoreBackend) GroupMeta(deckKey, group string) (deckstore.GroupMeta, bool) {
	return b.st.GroupMeta(deckKey, group)
}

func (b *deckStoreBackend) Item(deckKey, group string, idx int) (string, error) {
	return b.st.Item(deckKey, group, idx)
}

func (b *deckStoreBackend) Weights(deckKey, group string) ([]uint32, error) {
	return b.st.Weights(deckKey, group)
}

func (b *deckStoreBackend) GroupItems(deckKey, group string) []string {
	return b.st.GroupItems(deckKey, group)
}

func (b *deckStoreBackend) Close() error {
	return b.st.Close()
}

// Store 暴露底层 deckstore.Store，供导入 / 清理使用。
func (b *deckStoreBackend) Store() deckstore.Store {
	return b.st
}

// 测试钩子：文件库路径与内存库构造器，测试通过替换它们强制降级。
var deckStorePathFn = func() string { return filepath.Join("data", "_deck_cache", "decks.db") }

var newDeckStoreMemoryFn = func(log deckstore.Logger) (deckstore.Store, error) {
	return deckstore.OpenMemory(log)
}

// deckMemoryBackend 是 legacy 兜底：直接基于 DeckInfo.DeckItems。
type deckMemoryBackend struct {
	mu    sync.RWMutex
	decks map[string]*DeckInfo
	metas map[string]deckstore.GroupMeta // 惰性计算缓存，键 deckstore.Key
}

// Register 登记一个牌堆（reload 的 legacy 分支调用）。
func (b *deckMemoryBackend) Register(di *DeckInfo) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.decks == nil {
		b.decks = make(map[string]*DeckInfo)
	}
	b.decks[di.DeckKey] = di
	prefix := di.DeckKey + "\x00"
	for k := range b.metas {
		if strings.HasPrefix(k, prefix) {
			delete(b.metas, k)
		}
	}
}

func (b *deckMemoryBackend) GroupMeta(deckKey, group string) (deckstore.GroupMeta, bool) {
	key := deckstore.Key(deckKey, group)

	b.mu.RLock()
	if gm, ok := b.metas[key]; ok {
		b.mu.RUnlock()
		return gm, true
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()
	if gm, ok := b.metas[key]; ok {
		return gm, true
	}
	di, ok := b.decks[deckKey]
	if !ok {
		return deckstore.GroupMeta{}, false
	}
	items, ok := di.DeckItems[group]
	if !ok {
		return deckstore.GroupMeta{}, false
	}
	gm := computeDeckGroupMeta(items)
	if b.metas == nil {
		b.metas = make(map[string]deckstore.GroupMeta)
	}
	b.metas[key] = gm
	return gm, true
}

func (b *deckMemoryBackend) Item(deckKey, group string, idx int) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	di, ok := b.decks[deckKey]
	if !ok {
		return "", deckstore.ErrNotFound
	}
	items, ok := di.DeckItems[group]
	if !ok {
		return "", deckstore.ErrNotFound
	}
	if idx < 0 || idx >= len(items) {
		return "", deckstore.ErrNotFound
	}
	return items[idx], nil
}

func (b *deckMemoryBackend) Weights(deckKey, group string) ([]uint32, error) {
	b.mu.RLock()
	di, ok := b.decks[deckKey]
	if !ok {
		b.mu.RUnlock()
		return nil, deckstore.ErrNotFound
	}
	items, ok := di.DeckItems[group]
	b.mu.RUnlock()
	if !ok {
		return nil, deckstore.ErrNotFound
	}

	weights := make([]uint32, 0, len(items))
	weighted := false
	for _, s := range items {
		w, _ := extractWeight(s)
		if w > 1 {
			weighted = true
		}
		weights = append(weights, uint32(w))
	}
	if !weighted {
		return nil, nil
	}
	return weights, nil
}

func (b *deckMemoryBackend) GroupItems(deckKey, group string) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	di, ok := b.decks[deckKey]
	if !ok {
		return nil
	}
	return di.DeckItems[group]
}

func (b *deckMemoryBackend) Close() error { return nil }

// computeDeckGroupMeta 由 DeckItems 计算内存索引元数据（legacy 路径单块语义）。
func computeDeckGroupMeta(items []string) deckstore.GroupMeta {
	gm := deckstore.GroupMeta{ItemCount: len(items)}
	if len(items) > 0 {
		gm.ChunkStarts = []int{0}
	}
	for _, s := range items {
		w, _ := extractWeight(s)
		gm.TotalWeight += int64(w)
		if w > 1 {
			gm.Weighted = true
		}
	}
	return gm
}

// openDeckBackend 三级降级打开内容后端，返回后端与模式字符串（"file"/"memory-sqlite"/"legacy"）。
// log 可为 nil；不返回 error，逐级用 logger 记录失败原因。
func openDeckBackend(log *zap.SugaredLogger) (deckBackend, string) {
	var logr deckstore.Logger
	if log != nil {
		logr = log
	}

	if st, err := deckstore.Open(deckStorePathFn(), logr); err == nil {
		return &deckStoreBackend{st: st}, "file"
	} else if log != nil {
		log.Errorf("deckstore: 打开文件库失败，降级到内存 SQLite：%v", err)
	}

	if st, err := newDeckStoreMemoryFn(logr); err == nil {
		return &deckStoreBackend{st: st}, "memory-sqlite"
	} else if log != nil {
		log.Errorf("deckstore: 打开内存 SQLite 失败，降级到全内存后端：%v", err)
	}

	return &deckMemoryBackend{}, "legacy"
}

// diceSourceRand 把 DiceSource 适配为 deckstore.Rand。
type diceSourceRand struct{ src ds.DiceSource }

func (r diceSourceRand) Int63n(n int64) int64 {
	return int64(randIntnFromSource(r.src, int(n)))
}
