package deckstore

import (
	"strings"
	"testing"
)

// TestLargeChunkRejectedNotCached 验证单条成本超过 otter 单条上限（ChunkCacheCap/10）的块
// 不会进缓存：被拒次数被计数，但读取结果仍正确（性能退化，非正确性错误）。
func TestLargeChunkRejectedNotCached(t *testing.T) {
	dir := t.TempDir()
	store, err := openSQLite(dir+"/decks.db", nil)
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	s := store.(*sqliteStore)
	defer func() { _ = s.Close() }()

	big := strings.Repeat("A", ChunkCacheCap/10+1024)
	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	im.SetMeta(MetaKeyContentHash, []byte("h"))
	g := im.BeginGroup("g1")
	g.Add(big, 0)
	if _, err := g.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if err := im.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	for i := range 2 {
		got, err := s.Item("k", "g1", 0)
		if err != nil || got != big {
			t.Fatalf("第 %d 次 Item 读取失败: got len=%d err=%v", i+1, len(got), err)
		}
	}
	if n := s.cacheRejects.Load(); n != 2 {
		t.Fatalf("超大块每次读取都应被拒并计数，want 2, got %d", n)
	}
}

// TestLargeWeightsRejectedNotCached 验证权重数组成本超过 otter 单条上限（WeightCacheCap/10）时
// 不进缓存但可读。权重成本 = len(w)*4，故条目数需 ≥ WeightCacheCap/40 ≈ 5.2 万。
func TestLargeWeightsRejectedNotCached(t *testing.T) {
	dir := t.TempDir()
	store, err := openSQLite(dir+"/decks.db", nil)
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	s := store.(*sqliteStore)
	defer func() { _ = s.Close() }()

	const n = WeightCacheCap/10/4 + 2 // len(w)*4 > WeightCacheCap/10
	im, err := s.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	im.SetMeta(MetaKeyContentHash, []byte("h"))
	g := im.BeginGroup("g1")
	for i := range n {
		g.Add("x", 1+uint32(i%3)) // 加权组，权重数组非空且非全 1
	}
	if _, err := g.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if err := im.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	for i := range 2 {
		w, err := s.Weights("k", "g1")
		if err != nil || len(w) != n {
			t.Fatalf("第 %d 次 Weights 读取失败: len=%d err=%v", i+1, len(w), err)
		}
	}
	if got := s.cacheRejects.Load(); got != 2 {
		t.Fatalf("超大权重每次读取都应被拒并计数，want 2, got %d", got)
	}
}
