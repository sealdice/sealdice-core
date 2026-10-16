package dice

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"sealdice-core/utils/deckstore"
)

// setupReloadFixture 搭好临时 data/decks 目录并拷入夹具，随后把 CWD 切到临时目录。
// 返回临时目录路径；测试结束后（t.Cleanup）自动恢复原 CWD。
func setupReloadFixture(t *testing.T, names ...string) string {
	t.Helper()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	dir := t.TempDir()
	decksDir := filepath.Join(dir, "data", "decks")
	if err := os.MkdirAll(decksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(oldwd, "testdata", "decks", name))
		if err != nil {
			t.Fatalf("读取夹具 %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(decksDir, name), b, 0o644); err != nil {
			t.Fatalf("写夹具 %s: %v", name, err)
		}
	}
	t.Chdir(dir)
	return dir
}

// newReloadTestDice 构造用于 reload 测试的 Dice：deckStorePathFn 指向 dir 下库文件，Logger 静默。
func newReloadTestDice(t *testing.T, dir string) *Dice {
	t.Helper()
	oldPath := deckStorePathFn
	deckStorePathFn = func() string { return filepath.Join(dir, "data", "_deck_cache", "decks.db") }
	t.Cleanup(func() { deckStorePathFn = oldPath })
	d := &Dice{
		Logger:       zap.NewNop().Sugar(),
		StoreManager: &StoreManager{InstalledDecks: map[string]bool{}},
	}
	t.Cleanup(func() {
		if d.DeckStore != nil {
			_ = d.DeckStore.Close()
		}
	})
	return d
}

func TestDeckReloadBuildsListAndIndex(t *testing.T) {
	dir := setupReloadFixture(t, "sample.json", "sample.toml")
	d := newReloadTestDice(t, dir)

	DeckReload(d)

	if len(d.DeckList) != 2 {
		t.Fatalf("DeckList len = %d, want 2", len(d.DeckList))
	}
	byName := map[string]*DeckInfo{}
	for _, di := range d.DeckList {
		byName[di.Name] = di
	}
	json := byName["测试牌堆"]
	if json == nil {
		t.Fatalf("DeckList 缺少 测试牌堆: %+v", d.DeckList)
	}
	if byName["TOML牌堆"] == nil {
		t.Fatalf("DeckList 缺少 TOML牌堆: %+v", d.DeckList)
	}

	// DeckList 元数据正确（DeckItems 应保持 nil，文本已落库）。
	if json.Format != "Dice!" || json.Author != "作者甲" {
		t.Fatalf("测试牌堆 元数据 = %+v", json)
	}
	if json.DeckItems != nil {
		t.Fatalf("测试牌堆 DeckItems = %v, want nil", json.DeckItems)
	}

	// GroupMeta 命中。
	if _, ok := d.DeckStore.GroupMeta(json.DeckKey, "角色"); !ok {
		t.Fatalf("GroupMeta(%s, 角色) 未命中", json.DeckKey)
	}
	if _, ok := d.DeckStore.GroupMeta(json.DeckKey, "道具"); !ok {
		t.Fatalf("GroupMeta(%s, 道具) 未命中", json.DeckKey)
	}

	// deckCommandItemsList 非空。
	if len(d.deckCommandItemsList) == 0 {
		t.Fatal("deckCommandItemsList 为空")
	}
}

func TestDeckReloadIncrementalSkipsUnchanged(t *testing.T) {
	dir := setupReloadFixture(t, "sample.json")
	d := newReloadTestDice(t, dir)

	var count int
	oldHook := deckImportHook
	deckImportHook = func() { count++ }
	defer func() { deckImportHook = oldHook }()

	DeckReload(d)
	if count != 1 {
		t.Fatalf("第一次 reload 导入计数 = %d, want 1", count)
	}

	count = 0
	DeckReload(d)
	if count != 0 {
		t.Fatalf("第二次 reload 导入计数 = %d, want 0（未变文件应跳过解析）", count)
	}

	// 索引仍可用，DeckList 仍完整。
	if len(d.DeckList) != 1 {
		t.Fatalf("DeckList len = %d, want 1", len(d.DeckList))
	}
	di := d.DeckList[0]
	if _, ok := d.DeckStore.GroupMeta(di.DeckKey, "角色"); !ok {
		t.Fatal("第二次 reload 后 GroupMeta 未命中")
	}
	items := d.DeckStore.GroupItems(di.DeckKey, "角色")
	if len(items) != 3 {
		t.Fatalf("角色 组 = %v, want 3 条", items)
	}
}

func TestDeckReloadPicksUpChangesAndDeletes(t *testing.T) {
	dir := setupReloadFixture(t, "sample.json", "sample.toml")
	d := newReloadTestDice(t, dir)

	DeckReload(d)

	var jsonKey, tomlKey string
	for _, di := range d.DeckList {
		switch di.Name {
		case "测试牌堆":
			jsonKey = di.DeckKey
		case "TOML牌堆":
			tomlKey = di.DeckKey
		}
	}
	if jsonKey == "" || tomlKey == "" {
		t.Fatalf("未找到牌堆 key: json=%q toml=%q", jsonKey, tomlKey)
	}

	// 修改 sample.json：追加一个组。
	jsonPath := filepath.Join("data", "decks", "sample.json")
	b, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read sample.json: %v", err)
	}
	b = bytes.TrimRight(b, "\r\n\t ")
	b = bytes.TrimSuffix(b, []byte("}"))
	b = append(b, []byte(",\n  \"新组\": [\"新条目1\", \"新条目2\"]\n}")...)
	if err2 := os.WriteFile(jsonPath, b, 0o644); err2 != nil {
		t.Fatalf("write sample.json: %v", err2)
	}

	DeckReload(d)

	if _, ok := d.DeckStore.GroupMeta(jsonKey, "新组"); !ok {
		t.Fatal("新组 未命中（修改后应重建）")
	}
	if items := d.DeckStore.GroupItems(jsonKey, "新组"); len(items) != 2 || items[0] != "新条目1" || items[1] != "新条目2" {
		t.Fatalf("新组 = %v, want [新条目1 新条目2]", items)
	}

	// 删除 sample.toml。
	if err2 := os.Remove(filepath.Join("data", "decks", "sample.toml")); err2 != nil {
		t.Fatalf("remove sample.toml: %v", err2)
	}

	DeckReload(d)

	st := d.DeckStore.(*deckStoreBackend).Store()
	ids, err := st.DeckIDs()
	if err != nil {
		t.Fatalf("DeckIDs: %v", err)
	}
	for _, id := range ids {
		if id == tomlKey {
			t.Fatalf("DeckIDs 仍含已删除牌堆 %q", tomlKey)
		}
	}
	if _, ok := d.DeckStore.GroupMeta(tomlKey, "普通组"); ok {
		t.Fatal("已删除牌堆的 GroupMeta 应未命中")
	}
	// 未删除的 json 牌堆仍可用（含新增组）。
	if _, ok := d.DeckStore.GroupMeta(jsonKey, "新组"); !ok {
		t.Fatal("新组 在删除清理后应仍存在")
	}
}

func TestDeckReloadNoSourcesDoesNotOpenStore(t *testing.T) {
	dir := setupReloadFixture(t) // 空 data/decks
	d := newReloadTestDice(t, dir)

	DeckReload(d)

	if d.DeckStore != nil {
		t.Fatalf("DeckStore = %T, want nil（无来源不应打开库）", d.DeckStore)
	}
	if len(d.DeckList) != 0 {
		t.Fatalf("DeckList len = %d, want 0", len(d.DeckList))
	}
	if len(d.deckCommandItemsList) != 0 {
		t.Fatalf("deckCommandItemsList len = %d, want 0", len(d.deckCommandItemsList))
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "_deck_cache", "decks.db")); !os.IsNotExist(err) {
		t.Fatalf("不应创建 _deck_cache 库文件，stat err = %v", err)
	}
}

// TestDeckReloadRebuildsIndexFromExistingStore 模拟「进程重启」：先建立库并关闭后端，
// 再用全新 Dice（DeckStore==nil）指向同一目录 reload，文件未变 → 复用行内 DeckInfo 并由
// loadGroupMeta 兜底重建内存索引（新 store 索引为空、无 Commit 预载）。
func TestDeckReloadRebuildsIndexFromExistingStore(t *testing.T) {
	dir := setupReloadFixture(t, "sample.json")

	// 第一轮：建立库。
	d1 := newReloadTestDice(t, dir)
	DeckReload(d1)
	if len(d1.DeckList) != 1 {
		t.Fatalf("第一轮 DeckList len = %d, want 1", len(d1.DeckList))
	}
	key := d1.DeckList[0].DeckKey
	// 模拟进程退出：关闭后端（释放连接与内存索引）。
	if err := d1.DeckStore.Close(); err != nil {
		t.Fatalf("close d1 store: %v", err)
	}

	// 模拟重启：全新 Dice，DeckStore==nil，deckStorePathFn 指向同一目录。
	var count int
	oldHook := deckImportHook
	deckImportHook = func() { count++ }
	defer func() { deckImportHook = oldHook }()

	d2 := newReloadTestDice(t, dir)
	if d2.DeckStore != nil {
		t.Fatal("新 Dice 的 DeckStore 应为 nil（尚未打开）")
	}
	DeckReload(d2)

	if count != 0 {
		t.Fatalf("重启 reload 导入计数 = %d, want 0（文件未变应复用，不走导入）", count)
	}
	if len(d2.DeckList) != 1 {
		t.Fatalf("重启后 DeckList len = %d, want 1", len(d2.DeckList))
	}

	// ① 未变牌堆仍在内存索引中（loadGroupMeta 兜底重建）。
	if _, ok := d2.DeckStore.GroupMeta(key, "角色"); !ok {
		t.Fatal("重启后 GroupMeta(角色) 未命中（loadGroupMeta 兜底未生效）")
	}
	// ② Item 读原文。
	got, err := d2.DeckStore.Item(key, "道具", 0)
	if err != nil || got != "::3::长剑" {
		t.Fatalf("Item(道具, 0) = %q, %v, want ::3::长剑", got, err)
	}
	// ③ 加权组 Weights 可用。
	w, err := d2.DeckStore.Weights(key, "道具")
	if err != nil {
		t.Fatalf("Weights(道具): %v", err)
	}
	if len(w) != 3 || w[0] != 3 || w[1] != 1 || w[2] != 2 {
		t.Fatalf("Weights(道具) = %v, want [3 1 2]", w)
	}
}

func TestDeckReloadLegacyFallbackFillsDeckItems(t *testing.T) {
	dir := setupReloadFixture(t, "sample.json")
	d := newReloadTestDice(t, dir)

	// 文件库（deckStorePathFn 指向目录必然失败）与内存库都失败 → legacy。
	oldPath := deckStorePathFn
	deckStorePathFn = func() string { return dir }
	defer func() { deckStorePathFn = oldPath }()

	oldMem := newDeckStoreMemoryFn
	newDeckStoreMemoryFn = func(deckstore.Logger) (deckstore.Store, error) {
		return nil, errors.New("forced memory sqlite failure")
	}
	defer func() { newDeckStoreMemoryFn = oldMem }()

	DeckReload(d)

	if _, ok := d.DeckStore.(*deckMemoryBackend); !ok {
		t.Fatalf("DeckStore = %T, want *deckMemoryBackend", d.DeckStore)
	}
	if len(d.DeckList) != 1 {
		t.Fatalf("DeckList len = %d, want 1", len(d.DeckList))
	}
	di := d.DeckList[0]
	if len(di.DeckItems) == 0 {
		t.Fatal("DeckItems 未填充")
	}

	// 抽取可用（legacy 后端）。
	if _, ok := d.DeckStore.GroupMeta(di.DeckKey, "角色"); !ok {
		t.Fatal("legacy GroupMeta 未命中")
	}
	item, err := d.DeckStore.Item(di.DeckKey, "角色", 0)
	if err != nil || item != "骑士" {
		t.Fatalf("Item(角色, 0) = %q, %v, want 骑士", item, err)
	}
}
