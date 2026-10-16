//nolint:testpackage
package dice

// 本文件是牌堆存储的验收探针（基准 + 环境变量门控的真实数据验收），
// 对应设计 §11/§14：本地牌堆文本不常驻、额外磁盘 ≤ 原牌堆总量 45%、稳态 HeapAlloc 增量 ≤ 8MB。
// 探针只读原文件，且库文件始终落在 t.TempDir()/b.TempDir() 内，不写真实数据目录。

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/zap"

	"sealdice-core/utils/deckstore"
)

const (
	// deckAcceptanceMemBudget 是设计 §11 的稳态额外内存预算（HeapAlloc 增量）。
	deckAcceptanceMemBudget = 8 << 20
	// deckAcceptanceDiskRatio 是设计 §11/§14 的额外磁盘上限（相对原牌堆文件总量）。
	deckAcceptanceDiskRatio = 0.45

	// 合成牌堆规模：多组、单组远超 256 条块上限，总量约 20MB。
	synthDeckGroups        = 16
	synthDeckItemsPerGroup = 6000
	synthDeckItemBytes     = 200
)

// benchDeckDrawSink 防止编译器优化掉抽取结果。
var benchDeckDrawSink string

// synthDeckWords 是合成牌面的词组池，避开 {}/%/$ 等格式化元字符，
// 保证抽取文本原样返回、不引入递归解析噪声。多词组混排避免 zstd 遇到病态可压缩数据。
var synthDeckWords = []string{
	"春风", "夏雨", "秋叶", "冬雪", "青山", "碧水", "孤舟", "远客",
	"古道", "长亭", "明月", "残阳", "疏影", "暗香", "清泉", "幽谷",
	"寒江", "暮色", "晨光", "晚照", "飞鸟", "游鱼", "苍松", "翠竹",
}

// syntheticItemText 生成长度约 targetBytes 的确定性中文牌面（同一 (group, i) 恒等）。
func syntheticItemText(groupIdx, i, targetBytes int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "第%06d条-", i)
	for k := 0; b.Len() < targetBytes; k++ {
		b.WriteString(synthDeckWords[(i*3+k*7+groupIdx*11)%len(synthDeckWords)])
	}
	return b.String()
}

// writeSyntheticDiceE 生成 groups×itemsPerGroup 条、总量约 groups*itemsPerGroup*itemBytes 的
// DiceE JSON 牌堆，返回写入字节数。
func writeSyntheticDiceE(tb testing.TB, path string, groups, itemsPerGroup, itemBytes int) int64 {
	tb.Helper()
	f, err := os.Create(path)
	if err != nil {
		tb.Fatalf("创建合成牌堆 %s: %v", path, err)
	}
	w := bufio.NewWriterSize(f, 1<<20)
	var n int64
	fail := func(err error) {
		_ = f.Close()
		tb.Fatalf("写合成牌堆 %s: %v", path, err)
	}
	ws := func(s string) {
		m, err := w.WriteString(s)
		n += int64(m)
		if err != nil {
			fail(err)
		}
	}

	groupNames := make([]string, groups)
	ws(`{"_title":["合成大牌堆"],"_keys":[`)
	for gi := range groupNames {
		groupNames[gi] = fmt.Sprintf("组%02d", gi)
		if gi > 0 {
			ws(",")
		}
		ws(strconv.Quote(groupNames[gi]))
	}
	ws("]")
	for gi, g := range groupNames {
		ws(",")
		ws(strconv.Quote(g))
		ws(":[")
		for i := range itemsPerGroup {
			if i > 0 {
				ws(",")
			}
			ws(strconv.Quote(syntheticItemText(gi, i, itemBytes)))
		}
		ws("]")
	}
	ws("}\n")
	if err := w.Flush(); err != nil {
		fail(err)
	}
	if err := f.Close(); err != nil {
		tb.Fatalf("关闭合成牌堆 %s: %v", path, err)
	}
	return n
}

// setupDeckBenchDir 在 root 下建出 DecksDetect 约定的 data/decks 目录，返回其路径。
func setupDeckBenchDir(tb testing.TB, root string) string {
	tb.Helper()
	decksDir := filepath.Join(root, "data", "decks")
	if err := os.MkdirAll(decksDir, 0o755); err != nil {
		tb.Fatalf("MkdirAll(%s): %v", decksDir, err)
	}
	return decksDir
}

// setDeckStorePathForTest 把库路径钉在测试临时目录内，并在结束时不污染真实数据目录。
func setDeckStorePathForTest(tb testing.TB, dbPath string) {
	tb.Helper()
	oldPath := deckStorePathFn
	deckStorePathFn = func() string { return dbPath }
	tb.Cleanup(func() { deckStorePathFn = oldPath })
}

// deckHeapAlloc 连续 GC 后返回当前 HeapAlloc（近似稳态常驻堆）。
func deckHeapAlloc() uint64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc
}

// isDeckSourceFile 复刻 DecksDetect 的扩展名规则（info.yaml 除外）。
func isDeckSourceFile(path string) bool {
	if strings.EqualFold(filepath.Base(path), "info.yaml") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".jsonc", ".yml", ".yaml", ".toml", "":
		return true
	}
	return false
}

// copyDeckFile 是跨文件系统时的兜底复制。
func copyDeckFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// linkOrCopyDeckFiles 把真实牌堆目录里的牌堆文件（含 .deck 压缩包）硬链/复制到 dstDir，
// 保持目录结构，且绝不写入源目录。返回其中"牌堆源文件"的字节总量（.deck 不计）。
func linkOrCopyDeckFiles(t *testing.T, srcDir, dstDir string) int64 {
	t.Helper()
	var total int64
	err := filepath.Walk(srcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(srcDir, path)
		if rerr != nil {
			return rerr
		}
		if info.IsDir() {
			switch strings.ToLower(info.Name()) {
			case "assets", "images":
				return fs.SkipDir
			}
			return os.MkdirAll(filepath.Join(dstDir, rel), 0o755)
		}
		isDeck := isDeckSourceFile(path)
		isArchive := strings.EqualFold(filepath.Ext(path), ".deck")
		if !isDeck && !isArchive {
			return nil
		}
		dst := filepath.Join(dstDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Link(path, dst); err != nil {
			if cerr := copyDeckFile(path, dst); cerr != nil {
				return fmt.Errorf("硬链失败(%w)且复制失败(%w): %s", err, cerr, path)
			}
		}
		if isDeck {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		t.Fatalf("准备牌堆工作目录失败: %v", err)
	}
	return total
}

// deckFilesTotalSize 统计目录内所有"牌堆源文件"的字节总量（.deck 压缩包与 assets/images 不计）。
func deckFilesTotalSize(t *testing.T, dir string) int64 {
	t.Helper()
	var total int64
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch strings.ToLower(info.Name()) {
			case "assets", "images":
				return fs.SkipDir
			}
			return nil
		}
		if !isDeckSourceFile(path) {
			return nil
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		t.Fatalf("统计牌堆文件大小失败: %v", err)
	}
	return total
}

// deckDrawAcrossGroups 在已 reload 的库上轮转各组抽取 n 次（放回），返回实际抽取次数。
// executeDeck 的 deckDepth 只增不减（现状行为），每次抽取前清零以模拟独立命令。
func deckDrawAcrossGroups(t *testing.T, st deckstore.Store, d *Dice, ctx *MsgContext, n int) int {
	t.Helper()
	type target struct {
		di    *DeckInfo
		group string
	}
	var targets []target
	for _, di := range d.DeckList {
		groups, err := st.Groups(di.DeckKey)
		if err != nil {
			t.Fatalf("Groups(%s): %v", di.Name, err)
		}
		for _, g := range groups {
			if meta, ok := st.GroupMeta(di.DeckKey, g); ok && meta.ItemCount > 0 {
				targets = append(targets, target{di: di, group: g})
			}
		}
	}
	if len(targets) == 0 {
		t.Fatal("没有任何可抽取的牌组")
	}
	for i := range n {
		tg := targets[i%len(targets)]
		ctx.deckDepth = 0
		if _, err := executeDeck(ctx, tg.di, tg.group, false); err != nil {
			t.Fatalf("抽取 %s/%s: %v", tg.di.Name, tg.group, err)
		}
	}
	return n
}

// BenchmarkDeckReloadLarge 测 ~20MB 合成多组 DiceE 牌堆的全量 reload（解析+切块+zstd+落库+建索引）。
//
// 每轮使用全新库文件：DeckReload 是增量的，同一库重复 reload 会跳过未变文件，
// 无法反映"首次加载大牌堆"的成本，故每轮换库以强制走完整导入路径。
func BenchmarkDeckReloadLarge(b *testing.B) {
	root := b.TempDir()
	decksDir := setupDeckBenchDir(b, root)
	totalBytes := writeSyntheticDiceE(
		b, filepath.Join(decksDir, "big.json"),
		synthDeckGroups, synthDeckItemsPerGroup, synthDeckItemBytes,
	)
	b.Chdir(root)

	oldPath := deckStorePathFn
	var iter int
	deckStorePathFn = func() string {
		return filepath.Join(root, "data", "_deck_cache", fmt.Sprintf("decks-%d.db", iter))
	}
	b.Cleanup(func() { deckStorePathFn = oldPath })

	b.SetBytes(totalBytes)
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		iter = i
		d := &Dice{Logger: zap.NewNop().Sugar()}
		DeckReload(d)
		if _, ok := d.DeckStore.(*deckStoreBackend); !ok {
			b.Fatalf("后端 = %T, want *deckStoreBackend（文件库）", d.DeckStore)
		}
		if len(d.DeckList) == 0 {
			b.Fatal("DeckList 为空")
		}
		if err := d.DeckStore.Close(); err != nil {
			b.Fatalf("Close: %v", err)
		}
	}
}

// BenchmarkDeckDraw 测 reload 后连续抽取单条的 ns/op（读索引 → 命中块缓存/读单块）。
func BenchmarkDeckDraw(b *testing.B) {
	root := b.TempDir()
	decksDir := setupDeckBenchDir(b, root)
	writeSyntheticDiceE(
		b, filepath.Join(decksDir, "big.json"),
		synthDeckGroups, synthDeckItemsPerGroup, synthDeckItemBytes,
	)
	b.Chdir(root)
	setDeckStorePathForTest(b, filepath.Join(root, "data", "_deck_cache", "decks.db"))

	d := &Dice{Logger: zap.NewNop().Sugar()}
	b.Cleanup(func() {
		if d.DeckStore != nil {
			_ = d.DeckStore.Close()
		}
	})
	DeckReload(d)

	backend, ok := d.DeckStore.(*deckStoreBackend)
	if !ok {
		b.Fatalf("后端 = %T, want *deckStoreBackend（文件库）", d.DeckStore)
	}
	if len(d.DeckList) == 0 {
		b.Fatal("DeckList 为空")
	}
	di := d.DeckList[0]
	st := backend.Store()
	groups, err := st.Groups(di.DeckKey)
	if err != nil {
		b.Fatalf("Groups: %v", err)
	}
	if len(groups) == 0 {
		b.Fatal("牌堆没有任何组")
	}
	// 选条目最多的组：DiceE 会把 _title/_keys 等元数据数组也建成组，不能取 groups[0]。
	group := groups[0]
	meta, ok := st.GroupMeta(di.DeckKey, group)
	if !ok {
		b.Fatalf("组 %q 无元数据", group)
	}
	for _, g := range groups[1:] {
		if m, ok := st.GroupMeta(di.DeckKey, g); ok && m.ItemCount > meta.ItemCount {
			group, meta = g, m
		}
	}
	if meta.ItemCount == 0 {
		b.Fatalf("组 %q 无条目", group)
	}
	if len(meta.ChunkStarts) < 2 {
		b.Fatalf("组 %q 只有 %d 块，合成数据未跨块（期望每组件数 > %d）",
			group, len(meta.ChunkStarts), deckstore.ChunkMaxItems)
	}

	ctx := &MsgContext{Dice: d}
	ctx.diceRandSrc = newPCGDiceSource(drawTestSeed)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ctx.deckDepth = 0
		got, err := executeDeck(ctx, di, group, false)
		if err != nil {
			b.Fatalf("executeDeck: %v", err)
		}
		benchDeckDrawSink = got
	}
}

// TestDeckMemoryAcceptance 是真实牌堆的内存/磁盘验收探针，仅当 DECK_ACCEPTANCE_DIR
// 指向真实牌堆目录时运行，否则 Skip。
//
// 断言（设计 §11/§14）：
//   - 稳态 HeapAlloc 增量（GC 后基线 → reload + 抽 50 次 → GC 后）≤ 8MB；
//   - t.TempDir() 内 decks.db ≤ 目录内牌堆文件总大小的 45%。
//
// 全程只读原文件：库路径被覆盖到 t.TempDir()，不写真实数据目录。
func TestDeckMemoryAcceptance(t *testing.T) {
	srcDir := os.Getenv("DECK_ACCEPTANCE_DIR")
	if srcDir == "" {
		t.Skip("未设置 DECK_ACCEPTANCE_DIR，跳过真实牌堆内存/磁盘验收")
	}
	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("DECK_ACCEPTANCE_DIR=%q 不是有效目录: %v", srcDir, err)
	}

	work := t.TempDir()
	decksDir := setupDeckBenchDir(t, work)
	if linked := linkOrCopyDeckFiles(t, srcDir, decksDir); linked == 0 {
		t.Fatalf("DECK_ACCEPTANCE_DIR=%q 下没有可识别的牌堆文件", srcDir)
	}

	t.Chdir(work)
	setDeckStorePathForTest(t, filepath.Join(work, "data", "_deck_cache", "decks.db"))

	d := &Dice{Logger: zap.NewNop().Sugar()}
	closed := false
	t.Cleanup(func() {
		if !closed && d.DeckStore != nil {
			_ = d.DeckStore.Close()
		}
	})

	base := deckHeapAlloc()
	DeckReload(d)

	backend, ok := d.DeckStore.(*deckStoreBackend)
	if !ok {
		t.Fatalf("后端 = %T, want *deckStoreBackend（文件库）；真实牌堆验收不得降级", d.DeckStore)
	}
	if len(d.DeckList) == 0 {
		t.Fatal("DeckList 为空，未加载到牌堆")
	}
	for _, di := range d.DeckList {
		if di.DeckItems != nil {
			t.Fatalf("牌堆 %q 的 DeckItems 非 nil，文本未落盘", di.Name)
		}
	}

	ctx := &MsgContext{Dice: d}
	ctx.diceRandSrc = newPCGDiceSource(drawTestSeed)
	draws := deckDrawAcrossGroups(t, backend.Store(), d, ctx, 50)

	after := deckHeapAlloc()
	delta := int64(after) - int64(base)
	t.Logf("HeapAlloc: base=%d after=%d delta=%d (%.2f MB)；抽取 %d 次；牌堆 %d 个",
		base, after, delta, float64(delta)/(1<<20), draws, len(d.DeckList))
	if delta > deckAcceptanceMemBudget {
		t.Fatalf("稳态内存增量 %d 字节 (%.2f MB) 超过预算 %d 字节 (8 MB)",
			delta, float64(delta)/(1<<20), deckAcceptanceMemBudget)
	}

	// 关闭库：触发 WAL checkpoint 合并进主库文件，再测量持久化后的大小。
	if err2 := d.DeckStore.Close(); err2 != nil {
		t.Fatalf("Close: %v", err2)
	}
	closed = true
	d.DeckStore = nil

	dbPath := filepath.Join(work, "data", "_deck_cache", "decks.db")
	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat %s: %v", dbPath, err)
	}
	dbSize := dbInfo.Size()
	totalBytes := deckFilesTotalSize(t, decksDir)
	if totalBytes <= 0 {
		t.Fatalf("未统计到牌堆文件总大小: %s", decksDir)
	}
	ratio := float64(dbSize) / float64(totalBytes)
	t.Logf("decks.db=%d 字节, 牌堆文件合计=%d 字节, 比例=%.1f%%（上限 45%%）",
		dbSize, totalBytes, ratio*100)
	if dbSize > int64(float64(totalBytes)*deckAcceptanceDiskRatio) {
		t.Fatalf("decks.db=%d 字节 (%.1f%%) 超过牌堆总量 %d 字节的 45%%",
			dbSize, ratio*100, totalBytes)
	}
}
