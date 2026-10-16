package deckstore

import "testing"

// TestDeckGenInvalidatesOldCacheKeys 直接验证代号键的失效语义：
// 重导后代号变化，旧代号的缓存键不可达（未 Clear），读取必命中新代号键、返回新内容。
func TestDeckGenInvalidatesOldCacheKeys(t *testing.T) {
	dir := t.TempDir()
	store, err := openSQLite(dir+"/decks.db", nil)
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	s := store.(*sqliteStore)
	defer func() { _ = s.Close() }()

	importDeck := func(hash, text string) {
		t.Helper()
		im, err := s.BeginImport("k")
		if err != nil {
			t.Fatalf("BeginImport: %v", err)
		}
		im.SetMeta(genMetaKey, []byte(hash))
		g := im.BeginGroup("g1")
		g.Add(text, 0)
		if _, err := g.Finish(); err != nil {
			t.Fatalf("Finish: %v", err)
		}
		if err := im.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}

	importDeck("hash-A", "item-A")
	// 预热旧代号的块缓存键。
	if got, err := s.Item("k", "g1", 0); err != nil || got != "item-A" {
		t.Fatalf("A Item = %q, %v", got, err)
	}
	genA := s.genFor("k")
	if genA == "" {
		t.Fatal("导入写入 content_hash 后，代号不应为空")
	}

	// 重导同 deck，内容指纹变化。
	importDeck("hash-B", "item-B")
	genB := s.genFor("k")
	if genB == genA {
		t.Fatalf("重导后代号应变化，got 前后均为 %q", genA)
	}

	// 旧代号键仍留在缓存里（未并发 Clear），但已不可达：读取必须返回新内容而非旧内容。
	if got, err := s.Item("k", "g1", 0); err != nil || got != "item-B" {
		t.Fatalf("B Item = %q, %v（旧代号键仍命中？读到旧内容）", got, err)
	}
}
