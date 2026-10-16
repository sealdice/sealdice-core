package dice

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	ds "github.com/sealdice/dicescript"

	"sealdice-core/utils/deckstore"
)

// seqDiceSource 依次返回预设 uint64 序列，供确定性测试。
type seqDiceSource struct {
	vals []uint64
	i    int
}

func (s *seqDiceSource) Uint64() uint64 {
	if len(s.vals) == 0 {
		return 0
	}
	v := s.vals[s.i%len(s.vals)]
	s.i++
	return v
}

func TestOpenDeckBackendFileOK(t *testing.T) {
	oldPath := deckStorePathFn
	oldMem := newDeckStoreMemoryFn
	deckStorePathFn = func() string { return filepath.Join(t.TempDir(), "decks.db") }
	defer func() {
		deckStorePathFn = oldPath
		newDeckStoreMemoryFn = oldMem
	}()

	b, mode := openDeckBackend(nil)
	if mode != "file" {
		t.Fatalf("mode = %q, want %q", mode, "file")
	}
	sb, ok := b.(*deckStoreBackend)
	if !ok {
		t.Fatalf("backend = %T, want *deckStoreBackend", b)
	}
	if sb.Store() == nil {
		t.Fatal("Store() = nil")
	}
	defer func() { _ = b.Close() }()
}

func TestOpenDeckBackendFallsBackToMemorySQL(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root 可无视目录权限，跳过只读目录测试")
	}
	ro := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(ro, 0o500); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	defer func() { _ = os.Chmod(ro, 0o700) }()

	oldPath := deckStorePathFn
	oldMem := newDeckStoreMemoryFn
	deckStorePathFn = func() string { return filepath.Join(ro, "decks.db") }
	defer func() {
		deckStorePathFn = oldPath
		newDeckStoreMemoryFn = oldMem
	}()

	b, mode := openDeckBackend(nil)
	if mode != "memory-sqlite" {
		t.Fatalf("mode = %q, want %q", mode, "memory-sqlite")
	}
	sb, ok := b.(*deckStoreBackend)
	if !ok {
		t.Fatalf("backend = %T, want *deckStoreBackend", b)
	}
	defer func() { _ = b.Close() }()

	// 写后立即可读，钉住 :memory: 降级路径读写同库。
	st := sb.Store()
	im, err := st.BeginImport("k")
	if err != nil {
		t.Fatalf("BeginImport: %v", err)
	}
	g := im.BeginGroup("g")
	g.Add("hello", 0)
	if _, err2 := g.Finish(); err2 != nil {
		t.Fatalf("Finish: %v", err2)
	}
	if err2 := im.Commit(); err2 != nil {
		t.Fatalf("Commit: %v", err2)
	}
	got, err := st.Item("k", "g", 0)
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if got != "hello" {
		t.Fatalf("Item = %q, want %q", got, "hello")
	}
}

func TestOpenDeckBackendFallsBackToLegacy(t *testing.T) {
	oldPath := deckStorePathFn
	oldMem := newDeckStoreMemoryFn
	// 文件库路径指向目录（Open 必然失败，与 root 无关），内存库强制失败。
	deckStorePathFn = func() string { return t.TempDir() }
	newDeckStoreMemoryFn = func(deckstore.Logger) (deckstore.Store, error) {
		return nil, errors.New("forced memory sqlite failure")
	}
	defer func() {
		deckStorePathFn = oldPath
		newDeckStoreMemoryFn = oldMem
	}()

	b, mode := openDeckBackend(nil)
	if mode != "legacy" {
		t.Fatalf("mode = %q, want %q", mode, "legacy")
	}
	if _, ok := b.(*deckMemoryBackend); !ok {
		t.Fatalf("backend = %T, want *deckMemoryBackend", b)
	}
	_ = b.Close()
}

func TestDeckMemoryBackendDelegates(t *testing.T) {
	b := &deckMemoryBackend{}
	items := []string{"a", "b"}
	b.Register(&DeckInfo{
		DeckKey:   "k",
		DeckItems: map[string][]string{"g": items},
	})

	gm, ok := b.GroupMeta("k", "g")
	if !ok {
		t.Fatal("GroupMeta(k, g) 未命中")
	}
	if gm.ItemCount != 2 {
		t.Fatalf("ItemCount = %d, want 2", gm.ItemCount)
	}

	got, err := b.Item("k", "g", 1)
	if err != nil {
		t.Fatalf("Item(k, g, 1): %v", err)
	}
	if got != "b" {
		t.Fatalf("Item(k, g, 1) = %q, want %q", got, "b")
	}

	w, err := b.Weights("k", "g")
	if err != nil {
		t.Fatalf("Weights(k, g): %v", err)
	}
	if w != nil {
		t.Fatalf("Weights = %v, want nil（全默认权重）", w)
	}

	gi := b.GroupItems("k", "g")
	if len(gi) != len(items) {
		t.Fatalf("GroupItems len = %d, want %d", len(gi), len(items))
	}
	for i := range items {
		if gi[i] != items[i] {
			t.Fatalf("GroupItems[%d] = %q, want %q", i, gi[i], items[i])
		}
	}
	if &gi[0] != &items[0] {
		t.Fatal("GroupItems 应返回 DeckItems[group] 同一 slice")
	}
}

func TestDeckMemoryBackendWeightedAndMissing(t *testing.T) {
	b := &deckMemoryBackend{}
	b.Register(&DeckInfo{
		DeckKey:   "k",
		DeckItems: map[string][]string{"w": {"::3::x", "y"}},
	})

	gm, ok := b.GroupMeta("k", "w")
	if !ok {
		t.Fatal("GroupMeta(k, w) 未命中")
	}
	if gm.ItemCount != 2 || !gm.Weighted || gm.TotalWeight != 4 {
		t.Fatalf("GroupMeta = %+v, want ItemCount=2 Weighted=true TotalWeight=4", gm)
	}

	w, err := b.Weights("k", "w")
	if err != nil {
		t.Fatalf("Weights(k, w): %v", err)
	}
	if len(w) != 2 || w[0] != 3 || w[1] != 1 {
		t.Fatalf("Weights = %v, want [3 1]", w)
	}

	// 未命中：deck / group / 越界 idx 均返回 ErrNotFound。
	if _, ok := b.GroupMeta("missing", "w"); ok {
		t.Fatal("GroupMeta(missing, w) 应未命中")
	}
	if _, err := b.Item("k", "w", 2); !errors.Is(err, deckstore.ErrNotFound) {
		t.Fatalf("Item(k, w, 2) err = %v, want ErrNotFound", err)
	}
	if _, err := b.Weights("k", "missing"); !errors.Is(err, deckstore.ErrNotFound) {
		t.Fatalf("Weights(k, missing) err = %v, want ErrNotFound", err)
	}
}

func TestDiceSourceRandAdapter(t *testing.T) {
	seq := []uint64{0, 1, 0xffffffffffffffff, 42, 7, 123456789, 1<<63 | 1, 0xdeadbeef}
	makeSrc := func() *seqDiceSource { return &seqDiceSource{vals: seq} }

	for _, n := range []int64{1, 2, 3, 16, 17, 100, 1000000, 1 << 40} {
		got := diceSourceRand{src: makeSrc()}.Int63n(n)
		want := int64(randIntnFromSource(makeSrc(), int(n)))
		if got != want {
			t.Fatalf("Int63n(%d) = %d, want %d", n, got, want)
		}
		if got < 0 || got >= n {
			t.Fatalf("Int63n(%d) = %d 越界 [0,%d)", n, got, n)
		}
	}
}

var _ ds.DiceSource = (*seqDiceSource)(nil)
