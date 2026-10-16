//nolint:testpackage
package dice

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"

	"sealdice-core/utils/deckstore"
)

const drawTestSeed = uint64(0x5eed)

// drawFixtureJSON 生成抽取测试用 JSON 牌堆：
//   - 均匀：600 条（跨 3 块），验证单块读取 + 覆盖面。
//   - 加权：::1::×3 + ::99::，总权重 102。
//   - 五条 / 双：不放回 / 放回语义与跨块无关的小组。
//   - 引用 + _helper：验证私有组递归解析（_helper 不在 Command）。
func drawFixtureJSON() string {
	var b strings.Builder
	b.WriteString(`{"_title":["抽取测试牌堆"],"_keys":["均匀","加权","五条","双","引用"],"均匀":[`)
	for i := 0; i < 600; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%q", fmt.Sprintf("item%03d", i))
	}
	b.WriteString(`],"加权":["::1::轻甲","::1::轻乙","::1::轻丙","::99::重"],"五条":["a","b","c","d","e"],"双":["X","Y"],"引用":["引用{_helper}","裸文本"],"_helper":["辅助文本A","辅助文本B"]}`)
	return b.String()
}

// newDrawTestCtx 搭好临时 data/decks 并 reload，返回 Dice 与绑定固定 PCG DiceSource 的 MsgContext。
// legacy=true 时强制文件库与内存 SQLite 双双失败，走全内存 legacy 后端。
func newDrawTestCtx(t *testing.T, files map[string]string, legacy bool) (*Dice, *MsgContext) {
	t.Helper()
	dir := t.TempDir()
	decksDir := filepath.Join(dir, "data", "decks")
	if err := os.MkdirAll(decksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(decksDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("写夹具 %s: %v", name, err)
		}
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	oldPath := deckStorePathFn
	oldMem := newDeckStoreMemoryFn
	if legacy {
		// 指向目录 → 文件库打开失败；内存库强制失败 → legacy。
		deckStorePathFn = func() string { return dir }
		newDeckStoreMemoryFn = func(deckstore.Logger) (deckstore.Store, error) {
			return nil, errors.New("forced memory sqlite failure")
		}
	} else {
		deckStorePathFn = func() string { return filepath.Join(dir, "data", "_deck_cache", "decks.db") }
	}
	t.Cleanup(func() {
		deckStorePathFn = oldPath
		newDeckStoreMemoryFn = oldMem
	})

	d := &Dice{Logger: zap.NewNop().Sugar()}
	DeckReload(d)
	t.Cleanup(func() {
		if d.DeckStore != nil {
			_ = d.DeckStore.Close()
		}
	})

	ctx := &MsgContext{Dice: d}
	ctx.diceRandSrc = newPCGDiceSource(drawTestSeed)
	return d, ctx
}

func TestDrawUniformAndWeightedDistribution(t *testing.T) {
	d, ctx := newDrawTestCtx(t, map[string]string{"draw.json": drawFixtureJSON()}, false)
	di := d.DeckList[0]

	// 均匀组：固定种子放回抽 2000 次，每个结果都合法且覆盖面足够广。
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		got, err := executeDeck(ctx, di, "均匀", false)
		if err != nil {
			t.Fatalf("均匀 draw %d: %v", i, err)
		}
		if !strings.HasPrefix(got, "item") {
			t.Fatalf("均匀 draw %d = %q, want itemXXX", i, got)
		}
		seen[got] = true
	}
	if len(seen) < 500 {
		t.Fatalf("均匀组 2000 次抽取只覆盖 %d 个不同条目，期望 >= 500", len(seen))
	}

	// 加权组：重牌（::99::）占比 ≈ 99/102。
	// 注意 executeDeck 的 deckDepth 仅增不减（现状行为），单 ctx 总抽取需远小于 20000。
	const total = 5000
	heavy := 0
	for i := 0; i < total; i++ {
		got, err := executeDeck(ctx, di, "加权", false)
		if err != nil {
			t.Fatalf("加权 draw %d: %v", i, err)
		}
		if got == "重" {
			heavy++
		}
	}
	ratio := float64(heavy) / float64(total)
	if ratio < 99.0/102.0-0.05 || ratio > 99.0/102.0+0.05 {
		t.Fatalf("重牌占比 = %.4f, 期望 ≈ %.4f (±0.05)", ratio, 99.0/102.0)
	}
}

func TestDrawShuffleNoReplaceExhausts(t *testing.T) {
	d, ctx := newDrawTestCtx(t, map[string]string{"draw.json": drawFixtureJSON()}, false)
	di := d.DeckList[0]

	// 5 条组不放回连抽：前 5 次不重复。
	seen := map[string]bool{}
	for i := 0; i < 5; i++ {
		got, err := executeDeck(ctx, di, "五条", true)
		if err != nil {
			t.Fatalf("不放回 draw %d: %v", i, err)
		}
		if seen[got] {
			t.Fatalf("不放回前 5 次出现重复 %q", got)
		}
		seen[got] = true
	}

	// 第 6 次：池耗尽触发重建，仍返回五条之一。
	got, err := executeDeck(ctx, di, "五条", true)
	if err != nil {
		t.Fatalf("第 6 次 draw: %v", err)
	}
	if !strings.Contains("abcde", got) {
		t.Fatalf("第 6 次 = %q, want ∈ {a,b,c,d,e}", got)
	}
}

func TestDrawDiceSourceDeterministic(t *testing.T) {
	content := drawFixtureJSON()

	drawSeq := func(seed uint64) []string {
		d, ctx := newDrawTestCtx(t, map[string]string{"draw.json": content}, false)
		ctx.diceRandSrc = newPCGDiceSource(seed)
		di := d.DeckList[0]
		var seq []string
		for i := 0; i < 20; i++ {
			got, err := executeDeck(ctx, di, "均匀", false)
			if err != nil {
				t.Fatalf("draw %d: %v", i, err)
			}
			seq = append(seq, got)
		}
		return seq
	}

	seq1 := drawSeq(drawTestSeed)
	seq2 := drawSeq(drawTestSeed)
	if strings.Join(seq1, ",") != strings.Join(seq2, ",") {
		t.Fatalf("相同种子序列不一致:\n %v\n %v", seq1, seq2)
	}

	seq3 := drawSeq(drawTestSeed + 1)
	if strings.Join(seq1, ",") == strings.Join(seq3, ",") {
		t.Fatal("换种子后序列应不同")
	}
}

func TestDrawRecursionAndPrivateGroup(t *testing.T) {
	d, ctx := newDrawTestCtx(t, map[string]string{"draw.json": drawFixtureJSON()}, false)
	di := d.DeckList[0]

	// _helper 是私有组：不在 Command，但在 GroupMeta（存在性判断不得查 Command）。
	if _, ok := di.Command["_helper"]; ok {
		t.Fatal("_helper 不应出现在 Command")
	}
	if _, ok := d.DeckStore.GroupMeta(di.DeckKey, "_helper"); !ok {
		t.Fatal("_helper 应存在于 GroupMeta")
	}

	// {_helper} 可直接解析（私有组递归）。
	text, err := deckStringFormat(ctx, di, "{_helper}")
	if err != nil {
		t.Fatalf("{_helper}: %v", err)
	}
	if text != "辅助文本A" && text != "辅助文本B" {
		t.Fatalf("{_helper} = %q, want 辅助文本A/B", text)
	}

	// 引用组内嵌 {_helper} 能递归解析。
	found := false
	for i := 0; i < 40; i++ {
		got, err := executeDeck(ctx, di, "引用", false)
		if err != nil {
			t.Fatalf("引用 draw %d: %v", i, err)
		}
		if strings.HasPrefix(got, "引用辅助文本") {
			found = true
			break
		}
		if got != "裸文本" {
			t.Fatalf("引用 draw = %q", got)
		}
	}
	if !found {
		t.Fatal("40 次未抽到 引用{_helper}，私有组递归不可达")
	}

	// {%组} 放回 / {$组} 不放回语义。
	// {%双}{%双}：两次放回独立抽取，每字符 ∈ {X,Y}。
	got, err := deckStringFormat(ctx, di, "{%双}{%双}")
	if err != nil {
		t.Fatalf("放回引用解析失败: %v", err)
	}
	if len(got) != 2 || (got[0] != 'X' && got[0] != 'Y') || (got[1] != 'X' && got[1] != 'Y') {
		t.Fatalf("放回引用结果 = %q, want 两字符均 ∈ {X,Y}", got)
	}

	// {$双}{$双}：两次不放回（共享池）必不同。
	got2, err := deckStringFormat(ctx, di, "{$双}{$双}")
	if err != nil {
		t.Fatalf("不放回引用解析失败: %v", err)
	}
	if got2 != "XY" && got2 != "YX" {
		t.Fatalf("不放回引用结果 = %q, want XY 或 YX（不放回必不同）", got2)
	}
}

func TestDrawUnknownGroupTextUnchanged(t *testing.T) {
	d, ctx := newDrawTestCtx(t, map[string]string{"draw.json": drawFixtureJSON()}, false)
	di := d.DeckList[0]

	got, err := deckStringFormat(ctx, di, "{不存在的组}")
	if err != nil {
		t.Fatalf("deckStringFormat: %v", err)
	}
	if !strings.HasPrefix(got, "<%未知牌组-") || !strings.Contains(got, "不存在的组") {
		t.Fatalf("未知组文本 = %q, want 含未知牌组前缀", got)
	}
}

func TestDrawCloudMergeViaHTTP(t *testing.T) {
	t.Run("distinct_merge", func(t *testing.T) {
		var mu sync.Mutex
		secondCalled := false
		srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`["c1","a1"]`))
		}))
		defer srv1.Close()
		srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			secondCalled = true
			mu.Unlock()
			_, _ = w.Write([]byte(`["d1"]`))
		}))
		defer srv2.Close()

		toml := fmt.Sprintf(`
[meta]
title = "云端测试牌堆"
format_version = 1

["云端组"]
export = true
visible = true
cloud_extra = true
distinct = true
options_urls = ["%s"]
options = ["a1", "b1"]
`, srv1.URL)

		d, ctx := newDrawTestCtx(t, map[string]string{"cloud.toml": toml}, false)
		di := d.DeckList[0]

		// distinct=true：合并后 a1 只出现一次；不放回抽空应得 {a1,b1,c1}。
		seen := map[string]bool{}
		for i := 0; i < 3; i++ {
			got, err := executeDeck(ctx, di, "云端组", true)
			if err != nil {
				t.Fatalf("云端组 draw %d: %v", i, err)
			}
			if got != "a1" && got != "b1" && got != "c1" {
				t.Fatalf("云端组 draw = %q, want ∈ {a1,b1,c1}", got)
			}
			if seen[got] {
				t.Fatalf("distinct 合并后重复出现 %q", got)
			}
			seen[got] = true
		}
		if len(seen) != 3 {
			t.Fatalf("云端组抽空只得到 %d 个不同条目 %v, want 3 {a1,b1,c1}", len(seen), seen)
		}
		if secondCalled {
			t.Fatal("首个 URL 成功后不应请求第二个 URL")
		}
	})

	t.Run("first_url_404_discard", func(t *testing.T) {
		var mu sync.Mutex
		secondCalled := false
		srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv1.Close()
		srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			secondCalled = true
			mu.Unlock()
			_, _ = w.Write([]byte(`["c1"]`))
		}))
		defer srv2.Close()

		toml := fmt.Sprintf(`
[meta]
title = "云端测试牌堆"
format_version = 1

["云端组"]
export = true
visible = true
cloud_extra = true
distinct = true
options_urls = ["%s", "%s"]
options = ["a1", "b1"]
`, srv1.URL, srv2.URL)

		d, ctx := newDrawTestCtx(t, map[string]string{"cloud.toml": toml}, false)
		di := d.DeckList[0]

		// 首 URL 404 → 非 200 即弃：不请求第二个 URL，直接用本地 options。
		seen := map[string]bool{}
		for i := 0; i < 5; i++ {
			got, err := executeDeck(ctx, di, "云端组", false)
			if err != nil {
				t.Fatalf("云端组 draw %d: %v", i, err)
			}
			if got != "a1" && got != "b1" {
				t.Fatalf("404 时云端组 draw = %q, want ∈ {a1,b1}", got)
			}
			seen[got] = true
		}
		if secondCalled {
			t.Fatal("首 URL 404 后不应请求第二个 URL")
		}
	})
}

func TestDrawLegacyBackendSameResults(t *testing.T) {
	content := drawFixtureJSON()

	drawAll := func(legacy bool) map[string]bool {
		d, ctx := newDrawTestCtx(t, map[string]string{"draw.json": content}, legacy)
		di := d.DeckList[0]
		if legacy {
			if _, ok := d.DeckStore.(*deckMemoryBackend); !ok {
				t.Fatalf("legacy 后端 = %T, want *deckMemoryBackend", d.DeckStore)
			}
		}
		seen := map[string]bool{}
		for i := 0; i < 5; i++ {
			got, err := executeDeck(ctx, di, "五条", true)
			if err != nil {
				t.Fatalf("legacy=%v draw %d: %v", legacy, i, err)
			}
			seen[got] = true
		}
		return seen
	}

	sqliteSeen := drawAll(false)
	legacySeen := drawAll(true)

	if len(sqliteSeen) != 5 || len(legacySeen) != 5 {
		t.Fatalf("sqlite=%v legacy=%v, want 各 5 个不同条目", sqliteSeen, legacySeen)
	}
	for k := range sqliteSeen {
		if !legacySeen[k] {
			t.Fatalf("legacy 缺失可达文本 %q", k)
		}
	}
}
