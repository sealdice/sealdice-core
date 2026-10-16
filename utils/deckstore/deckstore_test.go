package deckstore_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"sealdice-core/utils/dboperator/engine/sqlite"
	"sealdice-core/utils/deckstore"
)

// OpenForTest 在临时目录下打开一个文件库，测试结束时自动关闭。
func OpenForTest(t *testing.T, dir string) deckstore.Store {
	t.Helper()
	s, err := deckstore.Open(filepath.Join(dir, "decks.db"), nil)
	if err != nil {
		t.Fatalf("deckstore.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestChunkRoundtrip(t *testing.T) {
	s := OpenForTest(t, t.TempDir())

	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	g := im.BeginGroup("g1")

	// 700 条（跨 256 条边界），其中 i%7==0 的条目 weight=3；文本含中文与换行。
	want := make([]string, 700)
	var wantTotal int64
	for i := range 700 {
		want[i] = fmt.Sprintf("第%d条\n内容😀", i)
		w := uint32(1)
		if i%7 == 0 {
			w = 3
		}
		wantTotal += int64(w)
		g.Add(want[i], w)
	}

	m, err := g.Finish()
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if m.ItemCount != 700 {
		t.Fatalf("ItemCount = %d, want 700", m.ItemCount)
	}
	if m.TotalWeight != wantTotal {
		t.Fatalf("TotalWeight = %d, want %d", m.TotalWeight, wantTotal)
	}
	if !m.Weighted {
		t.Fatal("Weighted = false, want true")
	}
	if len(m.ChunkStarts) < 3 {
		t.Fatalf("len(ChunkStarts) = %d, want >= 3", len(m.ChunkStarts))
	}

	im.SetMeta("content_hash", []byte("h1"))
	if err := im.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	for i := range 700 {
		got, err := s.Item("k", "g1", i)
		if err != nil {
			t.Fatalf("Item(%d): %v", i, err)
		}
		if got != want[i] {
			t.Fatalf("Item(%d) = %q, want %q", i, got, want[i])
		}
	}

	v, ok := s.Meta("k", "content_hash")
	if !ok {
		t.Fatal("Meta(content_hash) not found")
	}
	if string(v) != "h1" {
		t.Fatalf("Meta(content_hash) = %q, want h1", v)
	}
}

func TestChunkBoundaryByBytes(t *testing.T) {
	s := OpenForTest(t, t.TempDir())

	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	g := im.BeginGroup("g1")

	// 单条 >128KB ×3：按字节切块（每块 ≤1 条）。
	big := strings.Repeat("A", deckstore.ChunkMaxBytes+100)
	want := []string{big, big, big}
	for _, txt := range want {
		g.Add(txt, 0)
	}

	m, err := g.Finish()
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if m.ItemCount != 3 {
		t.Fatalf("ItemCount = %d, want 3", m.ItemCount)
	}
	if len(m.ChunkStarts) != 3 {
		t.Fatalf("len(ChunkStarts) = %d, want 3（每条自成一块）", len(m.ChunkStarts))
	}
	if err := im.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	for i := range 3 {
		got, err := s.Item("k", "g1", i)
		if err != nil {
			t.Fatalf("Item(%d): %v", i, err)
		}
		if got != want[i] {
			t.Fatalf("Item(%d) mismatch", i)
		}
	}
}

func TestUniformWeightsNull(t *testing.T) {
	s := OpenForTest(t, t.TempDir())

	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	g := im.BeginGroup("g1")
	for i := range 10 {
		g.Add(fmt.Sprintf("item%d", i), 0)
	}

	m, err := g.Finish()
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if m.Weighted {
		t.Fatal("Weighted = true, want false（全默认权重）")
	}
	if err2 := im.Commit(); err2 != nil {
		t.Fatalf("Commit: %v", err2)
	}

	w, err := s.Weights("k", "g1")
	if err != nil {
		t.Fatalf("Weights: %v", err)
	}
	if w != nil {
		t.Fatalf("Weights = %v, want nil", w)
	}
}

func TestUserVersionMismatchRebuilds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "decks.db")

	// 建库并导入数据。
	s, err := deckstore.Open(path, nil)
	if err != nil {
		t.Fatalf("deckstore.Open: %v", err)
	}
	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	g := im.BeginGroup("g1")
	g.Add("hello", 0)
	if _, err2 := g.Finish(); err2 != nil {
		t.Fatalf("Finish: %v", err2)
	}
	if err2 := im.Commit(); err2 != nil {
		t.Fatalf("Commit: %v", err2)
	}
	_ = s.Close()

	// 手动把 user_version 改为不匹配的值。
	db, err := sqlite.SQLiteDBInit(path, false)
	if err != nil {
		t.Fatalf("SQLiteDBInit: %v", err)
	}
	if err2 := db.Exec("PRAGMA user_version = 99").Error; err2 != nil {
		t.Fatalf("set user_version: %v", err2)
	}
	if pool, e := db.DB(); e == nil {
		_ = pool.Close()
	}

	// 重开：不匹配应触发整库重建，旧数据清空。
	s2, err := deckstore.Open(path, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s2.Close() }()

	ids, err := s2.DeckIDs()
	if err != nil {
		t.Fatalf("DeckIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("DeckIDs = %v, want 空（旧数据应已清空）", ids)
	}
	if _, ok := s2.GroupMeta("k", "g1"); ok {
		t.Fatal("GroupMeta 应未命中（旧数据应已清空）")
	}

	// 可重新导入。
	im2, err := s2.BeginImport("k")
	if err != nil {
		t.Fatalf("re-BeginImport: %v", err)
	}
	g2 := im2.BeginGroup("g1")
	g2.Add("world", 0)
	if _, err2 := g2.Finish(); err2 != nil {
		t.Fatalf("re-Finish: %v", err2)
	}
	if err2 := im2.Commit(); err2 != nil {
		t.Fatalf("re-Commit: %v", err2)
	}
	got, err := s2.Item("k", "g1", 0)
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if got != "world" {
		t.Fatalf("Item = %q, want world", got)
	}
}

func TestOpenMemoryWriteThenRead(t *testing.T) {
	s, err := deckstore.OpenMemory(nil)
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer func() { _ = s.Close() }()

	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	g := im.BeginGroup("g1")
	want := make([]string, 300)
	for i := range 300 {
		want[i] = fmt.Sprintf("item%d", i)
		g.Add(want[i], 0)
	}
	if _, err := g.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if err := im.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// 写后立即可读（钉住 :memory: 单连接）。
	for i := range 300 {
		got, err := s.Item("k", "g1", i)
		if err != nil {
			t.Fatalf("Item(%d): %v", i, err)
		}
		if got != want[i] {
			t.Fatalf("Item(%d) = %q, want %q", i, got, want[i])
		}
	}
}

func TestDeckIDsAndDeleteDeck(t *testing.T) {
	s := OpenForTest(t, t.TempDir())

	for _, id := range []string{"d1", "d2"} {
		im, err := s.BeginImport(id)
		if err != nil {
			t.Fatalf("BeginImport(%s): %v", id, err)
		}
		g := im.BeginGroup("g")
		g.Add("item", 0)
		if _, err := g.Finish(); err != nil {
			t.Fatalf("Finish(%s): %v", id, err)
		}
		if err := im.Commit(); err != nil {
			t.Fatalf("Commit(%s): %v", id, err)
		}
	}

	ids, err := s.DeckIDs()
	if err != nil {
		t.Fatalf("DeckIDs: %v", err)
	}
	if len(ids) != 2 || !containsStr(ids, "d1") || !containsStr(ids, "d2") {
		t.Fatalf("DeckIDs = %v, want [d1 d2]", ids)
	}

	if err2 := s.DeleteDeck("d1"); err2 != nil {
		t.Fatalf("DeleteDeck: %v", err2)
	}

	ids, err = s.DeckIDs()
	if err != nil {
		t.Fatalf("DeckIDs after delete: %v", err)
	}
	if len(ids) != 1 || ids[0] != "d2" {
		t.Fatalf("DeckIDs after delete = %v, want [d2]", ids)
	}
	if _, ok := s.GroupMeta("d1", "g"); ok {
		t.Fatal("GroupMeta(d1, g) 应未命中")
	}
	if _, err := s.Item("d1", "g", 0); !errors.Is(err, deckstore.ErrNotFound) {
		t.Fatalf("Item(d1) err = %v, want ErrNotFound", err)
	}
}

func TestReimportSameDeckInvalidatesCache(t *testing.T) {
	s := OpenForTest(t, t.TempDir())

	const n = 300 // 跨 256 块边界，产生多块
	importGroup := func(prefix string, weightFn func(i int) uint32) {
		im, err := s.BeginImport("k")
		if err != nil {
			t.Fatalf("BeginImport: %v", err)
		}
		g := im.BeginGroup("g1")
		for i := range n {
			g.Add(fmt.Sprintf("%s%d", prefix, i), weightFn(i))
		}
		if _, err := g.Finish(); err != nil {
			t.Fatalf("Finish: %v", err)
		}
		if err := im.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}

	// 内容 A：加权，权重全 5。读取若干 idx 预热块缓存，再读权重预热权重缓存。
	importGroup("A", func(i int) uint32 { return 5 })
	for _, idx := range []int{0, 299} {
		got, err := s.Item("k", "g1", idx)
		if err != nil || got != fmt.Sprintf("A%d", idx) {
			t.Fatalf("A Item(%d) = %q, %v", idx, got, err)
		}
	}
	wA, err := s.Weights("k", "g1")
	if err != nil {
		t.Fatalf("A Weights: %v", err)
	}
	if len(wA) != n || wA[0] != 5 || wA[n-1] != 5 {
		t.Fatalf("A Weights 应全为 5，got len=%d first=%d last=%d", len(wA), wA[0], wA[n-1])
	}

	// 重导同一 deck，内容 B：文本与权重都不同。
	importGroup("B", func(i int) uint32 { return uint32(i%3) + 1 })

	for _, idx := range []int{0, 299} {
		got, err2 := s.Item("k", "g1", idx)
		if err2 != nil || got != fmt.Sprintf("B%d", idx) {
			t.Fatalf("B Item(%d) = %q, %v，读到旧内容 A%d", idx, got, err2, idx)
		}
	}
	wB, err := s.Weights("k", "g1")
	if err != nil {
		t.Fatalf("B Weights: %v", err)
	}
	if len(wB) != n {
		t.Fatalf("B Weights len = %d, want %d", len(wB), n)
	}
	for i := range n {
		want := uint32(i%3) + 1
		if wB[i] != want {
			t.Fatalf("B weight[%d] = %d, want %d（读到旧权重）", i, wB[i], want)
		}
	}
}

func TestGroups(t *testing.T) {
	s := OpenForTest(t, t.TempDir())

	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	for _, name := range []string{"g1", "g2", "g3"} {
		g := im.BeginGroup(name)
		g.Add("item", 0)
		if _, err2 := g.Finish(); err2 != nil {
			t.Fatalf("Finish(%s): %v", name, err2)
		}
	}
	if err2 := im.Commit(); err2 != nil {
		t.Fatalf("Commit: %v", err2)
	}

	names, err := s.Groups("k")
	if err != nil {
		t.Fatalf("Groups: %v", err)
	}
	if len(names) != 3 || !containsStr(names, "g1") || !containsStr(names, "g2") || !containsStr(names, "g3") {
		t.Fatalf("Groups = %v, want 包含 [g1 g2 g3]", names)
	}

	// 不存在的 deck 返回空列表而非 error。
	names, err = s.Groups("missing")
	if err != nil {
		t.Fatalf("Groups(missing): %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("Groups(missing) = %v, want 空", names)
	}
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// TestReopenStoreReloadsGroupMeta 直接钉住 loadGroupMeta 兜底：重开既有文件库（新 store、索引为空）
// 后，GroupMeta 必须从 deck_group 表还原出多块 ChunkStarts，且 Item 跨块边界仍可读。
func TestReopenStoreReloadsGroupMeta(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "decks.db")

	// 建立库：300 条（跨 256 条块边界，产生多块）+ 一个加权组。
	s, err := deckstore.Open(path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	g := im.BeginGroup("g1")
	for i := range 300 {
		g.Add(fmt.Sprintf("item%d", i), 0)
	}
	if _, err2 := g.Finish(); err2 != nil {
		t.Fatalf("Finish(g1): %v", err2)
	}
	gw := im.BeginGroup("g2")
	gw.Add("::3::a", 3)
	gw.Add("b", 1)
	if _, err2 := gw.Finish(); err2 != nil {
		t.Fatalf("Finish(g2): %v", err2)
	}
	if err2 := im.Commit(); err2 != nil {
		t.Fatalf("Commit: %v", err2)
	}
	if err2 := s.Close(); err2 != nil {
		t.Fatalf("Close: %v", err2)
	}

	// 重新打开同一路径：全新 store，内存索引为空。
	s2, err := deckstore.Open(path, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s2.Close() }()

	gm, ok := s2.GroupMeta("k", "g1")
	if !ok {
		t.Fatal("重开后 GroupMeta(k, g1) 未命中（loadGroupMeta 兜底未生效）")
	}
	if gm.ItemCount != 300 {
		t.Fatalf("ItemCount = %d, want 300", gm.ItemCount)
	}
	if len(gm.ChunkStarts) < 2 {
		t.Fatalf("ChunkStarts = %v, want >= 2 块（跨 256 边界）", gm.ChunkStarts)
	}

	// 跨块边界读取，证明 chunk_starts 还原正确。
	for _, idx := range []int{0, 255, 256, 299} {
		got, err2 := s2.Item("k", "g1", idx)
		if err2 != nil || got != fmt.Sprintf("item%d", idx) {
			t.Fatalf("Item(%d) = %q, %v, want item%d", idx, got, err2, idx)
		}
	}

	// 加权组权重可用（Weighted 标志 + weight_blob 还原）。
	gm2, ok := s2.GroupMeta("k", "g2")
	if !ok {
		t.Fatal("重开后 GroupMeta(k, g2) 未命中")
	}
	if !gm2.Weighted || gm2.TotalWeight != 4 {
		t.Fatalf("GroupMeta(k, g2) = %+v, want Weighted=true TotalWeight=4", gm2)
	}
	w, err := s2.Weights("k", "g2")
	if err != nil {
		t.Fatalf("Weights(k, g2): %v", err)
	}
	if len(w) != 2 || w[0] != 3 || w[1] != 1 {
		t.Fatalf("Weights = %v, want [3 1]", w)
	}
}
