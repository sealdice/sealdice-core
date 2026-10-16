package dice

import (
	"bytes"
	"crypto/sha1"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.uber.org/zap"

	"sealdice-core/utils/deckstore"
)

const deckFixtureDir = "testdata/decks"

func newImportTestDice() *Dice {
	return &Dice{Logger: zap.NewNop().Sugar()}
}

func openImportTestStore(t *testing.T) deckstore.Store {
	t.Helper()
	st, err := deckstore.OpenMemory(nil)
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func readDeckFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(deckFixtureDir, name))
	if err != nil {
		t.Fatalf("读取夹具 %s: %v", name, err)
	}
	return b
}

// assertDeckMetaEqual 比较两个 DeckInfo 的"元数据"字段（不含 Filename/PackageID/DeckKey/DeckItems/Command）。
func assertDeckMetaEqual(t *testing.T, got, want *DeckInfo) {
	t.Helper()
	type meta struct {
		Enable, Cloud          bool
		Name, Author, Version  string
		License, Date          string
		UpdateDate, Desc       string
		Format, FileFormat     string
		ErrText, StoreID, Etag string
		FormatVersion          int64
		Info, UpdateUrls       []string
	}
	pack := func(di *DeckInfo) meta {
		return meta{
			Enable:        di.Enable,
			Cloud:         di.Cloud,
			Name:          di.Name,
			Author:        di.Author,
			Version:       di.Version,
			License:       di.License,
			Date:          di.Date,
			UpdateDate:    di.UpdateDate,
			Desc:          di.Desc,
			Format:        di.Format,
			FileFormat:    di.FileFormat,
			ErrText:       di.ErrText,
			StoreID:       di.StoreID,
			Etag:          di.Etag,
			FormatVersion: di.FormatVersion,
			Info:          di.Info,
			UpdateUrls:    di.UpdateUrls,
		}
	}
	g, w := pack(got), pack(want)
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("元数据不一致:\n got=%+v\nwant=%+v", g, w)
	}
}

// cloudInfosEqual 比较云牌组信息（nil 与空 map 视为相等）。
func cloudInfosEqual(a, b map[string]*CloudDeckItemInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if va.Distinct != vb.Distinct || !reflect.DeepEqual(va.OptionsUrls, vb.OptionsUrls) {
			return false
		}
	}
	return true
}

func TestDeckImportDiceE(t *testing.T) {
	d := newImportTestDice()
	st := openImportTestStore(t)

	t.Run("json", func(t *testing.T) {
		content := readDeckFixture(t, "sample.json")
		di, err := deckImportSource(d, st, deckSource{Filename: "sample.json"}, content)
		if err != nil {
			t.Fatalf("deckImportSource: %v", err)
		}
		if !di.Enable {
			t.Fatalf("Enable = false, ErrText = %q", di.ErrText)
		}
		if di.DeckItems != nil {
			t.Fatalf("DeckItems = %v, want nil（文本已落库）", di.DeckItems)
		}
		if di.Name != "测试牌堆" {
			t.Fatalf("Name = %q, want 测试牌堆", di.Name)
		}
		if di.Author != "作者甲" {
			t.Fatalf("Author = %q", di.Author)
		}
		if di.Desc != "这是第一行\n这是第二行" {
			t.Fatalf("Desc = %q", di.Desc)
		}
		if di.Format != "Dice!" || di.FileFormat != "json" || di.FormatVersion != 1 {
			t.Fatalf("Format/FileFormat/FormatVersion = %q/%q/%d", di.Format, di.FileFormat, di.FormatVersion)
		}
		if di.StoreID != "store-123" || di.Etag != "etag-abc" {
			t.Fatalf("StoreID/Etag = %q/%q", di.StoreID, di.Etag)
		}
		if !reflect.DeepEqual(di.UpdateUrls, []string{"https://example.com/deck.json"}) {
			t.Fatalf("UpdateUrls = %v", di.UpdateUrls)
		}

		// Command 可见性：_keys 与 _exports 命中项为 true；其余非下划线组无条目。
		if !di.Command["角色"] || !di.Command["道具"] || !di.Command["特殊"] {
			t.Fatalf("Command 缺失可见项: %v", di.Command)
		}
		if _, ok := di.Command["隐藏组"]; ok {
			t.Fatalf("隐藏组 不应出现在 Command 中: %v", di.Command)
		}
		if _, ok := di.Command["_helper"]; ok {
			t.Fatalf("_helper 不应出现在 Command 中: %v", di.Command)
		}

		// _helper 私有组已写入库。
		gm, ok := st.GroupMeta(di.DeckKey, "_helper")
		if !ok || gm.ItemCount != 2 {
			t.Fatalf("_helper 组: ok=%v meta=%+v", ok, gm)
		}

		// 权重组：原文保留 ::N:: 前缀，权重另存。
		gm, ok = st.GroupMeta(di.DeckKey, "道具")
		if !ok {
			t.Fatal("道具 组未命中")
		}
		if gm.ItemCount != 3 || !gm.Weighted || gm.TotalWeight != 6 {
			t.Fatalf("道具 组元数据 = %+v, want ItemCount=3 Weighted=true TotalWeight=6", gm)
		}
		items := st.GroupItems(di.DeckKey, "道具")
		if !reflect.DeepEqual(items, []string{"::3::长剑", "盾牌", "::2::药水"}) {
			t.Fatalf("道具 组原文 = %v", items)
		}
	})

	t.Run("jsonc", func(t *testing.T) {
		content := readDeckFixture(t, "sample.jsonc")
		di, err := deckImportSource(d, st, deckSource{Filename: "sample.jsonc"}, content)
		if err != nil {
			t.Fatalf("deckImportSource(jsonc): %v", err)
		}
		if di.FileFormat != "jsonc" {
			t.Fatalf("FileFormat = %q, want jsonc", di.FileFormat)
		}
		if di.Name != "JSONC牌堆" {
			t.Fatalf("Name = %q, want JSONC牌堆", di.Name)
		}
		if !di.Command["普通组"] {
			t.Fatalf("Command[普通组] = %v, want true", di.Command["普通组"])
		}
		items := st.GroupItems(di.DeckKey, "普通组")
		if !reflect.DeepEqual(items, []string{"选项1", "选项2"}) {
			t.Fatalf("普通组 = %v, want [选项1 选项2]", items)
		}
	})
}

func TestDeckImportSinaNya(t *testing.T) {
	d := newImportTestDice()
	st := openImportTestStore(t)
	content := readDeckFixture(t, "sample.yaml")
	di, err := deckImportSource(d, st, deckSource{Filename: "sample.yaml"}, content)
	if err != nil {
		t.Fatalf("deckImportSource(yaml): %v", err)
	}
	if !di.Enable {
		t.Fatalf("Enable = false, ErrText = %q", di.ErrText)
	}
	if di.Name != "YAML牌堆" || di.Format != "SinaNya" || di.FileFormat != "yaml" {
		t.Fatalf("Name/Format/FileFormat = %q/%q/%q", di.Name, di.Format, di.FileFormat)
	}
	if !di.Command["抽卡"] {
		t.Fatalf("Command[抽卡] = %v, want true", di.Command["抽卡"])
	}
	if !reflect.DeepEqual(di.Info, []string{"信息1", "信息2"}) {
		t.Fatalf("Info = %v", di.Info)
	}
	items := st.GroupItems(di.DeckKey, "抽卡")
	if !reflect.DeepEqual(items, []string{"结果A", "结果B"}) {
		t.Fatalf("抽卡 组 = %v", items)
	}
}

func TestDeckImportSeal(t *testing.T) {
	d := newImportTestDice()
	st := openImportTestStore(t)
	content := readDeckFixture(t, "sample.toml")
	di, err := deckImportSource(d, st, deckSource{Filename: "sample.toml"}, content)
	if err != nil {
		t.Fatalf("deckImportSource(toml): %v", err)
	}
	if !di.Enable {
		t.Fatalf("Enable = false, ErrText = %q", di.ErrText)
	}
	if di.Name != "TOML牌堆" || di.Format != "Seal" || di.FileFormat != "toml" {
		t.Fatalf("Name/Format/FileFormat = %q/%q/%q", di.Name, di.Format, di.FileFormat)
	}

	// 别名是"单条目组 {原名}"。
	alias := st.GroupItems(di.DeckKey, "别名甲")
	if !reflect.DeepEqual(alias, []string{"{复杂组}"}) {
		t.Fatalf("别名甲 = %v, want [\"{复杂组}\"]", alias)
	}
	alias = st.GroupItems(di.DeckKey, "别名乙")
	if !reflect.DeepEqual(alias, []string{"{复杂组}"}) {
		t.Fatalf("别名乙 = %v, want [\"{复杂组}\"]", alias)
	}

	// visible/export 组合。
	if di.Command["普通组"] != true {
		t.Fatalf("Command[普通组] = %v, want true", di.Command["普通组"])
	}
	if di.Command["_私有组"] != false {
		t.Fatalf("Command[_私有组] = %v, want false", di.Command["_私有组"])
	}
	if di.Command["复杂组"] != false {
		t.Fatalf("Command[复杂组] = %v, want false（export+visible=false）", di.Command["复杂组"])
	}
	if di.Command["别名甲"] != false {
		t.Fatalf("Command[别名甲] = %v, want false", di.Command["别名甲"])
	}
	if di.Command["云端组"] != true {
		t.Fatalf("Command[云端组] = %v, want true", di.Command["云端组"])
	}

	// Cloud 元数据保留在 DeckInfo（不联网）。
	if !di.Cloud {
		t.Fatal("Cloud = false, want true")
	}
	info := di.CloudDeckItemInfos["云端组"]
	if info == nil || !info.Distinct || !reflect.DeepEqual(info.OptionsUrls, []string{"https://example.com/cloud.json"}) {
		t.Fatalf("CloudDeckItemInfos[云端组] = %+v", info)
	}
}

func TestDeckImportMatchesLegacyParse(t *testing.T) {
	for _, name := range []string{"sample.json", "sample.jsonc", "sample.yaml", "sample.toml"} {
		t.Run(name, func(t *testing.T) {
			d := newImportTestDice()
			st := openImportTestStore(t)
			content := readDeckFixture(t, name)

			di, err := deckImportSource(d, st, deckSource{Filename: name}, content)
			if err != nil {
				t.Fatalf("deckImportSource: %v", err)
			}

			legacy := &DeckInfo{
				Filename:           name,
				DeckKey:            di.DeckKey,
				DeckItems:          map[string][]string{},
				Command:            map[string]bool{},
				CloudDeckItemInfos: map[string]*CloudDeckItemInfo{},
			}
			if !parseDeck(d, name, content, legacy) {
				t.Fatalf("parseDeck 失败: %s", legacy.ErrText)
			}

			assertDeckMetaEqual(t, di, legacy)
			if !reflect.DeepEqual(di.Command, legacy.Command) {
				t.Fatalf("Command 不一致:\n new=%v\nlegacy=%v", di.Command, legacy.Command)
			}
			if !cloudInfosEqual(di.CloudDeckItemInfos, legacy.CloudDeckItemInfos) {
				t.Fatalf("CloudDeckItemInfos 不一致:\n new=%v\nlegacy=%v", di.CloudDeckItemInfos, legacy.CloudDeckItemInfos)
			}

			for group, wantItems := range legacy.DeckItems {
				gm, ok := st.GroupMeta(di.DeckKey, group)
				if !ok {
					t.Fatalf("组 %q 未写入库", group)
				}
				wantMeta := computeDeckGroupMeta(wantItems)
				if gm.ItemCount != wantMeta.ItemCount || gm.TotalWeight != wantMeta.TotalWeight || gm.Weighted != wantMeta.Weighted {
					t.Fatalf("组 %q 元数据不一致: store=%+v legacy=%+v", group, gm, wantMeta)
				}
				gotItems := st.GroupItems(di.DeckKey, group)
				if !reflect.DeepEqual(gotItems, wantItems) {
					t.Fatalf("组 %q 文本不一致:\n store=%v\nlegacy=%v", group, gotItems, wantItems)
				}
			}
		})
	}
}

func TestDeckLoadMetaRowRoundtrip(t *testing.T) {
	for _, name := range []string{"sample.json", "sample.yaml"} {
		t.Run(name, func(t *testing.T) {
			d := newImportTestDice()
			st := openImportTestStore(t)
			content := readDeckFixture(t, name)

			di, err := deckImportSource(d, st, deckSource{Filename: name}, content)
			if err != nil {
				t.Fatalf("deckImportSource: %v", err)
			}

			rec, hash, ok := deckLoadMetaRow(st, di.DeckKey)
			if !ok {
				t.Fatal("deckLoadMetaRow 未命中")
			}
			wantHash := sha1.Sum(content)
			if !bytes.Equal(hash, wantHash[:]) {
				t.Fatalf("content_hash = %x, want %x", hash, wantHash)
			}
			if rec.DeckKey != di.DeckKey {
				t.Fatalf("DeckKey = %q, want %q", rec.DeckKey, di.DeckKey)
			}
			assertDeckMetaEqual(t, di, rec)
			if !reflect.DeepEqual(di.Command, rec.Command) {
				t.Fatalf("Command 不一致:\n import=%v\nroundtrip=%v", di.Command, rec.Command)
			}
		})
	}
}

func TestDeckImportDiceESkipsNonStringElements(t *testing.T) {
	d := newImportTestDice()
	st := openImportTestStore(t)
	content := readDeckFixture(t, "sample.mixed.json")

	di, err := deckImportSource(d, st, deckSource{Filename: "sample.mixed.json"}, content)
	if err != nil {
		t.Fatalf("deckImportSource: %v", err)
	}
	if !di.Enable {
		t.Fatalf("Enable = false, ErrText = %q", di.ErrText)
	}
	// 数组内非字符串元素静默跳过，有序保留字符串。
	if items := st.GroupItems(di.DeckKey, "混合组"); !reflect.DeepEqual(items, []string{"a", "b"}) {
		t.Fatalf("混合组 = %v, want [a b]（跳过非字符串）", items)
	}

	// 与 legacy parseDeck 等价。
	legacy := &DeckInfo{
		Filename:           "sample.mixed.json",
		DeckKey:            di.DeckKey,
		DeckItems:          map[string][]string{},
		Command:            map[string]bool{},
		CloudDeckItemInfos: map[string]*CloudDeckItemInfo{},
	}
	if !parseDeck(d, "sample.mixed.json", content, legacy) {
		t.Fatalf("parseDeck 失败: %s", legacy.ErrText)
	}
	if !reflect.DeepEqual(legacy.DeckItems["混合组"], []string{"a", "b"}) {
		t.Fatalf("legacy 混合组 = %v, want [a b]", legacy.DeckItems["混合组"])
	}
	if !reflect.DeepEqual(di.Command, legacy.Command) {
		t.Fatalf("Command 不一致:\n new=%v\nlegacy=%v", di.Command, legacy.Command)
	}
	gm, ok := st.GroupMeta(di.DeckKey, "混合组")
	if !ok {
		t.Fatal("混合组 未命中")
	}
	wantMeta := computeDeckGroupMeta(legacy.DeckItems["混合组"])
	if gm.ItemCount != wantMeta.ItemCount || gm.TotalWeight != wantMeta.TotalWeight || gm.Weighted != wantMeta.Weighted {
		t.Fatalf("混合组 元数据不一致: store=%+v legacy=%+v", gm, wantMeta)
	}
}

func TestDeckLoadMetaRowCloudRoundtrip(t *testing.T) {
	d := newImportTestDice()
	st := openImportTestStore(t)
	content := readDeckFixture(t, "sample.toml")

	di, err := deckImportSource(d, st, deckSource{Filename: "sample.toml"}, content)
	if err != nil {
		t.Fatalf("deckImportSource: %v", err)
	}
	if !di.Cloud {
		t.Fatal("导入后 Cloud = false, want true")
	}

	rec, _, ok := deckLoadMetaRow(st, di.DeckKey)
	if !ok {
		t.Fatal("deckLoadMetaRow 未命中")
	}
	if !rec.Cloud {
		t.Fatal("恢复后 Cloud = false, want true")
	}
	if !cloudInfosEqual(di.CloudDeckItemInfos, rec.CloudDeckItemInfos) {
		t.Fatalf("CloudDeckItemInfos 不一致:\n import=%+v\nroundtrip=%+v", di.CloudDeckItemInfos, rec.CloudDeckItemInfos)
	}
	info := rec.CloudDeckItemInfos["云端组"]
	if info == nil || !info.Distinct || !reflect.DeepEqual(info.OptionsUrls, []string{"https://example.com/cloud.json"}) {
		t.Fatalf("CloudDeckItemInfos[云端组] = %+v, want Distinct=true OptionsUrls=[https://example.com/cloud.json]", info)
	}

	// 恢复出的 Command 与普通组文本亦一致。
	if !reflect.DeepEqual(di.Command, rec.Command) {
		t.Fatalf("Command 不一致:\n import=%v\nroundtrip=%v", di.Command, rec.Command)
	}
	if items := st.GroupItems(di.DeckKey, "普通组"); !reflect.DeepEqual(items, []string{"甲", "乙"}) {
		t.Fatalf("普通组 文本 = %v, want [甲 乙]", items)
	}
}

func TestDeckImportParseErrorNotStored(t *testing.T) {
	d := newImportTestDice()
	st := openImportTestStore(t)
	src := deckSource{Filename: "broken.json"}
	content := []byte(`{"_title": ["坏"], "组": [未闭合}`)

	di, err := deckImportSource(d, st, src, content)
	if err == nil {
		t.Fatal("期望返回 error")
	}
	if di.Enable {
		t.Fatal("Enable 应为 false")
	}
	if di.ErrText == "" {
		t.Fatal("ErrText 应为空串之外的内容")
	}

	// 库中无该 deck_key 的行。
	if _, _, ok := deckLoadMetaRow(st, di.DeckKey); ok {
		t.Fatal("库中不应有该牌堆的行")
	}
	ids, err := st.DeckIDs()
	if err != nil {
		t.Fatalf("DeckIDs: %v", err)
	}
	for _, id := range ids {
		if id == di.DeckKey {
			t.Fatalf("DeckIDs 不应包含失败牌堆 %q", id)
		}
	}
}
