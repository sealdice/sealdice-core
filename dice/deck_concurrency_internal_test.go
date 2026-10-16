package dice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// concurrencyFixture 生成第 n 版牌堆内容：同名「组」，两组条目集各 5 条。
// 两版条目集不同但数量相同，保证 reload 每次都会改写文件并触发重建。
func concurrencyFixture(n int) string {
	if n%2 == 0 {
		return `{"_title":["并发牌堆"],"组":["甲1","甲2","甲3","甲4","甲5"]}`
	}
	return `{"_title":["并发牌堆"],"组":["乙1","乙2","乙3","乙4","乙5"]}`
}

// TestDeckConcurrentDrawAndReload 覆盖规格 §13：多消息上下文并发抽取 + reload 与抽取并发。
// 所有 reader 共享同一 Dice/DeckStore（各自独立 MsgContext），主 goroutine 反复改写夹具并
// DeckReload（原子换索引 + 清理缓存），验证 store 索引/LRU 加锁在 -race 下无数据竞争、无 panic。
// 已知并接受的窄窗口：换索引与缓存清理之间可能返回一次旧文本或瞬时 ErrNotFound，测试予以容忍。
func TestDeckConcurrentDrawAndReload(t *testing.T) {
	dir := t.TempDir()
	decksDir := filepath.Join(dir, "data", "decks")
	if err := os.MkdirAll(decksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	t.Chdir(dir)

	oldPath := deckStorePathFn
	deckStorePathFn = func() string { return filepath.Join(dir, "data", "_deck_cache", "decks.db") }
	t.Cleanup(func() { deckStorePathFn = oldPath })

	// 初始版本 + 首次 reload 建立共享 store。
	fixturePath := filepath.Join(decksDir, "concurrency.json")
	if err := os.WriteFile(fixturePath, []byte(concurrencyFixture(0)), 0o644); err != nil {
		t.Fatalf("写初始夹具: %v", err)
	}
	d := &Dice{Logger: zap.NewNop().Sugar()}
	DeckReload(d)
	t.Cleanup(func() {
		if d.DeckStore != nil {
			_ = d.DeckStore.Close()
		}
	})
	if len(d.DeckList) != 1 {
		t.Fatalf("DeckList len = %d, want 1", len(d.DeckList))
	}
	di := d.DeckList[0] // reader 共用的 DeckInfo 指针，reload 只重建新指针，不改写它。

	legal := map[string]bool{
		"甲1": true, "甲2": true, "甲3": true, "甲4": true, "甲5": true,
		"乙1": true, "乙2": true, "乙3": true, "乙4": true, "乙5": true,
	}

	const readers = 8
	const iterations = 200

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failures []string
	report := func(format string, args ...any) {
		mu.Lock()
		defer mu.Unlock()
		failures = append(failures, fmt.Sprintf(format, args...))
	}

	wg.Add(readers)
	for i := range readers {
		go func(seed int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					report("reader panic: %v", r)
				}
			}()
			ctx := &MsgContext{Dice: d}
			ctx.diceRandSrc = newPCGDiceSource(uint64(0xdead) + uint64(seed))
			// 预置 splitKey，跳过 InitSplitKey 的进程级 randSourceSplitKey 全局源
			// （那是消息管线的既有并发缺陷，与牌堆存储无关，测试不触发它）。
			ctx.SetSplitKey("concurrency")
			for range iterations {
				got, err := executeDeck(ctx, di, "组", false)
				if err != nil {
					continue // 接受瞬时窗口错误（ErrNotFound/牌组为空/抽取失败）
				}
				if !legal[got] {
					report("非法抽取结果 %q", got)
					return
				}
			}
		}(i)
	}

	// 主 goroutine 与 reader 并发：反复改写夹具并 reload（原子换索引）。
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range iterations {
			if err := os.WriteFile(fixturePath, []byte(concurrencyFixture(i+1)), 0o644); err != nil {
				report("改写夹具: %v", err)
				return
			}
			DeckReload(d)
		}
	}()

	wg.Wait()
	<-done

	if len(failures) > 0 {
		t.Fatalf("并发抽取/reload 失败 %d 条:\n%s", len(failures), strings.Join(failures, "\n"))
	}
}
