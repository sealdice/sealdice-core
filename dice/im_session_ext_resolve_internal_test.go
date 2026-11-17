package dice

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"sealdice-core/logger"
)

// TestGroupInfoUnmarshalResolvesExtPointerFromRegistry 验证反序列化时
// activatedExtList 中的扩展名被解析为全局共享的 *ExtInfo 指针，
// 而不是为每个群分配新的占位 ExtInfo 对象。
func TestGroupInfoUnmarshalResolvesExtPointerFromRegistry(t *testing.T) {
	sharedCoc := &ExtInfo{Name: "coc7", Version: "9.9.9"}
	prev := extJSONResolver
	extJSONResolver = func(name string) *ExtInfo {
		if name == "coc7" {
			return sharedCoc
		}
		return nil
	}
	defer func() { extJSONResolver = prev }()

	data := []byte(`{"groupId":"QQ-Group:1","activatedExtList":[{"name":"coc7","aliases":["c"],"version":"1.0"},{"name":"deleted_ext","version":"1.0"}]}`)
	var g GroupInfo
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}

	raw := g.GetActivatedExtListRaw()
	if len(raw) != 2 {
		t.Fatalf("expect 2 exts, got %d", len(raw))
	}
	if raw[0] != sharedCoc {
		t.Errorf("已注册扩展应解析为全局共享指针: shared=%p got=%p", sharedCoc, raw[0])
	}
	if raw[1] == nil || raw[1].Name != "deleted_ext" {
		t.Errorf("未注册扩展应保留名字占位, got %+v", raw[1])
	}
}

// TestGroupInfoUnmarshalWithoutResolverKeepsPlaceholders 验证未设置解析器时
// （如独立调用方）退回占位对象行为，且占位对象是轻量的。
func TestGroupInfoUnmarshalWithoutResolverKeepsPlaceholders(t *testing.T) {
	prev := extJSONResolver
	extJSONResolver = nil
	defer func() { extJSONResolver = prev }()

	data := []byte(`{"groupId":"QQ-Group:2","activatedExtList":[{"name":"coc7","version":"1.0"}]}`)
	var g GroupInfo
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}

	raw := g.GetActivatedExtListRaw()
	if len(raw) != 1 {
		t.Fatalf("expect 1 ext, got %d", len(raw))
	}
	if raw[0].Name != "coc7" {
		t.Errorf("占位对象应保留扩展名, got %q", raw[0].Name)
	}
}

// TestGroupInfoUnmarshalSkipsEmptyNameEntries 验证 null/缺名字的数组项被跳过
// （下游 GetActivatedExtList 与 MarshalJSON 本就会将其过滤）。
func TestGroupInfoUnmarshalSkipsEmptyNameEntries(t *testing.T) {
	prev := extJSONResolver
	extJSONResolver = nil
	defer func() { extJSONResolver = prev }()

	data := []byte(`{"groupId":"QQ-Group:3","activatedExtList":[null,{"version":"1.0"},{"name":"core"}]}`)
	var g GroupInfo
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}

	raw := g.GetActivatedExtListRaw()
	if len(raw) != 1 || raw[0].Name != "core" {
		t.Fatalf("空名条目应被跳过, got %v", raw)
	}
}

// TestGroupInfoMarshalKeepsExtEntryFormat 验证存储格式不变：
// 落盘时 activatedExtList 仍是 {name, aliases, version} 对象数组。
func TestGroupInfoMarshalKeepsExtEntryFormat(t *testing.T) {
	g := &GroupInfo{}
	g.SetActivatedExtList([]*ExtInfo{{Name: "coc7", Aliases: []string{"c"}, Version: "1.0"}}, nil)

	out, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	want := `"activatedExtList":[{"name":"coc7","aliases":["c"],"version":"1.0"}]`
	if !strings.Contains(string(out), want) {
		t.Errorf("落盘格式应保持不变\nwant contains: %s\ngot: %s", want, string(out))
	}
}

// BenchmarkGroupInfoUnmarshalSharedExt 内存收益基准：
// 5000 群 × 100 扩展场景下，命中解析器时群仅持有共享指针。
// 对照组为旧行为（解码为独立占位对象）的量级：约 5000×100×416B ≈ 208MB。
func BenchmarkGroupInfoUnmarshalSharedExt(b *testing.B) {
	shared := make([]*ExtInfo, 100)
	for i := range shared {
		shared[i] = &ExtInfo{Name: fmt.Sprintf("ext%d", i)}
	}
	prev := extJSONResolver
	extJSONResolver = func(name string) *ExtInfo {
		for _, e := range shared {
			if e.Name == name {
				return e
			}
		}
		return nil
	}
	defer func() { extJSONResolver = prev }()

	entries := make([]string, 0, 100)
	for _, e := range shared {
		entries = append(entries, `{"name":"`+e.Name+`"}`)
	}
	data := []byte(`{"groupId":"QQ-Group:1","activatedExtList":[` + strings.Join(entries, ",") + `]}`)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	b.ReportAllocs()
	b.ResetTimer()
	groups := make([]*GroupInfo, 0, 5000)
	for range 5000 {
		var g GroupInfo
		if err := json.Unmarshal(data, &g); err != nil {
			b.Fatal(err)
		}
		groups = append(groups, &g)
	}
	b.StopTimer()

	runtime.ReadMemStats(&after)
	runtime.GC()
	var end runtime.MemStats
	runtime.ReadMemStats(&end)
	b.ReportMetric(float64(after.TotalAlloc-before.TotalAlloc)/(1<<20), "alloc-MB-total")
	b.ReportMetric(float64(end.HeapAlloc-before.HeapAlloc)/(1<<20), "retained-MB")
	runtime.KeepAlive(groups)
}

// TestUnmarshalInternsPlaceholdersAcrossGroups 验证未注册扩展名的占位对象
// 在全局共享：不同群反序列化同一名字得到同一指针（共享池），
// 避免已删除插件在大量群中复活逐群分配的内存问题。
func TestUnmarshalInternsPlaceholdersAcrossGroups(t *testing.T) {
	prev := extJSONResolver
	extJSONResolver = nil
	defer func() { extJSONResolver = prev }()

	unmarshal := func() *GroupInfo {
		data := []byte(`{"groupId":"QQ-Group:1","activatedExtList":[{"name":"deleted_plugin"}]}`)
		var g GroupInfo
		if err := json.Unmarshal(data, &g); err != nil {
			t.Fatal(err)
		}
		return &g
	}

	g1, g2 := unmarshal(), unmarshal()
	p1 := g1.GetActivatedExtListRaw()[0]
	p2 := g2.GetActivatedExtListRaw()[0]
	if p1 != p2 {
		t.Errorf("同名未注册扩展应共享同一占位指针: %p vs %p", p1, p2)
	}
	if p1.Name != "deleted_plugin" {
		t.Errorf("占位对象应保留名字, got %q", p1.Name)
	}
}

// TestGetActivatedExtListKeepsUninstalledExtEntries 验证群消息触发延迟初始化时，
// 未安装（已删除）扩展的条目不被丢弃——这是删除后重装能恢复开启状态的前提。
func TestGetActivatedExtListKeepsUninstalledExtEntries(t *testing.T) {
	d := &Dice{Logger: logger.M(), ExtList: []*ExtInfo{{Name: "core"}}}
	group := &GroupInfo{ExtAppliedTime: 0}
	group.SetActivatedExtList([]*ExtInfo{
		{Name: "core"},
		{Name: "deleted_plugin"},
	}, nil)
	atomic.StoreInt64(&group.ExtAppliedTime, 0) // 强制走延迟初始化分支

	got := group.GetActivatedExtList(d)
	if len(got) != 2 {
		t.Fatalf("未安装扩展条目应保留, want 2, got %d: %v", len(got), got)
	}
	if got[1].Name != "deleted_plugin" {
		t.Errorf("got[1] 应为 deleted_plugin, got %q", got[1].Name)
	}
}

// TestReinstallRestoresGroupActivationState 端到端：删除的插件条目保留为占位，
// 重装（同名扩展回到 ExtList）后自动解析为活跃扩展，状态恢复。
func TestReinstallRestoresGroupActivationState(t *testing.T) {
	core := &ExtInfo{Name: "core"}
	d := &Dice{Logger: logger.M(), ExtList: []*ExtInfo{core}}
	group := &GroupInfo{}
	group.SetActivatedExtList([]*ExtInfo{
		core,
		{Name: "plugin_x"},
	}, nil)
	atomic.StoreInt64(&group.ExtAppliedTime, 0) // 强制走延迟初始化分支

	// 模拟运行期间：延迟初始化后 plugin_x 处于未安装状态但条目保留
	list := group.GetActivatedExtList(d)
	found := false
	for _, e := range list {
		if e.Name == "plugin_x" {
			found = true
			if e == core {
				t.Fatal("不应误解析为其他扩展")
			}
		}
	}
	if !found {
		t.Fatal("未安装条目应保留")
	}

	// 重装：plugin_x 回到 ExtList，再次触发解析
	pluginX := &ExtInfo{Name: "plugin_x"}
	d.ExtList = append(d.ExtList, pluginX)
	atomic.StoreInt64(&group.ExtAppliedTime, 0) // 重置以重新初始化
	group.extInitMu.Lock()
	tombstone := group.activatedExtList[len(group.activatedExtList)-1]
	group.extInitMu.Unlock()

	list = group.GetActivatedExtList(d)
	resolved := false
	for _, e := range list {
		if e.Name == "plugin_x" {
			resolved = true
			if e != pluginX {
				t.Errorf("重装后应解析为新扩展指针: want %p got %p", pluginX, e)
			}
			if e == tombstone {
				t.Error("不应仍是占位对象")
			}
		}
	}
	if !resolved {
		t.Fatal("重装后条目应恢复为活跃扩展")
	}
}

// TestMarshalPersistsDeletedWrapperName 验证群持有已标记删除的 wrapper 时，
// 落盘仍保留其名字（状态保留），而不是被过滤丢弃。
func TestMarshalPersistsDeletedWrapperName(t *testing.T) {
	g := &GroupInfo{}
	g.SetActivatedExtList([]*ExtInfo{
		{Name: "core"},
		{Name: "js_gone", IsWrapper: true, IsDeleted: true},
	}, nil)

	out, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"name":"js_gone"`) {
		t.Errorf("已删除 wrapper 的名字应保留在落盘数据中, got: %s", string(out))
	}
}

// TestSyncWrapperStatusKeepsDeletedWrapper 验证消息到达触发的同步
// 不再移除已删除的 wrapper（状态保留，等待重装）。
func TestSyncWrapperStatusKeepsDeletedWrapper(t *testing.T) {
	d := &Dice{Logger: logger.M(), ExtList: []*ExtInfo{{Name: "core"}}}
	group := &GroupInfo{}

	group.SetActivatedExtList([]*ExtInfo{
		{Name: "core"},
		{Name: "js_gone", IsWrapper: true, IsDeleted: true},
	}, nil)
	_ = group.GetActivatedExtList(d)
	atomic.StoreInt64(&d.ExtUpdateTime, 999999)
	atomic.StoreInt64(&group.ExtAppliedTime, 1)

	changed := group.SyncWrapperStatus(d)
	raw := group.GetActivatedExtListRaw()
	if len(raw) != 2 {
		t.Fatalf("已删除 wrapper 应保留, want 2, got %d", len(raw))
	}
	if raw[1].Name != "js_gone" {
		t.Errorf("got[1] 应为 js_gone, got %q", raw[1].Name)
	}
	if changed {
		t.Error("无移除操作时不应报告变更")
	}
}

// TestSyncExtensionsOnMessageResolvesReinstalledPlaceholder 复刻生产缺口：
// 跨重启删除（加载时条目为共享占位）后运行中重装插件，下一条群消息
// 触发 SyncExtensionsOnMessage 时，占位指针应被替换为真实扩展指针。
func TestSyncExtensionsOnMessageResolvesReinstalledPlaceholder(t *testing.T) {
	core := &ExtInfo{Name: "core"}
	d := &Dice{Logger: logger.M(), ExtList: []*ExtInfo{core}}
	group := &GroupInfo{}
	group.SetActivatedExtList([]*ExtInfo{
		core,
		extNamePlaceholder("plugin_x"), // 加载时未注册 → 共享占位
	}, nil)

	// 群完成首次初始化（消息路径中 SyncWrapperStatus 已跑过）
	_ = group.GetActivatedExtList(d)

	// 运行中重装：新 wrapper 进入注册表，版本号递增
	pluginX := &ExtInfo{Name: "plugin_x"}
	d.ExtList = append(d.ExtList, pluginX)
	d.ExtRegistryVersion++

	// 下一条消息
	group.SyncExtensionsOnMessage(d)

	raw := group.GetActivatedExtListRaw()
	var got *ExtInfo
	for _, e := range raw {
		if e.Name == "plugin_x" {
			got = e
		}
	}
	if got == nil {
		t.Fatal("重装后条目应保留在激活列表中")
	}
	if got != pluginX {
		t.Errorf("占位应被替换为真实扩展指针: want %p got %p", pluginX, got)
	}
}
